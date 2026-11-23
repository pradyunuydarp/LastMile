package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	locationpb "lastmile/gen/go/location"
	matchingpb "lastmile/gen/go/matching"
	"lastmile/internal/location"
	"lastmile/internal/matching"
)

// startGRPC serves one service over an in-memory connection and returns a
// client connection to it. Tests exercise the real gRPC contract, not mocks.
func startGRPC(t *testing.T, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	register(srv)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// newTestGateway returns a gateway wired to a real Matching service (and,
// optionally, a real Location service).
func newTestGateway(t *testing.T, withLocation bool) *Gateway {
	t.Helper()
	mconn := startGRPC(t, func(s *grpc.Server) {
		matchingpb.RegisterMatchingServiceServer(s, matching.NewServer(nil, "matching-test-pod"))
	})

	var loc locationpb.LocationServiceClient
	if withLocation {
		lconn := startGRPC(t, func(s *grpc.Server) {
			locationpb.RegisterLocationServiceServer(s, location.NewServer())
		})
		loc = locationpb.NewLocationServiceClient(lconn)
	}

	gw := NewGateway(nil, nil, loc, nil)
	gw.AttachMatching(matchingpb.NewMatchingServiceClient(mconn))
	gw.simTick, gw.simStopPause, gw.simPickupWait = time.Millisecond, time.Millisecond, 200*time.Millisecond
	return gw
}

func ecityDriver(id string, seats int) Driver {
	return Driver{
		ID: id, Name: "Ravi", CarDetails: "Toyota Innova", SeatsAvailable: seats,
		Route:    Route{ID: "route-" + id, TargetStationIDs: []string{"station-ecity"}, Destination: "Wipro Gate"},
		Latitude: 12.8456, Longitude: 77.6603,
	}
}

func postJSON(t *testing.T, handler http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	handler(rr, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(buf)))
	return rr
}

func TestSnapshotHandler(t *testing.T) {
	gw := newTestGateway(t, false)
	gw.drivers = []Driver{ecityDriver("driver-ramesh", 2)}

	rr := httptest.NewRecorder()
	gw.SnapshotHandler(rr, httptest.NewRequest(http.MethodGet, "/aggregates/snapshot", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	var snapshot BackendSnapshot
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &snapshot))
	assert.NotEmpty(t, snapshot.Drivers)
	assert.Len(t, snapshot.Stations, 9)
	assert.Len(t, snapshot.Riders, 3, "three bot riders are seeded for solo demos")
}

func TestMatchHandlerCreatesTrip(t *testing.T) {
	gw := newTestGateway(t, false)
	gw.drivers = []Driver{ecityDriver("driver-ramesh", 2)}

	rr := postJSON(t, gw.MatchHandler, "/matching/match", matchRequest{DriverID: "driver-ramesh", StationID: "station-ecity"})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var trip Trip
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &trip))
	assert.Equal(t, "driver-ramesh", trip.DriverID)
	assert.Equal(t, "bot-rider-meera", trip.RiderID, "the seeded rider waiting at Electronic City for Wipro Gate")

	snapshot := gw.snapshot()
	require.NotNil(t, snapshot.HighlightTrip)
	assert.Equal(t, trip.ID, snapshot.HighlightTrip.ID)
}

func TestDriverRequestsUsesMatchingService(t *testing.T) {
	gw := newTestGateway(t, false)
	pickup := gw.pickupPoints[0] // Electronic City
	gw.drivers = []Driver{ecityDriver("driver-1", 3)}
	gw.riders = []Rider{
		{ID: "rider-ok", Name: "Priya", Destination: "Wipro Gate", ArrivalTime: time.Now().Add(5 * time.Minute),
			StationID: "station-ecity", Status: "waiting", PickupPointID: pickup.ID, Pickup: &pickup},
		{ID: "rider-elsewhere", Name: "Kabir", Destination: "Forum Mall", ArrivalTime: time.Now().Add(5 * time.Minute),
			StationID: "station-ecity", Status: "waiting", PickupPointID: pickup.ID, Pickup: &pickup},
		{ID: "rider-taken", Name: "Anu", Destination: "Wipro Gate", ArrivalTime: time.Now().Add(5 * time.Minute),
			StationID: "station-ecity", Status: "matched", PickupPointID: pickup.ID, Pickup: &pickup},
	}

	rr := httptest.NewRecorder()
	gw.DriverRequestsHandler(rr, httptest.NewRequest(http.MethodGet, "/drivers/requests?driverId=driver-1", nil))
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var payload driverRequestsResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))
	require.Len(t, payload.Requests, 1, "only the waiting rider going to the driver's destination")
	assert.Equal(t, "rider-ok", payload.Requests[0].ID)
}

func TestBookRideHandler(t *testing.T) {
	gw := newTestGateway(t, false)
	pickup := gw.pickupPoints[0]
	gw.drivers = []Driver{ecityDriver("driver-2", 1)}
	gw.riders = nil

	rr := postJSON(t, gw.BookRideHandler, "/rides/book", bookRideRequest{
		Command: "book", Name: "Ananya", PickupPointID: pickup.ID, Destination: "Wipro Gate",
	})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var payload bookRideResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))
	assert.Equal(t, "awaiting_rider", payload.Status, "attempts=%+v rejections=%+v", payload.Attempts, payload.Rejections)
	require.NotNil(t, payload.Trip)
	require.NotEmpty(t, payload.Attempts)
	assert.True(t, payload.Attempts[0].Accepted)
	assert.Equal(t, "matching-test-pod", payload.MatchedBy, "the response says which Matching replica decided")
}

func TestBookRideExplainsRejections(t *testing.T) {
	gw := newTestGateway(t, false)
	pickup := gw.pickupPoints[0]
	gw.drivers = []Driver{ecityDriver("driver-full", 0)}

	rr := postJSON(t, gw.BookRideHandler, "/rides/book", bookRideRequest{
		Command: "book", Name: "Ananya", PickupPointID: pickup.ID, Destination: "Wipro Gate",
	})
	require.Equal(t, http.StatusOK, rr.Code)

	var payload bookRideResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))
	assert.Equal(t, "queued", payload.Status)
	require.Len(t, payload.Rejections, 1)
	assert.Equal(t, "no free seats", payload.Rejections[0].Reason)
	assert.Contains(t, payload.Message, "no free seats")
}

// TestBookRideDegradesWhenMatchingIsDown is the failure-isolation story in
// one test: with no Matching replica, booking answers politely instead of
// erroring, and everything else in the gateway keeps working.
func TestBookRideDegradesWhenMatchingIsDown(t *testing.T) {
	gw := NewGateway(nil, nil, nil, nil) // no Matching attached
	pickup := gw.pickupPoints[0]
	gw.drivers = []Driver{ecityDriver("driver-2", 1)}

	rr := postJSON(t, gw.BookRideHandler, "/rides/book", bookRideRequest{
		Command: "book", Name: "Ananya", PickupPointID: pickup.ID, Destination: "Wipro Gate",
	})
	require.Equal(t, http.StatusOK, rr.Code)
	var payload bookRideResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))
	assert.Equal(t, "matching_unavailable", payload.Status)

	snap := httptest.NewRecorder()
	gw.SnapshotHandler(snap, httptest.NewRequest(http.MethodGet, "/aggregates/snapshot", nil))
	assert.Equal(t, http.StatusOK, snap.Code, "the rest of the app is unaffected")

	reqs := httptest.NewRecorder()
	gw.DriverRequestsHandler(reqs, httptest.NewRequest(http.MethodGet, "/drivers/requests?driverId=driver-2", nil))
	assert.Equal(t, http.StatusServiceUnavailable, reqs.Code)
}

func TestDriverAcceptHandler(t *testing.T) {
	gw := newTestGateway(t, false)
	pickup := gw.pickupPoints[0]
	gw.riders = append([]Rider{{
		ID: "rider-accept", Name: "Sahana", Destination: "Wipro Gate", ArrivalTime: time.Now().Add(4 * time.Minute),
		StationID: pickup.StationID, Status: "waiting", PickupPointID: pickup.ID, Pickup: &pickup,
	}}, gw.riders...)
	gw.drivers = []Driver{ecityDriver("driver-accept", 2)}

	rr := postJSON(t, gw.DriverAcceptHandler, "/drivers/requests/accept", driverAcceptRequest{DriverID: "driver-accept", RiderID: "rider-accept"})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var trip Trip
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &trip))
	assert.Equal(t, "rider-accept", trip.RiderID)
	assert.Equal(t, pickup.ID, trip.PickupPointID)
}

// TestSimulatedTripEndToEnd drives a whole ride in-process: the driver plans
// a route, accepts a bot rider, the car drives to the station (crossing the
// approach zone), picks the rider up, and drops them at Wipro Gate.
func TestSimulatedTripEndToEnd(t *testing.T) {
	gw := newTestGateway(t, true)

	_, err := gw.configureDriverRoute(driverRouteRequest{
		DriverID: "driver-sim", Name: "Sim Driver", PickupPointIDs: []string{"pickup-station-ecity"}, Seats: 2,
	})
	require.NoError(t, err)
	_, err = gw.startDriverTrip(startTripRequest{DriverID: "driver-sim"})
	require.NoError(t, err)

	// The driver's queue (decided by Matching) contains the bot at E-City.
	queue, err := gw.driverRequests("driver-sim")
	require.NoError(t, err)
	require.NotEmpty(t, queue.Requests)
	assert.Equal(t, "bot-rider-meera", queue.Requests[0].ID)

	trip, err := gw.acceptTripFromRealtime("driver-sim", "bot-rider-meera")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return tripStatus(gw, trip.ID) == "pending" }, time.Second, 5*time.Millisecond,
		"bot riders auto-confirm")

	gw.runSimulatedTrip(context.Background(), "driver-sim", gw.pickupPointsForIDs([]string{"pickup-station-ecity"}))

	assert.Equal(t, "completed", tripStatus(gw, trip.ID))
	gw.mu.Lock()
	defer gw.mu.Unlock()
	pos := gw.lastPos["driver-sim"]
	assert.InDelta(t, 12.8467, pos[0], 0.0005, "car ends at Wipro Gate, the rider's destination")
	assert.InDelta(t, 77.6624, pos[1], 0.0005)
	assert.Equal(t, 2, gw.driverPlans["driver-sim"].SeatsAvailable, "seat is freed after drop-off")
	assert.False(t, gw.driverPlans["driver-sim"].Active, "plan ends after the last drop-off")
}

func tripStatus(gw *Gateway, id string) string {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	for _, t := range gw.trips {
		if t.ID == id {
			return t.Status
		}
	}
	return ""
}

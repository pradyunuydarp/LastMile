package matching

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lastmile/gen/go/matching"
)

var now = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

// eCity and konappana are real stations ~0.9 km apart.
var (
	eCity     = &pb.GeoPoint{Latitude: 12.8456, Longitude: 77.6600}
	konappana = &pb.GeoPoint{Latitude: 12.8519, Longitude: 77.6546}
	farAway   = &pb.GeoPoint{Latitude: 13.1986, Longitude: 77.7066} // the airport, ~40 km
)

func driver(mut ...func(*pb.DriverState)) *pb.DriverState {
	d := &pb.DriverState{
		DriverId:        "driver-asha",
		Name:            "Asha",
		PickupPointIds:  []string{"pickup-station-ecity", "pickup-station-konappana"},
		NextPickupIndex: 0,
		SeatsAvailable:  3,
		Destination:     "Wipro Gate",
		Position:        eCity,
		TripActive:      true,
	}
	for _, m := range mut {
		m(d)
	}
	return d
}

func rider(mut ...func(*pb.RiderRequest)) *pb.RiderRequest {
	r := &pb.RiderRequest{
		RiderId:       "rider-priya",
		Name:          "Priya",
		PickupPointId: "pickup-station-konappana",
		StationId:     "station-konappana",
		Destination:   "Wipro Gate",
		Pickup:        konappana,
		ArrivalUnix:   now.Add(5 * time.Minute).Unix(),
	}
	for _, m := range mut {
		m(r)
	}
	return r
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name       string
		driver     *pb.DriverState
		rider      *pb.RiderRequest
		wantOK     bool
		wantReason string
	}{
		{"happy path", driver(), rider(), true, ""},
		{"rule 1: trip not started",
			driver(func(d *pb.DriverState) { d.TripActive = false }), rider(),
			false, "driver has not started a trip"},
		{"rule 2: pickup not on route",
			driver(), rider(func(r *pb.RiderRequest) { r.PickupPointId = "pickup-station-hsr" }),
			false, "pickup is not on the driver's route"},
		{"rule 3: driver already passed pickup",
			driver(func(d *pb.DriverState) { d.NextPickupIndex = 2 }), rider(),
			false, "driver already passed this pickup"},
		{"rule 3: current stop still counts",
			driver(func(d *pb.DriverState) { d.NextPickupIndex = 1 }), rider(),
			true, ""},
		{"rule 4: no seats",
			driver(func(d *pb.DriverState) { d.SeatsAvailable = 0 }), rider(),
			false, "no free seats"},
		{"rule 5: different destination",
			driver(func(d *pb.DriverState) { d.Destination = "Forum Mall" }), rider(),
			false, `driver is going to "Forum Mall", rider to "Wipro Gate"`},
		{"rule 5: driver going anywhere",
			driver(func(d *pb.DriverState) { d.Destination = "unknown" }), rider(),
			true, ""},
		{"rule 6: driver too far away",
			driver(func(d *pb.DriverState) { d.Position = farAway }), rider(),
			false, "driver is 117 min away; rider would wait more than 15 min"},
		{"rule 6: unknown position is not rejected",
			driver(func(d *pb.DriverState) { d.Position = nil }), rider(),
			true, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := Check(tc.driver, tc.rider, now)
			assert.Equal(t, tc.wantOK, v.OK)
			assert.Equal(t, tc.wantReason, v.Reason)
		})
	}
}

func TestSameDestination(t *testing.T) {
	tests := []struct {
		driver, rider string
		want          bool
	}{
		{"Wipro Gate", "wipro gate", true},
		{"Wipro Gate · Electronic City", "Wipro Gate", true},
		{"unknown", "Forum Mall", true},
		{"", "Forum Mall", true},
		{"Forum Mall", "Wipro Gate", false},
	}
	for _, tc := range tests {
		t.Run(tc.driver+"->"+tc.rider, func(t *testing.T) {
			assert.Equal(t, tc.want, SameDestination(tc.driver, tc.rider))
		})
	}
}

func TestRankDriversOrdersByDistance(t *testing.T) {
	near := driver(func(d *pb.DriverState) { d.DriverId = "near"; d.Position = konappana })
	mid := driver(func(d *pb.DriverState) { d.DriverId = "mid"; d.Position = eCity })
	unknown := driver(func(d *pb.DriverState) { d.DriverId = "unknown"; d.Position = nil })
	full := driver(func(d *pb.DriverState) { d.DriverId = "full"; d.SeatsAvailable = 0 })

	got, rejected := RankDrivers(rider(), []*pb.DriverState{unknown, mid, full, near}, now)

	require.Len(t, got, 3)
	assert.Equal(t, []string{"near", "mid", "unknown"}, []string{got[0].DriverId, got[1].DriverId, got[2].DriverId})
	assert.Equal(t, float64(-1), got[2].DistanceMeters, "unknown distance is reported as -1")
	require.Len(t, rejected, 1)
	assert.Equal(t, "full", rejected[0].Id)
}

func TestFindRidersOrdersByArrival(t *testing.T) {
	late := rider(func(r *pb.RiderRequest) { r.RiderId = "late"; r.ArrivalUnix = now.Add(9 * time.Minute).Unix() })
	soon := rider(func(r *pb.RiderRequest) { r.RiderId = "soon"; r.ArrivalUnix = now.Add(2 * time.Minute).Unix() })
	elsewhere := rider(func(r *pb.RiderRequest) { r.RiderId = "elsewhere"; r.Destination = "Forum Mall" })

	got, rejected := FindRiders(driver(), []*pb.RiderRequest{late, elsewhere, soon}, now)

	require.Len(t, got, 2)
	assert.Equal(t, "soon", got[0].RiderId)
	assert.Equal(t, "late", got[1].RiderId)
	require.Len(t, rejected, 1)
	assert.Equal(t, "elsewhere", rejected[0].Id)
}

func TestServerEchoesReplicaName(t *testing.T) {
	s := NewServer(nil, "matching-abc123")
	s.now = func() time.Time { return now }

	resp, err := s.RankDrivers(context.Background(), &pb.RankDriversRequest{
		Rider:   rider(),
		Drivers: []*pb.DriverState{driver()},
	})
	require.NoError(t, err)
	assert.Equal(t, "matching-abc123", resp.ServedBy)
	require.Len(t, resp.Candidates, 1)
	assert.Equal(t, "driver-asha", resp.Candidates[0].DriverId)
}

func TestServerValidatesInput(t *testing.T) {
	s := NewServer(nil, "test")

	_, err := s.RankDrivers(context.Background(), &pb.RankDriversRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = s.FindRiders(context.Background(), &pb.FindRidersRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	driverpb "lastmile/gen/go/driver"
	gatewaypb "lastmile/gen/go/gateway"
	locationpb "lastmile/gen/go/location"
	userpb "lastmile/gen/go/user"

	"github.com/gorilla/websocket"
)

// Route mirrors the mobile Route type.
type PickupPoint struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	StationID   string  `json:"stationId"`
	StationName string  `json:"stationName"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type Route struct {
	ID               string        `json:"id"`
	TargetStationIDs []string      `json:"targetStationIds"`
	Destination      string        `json:"destination"`
	PickupPoints     []PickupPoint `json:"pickupPoints,omitempty"`
}

// Driver mirrors the mobile Driver type.
type Driver struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	CarDetails     string  `json:"carDetails"`
	SeatsAvailable int     `json:"seatsAvailable"`
	ETAMinutes     int     `json:"etaMinutes"`
	Status         string  `json:"status"`
	Route          Route   `json:"route"`
	Latitude       float64 `json:"latitude,omitempty"`
	Longitude      float64 `json:"longitude,omitempty"`
}

// Rider mirrors the mobile Rider type.
type Rider struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Destination   string       `json:"destination"`
	ArrivalTime   time.Time    `json:"arrivalTime"`
	StationID     string       `json:"stationId"`
	Status        string       `json:"status"`
	PickupPointID string       `json:"pickupPointId,omitempty"`
	Pickup        *PickupPoint `json:"pickup,omitempty"`
}

// Trip mirrors the mobile Trip type.
type Trip struct {
	ID            string       `json:"id"`
	DriverID      string       `json:"driverId"`
	RiderID       string       `json:"riderId"`
	StationID     string       `json:"stationId,omitempty"`
	Destination   string       `json:"destination,omitempty"`
	PickupPoint   *PickupPoint `json:"pickup,omitempty"`
	PickupPointID string       `json:"pickupPointId,omitempty"`
	ETAMinutes    int          `json:"etaMinutes,omitempty"`
	Status        string       `json:"status"`
	CreatedAt     time.Time    `json:"createdAt,omitempty"`
	CompletedAt   time.Time    `json:"completedAt,omitempty"`
	RoomID        string       `json:"roomId,omitempty"`
}

type pendingTripContext struct {
	Trip    Trip
	Rider   *Rider
	Pickup  *PickupPoint
	Station *Station
}

type driverPlan struct {
	DriverID       string
	PickupIDs      []string
	SeatsTotal     int
	SeatsAvailable int
	TargetStations []string
	Destination    string
	CurrentIndex   int
	Active         bool
	StartedAt      time.Time
	Simulated      bool
	simCancel      context.CancelFunc
}

type Station struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	NearbyAreas []string `json:"nearbyAreas"`
	LoadFactor  float64  `json:"loadFactor,omitempty"`
	Latitude    float64  `json:"latitude,omitempty"`
	Longitude   float64  `json:"longitude,omitempty"`
}

type BackendMetrics struct {
	PendingMatches int     `json:"pendingMatches"`
	RidersWaiting  int     `json:"ridersWaiting"`
	SeatsOpen      int     `json:"seatsOpen"`
	AvgWaitMinutes float64 `json:"avgWaitMinutes"`
	Version        string  `json:"version"`
}

type BackendSnapshot struct {
	Drivers       []Driver       `json:"drivers"`
	Riders        []Rider        `json:"riders"`
	Trips         []Trip         `json:"trips"`
	Stations      []Station      `json:"stations"`
	Metrics       BackendMetrics `json:"metrics"`
	HighlightTrip *Trip          `json:"highlightTrip,omitempty"`
	LastUpdated   time.Time      `json:"lastUpdated"`
}

type matchRequest struct {
	DriverID  string `json:"driverId"`
	StationID string `json:"stationId"`
}

type driverSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SeatsAvailable int    `json:"seatsAvailable"`
	NextStop       string `json:"nextStop"`
}

type riderRequestView struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Destination    string       `json:"destination"`
	ArrivalTime    time.Time    `json:"arrivalTime"`
	Station        Station      `json:"station"`
	Pickup         *PickupPoint `json:"pickup,omitempty"`
	Status         string       `json:"status"`
	DistanceMeters float64      `json:"distanceMeters"`
}

type driverRequestsResponse struct {
	Driver      driverSummary      `json:"driver"`
	Requests    []riderRequestView `json:"requests"`
	GeneratedAt time.Time          `json:"generatedAt"`
}

type driverAttempt struct {
	DriverID       string  `json:"driverId"`
	DriverName     string  `json:"driverName"`
	DistanceMeters float64 `json:"distanceMeters"`
	Accepted       bool    `json:"accepted"`
	Reason         string  `json:"reason,omitempty"`
}

type bookRideRequest struct {
	Command       string `json:"command"`
	RiderID       string `json:"riderId"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Destination   string `json:"destination"`
	StationID     string `json:"stationId"`
	PickupPointID string `json:"pickupPointId"`
}

type bookRideResponse struct {
	Status               string          `json:"status"`
	Message              string          `json:"message"`
	Rider                Rider           `json:"rider"`
	Station              Station         `json:"station"`
	Pickup               *PickupPoint    `json:"pickup"`
	RequestedDestination string          `json:"requestedDestination"`
	Attempts             []driverAttempt `json:"attempts"`
	Trip                 *Trip           `json:"trip,omitempty"`
}

type driverRouteRequest struct {
	DriverID       string   `json:"driverId"`
	Name           string   `json:"name"`
	CarDetails     string   `json:"carDetails"`
	PickupPointIDs []string `json:"pickupPointIds"`
	Seats          int      `json:"seats"`
}

type driverRouteResponse struct {
	DriverID       string        `json:"driverId"`
	PickupPoints   []PickupPoint `json:"pickupPoints"`
	SeatsTotal     int           `json:"seatsTotal"`
	SeatsAvailable int           `json:"seatsAvailable"`
	TargetStations []string      `json:"targetStations"`
	Destination    string        `json:"destination"`
}

type startTripRequest struct {
	DriverID string `json:"driverId"`
	Simulate bool   `json:"simulate"`
}

type driverAcceptRequest struct {
	DriverID string `json:"driverId"`
	RiderID  string `json:"riderId"`
}

// Gateway holds in-memory data and exposes the HTTP handlers expected by the mobile app.
type Gateway struct {
	mu             sync.Mutex
	logger         *slog.Logger
	drivers        []Driver
	riders         []Rider
	trips          []Trip
	stations       []Station
	pickupPoints   []PickupPoint
	driverPlans    map[string]*driverPlan
	driverClient   driverpb.DriverServiceClient
	locationClient locationpb.LocationServiceClient
	userClient     userpb.UserServiceClient
	hub            *RealtimeHub
	store          *Persistence
	pendingTrips   map[string]*pendingTripContext
	pushTokens     map[string]string
}

func NewGateway(logger *slog.Logger, driverClient driverpb.DriverServiceClient, locClient locationpb.LocationServiceClient, userClient userpb.UserServiceClient) *Gateway {
	l := slog.Default()
	if logger != nil {
		l = logger
	}

	now := time.Now()
	stations := defaultStations()
	pickups := defaultPickupPoints()

	findPickup := func(name string) *PickupPoint {
		for i := range pickups {
			if strings.EqualFold(pickups[i].Name, name) {
				pp := pickups[i]
				return &pp
			}
		}
		return nil
	}

	// Hardcoded riders and trips for demo purposes (can be replaced later)
	ensure := func(candidate *PickupPoint, fallbackIdx int) *PickupPoint {
		if candidate != nil {
			copyVal := *candidate
			return &copyVal
		}
		if fallbackIdx >= 0 && fallbackIdx < len(pickups) {
			pp := pickups[fallbackIdx]
			return &pp
		}
		return &PickupPoint{}
	}

	r1 := ensure(findPickup("Wipro Gate"), 0)
	r2 := ensure(findPickup("Siemens Campus"), 9)
	r3 := ensure(findPickup("D Mart Huskur"), 12)
	riders := []Rider{
		{ID: "rider-priya", Name: "Priya Sharma", Destination: r1.Name, ArrivalTime: now.Add(5 * time.Minute), StationID: r1.StationID, Status: "waiting", PickupPointID: r1.ID, Pickup: r1},
		{ID: "rider-rahul", Name: "Rahul Verma", Destination: r2.Name, ArrivalTime: now.Add(9 * time.Minute), StationID: r2.StationID, Status: "waiting", PickupPointID: r2.ID, Pickup: r2},
		{ID: "rider-anita", Name: "Anita Desai", Destination: r3.Name, ArrivalTime: now.Add(12 * time.Minute), StationID: r3.StationID, Status: "matched", PickupPointID: r3.ID, Pickup: r3},
	}

	trips := []Trip{
		{
			ID:            "trip-1",
			DriverID:      "driver-ramesh",
			RiderID:       "rider-priya",
			StationID:     r1.StationID,
			Destination:   r1.StationName,
			PickupPoint:   r1,
			PickupPointID: r1.ID,
			ETAMinutes:    5,
			Status:        "pending",
			CreatedAt:     now.Add(-2 * time.Minute),
		},
	}

	return &Gateway{
		logger:         l.With("component", "gateway"),
		drivers:        []Driver{}, // Drivers will be fetched dynamically
		riders:         riders,
		trips:          trips,
		stations:       stations,
		pickupPoints:   pickups,
		driverPlans:    make(map[string]*driverPlan),
		driverClient:   driverClient,
		locationClient: locClient,
		userClient:     userClient,
		pendingTrips:   make(map[string]*pendingTripContext),
		pushTokens:     make(map[string]string),
	}
}

func (g *Gateway) AttachHub(h *RealtimeHub) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.hub = h
	if h != nil {
		h.BindGateway(g)
	}
}

func (g *Gateway) AttachStore(store *Persistence) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.store = store
}

func (g *Gateway) snapshot() BackendSnapshot {
	g.mu.Lock()
	defer g.mu.Unlock()

	metrics := g.metrics()
	var highlight *Trip
	if len(g.trips) > 0 {
		highlight = &g.trips[0]
	}

	if g.driverClient != nil {
		resp, err := g.driverClient.ListDrivers(context.Background(), &driverpb.ListDriversRequest{})
		if err != nil {
			g.logger.Error("failed to list drivers", "err", err)
		} else {
			routeByDriver := make(map[string]*driverpb.Route)
			for _, r := range resp.Routes {
				routeByDriver[r.DriverId] = r
			}

			g.drivers = make([]Driver, 0, len(resp.Drivers))
			driverIDs := make([]string, 0, len(resp.Drivers))
			for _, d := range resp.Drivers {
				driverIDs = append(driverIDs, d.Id)
			}

			locMap := make(map[string]*locationpb.Location)
			if g.locationClient != nil && len(driverIDs) > 0 {
				locResp, err := g.locationClient.GetDriverLocations(context.Background(), &locationpb.GetDriverLocationsRequest{DriverIds: driverIDs})
				if err != nil {
					g.logger.Error("failed to get driver locations", "err", err)
				} else {
					for _, loc := range locResp.Locations {
						locMap[loc.DriverId] = loc
					}
				}
			}

			for _, d := range resp.Drivers {
				plan := g.driverPlans[d.Id]
				route := Route{}
				seats := 0
				if plan != nil {
					route = Route{
						ID:               plan.DriverID,
						TargetStationIDs: append([]string{}, plan.TargetStations...),
						Destination:      plan.Destination,
						PickupPoints:     g.pickupPointsForIDs(plan.PickupIDs),
					}
					seats = plan.SeatsAvailable
					if seats == 0 {
						seats = plan.SeatsTotal
					}
				} else if r := routeByDriver[d.Id]; r != nil {
					route = Route{
						ID:               r.Id,
						TargetStationIDs: append([]string{}, r.TargetStationIds...),
						Destination:      r.Destination,
					}
					seats = int(r.AvailableSeats)
				}

				if route.Destination == "" {
					route.Destination = "Unknown"
				}

				lat, lon := 0.0, 0.0
				if loc, ok := locMap[d.Id]; ok {
					lat, lon = loc.Latitude, loc.Longitude
				}

				g.drivers = append(g.drivers, Driver{
					ID:             d.Id,
					Name:           d.Name,
					CarDetails:     d.CarDetails,
					SeatsAvailable: seats,
					ETAMinutes:     5,
					Status:         "active",
					Route:          route,
					Latitude:       lat,
					Longitude:      lon,
				})
			}
		}
	}

	return BackendSnapshot{
		Drivers:       append([]Driver{}, g.drivers...),
		Riders:        append([]Rider{}, g.riders...),
		Trips:         append([]Trip{}, g.trips...),
		Stations:      append([]Station{}, g.stations...),
		Metrics:       metrics,
		HighlightTrip: highlight,
		LastUpdated:   time.Now().UTC(),
	}
}

// Snapshot returns a value copy of the current snapshot.
func (g *Gateway) Snapshot() BackendSnapshot {
	return g.snapshot()
}

// SnapshotProto returns the snapshot encoded as the gRPC contract.
func (g *Gateway) SnapshotProto() *gatewaypb.BackendSnapshot {
	view := g.snapshot()

	var highlight *gatewaypb.GatewayTrip
	if view.HighlightTrip != nil {
		highlight = toProtoTrip(*view.HighlightTrip)
	}

	return &gatewaypb.BackendSnapshot{
		Drivers:       toProtoDrivers(view.Drivers),
		Riders:        toProtoRiders(view.Riders),
		Trips:         toProtoTrips(view.Trips),
		Stations:      toProtoStations(view.Stations),
		Metrics:       toProtoMetrics(view.Metrics),
		HighlightTrip: highlight,
		LastUpdated:   view.LastUpdated.Format(time.RFC3339),
	}
}

func (g *Gateway) metrics() BackendMetrics {
	waiting := 0
	for _, rider := range g.riders {
		if rider.Status == "waiting" {
			waiting++
		}
	}

	avgWait := 0.0
	if waiting > 0 {
		total := 0.0
		now := time.Now()
		for _, rider := range g.riders {
			if rider.Status != "waiting" {
				continue
			}
			minutes := rider.ArrivalTime.Sub(now).Minutes()
			if minutes < 0 {
				minutes = 0
			}
			total += minutes
		}
		avgWait = total / float64(waiting)
	}

	seats := 0
	for _, driver := range g.drivers {
		seats += driver.SeatsAvailable
	}

	pending := 0
	for _, trip := range g.trips {
		if trip.Status == "pending" {
			pending++
		}
	}

	return BackendMetrics{
		PendingMatches: pending,
		RidersWaiting:  waiting,
		SeatsOpen:      seats,
		AvgWaitMinutes: avgWait,
		Version:        "go-gateway",
	}
}

func (g *Gateway) createTrip(driverID, stationID string) (Trip, error) {
	return g.createTripForRider(driverID, stationID, "")
}

func (g *Gateway) createTripForRider(driverID, stationID, riderID string) (Trip, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	driver, err := g.findDriver(driverID, stationID)
	if err != nil {
		return Trip{}, err
	}

	var rider *Rider
	if riderID != "" {
		rider, err = g.findRiderByID(riderID)
		if err != nil {
			return Trip{}, err
		}
		if rider.StationID != stationID {
			return Trip{}, fmt.Errorf("rider '%s' is not waiting at station '%s'", rider.ID, stationID)
		}
	} else {
		rider, err = g.findRider(stationID, driver.Route.Destination)
		if err != nil {
			return Trip{}, err
		}
	}

	if driver.SeatsAvailable <= 0 {
		return Trip{}, fmt.Errorf("driver '%s' has no seats left", driverID)
	}

	if !compatibleDestination(driver.Route.Destination, rider.Destination) {
		return Trip{}, fmt.Errorf("driver '%s' has a different destination", driverID)
	}

	driver.SeatsAvailable--
	if plan, ok := g.driverPlans[driver.ID]; ok {
		if plan.SeatsAvailable > 0 {
			plan.SeatsAvailable--
		}
	}

	var pickup *PickupPoint
	if rider.Pickup != nil {
		copyPickup := *rider.Pickup
		pickup = &copyPickup
	} else if rider.PickupPointID != "" {
		if pp, ok := g.pickupByID(rider.PickupPointID); ok {
			pickup = pp
		}
	}

	now := time.Now().UTC()
	trip := Trip{
		ID:            fmt.Sprintf("trip-%d", now.UnixNano()),
		DriverID:      driver.ID,
		RiderID:       rider.ID,
		StationID:     stationID,
		Destination:   rider.Destination,
		PickupPoint:   pickup,
		PickupPointID: rider.PickupPointID,
		ETAMinutes:    driver.ETAMinutes,
		Status:        "awaiting_rider",
		CreatedAt:     now,
	}

	g.trips = append([]Trip{trip}, g.trips...)
	rider.Status = "matched"

	g.logger.Info("match created",
		"tripId", trip.ID,
		"driverId", driverID,
		"riderId", rider.ID,
		"stationId", stationID)

	return trip, nil
}

// TriggerMatchProto creates a trip and returns the proto representation.
func (g *Gateway) TriggerMatchProto(driverID, stationID string) (*gatewaypb.GatewayTrip, error) {
	trip, err := g.createTrip(driverID, stationID)
	if err != nil {
		return nil, err
	}
	return toProtoTrip(trip), nil
}

func (g *Gateway) driverRequests(driverID string) (driverRequestsResponse, error) {
	g.snapshot()
	g.mu.Lock()
	defer g.mu.Unlock()

	driver, err := g.findDriver(driverID, "")
	if err != nil {
		return driverRequestsResponse{}, err
	}

	stationName := ""
	if len(driver.Route.TargetStationIDs) > 0 {
		if station, ok := g.stationByID(driver.Route.TargetStationIDs[0]); ok {
			stationName = station.Name
		} else {
			stationName = driver.Route.TargetStationIDs[0]
		}
	}

	var requests []riderRequestView
	for i := range g.riders {
		rider := g.riders[i]
		if rider.Status == "picked_up" {
			continue
		}
		if !routeContains(driver.Route.TargetStationIDs, rider.StationID) {
			continue
		}
		station, ok := g.stationByID(rider.StationID)
		if !ok {
			continue
		}
		pickup := rider.Pickup
		if pickup == nil && rider.PickupPointID != "" {
			if pp, found := g.pickupByID(rider.PickupPointID); found {
				pickup = pp
			}
		}
		distance := driverDistanceToPickup(driver, pickup, station)
		requests = append(requests, riderRequestView{
			ID:             rider.ID,
			Name:           rider.Name,
			Destination:    rider.Destination,
			ArrivalTime:    rider.ArrivalTime,
			Station:        *station,
			Pickup:         pickup,
			Status:         rider.Status,
			DistanceMeters: distance,
		})
	}

	sort.Slice(requests, func(i, j int) bool {
		return requests[i].ArrivalTime.Before(requests[j].ArrivalTime)
	})

	return driverRequestsResponse{
		Driver: driverSummary{
			ID:             driver.ID,
			Name:           driver.Name,
			SeatsAvailable: driver.SeatsAvailable,
			NextStop:       stationName,
		},
		Requests:    requests,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (g *Gateway) bookRide(payload bookRideRequest) (bookRideResponse, error) {
	command := strings.TrimSpace(strings.ToLower(payload.Command))
	if command != "book" {
		return bookRideResponse{}, fmt.Errorf("unsupported command '%s'", payload.Command)
	}

	station, pickup, inferredArea, err := g.resolveStation(payload)
	if err != nil {
		return bookRideResponse{}, err
	}
	if pickup == nil {
		return bookRideResponse{}, errors.New("pickup point required")
	}
	requestedDestination := strings.TrimSpace(payload.Destination)
	if requestedDestination == "" {
		requestedDestination = inferredArea
	}
	if requestedDestination == "" {
		requestedDestination = pickup.Name
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = "Guest Rider"
	}

	g.mu.Lock()
	rider := g.upsertRiderLocked(payload.RiderID, name, station, requestedDestination, pickup)
	candidates := g.driverCandidatesLocked(station, pickup)
	g.mu.Unlock()

	riderSnapshot := copyRider(rider)
	stationCopy := copyStation(station)
	pickupCopy := copyPickupPoint(pickup)

	if g.store != nil {
		go g.store.RecordRiderRequest(riderSnapshot, pickupCopy, "waiting")
	}

	attempts := make([]driverAttempt, len(candidates))
	copy(attempts, candidates)

	var trip *Trip
	var acceptedIndex = -1

	if g.hub == nil {
		for i := range attempts {
			result, err := g.createTripForRider(attempts[i].DriverID, station.ID, rider.ID)
			if err != nil {
				attempts[i].Reason = err.Error()
				continue
			}
			attempts[i].Accepted = true
			trip = &result
			acceptedIndex = i
			g.queueRiderApproval(result)
			break
		}
	} else if len(attempts) > 0 {
		go g.hub.EnqueueRiderRequest(riderSnapshot, stationCopy, pickupCopy, attempts)
	}

	status := "queued"
	message := fmt.Sprintf("No drivers available near %s yet", station.Name)
	if trip != nil && acceptedIndex >= 0 {
		status = "awaiting_rider"
		message = fmt.Sprintf("%s accepted. Confirm from your Riders tab.", attempts[acceptedIndex].DriverName)
	} else if g.hub != nil && len(attempts) > 0 {
		message = fmt.Sprintf("Contacting %d drivers near %s", len(attempts), station.Name)
	}

	return bookRideResponse{
		Status:               status,
		Message:              message,
		Rider:                riderSnapshot,
		Station:              *station,
		Pickup:               pickupCopy,
		RequestedDestination: requestedDestination,
		Attempts:             attempts,
		Trip:                 trip,
	}, nil
}

func toProtoDrivers(drivers []Driver) []*gatewaypb.GatewayDriver {
	out := make([]*gatewaypb.GatewayDriver, 0, len(drivers))
	for _, d := range drivers {
		out = append(out, &gatewaypb.GatewayDriver{
			Id:             d.ID,
			Name:           d.Name,
			CarDetails:     d.CarDetails,
			SeatsAvailable: int32(d.SeatsAvailable),
			EtaMinutes:     int32(d.ETAMinutes),
			Status:         d.Status,
			Route: &gatewaypb.GatewayRoute{
				Id:               d.Route.ID,
				TargetStationIds: d.Route.TargetStationIDs,
				Destination:      d.Route.Destination,
				PickupPoints:     toProtoPickupPoints(d.Route.PickupPoints),
			},
			Latitude:  d.Latitude,
			Longitude: d.Longitude,
		})
	}
	return out
}

func toProtoRiders(riders []Rider) []*gatewaypb.GatewayRider {
	out := make([]*gatewaypb.GatewayRider, 0, len(riders))
	for _, r := range riders {
		out = append(out, &gatewaypb.GatewayRider{
			Id:          r.ID,
			Name:        r.Name,
			Destination: r.Destination,
			ArrivalTime: r.ArrivalTime.Format(time.RFC3339),
			StationId:   r.StationID,
			Status:      r.Status,
		})
	}
	return out
}

func toProtoTrips(trips []Trip) []*gatewaypb.GatewayTrip {
	out := make([]*gatewaypb.GatewayTrip, 0, len(trips))
	for _, t := range trips {
		out = append(out, toProtoTrip(t))
	}
	return out
}

func toProtoTrip(t Trip) *gatewaypb.GatewayTrip {
	return &gatewaypb.GatewayTrip{
		Id:            t.ID,
		DriverId:      t.DriverID,
		RiderId:       t.RiderID,
		StationId:     t.StationID,
		Destination:   t.Destination,
		EtaMinutes:    int32(t.ETAMinutes),
		Status:        t.Status,
		CreatedAt:     t.CreatedAt.Format(time.RFC3339),
		PickupPointId: t.PickupPointID,
		Pickup:        toProtoPickupPoint(t.PickupPoint),
	}
}

func toProtoPickupPoints(points []PickupPoint) []*gatewaypb.GatewayPickupPoint {
	out := make([]*gatewaypb.GatewayPickupPoint, 0, len(points))
	for _, p := range points {
		out = append(out, toProtoPickupPoint(&p))
	}
	return out
}

func toProtoPickupPoint(p *PickupPoint) *gatewaypb.GatewayPickupPoint {
	if p == nil {
		return nil
	}
	return &gatewaypb.GatewayPickupPoint{
		Id:          p.ID,
		Name:        p.Name,
		StationId:   p.StationID,
		StationName: p.StationName,
		Latitude:    p.Latitude,
		Longitude:   p.Longitude,
	}
}

func toProtoStations(stations []Station) []*gatewaypb.GatewayStation {
	out := make([]*gatewaypb.GatewayStation, 0, len(stations))
	for _, s := range stations {
		out = append(out, &gatewaypb.GatewayStation{
			Id:          s.ID,
			Name:        s.Name,
			NearbyAreas: s.NearbyAreas,
			LoadFactor:  s.LoadFactor,
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
		})
	}
	return out
}

func toProtoMetrics(m BackendMetrics) *gatewaypb.BackendMetrics {
	return &gatewaypb.BackendMetrics{
		PendingMatches: int32(m.PendingMatches),
		RidersWaiting:  int32(m.RidersWaiting),
		SeatsOpen:      int32(m.SeatsOpen),
		AvgWaitMinutes: m.AvgWaitMinutes,
		Version:        "go-gateway",
	}
}

func (g *Gateway) findDriver(driverID, stationID string) (*Driver, error) {
	for i := range g.drivers {
		driver := &g.drivers[i]
		if driver.ID != driverID {
			continue
		}
		if stationID == "" {
			return driver, nil
		}
		for _, id := range driver.Route.TargetStationIDs {
			if id == stationID {
				return driver, nil
			}
		}
		return nil, fmt.Errorf("driver '%s' is not routed to station '%s'", driverID, stationID)
	}
	return nil, fmt.Errorf("driver '%s' not found", driverID)
}

func (g *Gateway) findRider(stationID, destination string) (*Rider, error) {
	for i := range g.riders {
		rider := &g.riders[i]
		if rider.StationID != stationID || rider.Status == "picked_up" {
			continue
		}
		if destination == "" || rider.Destination == "" || strings.EqualFold(rider.Destination, destination) {
			return rider, nil
		}
	}

	return nil, errors.New("no riders available for station with matching destination")
}

func (g *Gateway) findRiderByID(riderID string) (*Rider, error) {
	for i := range g.riders {
		if g.riders[i].ID == riderID {
			return &g.riders[i], nil
		}
	}
	return nil, fmt.Errorf("rider '%s' not found", riderID)
}

func routeContains(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func (g *Gateway) stationByID(id string) (*Station, bool) {
	for i := range g.stations {
		if g.stations[i].ID == id {
			return &g.stations[i], true
		}
	}
	return nil, false
}

func (g *Gateway) pickupByID(id string) (*PickupPoint, bool) {
	for i := range g.pickupPoints {
		if g.pickupPoints[i].ID == id {
			pp := g.pickupPoints[i]
			return &pp, true
		}
	}
	return nil, false
}

func (g *Gateway) pickupPointsForIDs(ids []string) []PickupPoint {
	out := make([]PickupPoint, 0, len(ids))
	for _, id := range ids {
		if pp, ok := g.pickupByID(id); ok {
			out = append(out, *pp)
		}
	}
	return out
}

func (g *Gateway) riderStationID(riderID string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.riders {
		if g.riders[i].ID == riderID {
			return g.riders[i].StationID, nil
		}
	}
	return "", fmt.Errorf("rider '%s' not found", riderID)
}

func driverDistance(driver *Driver, station *Station) float64 {
	if driver == nil || station == nil {
		return math.MaxFloat64
	}
	if (driver.Latitude == 0 && driver.Longitude == 0) || (station.Latitude == 0 && station.Longitude == 0) {
		return math.MaxFloat64
	}
	return haversineMeters(driver.Latitude, driver.Longitude, station.Latitude, station.Longitude)
}

func driverDistanceToPickup(driver *Driver, pickup *PickupPoint, fallbackStation *Station) float64 {
	if pickup != nil && pickup.Latitude != 0 && pickup.Longitude != 0 {
		return haversineMeters(driver.Latitude, driver.Longitude, pickup.Latitude, pickup.Longitude)
	}
	return driverDistance(driver, fallbackStation)
}

func (g *Gateway) resolveStation(payload bookRideRequest) (*Station, *PickupPoint, string, error) {
	if payload.PickupPointID != "" {
		pickup, ok := g.pickupByID(payload.PickupPointID)
		if !ok {
			return nil, nil, "", fmt.Errorf("unknown pickup '%s'", payload.PickupPointID)
		}
		station, ok := g.stationByID(pickup.StationID)
		if !ok {
			return nil, pickup, pickup.Name, fmt.Errorf("unknown station '%s'", pickup.StationID)
		}
		return station, pickup, pickup.Name, nil
	}

	if payload.StationID != "" {
		station, ok := g.stationByID(payload.StationID)
		if !ok {
			return nil, nil, "", fmt.Errorf("unknown station '%s'", payload.StationID)
		}
		return station, nil, strings.TrimSpace(payload.Destination), nil
	}

	text := strings.TrimSpace(payload.Address + " " + payload.Destination)
	if text == "" {
		return nil, nil, "", errors.New("address or pickup is required")
	}

	if pickup := g.matchPickupByText(text); pickup != nil {
		station, ok := g.stationByID(pickup.StationID)
		if !ok {
			return nil, pickup, pickup.Name, fmt.Errorf("unknown station '%s'", pickup.StationID)
		}
		return station, pickup, pickup.Name, nil
	}

	station, area := g.matchStationByText(text)
	if station == nil {
		return nil, nil, "", fmt.Errorf("could not infer station from '%s'", text)
	}
	var pickup *PickupPoint
	if area != "" {
		pickup = g.matchPickupByText(strings.ToLower(area))
	}
	return station, pickup, area, nil
}

func defaultDestination(station *Station) string {
	if station == nil {
		return ""
	}
	if len(station.NearbyAreas) > 0 {
		return station.NearbyAreas[0]
	}
	return station.Name
}

func (g *Gateway) upsertRiderLocked(riderID, name string, station *Station, destination string, pickup *PickupPoint) *Rider {
	arrival := time.Now().Add(7 * time.Minute)
	if riderID != "" {
		for i := range g.riders {
			if g.riders[i].ID == riderID {
				g.riders[i].Name = name
				g.riders[i].StationID = station.ID
				g.riders[i].Destination = destination
				g.riders[i].Status = "waiting"
				g.riders[i].ArrivalTime = arrival
				if pickup != nil {
					pp := *pickup
					g.riders[i].PickupPointID = pickup.ID
					g.riders[i].Pickup = &pp
				}
				return &g.riders[i]
			}
		}
	}

	newID := riderID
	if newID == "" {
		newID = fmt.Sprintf("rider-%d", time.Now().UnixNano())
	}
	var pickupCopy *PickupPoint
	pickupID := ""
	if pickup != nil {
		pp := *pickup
		pickupCopy = &pp
		pickupID = pickup.ID
	}

	rider := Rider{
		ID:            newID,
		Name:          name,
		Destination:   destination,
		ArrivalTime:   arrival,
		StationID:     station.ID,
		Status:        "waiting",
		PickupPointID: pickupID,
		Pickup:        pickupCopy,
	}
	g.riders = append([]Rider{rider}, g.riders...)
	return &g.riders[0]
}

func (g *Gateway) driverCandidatesLocked(station *Station, pickup *PickupPoint) []driverAttempt {
	if pickup == nil {
		return nil
	}
	attempts := make([]driverAttempt, 0, len(g.drivers))
	for i := range g.drivers {
		driver := &g.drivers[i]
		if station != nil && !routeContains(driver.Route.TargetStationIDs, station.ID) {
			continue
		}
		if plan, ok := g.driverPlans[driver.ID]; ok {
			if !plan.Active {
				continue
			}
			idx := indexOf(plan.PickupIDs, pickup.ID)
			if idx == -1 || idx < plan.CurrentIndex {
				continue
			}
			if plan.SeatsAvailable <= 0 {
				continue
			}
		} else if driver.SeatsAvailable <= 0 {
			continue
		}
		attempts = append(attempts, driverAttempt{
			DriverID:       driver.ID,
			DriverName:     driver.Name,
			DistanceMeters: driverDistanceToPickup(driver, pickup, station),
		})
	}
	sort.Slice(attempts, func(i, j int) bool {
		return attempts[i].DistanceMeters < attempts[j].DistanceMeters
	})
	return attempts
}

func indexOf(items []string, target string) int {
	for i, item := range items {
		if item == target {
			return i
		}
	}
	return -1
}

func (g *Gateway) matchStationByText(text string) (*Station, string) {
	lower := strings.ToLower(text)
	for i := range g.stations {
		station := &g.stations[i]
		for _, area := range station.NearbyAreas {
			if strings.Contains(lower, strings.ToLower(area)) {
				return station, area
			}
		}
		if strings.Contains(lower, strings.ToLower(station.Name)) {
			return station, station.Name
		}
	}
	return nil, ""
}

func (g *Gateway) matchPickupByText(text string) *PickupPoint {
	lower := strings.ToLower(text)
	for i := range g.pickupPoints {
		if strings.Contains(lower, strings.ToLower(g.pickupPoints[i].Name)) {
			pp := g.pickupPoints[i]
			return &pp
		}
	}
	return nil
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// SnapshotHandler returns the current snapshot expected by the mobile app.
func (g *Gateway) SnapshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, g.snapshot())
}

// MatchHandler triggers a match and returns the created trip.
func (g *Gateway) MatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload matchRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if payload.DriverID == "" || payload.StationID == "" {
		http.Error(w, "driverId and stationId are required", http.StatusBadRequest)
		return
	}

	trip, err := g.createTrip(payload.DriverID, payload.StationID)
	if err != nil {
		g.logger.Warn("match creation failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	g.queueRiderApproval(trip)

	writeJSON(w, http.StatusOK, trip)
}

func (g *Gateway) DriverRequestsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	driverID := r.URL.Query().Get("driverId")
	if driverID == "" {
		http.Error(w, "driverId required", http.StatusBadRequest)
		return
	}

	result, err := g.driverRequests(driverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (g *Gateway) DriverAcceptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload driverAcceptRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if payload.DriverID == "" || payload.RiderID == "" {
		http.Error(w, "driverId and riderId are required", http.StatusBadRequest)
		return
	}

	stationID, err := g.riderStationID(payload.RiderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	trip, err := g.createTripForRider(payload.DriverID, stationID, payload.RiderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	g.queueRiderApproval(trip)

	writeJSON(w, http.StatusOK, trip)
}

func (g *Gateway) acceptTripFromRealtime(driverID, riderID string) (Trip, error) {
	stationID, err := g.riderStationID(riderID)
	if err != nil {
		return Trip{}, err
	}
	trip, err := g.createTripForRider(driverID, stationID, riderID)
	if err != nil {
		return Trip{}, err
	}
	g.queueRiderApproval(trip)
	return trip, nil
}

func (g *Gateway) completeTrip(tripID string) (*Trip, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	var completed *Trip
	for i := range g.trips {
		if g.trips[i].ID == tripID {
			completed = &g.trips[i]
			break
		}
	}
	if completed == nil {
		return nil, fmt.Errorf("trip '%s' not found", tripID)
	}
	if completed.Status == "completed" {
		return completed, nil
	}

	completed.Status = "completed"
	completed.CompletedAt = time.Now().UTC()

	if driver, err := g.findDriver(completed.DriverID, ""); err == nil {
		driver.SeatsAvailable++
	}

	if plan, ok := g.driverPlans[completed.DriverID]; ok {
		if plan.SeatsAvailable < plan.SeatsTotal {
			plan.SeatsAvailable++
		}
	}

	if g.store != nil {
		g.store.RecordTrip(*completed)
		g.store.RecordTripEvent(tripID, "completed", map[string]any{
			"riderId": completed.RiderID,
		})
		g.store.UpdateRiderRequestStatus(completed.RiderID, "completed", completed.DriverID, tripID)
	}
	return completed, nil
}

func (g *Gateway) BookRideHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload bookRideRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	result, err := g.bookRide(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (g *Gateway) PickupPointsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	response := struct {
		PickupPoints []PickupPoint `json:"pickupPoints"`
	}{PickupPoints: g.pickupPoints}
	writeJSON(w, http.StatusOK, response)
}

func (g *Gateway) DriverRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload driverRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	result, err := g.configureDriverRoute(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (g *Gateway) DriverTripStartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload startTripRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	result, err := g.startDriverTrip(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (g *Gateway) SignUpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req userpb.SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.SignUp(r.Context(), &req)
	if err != nil {
		g.logger.Error("signup failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (g *Gateway) SignInHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req userpb.SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.SignIn(r.Context(), &req)
	if err != nil {
		g.logger.Error("signin failed", "err", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (g *Gateway) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req userpb.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.ForgotPassword(r.Context(), &req)
	if err != nil {
		g.logger.Error("forgot password failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (g *Gateway) LocationStreamHandler(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Query().Get("driverId")
	if driverID == "" {
		http.Error(w, "driverId required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		g.logger.Error("websocket upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	if g.locationClient == nil {
		g.logger.Error("location client not configured")
		return
	}

	stream, err := g.locationClient.SubscribeLocationUpdates(r.Context(), &locationpb.SubscribeLocationRequest{DriverId: driverID})
	if err != nil {
		g.logger.Error("failed to subscribe to location updates", "err", err)
		return
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			g.logger.Info("location stream ended", "err", err)
			break
		}

		if err := conn.WriteJSON(update); err != nil {
			g.logger.Info("websocket write failed", "err", err)
			break
		}
	}
}

func (g *Gateway) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.GetUser(r.Context(), &userpb.GetUserRequest{Id: userID})
	if err != nil {
		g.logger.Error("get user failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

type pushTokenRequest struct {
	UserID string `json:"userId"`
	Token  string `json:"token"`
}

func (g *Gateway) NotificationTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload pushTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.UserID) == "" || strings.TrimSpace(payload.Token) == "" {
		http.Error(w, "userId and token required", http.StatusBadRequest)
		return
	}
	g.setPushToken(payload.UserID, payload.Token)
	w.WriteHeader(http.StatusNoContent)
}

func (g *Gateway) afterTripFinalized(trip Trip, rider *Rider, pickup *PickupPoint, station *Station) {
	if g.store != nil {
		g.store.RecordTrip(trip)
		g.store.RecordTripEvent(trip.ID, "matched", map[string]any{
			"driverId": trip.DriverID,
			"riderId":  trip.RiderID,
		})
		g.store.UpdateRiderRequestStatus(trip.RiderID, "matched", trip.DriverID, trip.ID)
	}
	if g.hub != nil {
		go func(tp Trip, riderCopy *Rider, pickupCopy *PickupPoint, stationPtr *Station) {
			g.hub.CreateRoomForTrip(tp, pickupCopy, stationPtr, riderCopy)
			g.hub.RefreshDriverQueue(tp.DriverID)
		}(trip, rider, pickup, station)
	}
	g.sendPushNotification(trip.DriverID, "Rider confirmed", fmt.Sprintf("%s confirmed pickup", g.riderDisplayName(rider)), map[string]any{
		"tripId":  trip.ID,
		"riderId": trip.RiderID,
	})
}

func (g *Gateway) queueRiderApproval(trip Trip) {
	rider := g.riderSnapshot(trip.RiderID)
	pickupCopy := copyPickupPoint(trip.PickupPoint)
	var stationCopy *Station
	if station, ok := g.stationByID(trip.StationID); ok {
		stationCopy = copyStation(station)
	}

	g.mu.Lock()
	g.pendingTrips[trip.ID] = &pendingTripContext{Trip: trip, Rider: rider, Pickup: pickupCopy, Station: stationCopy}
	g.mu.Unlock()

	g.sendPushNotification(trip.RiderID, "Driver ready for pickup", fmt.Sprintf("%s is ready near %s", g.driverDisplayName(trip.DriverID), pickupName(pickupCopy)), map[string]any{
		"tripId": trip.ID,
		"driverId": trip.DriverID,
	})

	if g.hub != nil && rider != nil {
		g.hub.RequestRiderApproval(trip, pickupCopy, stationCopy, rider)
		return
	}

	go func(id string) {
		if _, err := g.finalizePendingTrip(id); err != nil {
			g.logger.Warn("finalize trip fallback failed", "tripId", id, "err", err)
		}
	}(trip.ID)
}

func (g *Gateway) finalizePendingTrip(tripID string) (*Trip, error) {
	g.mu.Lock()
	ctx, ok := g.pendingTrips[tripID]
	if !ok {
		g.mu.Unlock()
		return nil, fmt.Errorf("trip '%s' not pending approval", tripID)
	}
	delete(g.pendingTrips, tripID)
	var tripPtr *Trip
	for i := range g.trips {
		if g.trips[i].ID == tripID {
			g.trips[i].Status = "pending"
			tripPtr = &g.trips[i]
			break
		}
	}
	if tripPtr == nil {
		g.mu.Unlock()
		return nil, fmt.Errorf("trip '%s' not found", tripID)
	}
	trip := *tripPtr
	g.mu.Unlock()

	if g.hub != nil {
		g.hub.ClearApproval(tripID)
	}
	g.afterTripFinalized(trip, ctx.Rider, ctx.Pickup, ctx.Station)
	return &trip, nil
}

func (g *Gateway) cancelPendingTrip(tripID, reason string) error {
	g.mu.Lock()
	ctx, ok := g.pendingTrips[tripID]
	if !ok {
		g.mu.Unlock()
		return fmt.Errorf("trip '%s' not pending approval", tripID)
	}
	delete(g.pendingTrips, tripID)
	if ctx.Rider != nil {
		for i := range g.riders {
			if g.riders[i].ID == ctx.Rider.ID {
				g.riders[i].Status = "waiting"
			}
		}
	}
	for i := range g.trips {
		if g.trips[i].ID == tripID {
			g.trips = append(g.trips[:i], g.trips[i+1:]...)
			break
		}
	}
	if driver, err := g.findDriver(ctx.Trip.DriverID, ""); err == nil {
		driver.SeatsAvailable++
	}
	if plan, ok := g.driverPlans[ctx.Trip.DriverID]; ok {
		if plan.SeatsAvailable < plan.SeatsTotal {
			plan.SeatsAvailable++
		}
	}
	g.mu.Unlock()

	if g.store != nil {
		g.store.RecordTripEvent(tripID, "rider_declined", map[string]any{"reason": reason})
	}
	if g.hub != nil {
		g.hub.ClearApproval(tripID)
		g.hub.NotifyDriverTripCancelled(ctx.Trip.DriverID, tripID, reason)
	}
	g.sendPushNotification(ctx.Trip.DriverID, "Ride cancelled", cancelReasonMessage(reason), map[string]any{"tripId": tripID})
	return nil
}

func cancelReasonMessage(reason string) string {
	switch reason {
	case "rider_timeout":
		return "Rider approval timed out"
	case "rider_declined":
		return "Rider declined the trip"
	default:
		return reason
	}
}

func (g *Gateway) FinalizeTrip(tripID string) error {
	_, err := g.finalizePendingTrip(tripID)
	return err
}

func (g *Gateway) CancelTrip(tripID, reason string) error {
	return g.cancelPendingTrip(tripID, reason)
}

func (g *Gateway) publishDriverLocation(driverID string, lat, lon float64) {
	if g.hub != nil {
		g.hub.BroadcastLocation(driverID, lat, lon)
	}
}

func (g *Gateway) handlePickupCheckpoint(driverID string, pickup *PickupPoint) {
	if pickup == nil {
		return
	}

	if g.hub != nil {
		g.hub.PickupArrived(driverID, pickup.ID)
	}

	var tripCopy *Trip
	g.mu.Lock()
	for i := range g.trips {
		trip := &g.trips[i]
		if trip.DriverID == driverID && trip.PickupPointID == pickup.ID && trip.Status == "pending" {
			trip.Status = "in_progress"
			trip.CreatedAt = time.Now().UTC()
			tripCopy = copyTrip(trip)
			break
		}
	}
	g.mu.Unlock()

	if tripCopy != nil && g.store != nil {
		g.store.RecordTrip(*tripCopy)
		g.store.RecordTripEvent(tripCopy.ID, "pickup_reached", map[string]any{"pickupId": pickup.ID})
	}
}

func (g *Gateway) maybeCompleteTrips(driverID string, lat, lon float64) {
	g.mu.Lock()
	completed := make([]Trip, 0)
	for i := range g.trips {
		trip := &g.trips[i]
		if trip.DriverID != driverID || trip.Status != "in_progress" {
			continue
		}
		station, ok := g.stationByID(trip.StationID)
		if !ok {
			continue
		}
		if haversineMeters(lat, lon, station.Latitude, station.Longitude) <= 150 {
			trip.Status = "completed"
			trip.CompletedAt = time.Now().UTC()
			if driver, err := g.findDriver(driverID, ""); err == nil {
				driver.SeatsAvailable++
			}
			if plan, ok := g.driverPlans[driverID]; ok && plan.SeatsAvailable < plan.SeatsTotal {
				plan.SeatsAvailable++
			}
			completed = append(completed, *trip)
		}
	}
	g.mu.Unlock()

	for _, trip := range completed {
		if g.store != nil {
			g.store.RecordTrip(trip)
			g.store.RecordTripEvent(trip.ID, "dropoff_reached", map[string]any{"stationId": trip.StationID})
			g.store.UpdateRiderRequestStatus(trip.RiderID, "completed", trip.DriverID, trip.ID)
		}
		if g.hub != nil {
			g.hub.CompleteTrip(trip.ID, "auto-complete")
		}
	}
}

func copyPickupPoint(p *PickupPoint) *PickupPoint {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

func copyStation(s *Station) *Station {
	if s == nil {
		return nil
	}
	cp := *s
	return &cp
}

func copyRider(r *Rider) Rider {
	if r == nil {
		return Rider{}
	}
	cp := *r
	if r.Pickup != nil {
		cpPickup := *r.Pickup
		cp.Pickup = &cpPickup
	}
	return cp
}

func copyTrip(trip *Trip) *Trip {
	if trip == nil {
		return nil
	}
	cp := *trip
	cp.PickupPoint = copyPickupPoint(trip.PickupPoint)
	return &cp
}

func compatibleDestination(driverDest, riderDest string) bool {
	driverNorm := normalizeDestination(driverDest)
	riderNorm := normalizeDestination(riderDest)
	if driverNorm == "" || driverNorm == "unknown" || riderNorm == "" {
		return true
	}
	if driverNorm == riderNorm {
		return true
	}
	if strings.Contains(driverNorm, riderNorm) || strings.Contains(riderNorm, driverNorm) {
		return true
	}
	return false
}

func normalizeDestination(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func (g *Gateway) riderSnapshot(id string) *Rider {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.riders {
		if g.riders[i].ID == id {
			copy := g.riders[i]
			if copy.Pickup != nil {
				pp := *copy.Pickup
				copy.Pickup = &pp
			}
			return &copy
		}
	}
	return nil
}

func (g *Gateway) setPushToken(userID, token string) {
	g.mu.Lock()
	g.pushTokens[userID] = token
	g.mu.Unlock()
}

func (g *Gateway) getPushToken(userID string) string {
	g.mu.Lock()
	token := g.pushTokens[userID]
	g.mu.Unlock()
	return token
}

func (g *Gateway) driverDisplayName(driverID string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.drivers {
		if g.drivers[i].ID == driverID {
			return g.drivers[i].Name
		}
	}
	return "Driver"
}

func (g *Gateway) riderDisplayName(rider *Rider) string {
	if rider == nil || rider.Name == "" {
		return "Rider"
	}
	return rider.Name
}

func (g *Gateway) sendPushNotification(userID, title, body string, data map[string]any) {
	token := g.getPushToken(userID)
	if token == "" {
		return
	}
	payload := map[string]any{
		"to":    token,
		"title": title,
		"body":  body,
		"sound": "default",
	}
	if data != nil {
		payload["data"] = data
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		g.logger.Warn("push marshal failed", "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://exp.host/--/api/v2/push/send", bytes.NewReader(buf))
	if err != nil {
		g.logger.Warn("push request build failed", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		g.logger.Warn("push send failed", "err", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		g.logger.Warn("push send non-2xx", "status", resp.StatusCode)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (g *Gateway) UpdateLocationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DriverID  string  `json:"driverId"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	stream, err := g.locationClient.UpdateLocation(r.Context())
	if err != nil {
		g.logger.Error("failed to start location update stream", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = stream.Send(&locationpb.UpdateLocationRequest{
		Location: &locationpb.Location{
			DriverId:  req.DriverID,
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
		},
	})
	if err != nil {
		g.logger.Error("failed to send location update", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	passed := g.recordDriverLocationProgress(req.DriverID, req.Latitude, req.Longitude)

	// Close and receive response
	_, err = stream.CloseAndRecv()
	if err != nil {
		g.logger.Error("failed to close location update stream", "err", err)
		// Don't fail the request if we sent the update successfully
	}

	g.publishDriverLocation(req.DriverID, req.Latitude, req.Longitude)
	if passed != nil {
		g.handlePickupCheckpoint(req.DriverID, passed)
	}
	g.maybeCompleteTrips(req.DriverID, req.Latitude, req.Longitude)

	w.WriteHeader(http.StatusOK)
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	driverpb "lastmile/gen/go/driver"
	gatewaypb "lastmile/gen/go/gateway"
	locationpb "lastmile/gen/go/location"
	matchingpb "lastmile/gen/go/matching"
	userpb "lastmile/gen/go/user"
	"lastmile/internal/pkg/geo"

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
	// Bot riders are seeded demo riders that auto-confirm a driver's offer.
	Bot bool `json:"bot,omitempty"`
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
	PickedUpAt    time.Time    `json:"pickedUpAt,omitempty"`
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
	Name           string
	CarDetails     string
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
	// Rejections lists drivers the Matching service ruled out, and why.
	Rejections []driverAttempt `json:"rejections,omitempty"`
	// MatchedBy is the Matching replica (pod) that answered.
	MatchedBy string `json:"matchedBy,omitempty"`
}

type driverRouteRequest struct {
	DriverID       string   `json:"driverId"`
	Name           string   `json:"name"`
	CarDetails     string   `json:"carDetails"`
	PickupPointIDs []string `json:"pickupPointIds"`
	Seats          int      `json:"seats"`
	Destination    string   `json:"destination"`
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

// Gateway is LastMile's API gateway / backend-for-frontend: it serves the
// web and mobile clients over HTTP + socket.io and talks gRPC to the services.
//
// Lock discipline: g.mu guards the fields below. Never hold it while making
// a network call (gRPC, Postgres, push) — a slow downstream would freeze
// every request in the gateway. Copy what you need, unlock, then call.
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
	matchingClient matchingpb.MatchingServiceClient
	hub            *RealtimeHub
	store          *Persistence
	pendingTrips   map[string]*pendingTripContext
	pushTokens     map[string]string

	// lastPos caches every driver's last reported position, so a Location
	// service restart doesn't wipe drivers off the map.
	lastPos map[string][2]float64
	// lastMatchedBy / matchedByCount record which Matching replicas answered.
	lastMatchedBy  string
	matchedByCount map[string]int
	// driverServiceUp is false while the Driver service is unreachable; the
	// snapshot is then served from the last good copy.
	driverServiceUp bool
	statusChecker   *StatusChecker

	// Simulation pacing (shortened in tests).
	simTick       time.Duration
	simStopPause  time.Duration
	simPickupWait time.Duration
}

func NewGateway(logger *slog.Logger, driverClient driverpb.DriverServiceClient, locClient locationpb.LocationServiceClient, userClient userpb.UserServiceClient) *Gateway {
	l := slog.Default()
	if logger != nil {
		l = logger
	}

	stations := defaultStations()

	return &Gateway{
		logger:          l.With("component", "gateway"),
		drivers:         []Driver{}, // refreshed from the Driver service
		riders:          seedRiders(time.Now()),
		trips:           []Trip{},
		stations:        stations,
		pickupPoints:    pickupPointsFor(stations),
		driverPlans:     make(map[string]*driverPlan),
		driverClient:    driverClient,
		locationClient:  locClient,
		userClient:      userClient,
		pendingTrips:    make(map[string]*pendingTripContext),
		pushTokens:      make(map[string]string),
		lastPos:         make(map[string][2]float64),
		matchedByCount:  make(map[string]int),
		driverServiceUp: true,
		simTick:         700 * time.Millisecond,
		simStopPause:    2 * time.Second,
		simPickupWait:   20 * time.Second,
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

// refreshDrivers pulls driver profiles from the Driver service and positions
// from the Location service, then rebuilds g.drivers.
//
// Both calls happen WITHOUT holding g.mu and with a short deadline. The
// original version held the gateway-wide lock across an un-timed gRPC call,
// so one slow Driver pod froze every HTTP request in the system.
func (g *Gateway) refreshDrivers() {
	if g.driverClient == nil {
		g.refreshFromPlansOnly()
		return
	}

	var profiles []*driverpb.Driver
	var routes []*driverpb.Route
	driverUp := false

	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	resp, err := g.driverClient.ListDrivers(ctx, &driverpb.ListDriversRequest{})
	cancel()
	if err != nil {
		g.logger.Warn("driver service unavailable; serving cached drivers", "err", err)
	} else {
		profiles, routes, driverUp = resp.Drivers, resp.Routes, true
	}

	g.mu.Lock()
	g.driverServiceUp = driverUp
	if !driverUp {
		g.mu.Unlock()
		return // keep the last good g.drivers
	}

	ids := make([]string, 0, len(profiles))
	for _, d := range profiles {
		ids = append(ids, d.Id)
	}
	g.mu.Unlock()

	locMap := make(map[string]*locationpb.Location)
	if g.locationClient != nil && len(ids) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		locResp, err := g.locationClient.GetDriverLocations(ctx, &locationpb.GetDriverLocationsRequest{DriverIds: ids})
		cancel()
		if err != nil {
			g.logger.Warn("location service unavailable; using cached positions", "err", err)
		} else {
			for _, loc := range locResp.Locations {
				locMap[loc.DriverId] = loc
			}
		}
	}

	g.mu.Lock()
	routeByDriver := make(map[string]*driverpb.Route)
	for _, r := range routes {
		routeByDriver[r.DriverId] = r
	}

	seen := make(map[string]bool, len(profiles))
	drivers := make([]Driver, 0, len(profiles))
	for _, d := range profiles {
		seen[d.Id] = true
		lat, lon := 0.0, 0.0
		if loc, ok := locMap[d.Id]; ok {
			lat, lon = loc.Latitude, loc.Longitude
		}
		drivers = append(drivers, g.driverViewLocked(d.Id, d.Name, d.CarDetails, routeByDriver[d.Id], lat, lon))
	}

	// Self-healing: the Driver service keeps state in memory, so if its pod
	// restarts it forgets everyone. Drivers we still have plans for are kept
	// visible and re-registered in the background.
	var orphans []*driverPlan
	for id, plan := range g.driverPlans {
		if seen[id] {
			continue
		}
		drivers = append(drivers, g.driverViewLocked(id, plan.Name, plan.CarDetails, nil, 0, 0))
		cp := *plan
		orphans = append(orphans, &cp)
	}
	sort.Slice(drivers, func(i, j int) bool { return drivers[i].ID < drivers[j].ID })
	g.drivers = drivers
	g.mu.Unlock()

	for _, plan := range orphans {
		g.logger.Info("re-registering driver lost by Driver service", "driverId", plan.DriverID)
		go g.ensureDriverProfile(driverRouteRequest{DriverID: plan.DriverID, Name: plan.Name, CarDetails: plan.CarDetails},
			plan.TargetStations, plan.Destination, plan.SeatsTotal)
	}
}

// refreshFromPlansOnly is used when no Driver service is wired (unit tests,
// standalone runs): keep the drivers we have, refresh their plan-derived
// fields, and add any driver that only exists as a plan.
func (g *Gateway) refreshFromPlansOnly() {
	g.mu.Lock()
	defer g.mu.Unlock()
	have := make(map[string]bool, len(g.drivers))
	for i, d := range g.drivers {
		have[d.ID] = true
		if _, ok := g.driverPlans[d.ID]; ok {
			g.drivers[i] = g.driverViewLocked(d.ID, d.Name, d.CarDetails, nil, d.Latitude, d.Longitude)
		}
	}
	for id, plan := range g.driverPlans {
		if !have[id] {
			g.drivers = append(g.drivers, g.driverViewLocked(id, plan.Name, plan.CarDetails, nil, 0, 0))
		}
	}
}

// driverViewLocked merges a driver profile with the gateway's trip plan.
func (g *Gateway) driverViewLocked(id, name, car string, route *driverpb.Route, lat, lon float64) Driver {
	view := Route{}
	seats := 0
	if plan := g.driverPlans[id]; plan != nil {
		view = Route{
			ID:               plan.DriverID,
			TargetStationIDs: append([]string{}, plan.TargetStations...),
			Destination:      plan.Destination,
			PickupPoints:     g.pickupPointsForIDs(plan.PickupIDs),
		}
		seats = plan.SeatsAvailable
		if name == "" {
			name = plan.Name
		}
	} else if route != nil {
		view = Route{
			ID:               route.Id,
			TargetStationIDs: append([]string{}, route.TargetStationIds...),
			Destination:      route.Destination,
		}
		seats = int(route.AvailableSeats)
	}
	if view.Destination == "" {
		view.Destination = "Unknown"
	}
	if name == "" {
		name = "Driver"
	}
	if plat, plon, ok := g.positionLocked(id, lat, lon); ok {
		lat, lon = plat, plon
	}
	status := "idle"
	if plan := g.driverPlans[id]; plan != nil && plan.Active {
		status = "active"
	}
	return Driver{
		ID: id, Name: name, CarDetails: car, SeatsAvailable: seats, ETAMinutes: 5,
		Status: status, Route: view, Latitude: lat, Longitude: lon,
	}
}

// positionLocked prefers a fresh position, falling back to the gateway cache.
func (g *Gateway) positionLocked(driverID string, lat, lon float64) (float64, float64, bool) {
	if geo.Known(lat, lon) {
		return lat, lon, true
	}
	if p, ok := g.lastPos[driverID]; ok {
		return p[0], p[1], true
	}
	return 0, 0, false
}

func (g *Gateway) snapshot() BackendSnapshot {
	g.refreshDrivers()

	g.mu.Lock()
	defer g.mu.Unlock()

	var highlight *Trip
	if len(g.trips) > 0 {
		t := *copyTrip(&g.trips[0])
		highlight = &t
	}

	return BackendSnapshot{
		Drivers:       append([]Driver{}, g.drivers...),
		Riders:        append([]Rider{}, g.riders...),
		Trips:         append([]Trip{}, g.trips...),
		Stations:      append([]Station{}, g.stations...),
		Metrics:       g.metrics(),
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

// driverRequests lists the waiting riders this driver can pick up. The
// decision is made by the Matching service (FindRiders), so the driver's
// queue and the rider's booking always apply the same rules.
func (g *Gateway) driverRequests(driverID string) (driverRequestsResponse, error) {
	g.refreshDrivers()

	g.mu.Lock()
	driver, err := g.findDriver(driverID, "")
	if err != nil {
		g.mu.Unlock()
		return driverRequestsResponse{}, err
	}
	state := g.driverStateLocked(driver)
	summary := driverSummary{
		ID:             driver.ID,
		Name:           driver.Name,
		SeatsAvailable: int(state.SeatsAvailable),
		NextStop:       g.nextStopNameLocked(driver),
	}

	byID := make(map[string]Rider)
	riderReqs := make([]*matchingpb.RiderRequest, 0)
	for i := range g.riders {
		r := &g.riders[i]
		if r.Status != "waiting" {
			continue
		}
		byID[r.ID] = copyRider(r)
		riderReqs = append(riderReqs, riderRequestFor(r))
	}
	g.mu.Unlock()

	out := driverRequestsResponse{Driver: summary, Requests: []riderRequestView{}, GeneratedAt: time.Now().UTC()}
	if len(riderReqs) == 0 {
		return out, nil
	}

	found, err := g.findRiders(state, riderReqs)
	if err != nil {
		return out, err
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	for _, c := range found.Riders {
		r, ok := byID[c.RiderId]
		if !ok {
			continue
		}
		station, ok := g.stationByID(r.StationID)
		if !ok {
			continue
		}
		pickup := r.Pickup
		if pickup == nil && r.PickupPointID != "" {
			pickup, _ = g.pickupByID(r.PickupPointID)
		}
		out.Requests = append(out.Requests, riderRequestView{
			ID:             r.ID,
			Name:           r.Name,
			Destination:    r.Destination,
			ArrivalTime:    r.ArrivalTime,
			Station:        *station,
			Pickup:         pickup,
			Status:         r.Status,
			DistanceMeters: c.DistanceMeters,
		})
	}
	return out, nil
}

// nextStopNameLocked names the next pickup on the driver's plan.
func (g *Gateway) nextStopNameLocked(d *Driver) string {
	if plan, ok := g.driverPlans[d.ID]; ok && plan.CurrentIndex < len(plan.PickupIDs) {
		if pp, ok := g.pickupByID(plan.PickupIDs[plan.CurrentIndex]); ok {
			return pp.StationName
		}
	}
	if len(d.Route.TargetStationIDs) > 0 {
		if st, ok := g.stationByID(d.Route.TargetStationIDs[0]); ok {
			return st.Name
		}
		return d.Route.TargetStationIDs[0]
	}
	return ""
}

func (g *Gateway) bookRide(payload bookRideRequest) (bookRideResponse, error) {
	command := strings.TrimSpace(strings.ToLower(payload.Command))
	if command != "book" {
		return bookRideResponse{}, fmt.Errorf("unsupported command '%s'", payload.Command)
	}

	g.mu.Lock()
	station, pickup, inferredArea, err := g.resolveStation(payload)
	g.mu.Unlock()
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

	g.refreshDrivers()

	g.mu.Lock()
	rider := g.upsertRiderLocked(payload.RiderID, name, station, requestedDestination, pickup)
	riderReq := riderRequestFor(rider)
	states := make([]*matchingpb.DriverState, 0, len(g.drivers))
	names := make(map[string]string, len(g.drivers))
	for i := range g.drivers {
		states = append(states, g.driverStateLocked(&g.drivers[i]))
		names[g.drivers[i].ID] = g.drivers[i].Name
	}
	riderSnapshot := copyRider(rider)
	g.mu.Unlock()

	stationCopy := copyStation(station)
	pickupCopy := copyPickupPoint(pickup)

	if g.store != nil {
		go g.store.RecordRiderRequest(riderSnapshot, pickupCopy, "waiting")
	}

	// Ask the Matching service who can take this rider.
	ranked, err := g.rankDrivers(riderReq, states)
	if err != nil {
		return bookRideResponse{
			Status:               "matching_unavailable",
			Message:              "Matching is busy right now. You're on the list; try again in a few seconds.",
			Rider:                riderSnapshot,
			Station:              *station,
			Pickup:               pickupCopy,
			RequestedDestination: requestedDestination,
			Attempts:             []driverAttempt{},
		}, nil
	}

	attempts := make([]driverAttempt, 0, len(ranked.Candidates))
	for _, c := range ranked.Candidates {
		attempts = append(attempts, driverAttempt{DriverID: c.DriverId, DriverName: c.Name, DistanceMeters: c.DistanceMeters})
	}
	rejections := make([]driverAttempt, 0, len(ranked.Rejections))
	for _, r := range ranked.Rejections {
		rejections = append(rejections, driverAttempt{DriverID: r.Id, DriverName: names[r.Id], Reason: r.Reason, DistanceMeters: -1})
	}

	var trip *Trip
	var acceptedIndex = -1

	if g.hub == nil {
		for i := range attempts {
			result, err := g.createTripForRider(attempts[i].DriverID, station.ID, riderSnapshot.ID)
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
	if len(rejections) > 0 {
		message += fmt.Sprintf(" (%d ruled out, e.g. %s: %s)", len(rejections), rejections[0].DriverName, rejections[0].Reason)
	}
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
		Rejections:           rejections,
		MatchedBy:            ranked.GetServedBy(),
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
	return nil, fmt.Errorf("driver '%s': %w", driverID, errDriverNotFound)
}

// errDriverNotFound means the driver hasn't planned a route yet.
var errDriverNotFound = errors.New("driver not found (plan a route first)")

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
	return geo.HaversineMeters(lat1, lon1, lat2, lon2)
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
	if errors.Is(err, errMatchingUnavailable) {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
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
		g.logger.Warn("accept trip failed", "driverId", payload.DriverID, "riderId", payload.RiderID, "err", err)
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

// completeTrip marks a trip as dropped off and frees the seat. It is
// idempotent: completing an already-completed trip is a no-op.
func (g *Gateway) completeTrip(tripID string) (*Trip, error) {
	g.mu.Lock()
	var completed *Trip
	for i := range g.trips {
		if g.trips[i].ID == tripID {
			completed = &g.trips[i]
			break
		}
	}
	if completed == nil {
		g.mu.Unlock()
		return nil, fmt.Errorf("trip '%s' not found", tripID)
	}
	if completed.Status == "completed" {
		out := copyTrip(completed)
		g.mu.Unlock()
		return out, nil
	}

	completed.Status = "completed"
	completed.CompletedAt = time.Now().UTC()
	g.setRiderStatusLocked(completed.RiderID, "completed")
	if driver, err := g.findDriver(completed.DriverID, ""); err == nil {
		driver.SeatsAvailable++
	}
	if plan, ok := g.driverPlans[completed.DriverID]; ok && plan.SeatsAvailable < plan.SeatsTotal {
		plan.SeatsAvailable++
	}
	out := copyTrip(completed)
	g.mu.Unlock()

	// Database writes happen after unlocking (never do I/O under g.mu).
	if g.store != nil {
		g.store.RecordTrip(*out)
		g.store.RecordTripEvent(tripID, "completed", map[string]any{"riderId": out.RiderID})
		g.store.UpdateRiderRequestStatus(out.RiderID, "completed", out.DriverID, tripID)
	}
	return out, nil
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
	g.mu.Lock()
	points := append([]PickupPoint(nil), g.pickupPoints...)
	g.mu.Unlock()
	response := struct {
		PickupPoints []PickupPoint `json:"pickupPoints"`
	}{PickupPoints: points}
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

	g.mu.Lock()
	var stationCopy *Station
	if station, ok := g.stationByID(trip.StationID); ok {
		stationCopy = copyStation(station)
	}
	g.pendingTrips[trip.ID] = &pendingTripContext{Trip: trip, Rider: rider, Pickup: pickupCopy, Station: stationCopy}
	g.mu.Unlock()

	g.sendPushNotification(trip.RiderID, "Driver ready for pickup", fmt.Sprintf("%s is ready near %s", g.driverDisplayName(trip.DriverID), pickupName(pickupCopy)), map[string]any{
		"tripId":   trip.ID,
		"driverId": trip.DriverID,
	})

	if g.hub != nil && rider != nil && !rider.Bot {
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
	g.mu.Lock()
	g.lastPos[driverID] = [2]float64{lat, lon}
	for i := range g.drivers {
		if g.drivers[i].ID == driverID {
			g.drivers[i].Latitude, g.drivers[i].Longitude = lat, lon
		}
	}
	g.mu.Unlock()
	if g.hub != nil {
		g.hub.BroadcastLocation(driverID, lat, lon)
	}
}

// handlePickupCheckpoint marks every confirmed trip at this pickup as
// picked up. (It used to stop after the first one, stranding any second
// rider waiting at the same station.)
func (g *Gateway) handlePickupCheckpoint(driverID string, pickup *PickupPoint) {
	if pickup == nil {
		return
	}

	if g.hub != nil {
		g.hub.PickupArrived(driverID, pickup.ID)
	}

	var picked []Trip
	g.mu.Lock()
	now := time.Now().UTC()
	for i := range g.trips {
		trip := &g.trips[i]
		if trip.DriverID == driverID && trip.PickupPointID == pickup.ID && trip.Status == "pending" {
			trip.Status = "in_progress"
			trip.PickedUpAt = now
			picked = append(picked, *copyTrip(trip))
			for j := range g.riders {
				if g.riders[j].ID == trip.RiderID {
					g.riders[j].Status = "picked_up"
				}
			}
		}
	}
	g.mu.Unlock()

	for _, t := range picked {
		g.logger.Info("rider picked up", "tripId", t.ID, "riderId", t.RiderID, "pickup", pickup.ID)
		if g.store != nil {
			g.store.RecordTrip(t)
			g.store.RecordTripEvent(t.ID, "pickup_reached", map[string]any{"pickupId": pickup.ID})
		}
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
		station, _ := g.stationByID(trip.StationID)
		dropLat, dropLon, ok := dropoffFor(trip.Destination, station)
		if !ok {
			continue
		}
		// Drop-off is at the rider's DESTINATION. (The original code checked
		// the pickup station, so every trip "completed" the instant the rider
		// got in.)
		if haversineMeters(lat, lon, dropLat, dropLon) <= 150 {
			trip.Status = "completed"
			trip.CompletedAt = time.Now().UTC()
			if driver, err := g.findDriver(driverID, ""); err == nil {
				driver.SeatsAvailable++
			}
			if plan, ok := g.driverPlans[driverID]; ok && plan.SeatsAvailable < plan.SeatsTotal {
				plan.SeatsAvailable++
			}
			g.setRiderStatusLocked(trip.RiderID, "completed")
			g.logger.Info("rider dropped off", "tripId", trip.ID, "riderId", trip.RiderID, "destination", trip.Destination)
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

func (g *Gateway) setRiderStatusLocked(riderID, status string) {
	for i := range g.riders {
		if g.riders[i].ID == riderID {
			g.riders[i].Status = status
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

	if strings.TrimSpace(req.DriverID) == "" {
		http.Error(w, "driverId required", http.StatusBadRequest)
		return
	}
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		http.Error(w, "latitude/longitude out of range", http.StatusBadRequest)
		return
	}

	// Same path as simulated pings: Location service, realtime fan-out,
	// pickup/drop-off checkpoints and the proximity trigger.
	g.ingestLocation(r.Context(), req.DriverID, req.Latitude, req.Longitude)
	w.WriteHeader(http.StatusOK)
}

// TripPickupHandler lets the driver confirm a pickup by hand ("rider is in
// the car"). For simulated trips the car jumps to the pickup and carries on.
func (g *Gateway) TripPickupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tripID := r.URL.Query().Get("tripId")
	if tripID == "" {
		http.Error(w, "tripId required", http.StatusBadRequest)
		return
	}

	// Everything is read and written under one lock, via the slice index.
	// (The old code kept a *Trip into g.trips across unlock/relock; any
	// append in between reallocates the slice, so the write went to a stale
	// copy and the pickup was silently lost.)
	g.mu.Lock()
	idx := -1
	for i := range g.trips {
		if g.trips[i].ID == tripID {
			idx = i
			break
		}
	}
	if idx < 0 {
		g.mu.Unlock()
		http.Error(w, "trip not found", http.StatusNotFound)
		return
	}
	trip := &g.trips[idx]
	switch trip.Status {
	case "pending", "awaiting_pickup", "in_progress":
	default:
		status := trip.Status
		g.mu.Unlock()
		http.Error(w, fmt.Sprintf("trip is %s", status), http.StatusConflict)
		return
	}
	alreadyPicked := trip.Status == "in_progress"
	trip.Status = "in_progress"
	if trip.PickedUpAt.IsZero() {
		trip.PickedUpAt = time.Now().UTC()
	}
	tripCopy := *copyTrip(trip)
	pickup, hasPickup := g.pickupByID(trip.PickupPointID)

	var simCtx context.Context
	var remaining []PickupPoint
	if plan, ok := g.driverPlans[trip.DriverID]; ok && plan.Simulated && !alreadyPicked {
		if plan.simCancel != nil {
			plan.simCancel()
		}
		var cancel context.CancelFunc
		simCtx, cancel = context.WithCancel(context.Background())
		plan.simCancel = cancel
		// Skip past this pickup, then continue the rest of the plan.
		if i := indexOf(plan.PickupIDs, trip.PickupPointID); i >= plan.CurrentIndex {
			plan.CurrentIndex = i + 1
		}
		remaining = g.pickupPointsForIDs(plan.PickupIDs[min(plan.CurrentIndex, len(plan.PickupIDs)):])
	}
	g.mu.Unlock()

	if !alreadyPicked {
		if g.hub != nil {
			g.hub.PickupArrived(tripCopy.DriverID, tripCopy.PickupPointID)
		}
		if hasPickup {
			g.publishDriverLocation(tripCopy.DriverID, pickup.Latitude, pickup.Longitude)
		}
		if g.store != nil {
			g.store.RecordTrip(tripCopy)
			g.store.RecordTripEvent(tripCopy.ID, "pickup_confirmed", map[string]any{"pickupId": tripCopy.PickupPointID})
		}
	}
	if simCtx != nil {
		go g.runSimulatedTrip(simCtx, tripCopy.DriverID, remaining)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "picked_up"})
}

func (g *Gateway) TripDropoffHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tripID := r.URL.Query().Get("tripId")
	if tripID == "" {
		http.Error(w, "tripId required", http.StatusBadRequest)
		return
	}

	completed, err := g.completeTrip(tripID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if g.hub != nil {
		g.hub.CompleteTrip(tripID, "manual_dropoff")
	}

	writeJSON(w, http.StatusOK, completed)
}

func (g *Gateway) SimulateTripHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tripID := r.URL.Query().Get("tripId")
	if tripID == "" {
		http.Error(w, "tripId required", http.StatusBadRequest)
		return
	}

	g.mu.Lock()
	var targetTrip *Trip
	for i := range g.trips {
		if g.trips[i].ID == tripID {
			targetTrip = &g.trips[i]
			break
		}
	}
	if targetTrip == nil {
		g.mu.Unlock()
		http.Error(w, "trip not found", http.StatusNotFound)
		return
	}
	driverID := targetTrip.DriverID
	plan, ok := g.driverPlans[driverID]
	if !ok {
		g.mu.Unlock()
		http.Error(w, "driver plan not found", http.StatusNotFound)
		return
	}
	if plan.simCancel != nil {
		g.mu.Unlock()
		// Already simulating, nothing to start
		w.WriteHeader(http.StatusOK)
		return
	}

	// Start simulation
	pickups := g.pickupPointsForIDs(plan.PickupIDs)
	if plan.CurrentIndex > 0 && plan.CurrentIndex < len(pickups) {
		pickups = pickups[plan.CurrentIndex:]
	} else if plan.CurrentIndex >= len(pickups) {
		// Already finished route?
		// Maybe just target stations?
		// But for now, let's restart if finished or do nothing?
		// User wants to simulate current trip segment.
		// If index is end, maybe we just complete?
		// Let's assume restart if finished, or just nothing.
		if plan.CurrentIndex >= len(plan.PickupIDs) {
			pickups = []PickupPoint{} // Should trigger station logic?
		}
	}

	simCtx, cancel := context.WithCancel(context.Background())
	plan.simCancel = cancel
	plan.Simulated = true
	g.mu.Unlock()

	go g.runSimulatedTrip(simCtx, driverID, pickups)

	writeJSON(w, http.StatusOK, map[string]string{"status": "simulation_started"})
}

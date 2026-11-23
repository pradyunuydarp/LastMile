package api

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	driverpb "lastmile/gen/go/driver"
	locationpb "lastmile/gen/go/location"
	"lastmile/internal/pkg/geo"
)

// Simulation tuning. A simulated car moves in small steps so the map animates
// smoothly and the Location service sees it ENTER a station's approach zone
// before it arrives — which is the spec's matching trigger.
const (
	simStepMeters = 60.0
	simMaxSteps   = 45
)

func (g *Gateway) configureDriverRoute(payload driverRouteRequest) (driverRouteResponse, error) {
	driverID := strings.TrimSpace(payload.DriverID)
	if driverID == "" {
		return driverRouteResponse{}, fmt.Errorf("driverId is required")
	}

	g.mu.Lock()
	normalized := g.normalizePickupIDs(payload.PickupPointIDs)
	g.mu.Unlock()
	if len(normalized) == 0 {
		return driverRouteResponse{}, fmt.Errorf("select at least one pickup point")
	}
	seats := payload.Seats
	if seats <= 0 {
		seats = 1
	}
	if seats > 8 {
		return driverRouteResponse{}, fmt.Errorf("seats must be between 1 and 8")
	}

	g.mu.Lock()
	targets := g.stationIDsForPickupIDs(normalized)
	pickups := g.pickupPointsForIDs(normalized)
	g.mu.Unlock()
	destination := strings.TrimSpace(payload.Destination)
	if destination == "" {
		destination = "unknown" // no fixed destination: accept any rider
	}

	// Register with the Driver service BEFORE storing the plan, so the
	// snapshot loop never sees a plan without a profile and mistakes it for
	// a driver the Driver service "lost". If this fails, the plan is still
	// stored and the snapshot loop retries the registration.
	g.ensureDriverProfile(payload, targets, destination, seats)

	g.mu.Lock()
	plan := &driverPlan{
		DriverID:       driverID,
		Name:           payload.Name,
		CarDetails:     payload.CarDetails,
		PickupIDs:      normalized,
		SeatsTotal:     seats,
		SeatsAvailable: seats,
		TargetStations: targets,
		Destination:    destination,
	}
	if existing, ok := g.driverPlans[driverID]; ok && existing.simCancel != nil {
		existing.simCancel()
	}
	g.driverPlans[driverID] = plan
	planCopy := *plan
	g.mu.Unlock()

	if g.store != nil {
		go g.store.SaveDriverRoute(driverID, &planCopy, pickups)
	}
	if g.hub != nil {
		go g.hub.RefreshDriverQueue(driverID)
	}

	return driverRouteResponse{
		DriverID:       driverID,
		PickupPoints:   pickups,
		SeatsTotal:     seats,
		SeatsAvailable: seats,
		TargetStations: targets,
		Destination:    destination,
	}, nil
}

// normalizePickupIDs de-duplicates and validates pickup IDs. Callers hold g.mu.
func (g *Gateway) normalizePickupIDs(ids []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if _, valid := g.pickupByID(id); !valid {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// stationIDsForPickupIDs maps pickups to their stations. Callers hold g.mu.
func (g *Gateway) stationIDsForPickupIDs(ids []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if pickup, ok := g.pickupByID(id); ok {
			if _, dup := seen[pickup.StationID]; dup {
				continue
			}
			seen[pickup.StationID] = struct{}{}
			out = append(out, pickup.StationID)
		}
	}
	return out
}

// ensureDriverProfile registers the driver and route with the Driver service.
// Failure is logged, not fatal: the gateway keeps the plan and the snapshot
// loop re-registers the driver once the service is back.
func (g *Gateway) ensureDriverProfile(payload driverRouteRequest, targets []string, destination string, seats int) {
	if g.driverClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := g.driverClient.RegisterDriver(ctx, &driverpb.RegisterDriverRequest{
		Driver: &driverpb.Driver{Id: payload.DriverID, Name: payload.Name, CarDetails: payload.CarDetails},
	})
	if err != nil {
		g.logger.Warn("register driver failed", "driverId", payload.DriverID, "err", err)
		return
	}

	_, err = g.driverClient.RegisterRoute(ctx, &driverpb.RegisterRouteRequest{
		Route: &driverpb.Route{
			DriverId:         payload.DriverID,
			TargetStationIds: targets,
			AvailableSeats:   int32(seats),
			Destination:      destination,
		},
	})
	if err != nil {
		g.logger.Warn("register route failed", "driverId", payload.DriverID, "err", err)
	}
}

func (g *Gateway) startDriverTrip(payload startTripRequest) (driverRouteResponse, error) {
	driverID := strings.TrimSpace(payload.DriverID)
	if driverID == "" {
		return driverRouteResponse{}, fmt.Errorf("driverId is required")
	}

	g.mu.Lock()
	plan, ok := g.driverPlans[driverID]
	if !ok {
		g.mu.Unlock()
		return driverRouteResponse{}, fmt.Errorf("configure a route before starting a trip")
	}
	plan.Active = true
	plan.StartedAt = time.Now()
	plan.SeatsAvailable = plan.SeatsTotal
	plan.CurrentIndex = 0
	plan.Simulated = payload.Simulate
	pickups := g.pickupPointsForIDs(plan.PickupIDs)

	if plan.simCancel != nil {
		plan.simCancel()
		plan.simCancel = nil
	}

	planResp := driverRouteResponse{
		DriverID:       driverID,
		PickupPoints:   pickups,
		SeatsTotal:     plan.SeatsTotal,
		SeatsAvailable: plan.SeatsAvailable,
		TargetStations: append([]string{}, plan.TargetStations...),
		Destination:    plan.Destination,
	}

	var simCtx context.Context
	if payload.Simulate && len(pickups) > 0 {
		var cancel context.CancelFunc
		simCtx, cancel = context.WithCancel(context.Background())
		plan.simCancel = cancel
	}
	planCopy := *plan
	g.mu.Unlock()

	if simCtx != nil {
		go g.runSimulatedTrip(simCtx, driverID, pickups)
	}
	if g.store != nil {
		go g.store.UpdateDriverRouteStatus(driverID, &planCopy, "active")
	}
	if g.hub != nil {
		go g.hub.RefreshDriverQueue(driverID)
	}
	return planResp, nil
}

// runSimulatedTrip drives the car through each remaining pickup in order,
// then to each on-board rider's drop-off, then ends the plan.
func (g *Gateway) runSimulatedTrip(ctx context.Context, driverID string, waypoints []PickupPoint) {
	lat, lon := g.simStart(driverID, waypoints)

	for _, wp := range waypoints {
		if !g.driveTo(ctx, driverID, &lat, &lon, wp.Latitude, wp.Longitude) {
			return
		}
		// At the stop: wait briefly for riders who accepted but haven't
		// confirmed yet, then pick up everyone who is ready.
		g.waitAtPickup(ctx, driverID, wp)
	}

	if !g.runDropoffs(ctx, driverID, &lat, &lon) {
		return
	}
	g.finishPlan(driverID)
}

// simStart picks where the simulated car begins: its last known position,
// or ~2 km before the first stop (outside the 1.5 km approach zone, so the
// students can watch the zone trigger fire).
func (g *Gateway) simStart(driverID string, waypoints []PickupPoint) (float64, float64) {
	g.mu.Lock()
	p, ok := g.lastPos[driverID]
	g.mu.Unlock()
	if ok {
		return p[0], p[1]
	}
	if len(waypoints) == 0 {
		return 0, 0
	}
	return waypoints[0].Latitude - 0.014, waypoints[0].Longitude - 0.012
}

// runDropoffs drives to the nearest on-board rider's destination until
// nobody is left in the car. Returns false if cancelled.
func (g *Gateway) runDropoffs(ctx context.Context, driverID string, lat, lon *float64) bool {
	for i := 0; i < 10; i++ { // bounded: a car has at most a handful of seats
		toLat, toLon, ok := g.nextDropoff(driverID, *lat, *lon)
		if !ok {
			return true
		}
		if !g.driveTo(ctx, driverID, lat, lon, toLat, toLon) {
			return false
		}
		if !g.sleep(ctx, g.simStopPause) {
			return false
		}
	}
	return true
}

// nextDropoff returns the closest drop-off among this driver's in-progress trips.
func (g *Gateway) nextDropoff(driverID string, lat, lon float64) (float64, float64, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	best, bestLat, bestLon := math.Inf(1), 0.0, 0.0
	for i := range g.trips {
		t := &g.trips[i]
		if t.DriverID != driverID || t.Status != "in_progress" {
			continue
		}
		station, _ := g.stationByID(t.StationID)
		dLat, dLon, ok := dropoffFor(t.Destination, station)
		if !ok {
			continue
		}
		if d := haversineMeters(lat, lon, dLat, dLon); d < best {
			best, bestLat, bestLon = d, dLat, dLon
		}
	}
	return bestLat, bestLon, !math.IsInf(best, 1)
}

// waitAtPickup holds the car at a stop for up to simPickupWait while any
// rider here is still confirming, then picks up everyone who is ready.
func (g *Gateway) waitAtPickup(ctx context.Context, driverID string, wp PickupPoint) {
	deadline := time.Now().Add(g.simPickupWait)
	for time.Now().Before(deadline) && g.hasUnconfirmedAt(driverID, wp.ID) {
		if !g.sleep(ctx, 500*time.Millisecond) {
			return
		}
	}
	g.handlePickupCheckpoint(driverID, &wp)
	g.sleep(ctx, g.simStopPause)
}

func (g *Gateway) hasUnconfirmedAt(driverID, pickupID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.trips {
		t := &g.trips[i]
		if t.DriverID == driverID && t.PickupPointID == pickupID && t.Status == "awaiting_rider" {
			return true
		}
	}
	return false
}

func (g *Gateway) finishPlan(driverID string) {
	g.mu.Lock()
	plan, ok := g.driverPlans[driverID]
	if ok {
		plan.Active = false
		plan.simCancel = nil
	}
	var planCopy driverPlan
	if ok {
		planCopy = *plan
	}
	g.mu.Unlock()
	if !ok {
		return
	}
	g.logger.Info("simulated trip finished", "driverId", driverID)
	if g.store != nil {
		g.store.UpdateDriverRouteStatus(driverID, &planCopy, "completed")
	}
	if g.hub != nil {
		g.hub.RefreshDriverQueue(driverID)
	}
}

// driveTo moves the car from (lat, lon) to the target in small steps, sending
// each step as a GPS ping. Returns false if cancelled.
func (g *Gateway) driveTo(ctx context.Context, driverID string, lat, lon *float64, toLat, toLon float64) bool {
	if !geo.Known(*lat, *lon) {
		*lat, *lon = toLat, toLon
	}
	dist := haversineMeters(*lat, *lon, toLat, toLon)
	steps := int(math.Ceil(dist / simStepMeters))
	steps = max(1, min(steps, simMaxSteps))

	fromLat, fromLon := *lat, *lon
	for i := 1; i <= steps; i++ {
		nLat, nLon := geo.Lerp(fromLat, fromLon, toLat, toLon, float64(i)/float64(steps))
		if ctx.Err() != nil {
			return false
		}
		g.ingestLocation(ctx, driverID, nLat, nLon)
		*lat, *lon = nLat, nLon
		if !g.sleep(ctx, g.simTick) {
			return false
		}
	}
	return true
}

func (g *Gateway) sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// ingestLocation is the single path every GPS ping takes, real or simulated:
//
//  1. Location service stores it, fans it out, and runs proximity detection
//  2. riders in the trip room see the car move (socket.io)
//  3. pickups are ticked off as the car passes them
//  4. riders whose destination we've reached are dropped off
//  5. if the car just ENTERED a station's approach zone, we run matching
func (g *Gateway) ingestLocation(ctx context.Context, driverID string, lat, lon float64) {
	resp, err := g.pushLocationUpdate(ctx, driverID, lat, lon)
	g.publishDriverLocation(driverID, lat, lon)
	if passed := g.recordDriverLocationProgress(driverID, lat, lon); passed != nil {
		g.handlePickupCheckpoint(driverID, passed)
	}
	g.maybeCompleteTrips(driverID, lat, lon)
	if err == nil && resp.GetEnteredZone() {
		g.onApproachingStation(driverID, resp.GetNearbyStationId(), resp.GetDistanceToStationMeters())
	}
}

// onApproachingStation is the proximity trigger from the spec: the driver is
// a few minutes from a station on their route, so ask Matching who to pick up
// and push the result to the driver.
func (g *Gateway) onApproachingStation(driverID, stationID string, distance float64) {
	g.mu.Lock()
	plan, ok := g.driverPlans[driverID]
	onRoute := false
	if ok && plan.Active {
		for _, id := range plan.PickupIDs[min(plan.CurrentIndex, len(plan.PickupIDs)):] {
			if pp, found := g.pickupByID(id); found && pp.StationID == stationID {
				onRoute = true
			}
		}
	}
	stationName := stationID
	if st, found := g.stationByID(stationID); found {
		stationName = st.Name
	}
	g.mu.Unlock()
	if !onRoute {
		return
	}

	etaMinutes := distance / 1000 / 20 * 60
	g.logger.Info("proximity trigger: driver approaching station",
		"driverId", driverID, "stationId", stationID, "distanceMeters", int(distance), "etaMinutes", math.Round(etaMinutes*10)/10)
	if g.hub != nil {
		g.hub.DriverApproaching(driverID, stationID, stationName, distance, etaMinutes)
		g.hub.RefreshDriverQueue(driverID)
	}
}

func (g *Gateway) pushLocationUpdate(ctx context.Context, driverID string, lat, lon float64) (*locationpb.UpdateLocationResponse, error) {
	if g.locationClient == nil {
		return nil, fmt.Errorf("location client not configured")
	}
	callCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	stream, err := g.locationClient.UpdateLocation(callCtx)
	if err != nil {
		g.logger.Warn("location stream failed", "driverId", driverID, "err", err)
		return nil, err
	}
	if err := stream.Send(&locationpb.UpdateLocationRequest{Location: &locationpb.Location{DriverId: driverID, Latitude: lat, Longitude: lon}}); err != nil {
		g.logger.Warn("location send failed", "driverId", driverID, "err", err)
		return nil, err
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		g.logger.Warn("location close failed", "driverId", driverID, "err", err)
		return nil, err
	}
	return resp, nil
}

// recordDriverLocationProgress advances the plan when the car reaches its
// next pickup, and returns that pickup.
func (g *Gateway) recordDriverLocationProgress(driverID string, lat, lon float64) *PickupPoint {
	g.mu.Lock()
	defer g.mu.Unlock()

	plan, ok := g.driverPlans[driverID]
	if !ok || plan.CurrentIndex >= len(plan.PickupIDs) {
		return nil
	}
	nextID := plan.PickupIDs[plan.CurrentIndex]
	pickup, ok := g.pickupByID(nextID)
	if !ok {
		plan.CurrentIndex++
		return nil
	}
	if haversineMeters(lat, lon, pickup.Latitude, pickup.Longitude) <= 120 {
		plan.CurrentIndex++
		pp := *pickup
		return &pp
	}
	return nil
}

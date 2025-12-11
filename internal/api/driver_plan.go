package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	driverpb "lastmile/gen/go/driver"
	locationpb "lastmile/gen/go/location"
)

func (g *Gateway) configureDriverRoute(payload driverRouteRequest) (driverRouteResponse, error) {
	driverID := strings.TrimSpace(payload.DriverID)
	if driverID == "" {
		return driverRouteResponse{}, fmt.Errorf("driverId is required")
	}
	normalized := g.normalizePickupIDs(payload.PickupPointIDs)
	if len(normalized) == 0 {
		return driverRouteResponse{}, fmt.Errorf("select at least one pickup point")
	}
	seats := payload.Seats
	if seats <= 0 {
		seats = 1
	}
	targets := g.stationIDsForPickupIDs(normalized)
	if len(targets) == 0 {
		return driverRouteResponse{}, fmt.Errorf("unable to infer metro stations for selected pickups")
	}
	destination := payload.Destination
	if destination == "" {
		destination = g.pickupDisplayName(normalized[len(normalized)-1])
	}
	pickups := g.pickupPointsForIDs(normalized)

	plan := &driverPlan{
		DriverID:       driverID,
		PickupIDs:      normalized,
		SeatsTotal:     seats,
		SeatsAvailable: seats,
		TargetStations: targets,
		Destination:    destination,
	}

	g.mu.Lock()
	if existing, ok := g.driverPlans[driverID]; ok && existing.simCancel != nil {
		existing.simCancel()
	}
	g.driverPlans[driverID] = plan
	g.mu.Unlock()

	if g.store != nil {
		go g.store.SaveDriverRoute(driverID, plan, pickups)
	}
	if g.hub != nil {
		go g.hub.RefreshDriverQueue(driverID)
	}

	g.ensureDriverProfile(payload, targets, destination, seats)

	return driverRouteResponse{
		DriverID:       driverID,
		PickupPoints:   pickups,
		SeatsTotal:     seats,
		SeatsAvailable: seats,
		TargetStations: targets,
		Destination:    destination,
	}, nil
}

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

func (g *Gateway) pickupDisplayName(id string) string {
	if pickup, ok := g.pickupByID(id); ok {
		return fmt.Sprintf("%s · %s", pickup.Name, pickup.StationName)
	}
	return ""
}

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
	targets := append([]string{}, plan.TargetStations...)
	destination := plan.Destination

	if plan.simCancel != nil {
		plan.simCancel()
		plan.simCancel = nil
	}

	var simulatePickups []PickupPoint
	if payload.Simulate && g.locationClient != nil {
		simulatePickups = pickups
	}

	planResp := driverRouteResponse{
		DriverID:       driverID,
		PickupPoints:   pickups,
		SeatsTotal:     plan.SeatsTotal,
		SeatsAvailable: plan.SeatsAvailable,
		TargetStations: targets,
		Destination:    destination,
	}

	var simCtx context.Context
	var cancel context.CancelFunc
	if payload.Simulate && g.locationClient != nil && len(simulatePickups) > 0 {
		simCtx, cancel = context.WithCancel(context.Background())
		plan.simCancel = cancel
	}
	g.mu.Unlock()

	if simCtx != nil {
		go g.runSimulatedTrip(simCtx, driverID, simulatePickups)
	}

	if g.store != nil {
		go g.store.UpdateDriverRouteStatus(driverID, plan, "active")
	}
	if g.hub != nil {
		go g.hub.RefreshDriverQueue(driverID)
	}

	return planResp, nil
}

func (g *Gateway) runSimulatedTrip(ctx context.Context, driverID string, waypoints []PickupPoint) {
	if g.locationClient == nil {
		return
	}
	for _, wp := range waypoints {
		if g.simulatedHop(ctx, driverID, wp.Latitude, wp.Longitude) {
			return
		}
		passed := g.recordDriverLocationProgress(driverID, wp.Latitude, wp.Longitude)
		if passed != nil {
			g.handlePickupCheckpoint(driverID, passed)
		}
		g.maybeCompleteTrips(driverID, wp.Latitude, wp.Longitude)
		if station, ok := g.stationByID(wp.StationID); ok {
			if g.simulatedHop(ctx, driverID, station.Latitude, station.Longitude) {
				return
			}
			g.maybeCompleteTrips(driverID, station.Latitude, station.Longitude)
		}
	}
}

func (g *Gateway) simulatedHop(ctx context.Context, driverID string, lat, lon float64) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	_ = g.pushLocationUpdate(ctx, driverID, lat, lon)
	g.publishDriverLocation(driverID, lat, lon)
	select {
	case <-ctx.Done():
		return true
	case <-time.After(3 * time.Second):
	}
	return false
}

func (g *Gateway) pushLocationUpdate(ctx context.Context, driverID string, lat, lon float64) error {
	if g.locationClient == nil {
		return nil
	}
	stream, err := g.locationClient.UpdateLocation(ctx)
	if err != nil {
		g.logger.Warn("simulate location stream failed", "driverId", driverID, "err", err)
		return err
	}
	if err := stream.Send(&locationpb.UpdateLocationRequest{Location: &locationpb.Location{DriverId: driverID, Latitude: lat, Longitude: lon}}); err != nil {
		g.logger.Warn("simulate location send failed", "driverId", driverID, "err", err)
		return err
	}
	if _, err := stream.CloseAndRecv(); err != nil {
		g.logger.Warn("simulate location close failed", "driverId", driverID, "err", err)
		return err
	}
	return nil
}

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
	distance := haversineMeters(lat, lon, pickup.Latitude, pickup.Longitude)
	if distance <= 120 {
		plan.CurrentIndex++
		pp := *pickup
		return &pp
	}
	return nil
}

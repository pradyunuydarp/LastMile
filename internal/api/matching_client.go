package api

import (
	"context"
	"errors"
	"time"

	matchingpb "lastmile/gen/go/matching"
)

// errMatchingUnavailable is returned when no Matching replica answered.
var errMatchingUnavailable = errors.New("matching service unavailable")

// matchingTimeout bounds every call to the Matching service. Retries on other
// replicas happen inside this budget (see chassis.Dial's retry policy).
const matchingTimeout = 2 * time.Second

// AttachMatching wires the Matching service client. Without it, bookings and
// driver queues report "matching unavailable" instead of guessing.
func (g *Gateway) AttachMatching(client matchingpb.MatchingServiceClient) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.matchingClient = client
}

// driverStateLocked builds the Matching view of one driver. Callers hold g.mu.
func (g *Gateway) driverStateLocked(d *Driver) *matchingpb.DriverState {
	state := &matchingpb.DriverState{
		DriverId:       d.ID,
		Name:           d.Name,
		SeatsAvailable: int32(d.SeatsAvailable),
		Destination:    d.Route.Destination,
	}
	if lat, lon, ok := g.positionLocked(d.ID, d.Latitude, d.Longitude); ok {
		state.Position = &matchingpb.GeoPoint{Latitude: lat, Longitude: lon}
	}

	if plan, ok := g.driverPlans[d.ID]; ok {
		state.PickupPointIds = append([]string(nil), plan.PickupIDs...)
		state.NextPickupIndex = int32(plan.CurrentIndex)
		state.SeatsAvailable = int32(plan.SeatsAvailable)
		state.Destination = plan.Destination
		state.TripActive = plan.Active
		return state
	}

	// A driver registered straight through the Driver service (no plan in
	// the gateway): treat every target station as a pickup, always active.
	for _, stationID := range d.Route.TargetStationIDs {
		state.PickupPointIds = append(state.PickupPointIds, "pickup-"+stationID)
	}
	state.TripActive = true
	return state
}

func riderRequestFor(r *Rider) *matchingpb.RiderRequest {
	req := &matchingpb.RiderRequest{
		RiderId:       r.ID,
		Name:          r.Name,
		PickupPointId: r.PickupPointID,
		StationId:     r.StationID,
		Destination:   r.Destination,
		ArrivalUnix:   r.ArrivalTime.Unix(),
	}
	if r.Pickup != nil {
		req.Pickup = &matchingpb.GeoPoint{Latitude: r.Pickup.Latitude, Longitude: r.Pickup.Longitude}
	}
	return req
}

func (g *Gateway) rankDrivers(rider *matchingpb.RiderRequest, drivers []*matchingpb.DriverState) (*matchingpb.RankDriversResponse, error) {
	g.mu.Lock()
	client := g.matchingClient
	g.mu.Unlock()
	if client == nil {
		return nil, errMatchingUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), matchingTimeout)
	defer cancel()
	resp, err := client.RankDrivers(ctx, &matchingpb.RankDriversRequest{Rider: rider, Drivers: drivers})
	if err != nil {
		g.logger.Warn("matching RankDrivers failed", "riderId", rider.GetRiderId(), "err", err)
		return nil, errMatchingUnavailable
	}
	g.noteMatchedBy(resp.GetServedBy())
	return resp, nil
}

func (g *Gateway) findRiders(driver *matchingpb.DriverState, riders []*matchingpb.RiderRequest) (*matchingpb.FindRidersResponse, error) {
	g.mu.Lock()
	client := g.matchingClient
	g.mu.Unlock()
	if client == nil {
		return nil, errMatchingUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), matchingTimeout)
	defer cancel()
	resp, err := client.FindRiders(ctx, &matchingpb.FindRidersRequest{Driver: driver, Riders: riders})
	if err != nil {
		g.logger.Warn("matching FindRiders failed", "driverId", driver.GetDriverId(), "err", err)
		return nil, errMatchingUnavailable
	}
	g.noteMatchedBy(resp.GetServedBy())
	return resp, nil
}

// noteMatchedBy remembers which Matching replicas have been answering, so the
// UI and /system/status can show load spreading across pods.
func (g *Gateway) noteMatchedBy(instance string) {
	if instance == "" {
		return
	}
	g.mu.Lock()
	g.lastMatchedBy = instance
	g.matchedByCount[instance]++
	g.mu.Unlock()
}

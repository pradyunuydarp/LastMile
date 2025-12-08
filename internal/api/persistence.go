package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Persistence struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewPersistence(ctx context.Context, dsn string, logger *slog.Logger) (*Persistence, error) {
	l := logger
	if l == nil {
		l = slog.Default()
	}
	if dsn == "" {
		return &Persistence{logger: l}, nil
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &Persistence{
		pool:   pool,
		logger: l,
	}, nil
}

func (p *Persistence) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}

func (p *Persistence) SaveDriverRoute(driverID string, plan *driverPlan, pickups []PickupPoint) {
	if p == nil || p.pool == nil || plan == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		p.logger.Warn("save driver route: begin tx failed", "driverId", driverID, "err", err)
		return
	}
	defer tx.Rollback(ctx)

	meta := map[string]any{
		"pickup_ids":   plan.PickupIDs,
		"destination":  plan.Destination,
		"target_stops": plan.TargetStations,
	}
	metaBytes, _ := json.Marshal(meta)

	var routeID string
	err = tx.QueryRow(ctx, `
		insert into driver_routes (driver_id, seats_total, seats_available, status, active, simulate, metadata, started_at, completed_at)
		values ($1, $2, $3, $4, $5, $6, $7::jsonb, null, null)
		on conflict (driver_id) do update set
			seats_total = excluded.seats_total,
			seats_available = excluded.seats_available,
			status = excluded.status,
			active = excluded.active,
			simulate = excluded.simulate,
			metadata = excluded.metadata,
			started_at = driver_routes.started_at,
			completed_at = driver_routes.completed_at
		returning id
	`, driverID, plan.SeatsTotal, plan.SeatsAvailable, "configured", plan.Active, plan.Simulated, metaBytes).Scan(&routeID)
	if err != nil {
		p.logger.Warn("save driver route: upsert failed", "driverId", driverID, "err", err)
		return
	}

	if _, err := tx.Exec(ctx, `delete from driver_route_pickups where route_id=$1`, routeID); err != nil {
		p.logger.Warn("save driver route: cleanup pickups failed", "driverId", driverID, "err", err)
		return
	}

	for idx, pickup := range pickups {
		_, err := tx.Exec(ctx, `
			insert into driver_route_pickups (route_id, sequence, pickup_id, pickup_name, station_id, station_name, latitude, longitude)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
		`, routeID, idx, pickup.ID, pickup.Name, pickup.StationID, pickup.StationName, pickup.Latitude, pickup.Longitude)
		if err != nil {
			p.logger.Warn("save driver route: insert pickup failed", "driverId", driverID, "pickupId", pickup.ID, "err", err)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		p.logger.Warn("save driver route: commit failed", "driverId", driverID, "err", err)
	}
}

func (p *Persistence) UpdateDriverRouteStatus(driverID string, plan *driverPlan, status string) {
	if p == nil || p.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := p.pool.Exec(ctx, `
		update driver_routes
		set status=$2,
			active=$3,
			seats_available=$4,
			started_at=coalesce(started_at, case when $3 then now() end),
			completed_at=case when $2='completed' then now() else completed_at end
		where driver_id=$1
	`, driverID, status, plan != nil && plan.Active, planSeats(plan))
	if err != nil {
		p.logger.Warn("update driver route status failed", "driverId", driverID, "err", err)
	}
}

func planSeats(plan *driverPlan) int {
	if plan == nil {
		return 0
	}
	return plan.SeatsAvailable
}

func (p *Persistence) RecordRiderRequest(rider Rider, pickup *PickupPoint, status string) {
	if p == nil || p.pool == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stationID := rider.StationID
	stationName := ""
	if pickup != nil {
		stationID = pickup.StationID
		stationName = pickup.StationName
	}

	_, err := p.pool.Exec(ctx, `
		insert into rider_requests (rider_id, pickup_id, pickup_name, station_id, station_name, destination, status, context)
		values ($1,$2,$3,$4,$5,$6,$7,$8::jsonb)
		on conflict (id) do nothing
	`, rider.ID, rider.PickupPointID, pickupName(pickup), stationID, stationName, rider.Destination, status, contextPayload(map[string]any{
		"arrival_time": rider.ArrivalTime,
	}))
	if err != nil {
		p.logger.Warn("record rider request failed", "riderId", rider.ID, "err", err)
	}
}

func (p *Persistence) UpdateRiderRequestStatus(riderID, status, driverID, tripID string) {
	if p == nil || p.pool == nil || riderID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := p.pool.Exec(ctx, `
		update rider_requests
		set status=$2,
			matched_driver_id=$3,
			matched_trip_id=$4
		where rider_id=$1
	`, riderID, status, driverID, tripID)
	if err != nil {
		p.logger.Warn("update rider request status failed", "riderId", riderID, "err", err)
	}
}

func (p *Persistence) RecordTrip(trip Trip) {
	if p == nil || p.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stationName := trip.Destination
	if trip.PickupPoint != nil && trip.PickupPoint.StationName != "" {
		stationName = trip.PickupPoint.StationName
	}

	_, err := p.pool.Exec(ctx, `
		insert into trips (id, driver_id, rider_id, pickup_id, pickup_name, station_id, station_name, status, seats_snapshot, started_at, destination, metadata)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb)
		on conflict (id) do update set
			status=excluded.status,
			seats_snapshot=excluded.seats_snapshot,
			completed_at=case when excluded.status='completed' then now() else trips.completed_at end,
			metadata=excluded.metadata
	`, trip.ID, trip.DriverID, trip.RiderID, trip.PickupPointID, pickupName(trip.PickupPoint), trip.StationID, stationName, trip.Status, trip.ETAMinutes, trip.CreatedAt, trip.Destination, contextPayload(map[string]any{
		"pickup":  trip.PickupPoint,
		"status":  trip.Status,
		"created": trip.CreatedAt,
	}))
	if err != nil {
		p.logger.Warn("record trip failed", "tripId", trip.ID, "err", err)
	}
}

func (p *Persistence) RecordTripEvent(tripID, eventType string, payload map[string]any) {
	if p == nil || p.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := p.pool.Exec(ctx, `
		insert into trip_events (trip_id, event_type, payload)
		values ($1,$2,$3::jsonb)
	`, tripID, eventType, contextPayload(payload))
	if err != nil {
		p.logger.Warn("record trip event failed", "tripId", tripID, "event", eventType, "err", err)
	}
}

func pickupName(p *PickupPoint) string {
	if p == nil {
		return ""
	}
	return p.Name
}

func contextPayload(payload map[string]any) []byte {
	if payload == nil {
		payload = map[string]any{}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func (p *Persistence) Healthy() bool {
	if p == nil || p.pool == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := p.pool.Ping(ctx); err != nil {
		return false
	}
	return true
}

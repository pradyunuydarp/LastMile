package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Persistence writes the gateway's ride history to Postgres (Supabase in
// production). The schema lives in supabase/migrations; keep the two in step.
//
// Every write is best effort: it has its own deadline, logs a warning on
// failure and never fails the request that triggered it. The live state of
// the system stays in memory; these tables are the audit trail.
type Persistence struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// transactionPoolerPort is the port of Supabase's transaction-mode pooler
// (Supavisor). Poolers in transaction mode hand each transaction to a
// different server connection, which breaks pgx's default of caching named
// prepared statements per connection.
const transactionPoolerPort = 6543

// NewPersistence connects a small pool to dsn (a postgres:// URI). The
// connection is lazy: a database that is down at startup is retried on
// every write, so the gateway still starts.
func NewPersistence(ctx context.Context, dsn string, logger *slog.Logger) (*Persistence, error) {
	l := logger
	if l == nil {
		l = slog.Default()
	}
	if dsn == "" {
		return &Persistence{logger: l}, nil
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// Supabase's free plan allows few connections; the gateway's writes are
	// small and short.
	cfg.MaxConns = 4
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	if cfg.ConnConfig.Port == transactionPoolerPort && !strings.Contains(dsn, "default_query_exec_mode") {
		// One round trip per query, no named prepared statements: safe
		// behind a transaction pooler. The SQL below casts its parameters
		// explicitly so it works in every exec mode.
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	p := &Persistence{pool: pool, logger: l}
	if err := pool.Ping(ctx); err != nil {
		l.Warn("database not reachable yet; history writes will retry", "host", cfg.ConnConfig.Host, "err", err)
	} else {
		l.Info("persistence enabled", "host", cfg.ConnConfig.Host, "port", cfg.ConnConfig.Port, "exec_mode", cfg.ConnConfig.DefaultQueryExecMode.String())
	}
	return p, nil
}

func (p *Persistence) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}

// SaveDriverRoute stores the driver's newly planned route and its ordered
// pickups. A new plan replaces the previous one, so its timestamps restart.
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
	defer func() { _ = tx.Rollback(ctx) }() // no-op after Commit

	meta := map[string]any{
		"pickup_ids":   plan.PickupIDs,
		"destination":  plan.Destination,
		"target_stops": plan.TargetStations,
	}

	var routeID string
	err = tx.QueryRow(ctx, `
		insert into driver_routes (driver_id, seats_total, seats_available, status, active, simulate, metadata, started_at, completed_at)
		values ($1::text, $2::int, $3::int, $4::text, $5::boolean, $6::boolean, $7::jsonb, null, null)
		on conflict (driver_id) do update set
			seats_total = excluded.seats_total,
			seats_available = excluded.seats_available,
			status = excluded.status,
			active = excluded.active,
			simulate = excluded.simulate,
			metadata = excluded.metadata,
			started_at = null,
			completed_at = null
		returning id::text
	`, driverID, plan.SeatsTotal, plan.SeatsAvailable, "configured", plan.Active, plan.Simulated, jsonText(meta)).Scan(&routeID)
	if err != nil {
		p.logger.Warn("save driver route: upsert failed", "driverId", driverID, "err", err)
		return
	}

	if _, err := tx.Exec(ctx, `delete from driver_route_pickups where route_id = $1::uuid`, routeID); err != nil {
		p.logger.Warn("save driver route: cleanup pickups failed", "driverId", driverID, "err", err)
		return
	}

	for idx, pickup := range pickups {
		_, err := tx.Exec(ctx, `
			insert into driver_route_pickups (route_id, sequence, pickup_id, pickup_name, station_id, station_name, latitude, longitude)
			values ($1::uuid, $2::int, $3::text, $4::text, $5::text, $6::text, $7::float8, $8::float8)
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

// UpdateDriverRouteStatus records that the route started ("active") or
// finished ("completed").
func (p *Persistence) UpdateDriverRouteStatus(driverID string, plan *driverPlan, status string) {
	if p == nil || p.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := p.pool.Exec(ctx, `
		update driver_routes
		set status = $2::text,
			active = $3::boolean,
			seats_available = $4::int,
			simulate = coalesce($5::boolean, simulate),
			started_at = coalesce(started_at, case when $3::boolean then now() end),
			completed_at = case when $2::text = 'completed' then now() else completed_at end
		where driver_id = $1::text
	`, driverID, status, plan != nil && plan.Active, planSeats(plan), planSimulated(plan))
	if err != nil {
		p.logger.Warn("update driver route status failed", "driverId", driverID, "err", err)
	}
}

// planSimulated is nil (keep the stored value) when there is no plan.
func planSimulated(plan *driverPlan) *bool {
	if plan == nil {
		return nil
	}
	return &plan.Simulated
}

func planSeats(plan *driverPlan) int {
	if plan == nil {
		return 0
	}
	return plan.SeatsAvailable
}

// RecordRiderRequest stores one booking. (Seeded bot riders never book, so
// they only show up in trips.)
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
		values ($1::text, $2::text, $3::text, $4::text, $5::text, $6::text, $7::text, $8::jsonb)
	`, rider.ID, rider.PickupPointID, pickupName(pickup), stationID, stationName, rider.Destination, status, jsonText(map[string]any{
		"arrival_time": rider.ArrivalTime,
	}))
	if err != nil {
		p.logger.Warn("record rider request failed", "riderId", rider.ID, "err", err)
	}
}

// UpdateRiderRequestStatus moves the rider's most recent booking along
// (matched, completed). Older bookings by the same rider keep their status.
func (p *Persistence) UpdateRiderRequestStatus(riderID, status, driverID, tripID string) {
	if p == nil || p.pool == nil || riderID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := p.pool.Exec(ctx, `
		update rider_requests
		set status = $2::text,
			matched_driver_id = $3::text,
			matched_trip_id = $4::text
		where id = (
			select id from rider_requests
			where rider_id = $1::text
			order by requested_at desc
			limit 1
		)
	`, riderID, status, driverID, tripID)
	if err != nil {
		p.logger.Warn("update rider request status failed", "riderId", riderID, "err", err)
	}
}

// RecordTrip upserts the trip's current state.
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
		insert into trips (id, driver_id, rider_id, pickup_id, pickup_name, station_id, station_name, status,
			eta_minutes, started_at, picked_up_at, completed_at, destination, metadata)
		values ($1::text, $2::text, $3::text, $4::text, $5::text, $6::text, $7::text, $8::text,
			$9::int, coalesce($10::timestamptz, now()), $11::timestamptz,
			coalesce($12::timestamptz, case when $8::text = 'completed' then now() end),
			$13::text, $14::jsonb)
		on conflict (id) do update set
			status = excluded.status,
			eta_minutes = excluded.eta_minutes,
			picked_up_at = coalesce(excluded.picked_up_at, trips.picked_up_at),
			completed_at = coalesce(excluded.completed_at, trips.completed_at),
			metadata = excluded.metadata
	`, trip.ID, trip.DriverID, trip.RiderID, trip.PickupPointID, pickupName(trip.PickupPoint), trip.StationID, stationName, trip.Status,
		trip.ETAMinutes, timeOrNil(trip.CreatedAt), timeOrNil(trip.PickedUpAt), timeOrNil(trip.CompletedAt),
		trip.Destination, jsonText(map[string]any{
			"pickup":  trip.PickupPoint,
			"status":  trip.Status,
			"created": trip.CreatedAt,
		}))
	if err != nil {
		p.logger.Warn("record trip failed", "tripId", trip.ID, "err", err)
	}
}

// RecordTripEvent appends to the trip's event log.
func (p *Persistence) RecordTripEvent(tripID, eventType string, payload map[string]any) {
	if p == nil || p.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := p.pool.Exec(ctx, `
		insert into trip_events (trip_id, event_type, payload)
		values ($1::text, $2::text, $3::jsonb)
	`, tripID, eventType, jsonText(payload))
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

// jsonText encodes payload for a $n::jsonb parameter. It is a string, not
// []byte: with the simple or exec protocols pgx would send []byte as bytea.
func jsonText(payload map[string]any) string {
	if payload == nil {
		return "{}"
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// timeOrNil maps Go's zero time (field never set) to SQL NULL instead of
// the year 1.
func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
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

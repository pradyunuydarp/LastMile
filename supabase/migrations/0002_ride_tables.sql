-- 0002_ride_tables.sql
--
-- History tables the gateway writes to when PERSISTENCE_DSN is set
-- (internal/api/persistence.go). The live system state stays in memory in the
-- services; these tables are an audit trail: who planned which route, who
-- asked for a ride, and every step of every trip.
--
-- Column types follow what persistence.go sends: ids are text because
-- drivers, riders and trips use ids like "smoke-driver-42" and
-- "trip-1732350000", not UUIDs.

-- One row per driver: the route they most recently planned.
create table if not exists public.driver_routes (
  id              uuid primary key default gen_random_uuid(),
  driver_id       text not null unique,
  seats_total     integer not null default 1,
  seats_available integer not null default 1,
  status          text not null default 'idle',  -- configured | active | completed
  active          boolean not null default false,
  started_at      timestamptz,
  completed_at    timestamptz,
  simulate        boolean not null default false,
  metadata        jsonb not null default '{}'::jsonb
);

-- The ordered pickup points of a route (rewritten whenever the route is saved).
create table if not exists public.driver_route_pickups (
  route_id     uuid not null references public.driver_routes (id) on delete cascade,
  sequence     integer not null,
  pickup_id    text not null,
  pickup_name  text not null,
  station_id   text not null,
  station_name text not null,
  latitude     double precision,
  longitude    double precision,
  passed_at    timestamptz,
  primary key (route_id, sequence)
);

-- One row per booking a rider makes.
create table if not exists public.rider_requests (
  id                uuid primary key default gen_random_uuid(),
  rider_id          text not null,
  pickup_id         text not null,
  pickup_name       text not null,
  station_id        text not null,
  station_name      text not null,
  destination       text,
  status            text not null default 'waiting',  -- waiting | matched | completed
  requested_at      timestamptz not null default now(),
  matched_driver_id text,
  matched_trip_id   text,
  context           jsonb not null default '{}'::jsonb
);

-- One row per trip, upserted on every state change.
create table if not exists public.trips (
  id             text primary key,
  driver_id      text not null,
  rider_id       text not null,
  pickup_id      text not null,
  pickup_name    text not null,
  station_id     text not null,
  station_name   text not null,
  status         text not null default 'pending',  -- scheduled | picked_up | completed
  seats_snapshot integer,
  eta_minutes    integer,
  started_at     timestamptz default now(),
  picked_up_at   timestamptz,
  completed_at   timestamptz,
  destination    text,
  metadata       jsonb not null default '{}'::jsonb
);

-- Append-only log of trip events (matched, pickup_reached, dropoff_reached...).
-- trip_id deliberately has no foreign key: the gateway also logs events for
-- offers that never became a stored trip (for example "rider_declined").
create table if not exists public.trip_events (
  id          uuid primary key default gen_random_uuid(),
  trip_id     text,
  event_type  text not null,
  payload     jsonb not null default '{}'::jsonb,
  recorded_at timestamptz not null default now()
);

-- Bring databases created by the old schema.sql up to the same shape --------
alter table public.trips add column if not exists eta_minutes integer;
alter table public.trips add column if not exists picked_up_at timestamptz;

-- The old foreign key rejected events for offers that were declined before
-- a trip row existed ("violates foreign key constraint trip_events_trip_id_fkey").
alter table public.trip_events drop constraint if exists trip_events_trip_id_fkey;

-- Indexes ----------------------------------------------------------------------
create index if not exists idx_rider_requests_pickup_status on public.rider_requests (pickup_id, status);
create index if not exists idx_rider_requests_driver on public.rider_requests (matched_driver_id);
-- persistence.go updates "the rider's latest request", so look that up fast.
create index if not exists idx_rider_requests_rider_requested on public.rider_requests (rider_id, requested_at desc);
create index if not exists idx_trips_driver_status on public.trips (driver_id, status);
create index if not exists idx_trips_rider_status on public.trips (rider_id, status);
create index if not exists idx_trip_events_trip on public.trip_events (trip_id, recorded_at desc);

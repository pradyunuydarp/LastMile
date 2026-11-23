-- 0003_ride_tables_rls.sql
--
-- Supabase exposes every table in the public schema through its REST API.
-- Without Row Level Security, anyone holding the project's anon key (which
-- ships inside browser apps) could read or rewrite the ride history.
--
-- Turning RLS on with no policies means: the anon and authenticated roles see
-- nothing and can change nothing. The gateway is not affected: it connects
-- with the database URL as the tables' owner (postgres), and owners bypass
-- RLS. To let signed-in users read their own trips later, add a policy, e.g.
--   create policy "Riders read own trips" on public.trips
--     for select to authenticated using (rider_id = (select auth.uid())::text);
alter table public.driver_routes        enable row level security;
alter table public.driver_route_pickups enable row level security;
alter table public.rider_requests       enable row level security;
alter table public.trips                enable row level security;
alter table public.trip_events          enable row level security;

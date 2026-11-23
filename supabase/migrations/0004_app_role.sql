-- A least-privilege database role for the gateway at runtime.
--
-- The gateway only reads and writes the five ride tables, so it should not
-- connect as `postgres` (which can do anything, including in auth.*). This
-- migration creates `lastmile_app` WITHOUT a password and gives it exactly
-- what persistence.go needs. Giving it a password is a one-time, out-of-band
-- step so no secret ever lands in the repo (see docs/deploy.md):
--
--   alter role lastmile_app with login password '<generated>';
--
-- Through Supabase's pooler the user name is `lastmile_app.<project-ref>`.

do $$
begin
  if not exists (select 1 from pg_roles where rolname = 'lastmile_app') then
    create role lastmile_app nologin;
  end if;
end
$$;

grant usage on schema public to lastmile_app;

grant select, insert, update, delete on
  public.driver_routes,
  public.driver_route_pickups,
  public.rider_requests,
  public.trips,
  public.trip_events
to lastmile_app;

-- Row level security is on for these tables (0003) and no other role has a
-- policy, so this role sees every row and nobody else (anon, authenticated)
-- sees any.
drop policy if exists "lastmile_app full access" on public.driver_routes;
create policy "lastmile_app full access" on public.driver_routes
  for all to lastmile_app using (true) with check (true);

drop policy if exists "lastmile_app full access" on public.driver_route_pickups;
create policy "lastmile_app full access" on public.driver_route_pickups
  for all to lastmile_app using (true) with check (true);

drop policy if exists "lastmile_app full access" on public.rider_requests;
create policy "lastmile_app full access" on public.rider_requests
  for all to lastmile_app using (true) with check (true);

drop policy if exists "lastmile_app full access" on public.trips;
create policy "lastmile_app full access" on public.trips
  for all to lastmile_app using (true) with check (true);

drop policy if exists "lastmile_app full access" on public.trip_events;
create policy "lastmile_app full access" on public.trip_events
  for all to lastmile_app using (true) with check (true);

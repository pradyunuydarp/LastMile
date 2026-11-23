-- ci/supabase-emulation.sql
--
-- Makes supabase/migrations apply on a plain PostgreSQL 16 server (the CI
-- service container) by creating the small part of a Supabase project they
-- depend on:
--   * the API roles anon, authenticated and service_role
--   * schema auth with auth.users (only the columns the migrations use, plus
--     a few common ones) and the auth.uid() / auth.role() / auth.jwt() helpers
--   * Supabase's default grants, so RLS (not missing privileges) is what
--     decides what anon can see, exactly as on Supabase
--
-- Run it ONCE, as a superuser, BEFORE scripts/db-migrate.sh:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f ci/supabase-emulation.sql
-- It is idempotent. Never run it against a real Supabase project (it refuses).

set client_min_messages = warning;

do $$
begin
  if exists (select 1 from pg_roles where rolname = 'supabase_admin')
     or exists (select 1 from pg_namespace where nspname = 'supabase_functions') then
    raise exception 'this looks like a real Supabase database: ci/supabase-emulation.sql is only for plain PostgreSQL in CI';
  end if;
end
$$;

-- Roles ----------------------------------------------------------------------
do $$
begin
  if not exists (select 1 from pg_roles where rolname = 'anon') then
    create role anon nologin noinherit;
  end if;
  if not exists (select 1 from pg_roles where rolname = 'authenticated') then
    create role authenticated nologin noinherit;
  end if;
  if not exists (select 1 from pg_roles where rolname = 'service_role') then
    create role service_role nologin noinherit bypassrls;
  end if;
end
$$;

-- auth schema ----------------------------------------------------------------
create schema if not exists auth;

create table if not exists auth.users (
  instance_id        uuid,
  id                 uuid primary key default gen_random_uuid(),
  aud                varchar(255),
  role               varchar(255),
  email              varchar(255),
  encrypted_password varchar(255),
  raw_app_meta_data  jsonb default '{}'::jsonb,
  raw_user_meta_data jsonb default '{}'::jsonb,
  created_at         timestamptz default now(),
  updated_at         timestamptz default now()
);

-- Same contract as Supabase: PostgREST puts the caller's JWT claims in
-- request.jwt.claims (older versions: request.jwt.claim.sub). Tests can
-- impersonate a user with:
--   set local role authenticated;
--   set local request.jwt.claims = '{"sub": "<uuid>", "role": "authenticated"}';
create or replace function auth.jwt() returns jsonb
language sql stable
as $$
  select coalesce(nullif(current_setting('request.jwt.claims', true), ''), '{}')::jsonb
$$;

create or replace function auth.uid() returns uuid
language sql stable
as $$
  select coalesce(
    nullif(current_setting('request.jwt.claim.sub', true), ''),
    auth.jwt() ->> 'sub'
  )::uuid
$$;

create or replace function auth.role() returns text
language sql stable
as $$
  select coalesce(
    nullif(current_setting('request.jwt.claim.role', true), ''),
    auth.jwt() ->> 'role'
  )::text
$$;

-- Grants (mirrors a fresh Supabase project) ----------------------------------
grant usage on schema public to anon, authenticated, service_role;
grant usage on schema auth to anon, authenticated, service_role;
grant execute on function auth.uid(), auth.role(), auth.jwt() to anon, authenticated, service_role;

-- Tables, sequences and functions that the migration user creates in public
-- are reachable by the API roles; RLS policies then decide row access.
alter default privileges in schema public grant all on tables to anon, authenticated, service_role;
alter default privileges in schema public grant all on sequences to anon, authenticated, service_role;
alter default privileges in schema public grant execute on functions to anon, authenticated, service_role;

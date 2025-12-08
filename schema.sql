create extension if not exists "pgcrypto";

-- Create a table for public profiles
create table if not exists profiles (
  id uuid references auth.users not null primary key,
  updated_at timestamp with time zone,
  username text unique,
  full_name text,
  avatar_url text,
  role text check (role in ('rider', 'driver', 'admin')) default 'rider',
  constraint username_length check (char_length(username) >= 3)
);

-- Set up Row Level Security (RLS)
alter table profiles enable row level security;

do $$
begin
  if not exists (select 1 from pg_policies where policyname = 'Public profiles are viewable by everyone.' and tablename = 'profiles') then
    create policy "Public profiles are viewable by everyone." on profiles
      for select using (true);
  end if;
end$$;

do $$
begin
  if not exists (select 1 from pg_policies where policyname = 'Users can insert their own profile.' and tablename = 'profiles') then
    create policy "Users can insert their own profile." on profiles
      for insert with check (auth.uid() = id);
  end if;
end$$;

do $$
begin
  if not exists (select 1 from pg_policies where policyname = 'Users can update own profile.' and tablename = 'profiles') then
    create policy "Users can update own profile." on profiles
      for update using (auth.uid() = id);
  end if;
end$$;

-- This trigger automatically creates a profile entry when a new user signs up via Supabase Auth.
create or replace function public.handle_new_user()
returns trigger as $$
begin
  insert into public.profiles (id, full_name, avatar_url, role)
  values (new.id, new.raw_user_meta_data->>'full_name', new.raw_user_meta_data->>'avatar_url', coalesce(new.raw_user_meta_data->>'role', 'rider'))
  on conflict (id) do update set
    full_name = excluded.full_name,
    avatar_url = excluded.avatar_url,
    role = excluded.role;
  return new;
end;
$$ language plpgsql security definer;

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created
  after insert on auth.users
  for each row execute procedure public.handle_new_user();

-- Realtime routing + trips ----------------------------------------------------

create table if not exists driver_routes (
  id uuid primary key default gen_random_uuid(),
  driver_id text not null unique,
  seats_total integer not null default 1,
  seats_available integer not null default 1,
  status text not null default 'idle',
  active boolean not null default false,
  started_at timestamptz,
  completed_at timestamptz,
  simulate boolean not null default false,
  metadata jsonb not null default '{}'::jsonb
);

create table if not exists driver_route_pickups (
  route_id uuid references driver_routes(id) on delete cascade,
  sequence integer not null,
  pickup_id text not null,
  pickup_name text not null,
  station_id text not null,
  station_name text not null,
  latitude double precision,
  longitude double precision,
  passed_at timestamptz,
  primary key (route_id, sequence)
);

create table if not exists rider_requests (
  id uuid primary key default gen_random_uuid(),
  rider_id text not null,
  pickup_id text not null,
  pickup_name text not null,
  station_id text not null,
  station_name text not null,
  destination text,
  status text not null default 'waiting',
  requested_at timestamptz not null default now(),
  matched_driver_id text,
  matched_trip_id text,
  context jsonb not null default '{}'::jsonb
);

create index if not exists idx_rider_requests_pickup_status on rider_requests (pickup_id, status);
create index if not exists idx_rider_requests_driver on rider_requests (matched_driver_id);

create table if not exists trips (
  id text primary key,
  driver_id text not null,
  rider_id text not null,
  pickup_id text not null,
  pickup_name text not null,
  station_id text not null,
  station_name text not null,
  status text not null default 'pending',
  seats_snapshot integer,
  started_at timestamptz default now(),
  completed_at timestamptz,
  destination text,
  metadata jsonb not null default '{}'::jsonb
);

create index if not exists idx_trips_driver_status on trips (driver_id, status);
create index if not exists idx_trips_rider_status on trips (rider_id, status);

create table if not exists trip_events (
  id uuid primary key default gen_random_uuid(),
  trip_id text references trips(id) on delete cascade,
  event_type text not null,
  payload jsonb not null default '{}'::jsonb,
  recorded_at timestamptz not null default now()
);

create index if not exists idx_trip_events_trip on trip_events (trip_id, recorded_at desc);

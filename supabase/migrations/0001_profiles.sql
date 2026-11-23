-- 0001_profiles.sql
--
-- public.profiles: one row per Supabase Auth user. The User service reads it
-- (through Supabase's REST API) to get a user's display name and role.
--
-- Every migration in this folder must be safe to run against a database that
-- already has the objects, because older LastMile projects were set up by
-- pasting the old root schema.sql into the Supabase SQL editor. Hence
-- "if not exists", "drop ... if exists" before "create", and "create or replace".
--
-- scripts/db-migrate.sh runs each file inside one transaction, so do not put
-- BEGIN/COMMIT (or CREATE INDEX CONCURRENTLY) in migration files.
--
-- gen_random_uuid() is built into PostgreSQL 13+, so the old
-- `create extension pgcrypto` line is gone.

create table if not exists public.profiles (
  id         uuid primary key references auth.users (id) on delete cascade,
  updated_at timestamptz,
  username   text unique,
  full_name  text,
  avatar_url text,
  role       text default 'rider' check (role in ('rider', 'driver', 'admin')),
  constraint username_length check (char_length(username) >= 3)
);

-- Databases created from the old schema.sql have the foreign key without
-- ON DELETE CASCADE, which makes "Delete user" in the Supabase dashboard fail
-- for anyone who has a profile. Recreate it with the cascade.
alter table public.profiles drop constraint if exists profiles_id_fkey;
alter table public.profiles
  add constraint profiles_id_fkey
  foreign key (id) references auth.users (id) on delete cascade;

-- Row Level Security --------------------------------------------------------
-- The User service queries profiles with the project's anon key, so
-- profiles stay publicly readable. Writes are limited to the row's owner.
alter table public.profiles enable row level security;

drop policy if exists "Public profiles are viewable by everyone." on public.profiles;
create policy "Public profiles are viewable by everyone." on public.profiles
  for select using (true);

drop policy if exists "Users can insert their own profile." on public.profiles;
create policy "Users can insert their own profile." on public.profiles
  for insert with check ((select auth.uid()) = id);

drop policy if exists "Users can update own profile." on public.profiles;
create policy "Users can update own profile." on public.profiles
  for update using ((select auth.uid()) = id);

-- Sign-up trigger -------------------------------------------------------------
-- Creates the profile when Supabase Auth inserts a user. The role comes from
-- the sign-up metadata the User service sends ("rider" or "driver").
--
-- Two fixes compared with the old schema.sql:
--   * any other value (including "" and "admin") becomes "rider". Before,
--     anyone could make themselves an admin by signing up with
--     {"role": "admin"}, and an unexpected value failed the check constraint,
--     which made the whole sign-up fail with "Database error saving new user".
--   * `set search_path = ''` (Supabase's advice for SECURITY DEFINER
--     functions), so every name below is schema-qualified.
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = ''
as $$
declare
  requested_role text := lower(coalesce(new.raw_user_meta_data ->> 'role', ''));
begin
  insert into public.profiles (id, full_name, avatar_url, role)
  values (
    new.id,
    new.raw_user_meta_data ->> 'full_name',
    new.raw_user_meta_data ->> 'avatar_url',
    case when requested_role in ('rider', 'driver') then requested_role else 'rider' end
  )
  on conflict (id) do update set
    full_name  = excluded.full_name,
    avatar_url = excluded.avatar_url,
    role       = excluded.role;
  return new;
end;
$$;

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_user();

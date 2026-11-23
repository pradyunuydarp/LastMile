#!/usr/bin/env bash
# Apply LastMile's database migrations (supabase/migrations/NNNN_name.sql).
#
#   SUPABASE_DB_URL='postgresql://...' scripts/db-migrate.sh           # apply pending
#   SUPABASE_DB_URL='postgresql://...' scripts/db-migrate.sh --status  # show, change nothing
#
# SUPABASE_DB_URL is a postgres URI. For Supabase use the pooler URI from
# Project settings -> Database -> Connection string (session mode, port 5432,
# is the safest choice; transaction mode, port 6543, works too).
#
# What it does, every time (it runs on every deploy):
#   1. Creates lastmile_meta.schema_migrations if needed. It records each
#      applied file's name and sha256.
#   2. Refuses to continue if a file that was already applied has changed
#      since (checksum mismatch). Applied migrations are history: to change
#      the schema, add a new file with the next number instead.
#   3. Applies every file not applied yet, in filename order. Each file runs in
#      its own transaction together with its tracking row, so a failing file
#      leaves nothing half-applied, and the next run retries it.
#   4. Prints what it did. A second run in a row does nothing.
#
# Rules for migration files: name them NNNN_snake_case.sql, never edit one
# that has been applied anywhere, and don't use BEGIN/COMMIT or
# CREATE INDEX CONCURRENTLY inside them (each file already is a transaction).
# Write them so they also succeed on a database where the objects already
# exist ("if not exists", "drop ... if exists" then "create").
#
# Needs: bash, psql (any version >= 10), sha256sum or shasum.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIR="${MIGRATIONS_DIR:-$ROOT/supabase/migrations}"

say()  { printf '\033[1;36m▸ %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

usage() {
  sed -n '2,5p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

mode=apply
case "${1:-}" in
  "") ;;
  --status) mode=status ;;
  -h | --help) usage 0 ;;
  *) usage 2 ;;
esac

[ -n "${SUPABASE_DB_URL:-}" ] || die "SUPABASE_DB_URL is not set (a postgresql:// URI; see docs/deploy.md)"
command -v psql >/dev/null 2>&1 || die "psql not found (apt-get install postgresql-client / brew install libpq)"

# Fail fast instead of hanging on an unreachable host; name the connection
# so it is easy to spot in Supabase's pg_stat_activity.
export PGCONNECT_TIMEOUT="${PGCONNECT_TIMEOUT:-20}"
export PGAPPNAME="${PGAPPNAME:-lastmile-db-migrate}"

# -X: ignore ~/.psqlrc. Never echo SUPABASE_DB_URL: it contains the password.
db() { psql "$SUPABASE_DB_URL" -X -q -v ON_ERROR_STOP=1 "$@"; }

# Checksum of the file's contents with carriage returns removed, so a
# Windows checkout (CRLF) of an unchanged file still matches.
checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    tr -d '\r' <"$1" | sha256sum | cut -d' ' -f1
  else
    tr -d '\r' <"$1" | shasum -a 256 | cut -d' ' -f1
  fi
}

# ---- 1. find and sanity-check the migration files --------------------------
[ -d "$DIR" ] || die "migrations directory not found: $DIR"
files=()
while IFS= read -r f; do files+=("$f"); done < <(find "$DIR" -maxdepth 1 -type f -name '*.sql' | LC_ALL=C sort)
[ "${#files[@]}" -gt 0 ] || die "no .sql files in $DIR"

for f in "${files[@]}"; do
  name="$(basename "$f")"
  [[ "$name" =~ ^[0-9]{4}_[a-z0-9_]+\.sql$ ]] || die "$name: migration files must be named NNNN_snake_case.sql"
  if grep -Eiq '^[[:space:]]*(begin|commit|rollback|start[[:space:]]+transaction)[[:space:]]*;' "$f"; then
    die "$name: remove BEGIN/COMMIT, db-migrate.sh already runs each file in a transaction"
  fi
done
dups="$(for f in "${files[@]}"; do basename "$f" | cut -c1-4; done | sort | uniq -d | tr '\n' ' ')"
[ -z "$dups" ] || die "two migration files share the number(s): $dups"

# ---- 2. tracking table -------------------------------------------------------
if [ "$mode" = apply ]; then
  # Same advisory lock as below, so two runs starting at once don't race to
  # create the table.
  db --single-transaction \
     -c "set local client_min_messages = warning" \
     -c "do \$\$ begin perform pg_advisory_xact_lock(hashtext('lastmile_meta.schema_migrations')); end \$\$" \
     -c "create schema if not exists lastmile_meta" \
     -c "create table if not exists lastmile_meta.schema_migrations (
           filename   text primary key,
           sha256     text not null,
           applied_at timestamptz not null default now(),
           applied_by text not null default current_user,
           git_commit text
         )"
  table_exists=t
else
  table_exists="$(db -At -c "select to_regclass('lastmile_meta.schema_migrations') is not null")"
fi

# "filename sha256" per line (no associative arrays: macOS still ships bash 3.2).
applied_list=""
if [ "$table_exists" = t ]; then
  applied_list="$(db -At -F ' ' -c "select filename, sha256 from lastmile_meta.schema_migrations order by filename")"
fi
applied_sum() { printf '%s\n' "$applied_list" | awk -v n="$1" '$1 == n { print $2 }'; }

# ---- 3. refuse to continue if history was rewritten -------------------------
mismatch=0
pending=()
for f in "${files[@]}"; do
  name="$(basename "$f")"
  sum="$(checksum "$f")"
  was="$(applied_sum "$name")"
  if [ -z "$was" ]; then
    pending+=("$f")
  elif [ "$was" != "$sum" ]; then
    printf '\033[31m✗ %s was already applied, but the file has changed since.\033[0m\n' "$name" >&2
    printf '    applied sha256: %s\n    file    sha256: %s\n' "$was" "$sum" >&2
    mismatch=1
  fi
done
while read -r name _; do
  [ -z "$name" ] || [ -f "$DIR/$name" ] || warn "$name is recorded as applied but is not in $DIR (deleted or renamed?)"
done <<<"$applied_list"
if [ "$mismatch" = 1 ]; then
  die "refusing to migrate: restore the original file(s) and put the change in a new migration with the next number"
fi

if [ "$mode" = status ]; then
  say "Migrations in $DIR"
  for f in "${files[@]}"; do
    name="$(basename "$f")"
    if [ -n "$(applied_sum "$name")" ]; then ok "$name (applied)"; else printf '  · %s (pending)\n' "$name"; fi
  done
  exit 0
fi

if [ "${#pending[@]}" -eq 0 ]; then
  ok "database is up to date (${#files[@]} migrations applied)"
  exit 0
fi

# ---- 4. apply pending files, one transaction each -----------------------------
say "Applying ${#pending[@]} migration(s)"
for f in "${pending[@]}"; do
  name="$(basename "$f")"
  sum="$(checksum "$f")"
  path="${f//\'/\'\'}" # quote for psql's \i

  # The advisory lock serialises concurrent runs (two deploys at once); the
  # second one then sees the row and skips. Everything here, the file
  # included, commits or rolls back as one transaction (--single-transaction).
  if ! db --single-transaction \
    -v name="$name" -v sha="$sum" -v commit="${GITHUB_SHA:-}" \
    -f - <<SQL; then
set local client_min_messages = warning;
set local lock_timeout = '20s';
set local statement_timeout = '5min';
select pg_advisory_xact_lock(hashtext('lastmile_meta.schema_migrations')) \gset lock_
select exists (select 1 from lastmile_meta.schema_migrations where filename = :'name') as already_applied \gset
\if :already_applied
\echo '  ↷' :name 'was applied by a concurrent run; skipping'
\else
\i '${path}'
insert into lastmile_meta.schema_migrations (filename, sha256, git_commit)
values (:'name', :'sha', nullif(:'commit', ''));
\endif
SQL
    die "$name failed and was rolled back; nothing from it was applied. Fix it and run again."
  fi
  ok "$name"
done
ok "done: ${#files[@]} migrations applied"

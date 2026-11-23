#!/usr/bin/env bash
# End-to-end smoke test: one complete ride through the real services.
#
#   scripts/smoke.sh                       # against localhost:8082
#   GATEWAY=http://localhost:8082 scripts/smoke.sh
#
# Works against `make run-local`, `docker compose up`, or Kubernetes with
#   kubectl -n lastmile port-forward svc/gateway 8082:80
#
# Story: driver Asha plans a route through Electronic City, starts a
# simulated trip, accepts the bot rider waiting there, drives her to Wipro
# Gate. Needs python3 for JSON parsing.
set -euo pipefail
G="${GATEWAY:-http://localhost:8082}"
DRIVER="smoke-driver-$$"

say()  { printf '\033[1;36m▸ %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
die()  { printf '  \033[31m✗ %s\033[0m\n' "$*"; exit 1; }
json() { python3 -c "import sys,json; d=json.load(sys.stdin); print($1)"; }
post() { curl -sf -XPOST "$G$1" -H 'Content-Type: application/json' -d "$2"; }

say "1. Every service healthy?"
curl -sf "$G/system/status" | json "'\n'.join(f\"  {s['name']:<13} {s['status']}\" for s in d['services'])"
down=$(curl -sf "$G/system/status" | json "sum(s['status']!='SERVING' for s in d['services'])")
[ "$down" = "0" ] && ok "all services SERVING" || die "$down service(s) not serving"

say "2. Demo login (User service)"
post /auth/signin '{"email":"driver@lastmile.demo","password":"lastmile"}' | json "'  signed in as ' + d['user']['name']"

say "3. Driver plans a route: Electronic City -> Konappana, 3 seats"
post /drivers/routes "{\"driverId\":\"$DRIVER\",\"name\":\"Smoke Driver\",\"carDetails\":\"Test car\",\"pickupPointIds\":[\"pickup-station-ecity\",\"pickup-station-konappana\"],\"seats\":3}" >/dev/null
ok "route saved (Driver service + gateway plan)"

say "4. Start a simulated trip; ask Matching who to pick up"
post /drivers/trip/start "{\"driverId\":\"$DRIVER\",\"simulate\":true}" >/dev/null
sleep 1
curl -sf "$G/drivers/requests?driverId=$DRIVER" | json "'\n'.join(f\"  {r['name']:<14} {r['station']['name']} -> {r['destination']}\" for r in d['requests'])"
rider=$(curl -sf "$G/drivers/requests?driverId=$DRIVER" | json "next((r['id'] for r in d['requests'] if r['station']['id']=='station-ecity'), '')")
[ -n "$rider" ] || die "no rider waiting at Electronic City (bots are seeded at gateway start; restart it to reset)"

say "5. Accept $rider"
trip=$(post /drivers/requests/accept "{\"driverId\":\"$DRIVER\",\"riderId\":\"$rider\"}" | json "d['id']")
ok "trip $trip created"

say "6. Watch the car (Location service ingests every GPS step)"
last=""
for _ in $(seq 1 90); do
  status=$(curl -sf "$G/aggregates/snapshot" | json "next((t['status'] for t in d['trips'] if t['id']=='$trip'), 'missing')")
  if [ "$status" != "$last" ]; then ok "trip is $status"; last=$status; fi
  [ "$status" = "completed" ] && break
  sleep 1
done
[ "$last" = "completed" ] || die "trip did not complete in 90s (last status: $last)"

say "7. Who decided? (Matching replicas that answered)"
curl -sf "$G/system/status" | json "'  ' + str(d['matching']['servedBy'])"
printf '\n\033[1;32mSmoke test passed: booked, picked up, dropped off.\033[0m\n'

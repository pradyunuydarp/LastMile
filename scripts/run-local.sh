#!/usr/bin/env bash
# Run every LastMile service as a local process — no Docker, no Kubernetes.
#
#   scripts/run-local.sh     # build + start everything (Ctrl-C stops all)
#
# One replica of each service. To see Matching scale out, use
# `docker compose up --scale matching=3` or the Kubernetes demo.
#
# Logs go to .run/logs/<service>.log; `tail -f .run/logs/*.log` to watch.
set -euo pipefail
cd "$(dirname "$0")/.."

mkdir -p bin .run/logs
echo "▸ building services into ./bin"
for svc in station driver rider trip notification location user matching gateway loadgen; do
  go build -o "bin/$svc" "./cmd/$svc"
done

pids=()
cleanup() {
  echo; echo "▸ stopping ${#pids[@]} processes"
  kill "${pids[@]}" 2>/dev/null || true
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

start() { # name, logfile, env...
  local name=$1 log=$2; shift 2
  env "$@" "bin/$name" >"$log" 2>&1 &
  pids+=($!)
  printf '  %-14s pid %-6s log %s\n' "$name" "$!" "$log"
}

echo "▸ starting services"
start station      .run/logs/station.log
start driver       .run/logs/driver.log
start rider        .run/logs/rider.log
start trip         .run/logs/trip.log
start notification .run/logs/notification.log
start user         .run/logs/user.log
start location     .run/logs/location.log STATION_ADDR=localhost:50056

start matching     .run/logs/matching.log POD_NAME=matching-local

sleep 1
start gateway .run/logs/gateway.log

echo
echo "  gateway  http://localhost:8082   (try: curl localhost:8082/system/status)"
echo "  web      cd web && pnpm dev       (http://localhost:5173)"
echo "  accounts driver@lastmile.demo / rider@lastmile.demo   password: lastmile"
echo
wait

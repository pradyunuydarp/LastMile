---
name: run-lastmile
description: Start the LastMile system (local processes or docker compose), check every service is healthy, and drive a complete ride end-to-end to prove a change works. Use when asked to run, start, demo, smoke-test or "see it working", or after changing a service, the gateway, or the web app.
---

# Run LastMile and prove it works

LastMile is 8 gRPC microservices + an HTTP/socket.io gateway + a React web app.
Pick the lightest way to run it that proves what you changed.

| Need | Command | Notes |
|---|---|---|
| Unit/contract tests | `make race` | Always run first. Gateway tests use real Matching/Location servers over bufconn. |
| Whole system, no Docker | `scripts/run-local.sh` | Builds to `./bin`, logs to `.run/logs/<svc>.log`, Ctrl-C stops all. One replica each. |
| Multiple Matching replicas | `docker compose up --build -d --scale matching=3` | Web on :8080, gateway on :8082. |
| Kubernetes | see the `k8s-demo` skill | |

## Verify, don't assume

1. Health of every service, as the gateway sees it:
   ```bash
   curl -s localhost:8082/system/status | python3 -m json.tool
   ```
   All eight should be `SERVING`. `matching.servedBy` shows which replicas answered.
2. A full ride through real services:
   ```bash
   scripts/smoke.sh
   ```
   Plans a route, starts a simulated trip, accepts the seeded bot rider at
   Electronic City, waits for `pending → in_progress → completed`.
3. Web UI: `cd web && pnpm install && pnpm dev` → http://localhost:5173.
   Demo accounts (password `lastmile`): `driver@lastmile.demo`,
   `driver2@lastmile.demo`, `rider@lastmile.demo`, `rider2@lastmile.demo`.
   Two-person flow: driver → Trips → pick stations → Save Route → Simulated
   Trip; rider → Book → station → destination → Book Ride; driver accepts the
   live offer (Riders tab); rider approves.

## Reading the logs

Every service logs JSON with `service` and `instance`. The interesting events:

```bash
grep -h "proximity trigger\|entered station zone\|picked up\|dropped off\|found riders" .run/logs/*.log
```

## Gotchas

- Bot riders (`bot-rider-*`) are seeded when the **gateway** starts and are
  consumed by rides. Restart the gateway to reset them.
- Without `SUPABASE_URL`/`SUPABASE_KEY` the User service runs in demo mode
  (in-memory accounts). That is expected, not an error.
- Map tiles come from openstreetmap.org; in a sandbox without internet the
  map is grey but everything else works.
- If port 8082 is busy, an old gateway is still running:
  `pkill -f '^bin/gateway$'`.

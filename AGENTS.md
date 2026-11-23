# LastMile: guide for AI agents and humans

Metro last-mile ride sharing as gRPC microservices on Kubernetes. Riders arrive
at a metro station and need a short ride to a nearby area; drivers plan a
route through stations with free seats. When a driver gets within a few
minutes of a station, matching runs and pairs them with waiting riders going
to the same place.

This file is the single source of truth for every coding agent (CLAUDE.md and
GEMINI.md import it). Keep it accurate when the code changes.

## Architecture

```
web (React) / mobile (Expo)
        │ HTTP + socket.io
        ▼
gateway (cmd/gateway, internal/api) ── /system/status, /healthz
        │ gRPC via chassis.Dial (round_robin + retries + deadlines)
        ├── user          :50052  auth + profiles (Supabase or in-memory demo accounts)
        ├── driver        :50051  driver profiles and routes
        ├── rider         :50055  rider registration
        ├── station       :50056  station metadata (seeded from internal/catalog)
        ├── location      :50054  GPS ingest, live fan-out, approach-zone detection
        ├── matching      :50053  STATELESS matching rules; the one we scale 1→5
        ├── trip          :50057  trip lifecycle API
        └── notification  :50058  notification API
```

Key flows:
- **Booking:** rider books → gateway asks `Matching.RankDrivers` → offers go
  to drivers over socket.io one at a time → driver accepts → rider approves →
  trip room opens.
- **Proximity trigger:** every GPS ping goes through `Location.UpdateLocation`;
  when a driver *enters* a station's 1.5 km approach zone the gateway calls
  `Matching.FindRiders` and pushes the result to the driver.
- **Trip:** `awaiting_rider → pending → in_progress (at pickup) → completed
  (at the rider's destination)`.

Known architectural debts (see the `microservices-review` skill): the gateway
still owns rider/trip/plan state; no token auth at the gateway; booking is not
a saga; stores are in-memory.

## Layout

- `api/*.proto` — gRPC contracts. Generated Go in `gen/go/` (`make proto`).
- `cmd/<service>/main.go` — ~15-line entrypoints built on the chassis.
- `internal/<service>/` — service logic + tests.
- `internal/api/` — the gateway (HTTP handlers, socket.io hub, simulation).
- `internal/pkg/chassis/` — health, reflection, graceful shutdown, keepalive,
  client dialing. Every service and client uses it.
- `internal/catalog/` — Bengaluru stations and nearby areas (reference data).
- `web/` — React + Vite + Tailwind. `mobile/` — Expo app.
- `k8s/` — manifests. `docker-compose.yml` — same system without Kubernetes.
- `scripts/run-local.sh`, `scripts/smoke.sh` — run everything, prove a ride works.
- `docs/demo/` — the class demo (runbook, slides, interactive page).
- `docs/history/` — earlier planning notes and agent prompts.

## Commands

```bash
make race            # all Go tests with the race detector (run before every commit)
make lint            # gofmt + go vet
scripts/run-local.sh # every service as a local process
scripts/smoke.sh     # one full ride against localhost:8082
docker compose up --build --scale matching=3
make k8s-up          # minikube: build images in-cluster + deploy
cd web && pnpm install && pnpm dev
```

Demo accounts when Supabase is not configured (password `lastmile`):
`driver@lastmile.demo`, `driver2@lastmile.demo`, `rider@lastmile.demo`,
`rider2@lastmile.demo`. Three bot riders are seeded at gateway start.

## Rules that keep this codebase healthy

Go
- `gofmt` everything. Context first argument. Return errors, don't panic.
- **Never hold a mutex while doing I/O** (gRPC, Postgres, HTTP). Copy what
  you need, unlock, call. The gateway once froze entirely because of this.
- **Every outbound call gets a deadline** (`context.WithTimeout`).
- Shared state behind `sync.Mutex`/`RWMutex`; `proto.Clone` what you store.
  `go test -race` must stay clean.
- Never keep a pointer into a slice across unlock/relock.
- Validate inputs; nil-check nested proto messages.
- Dial other services with `chassis.Dial`; build servers with `chassis.New`.

gRPC / proto
- Messages `UpperCamelCase`, fields `snake_case`, RPCs are verbs.
- Never reuse or renumber fields; `reserved` removed ones.

Tests
- Table-driven `Test<Thing>` with subtests. Pure rules get pure tests
  (`internal/matching/server_test.go`); service contracts are tested against
  real servers over `bufconn` (`internal/api/gateway_test.go`).

Commits & PRs
- Imperative subject (`Add driver matching scorer`). One concern per commit.
- PRs include what changed, why, and evidence (`make race`, smoke output,
  screenshots for UI).

Security
- No secrets in the repo or `k8s/*.yaml`; use env vars / Kubernetes Secrets.
- No PII (emails, tokens, passwords) in logs.

## Course spec (Project 1: LastMile)

Required services: User, Driver, Rider, Matching, Trip, Notification,
Location, Station — all gRPC, all deployable on Kubernetes. Demo must show:
matching riders with drivers; the system continuing to work when a service
fails; Matching scaling from 1 to 5 replicas (HPA or manual) while matches
continue. Matching triggers when a driver nears a station ("nearby" areas per
station) and the rider's destination must equal the driver's. In-memory
stores are acceptable. Full text: `docs/spec/overall-spec.md`,
`docs/spec/MicroserviceAppSpec.pdf`.

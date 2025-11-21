# Repository Guidelines

## Project Structure & Module Organization
- Go backend lives under `cmd/<service>/main.go` with shared code in `internal/<pkg>/` and reusable helpers in `pkg/`.  
- API definitions belong in `api/<service>.proto`; generated stubs go to `gen/go/`.  
- Frontend (if used) sits in `ui/`. Tests accompany code in the same package (`*_test.go`) or `ui/src/__tests__/`.

## Build, Test, and Development Commands
- `go mod tidy` – sync Go dependencies.  
- `go test ./...` – run all Go unit tests.  
- `go run ./cmd/<service>` – start a specific Go service locally.  
- `npm install && npm test` (run inside `ui/`) – install and test the React UI.  
- `npm start` (in `ui/`) – run the UI against local services.

## Coding Style & Naming Conventions
- Go: `gofmt` is required (`gofmt -w` before committing). Favor small interfaces, context-aware funcs (`ctx context.Context` first argument), and explicit errors. Packages use lowercase, no underscores; files are snake_case; exported identifiers are CamelCase.  
- Proto: use `snake_case` fields, `UpperCamelCase` messages/services; keep RPC names verb-based (e.g., `ListDrivers`).  
- JS/TS (UI): Prettier defaults; components in `PascalCase.tsx`; hooks in `useX.ts`.

## Testing Guidelines
- Go: table-driven tests named `Test<Thing>`; use subtests for variants. Prefer fast, hermetic unit tests; integration tests may live in `internal/<pkg>/integration`.  
- Aim to cover handlers, matching logic, and critical error paths.  
- UI: Jest/RTL tests in `__tests__`; favor role-based queries.

## Commit & Pull Request Guidelines
- Commits: imperative, concise subject (e.g., `Add driver matching scorer`, `Fix rider ETA parsing`). Group related changes; avoid mixed concerns.  
- PRs: include summary, testing evidence (`go test ./...`, UI tests if touched), and references to issues or spec sections. Add screenshots/GIFs for UI changes.

## Security & Configuration Tips
- Keep secrets out of the repo; use environment variables or Kubernetes secrets.  
- Validate all external input (HTTP/gRPC); propagate context deadlines; log without leaking PII.  
- Container images should be minimal (e.g., `distroless` or `alpine`) and expose only needed ports.

## Project 1 (LastMile) Quick Instructions
- Goal: microservice app for metro last‑mile drop service with Riders/Drivers; matching occurs minutes before driver reaches station.  
- Core services: User (auth/profiles), Driver (routes/stations/seats/location), Rider (arrival time/destination/status), Matching (location/time/destination match + notifications), Trip (lifecycle), Notification (push/real‑time), Location (ingest driver location + proximity), Station (station metadata/nearby areas).  
- API: define gRPC protos under `api/` (messages/fields in snake_case; RPCs verb-based). Generate stubs to `gen/go/`.  
- Minimal flow to demo: ingest station metadata → drivers register route + seats → riders post arrival time + destination → driver location updates hit Location svc → Matching triggers when driver nears station and destinations align → Trip created → Notification sent.  
- Kubernetes demo expectations: deploy all services; show matching works; demonstrate resilience by killing a service pod; scale Matching svc 1→5 replicas (HPA or manual) and show continued matches.  
- Start point suggestion: build Station + Driver + Rider + Matching first with in-memory stores; add Trip/Notification later; add tests for matching logic and proximity triggers.

## Project 1 (LastMile) Full Spec (from MicroserviceAppSpec.pdf)
- Purpose: connect metro riders with nearby drivers for short “last‑mile” drops; matching happens just minutes before a driver reaches the metro station. Riders publish arrival time + destination; drivers publish route, target station(s), and free seats, and stream live location updates.  
- Station metadata: each station maintains a list of “nearby” areas; when a driver’s live location enters a nearby area, matching can trigger. Rider destination must match the driver’s target destination for a ride to be assigned.  
- Required microservices (all gRPC, Kubernetes‑deployable):  
  - **User** – rider/driver profiles; authN/authZ.  
  - **Driver** – register routes, metro stations, seats; update live location + pickup status.  
  - **Rider** – register metro arrival time + destination; track ride status.  
  - **Matching** – match on location/time/destination; notify both parties.  
  - **Trip** – manage lifecycle (scheduled/active/completed); track pickup + drop‑off.  
  - **Notification** – push/real‑time updates to riders and drivers.  
  - **Location** – ingest real‑time driver GPS updates; detect proximity for triggers.  
  - **Station** – maintain station metadata; map stations to nearby areas.  
- Deployment/demo requirements: run on Kubernetes; prove fault resilience by killing a service while system continues operating; scale Matching service from 1 to 5 replicas (HPA or manual) and keep matching functional.  
- API expectations: publish Google gRPC service definitions; messages use `snake_case` fields, RPCs are verbs (e.g., `ListDrivers`, `RegisterRoute`).  
- Data sources: in‑memory stores are acceptable for the first demo; design for pluggable persistence later.  
- Success criteria: reliable matches near driver arrival; consistent trip state tracking; timely notifications; operational under single‑service failures; horizontal scalability of Matching.

## Project Context
- Audience: commuters requesting short rides from metro stations; drivers offering routes with limited seats.  
- Success criteria: reliable matches near station arrival time; consistent trip state tracking; timely notifications; system stays operational during single-service failures and scales matching under load.  
- Constraints: use gRPC between services; target Kubernetes deployment; prefer small, independently deployable services with clear contracts.  
- Tech defaults: Go for services; React (optional) for UI; in-memory storage acceptable initially, but design for pluggable persistence.  
- Observability: add structured logging and basic health/readiness endpoints per service to aid Kubernetes liveness/readiness probes.

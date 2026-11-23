# LastMile architecture, mapped to *Microservices Patterns*

Chapter references are to Chris Richardson, *Microservices Patterns: With
examples in Java* (Manning, 2018). Status: ✅ in place, 🟡 partial, ❌ not yet.

## Services (decompose by business capability — ch. 2)

| Service | Capability | State | Scales? |
|---|---|---|---|
| User | identity: sign-up, sign-in, profiles | Supabase or in-memory | 1 replica (in-memory) |
| Driver | driver profiles + registered routes | in-memory | 1 replica |
| Rider | rider registration | in-memory | 1 replica |
| Station | station catalog + nearby areas | seeded in-memory | yes (read-only) |
| Location | GPS ingest, live fan-out, approach-zone detection | in-memory last positions | 1 replica (subscribers are local) |
| **Matching** | **who rides with whom** | **none** | **yes, 1–5 (HPA)** |
| Trip | trip lifecycle API | in-memory | 1 replica |
| Notification | notification API | none | yes |
| Gateway | edge: HTTP + socket.io ↔ gRPC | rider/trip/plan state (debt) | 1 replica |

## Pattern map

| Pattern (chapter) | Where | Status |
|---|---|---|
| Decompose by business capability (2) | eight services above | ✅ |
| Remote procedure invocation, gRPC (3) | `api/*.proto`, verb RPCs, snake_case | ✅ |
| Client-side discovery + load balancing (3) | headless Service + `dns:///` + `round_robin` in `chassis.Dial` | ✅ |
| Timeouts / retries / fallbacks (3) | every call has a deadline; retry on UNAVAILABLE; cached snapshot when Driver is down; "matching unavailable" when no replica answers | ✅ |
| Circuit breaker (3) | — | ❌ (retries are bounded; no breaker state) |
| Asynchronous messaging (3) | socket.io pushes to clients; service-to-service is all sync | 🟡 |
| Saga (4) | booking: offer → accept → rider approval → trip, with compensations (decline/timeout restores the seat and re-queues the rider). Orchestrated by the gateway, in memory | 🟡 not durable |
| Database per service (5) | each service owns its map; but the gateway owns riders/trips/plans | 🟡 |
| API gateway / BFF (8) | `cmd/gateway`: HTTP + socket.io for browsers, gRPC fan-out | ✅ (too much business logic — see debts) |
| API composition (7) | `/aggregates/snapshot` composes Driver + Location + gateway state | ✅ |
| CQRS (7) | — | ❌ |
| Testing: contract tests (9–10) | gateway tested against real Matching/Location servers over bufconn; chassis tested with 3 real TCP servers | 🟡 in-process, not consumer-driven |
| Microservice chassis (11) | `internal/pkg/chassis` | ✅ |
| Health check API (11) | `grpc.health.v1` on every service, k8s gRPC probes, gateway `/healthz` | ✅ |
| Log aggregation (11) | structured JSON logs with `service` + `instance` | 🟡 ready to ship, no aggregator |
| Distributed tracing (11) | — | ❌ |
| Externalized configuration (11) | env vars, Kubernetes Secret for Supabase | ✅ |
| Access token (11) | sign-in returns a token, gateway does not verify it | ❌ |
| Service per container + Kubernetes (12) | distroless images, one Deployment per service, HPA | ✅ |
| Strangler application (13) | Matching was extracted out of the gateway into a real service; Trip is next | 🟡 |

## Key decisions

**Matching is stateless.** Every request carries the driver states and the
rider request; the service applies pure rules (`internal/matching/rules.go`)
and stores nothing. That is the whole reason it can scale 1→5 and survive a
killed pod with zero failed requests. The cost: the caller must send the
facts, so the gateway remains the owner of plan state for now.

**Headless Service for Matching.** gRPC multiplexes calls over one long-lived
HTTP/2 connection. Behind a normal ClusterIP Service that connection lands on
one pod, so all traffic goes there no matter how many replicas exist. A
headless Service lets DNS return every pod; `round_robin` spreads calls;
`MaxConnectionAge` makes clients re-resolve so new pods get traffic within
~30s. Verified: two replicas at 50/50 rebalanced to 25/25/25/25 after scaling
to four, at ~2,700 req/s with zero errors.

**Proximity is an edge, not a level.** Location reports `entered_zone` only on
the ping where a driver crosses into a station's 1.5 km approach zone (~4–5
minutes out at city speed). Matching runs on that edge, which is exactly the
spec's "match just minutes before the driver reaches the station" without
re-running on every ping.

**Graceful degradation over hard failure.** If Matching is down, booking
answers "matching unavailable, try again" and the rest of the app keeps
working. If the Driver service restarts empty, the gateway keeps serving its
last snapshot and re-registers the drivers it knows about.

## Debts and next steps

1. Extract trip lifecycle into the Trip service and rider requests into the
   Rider service (strangler pattern, ch. 13). The gateway should only route.
2. Make booking a durable orchestration saga with an outbox (ch. 3–4).
3. Verify access tokens at the gateway (ch. 11).
4. Publish location pings as events (NATS/Kafka) instead of synchronous calls.
5. OpenTelemetry tracing across gateway → services.

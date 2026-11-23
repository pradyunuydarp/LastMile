---
name: microservices-review
description: Review a LastMile change (or the whole repo) against microservices patterns from Chris Richardson's "Microservices Patterns" - service boundaries, data ownership, synchronous call safety, resilience, scalability, observability. Use when asked to review, audit, critique the architecture, or check a PR for distributed-systems bugs.
---

# Microservices review for LastMile

Review like an SRE who has been paged at 3am. For each finding give: file and
line, what breaks, a concrete failure scenario, and the pattern that fixes it.
Verify claims by running code (`go test -race`, a failing test, a curl) rather
than asserting.

## Checklist

### 1. Boundaries & data (Richardson ch. 2, "Decomposition"; ch. 5 "Database per service")
- Does each service own its data, or does the gateway hold another
  service's state? *Known debt:* the gateway still owns riders, trips and
  driver plans (`internal/api/gateway.go`). Trip/Rider/Notification services
  have contracts but are not yet the source of truth.
- Does a change add business logic to the gateway? Push it into a service.
  The gateway should route, compose and translate (API Gateway pattern, ch. 8).

### 2. Synchronous calls (ch. 3, "Interprocess communication")
- Every outbound gRPC call has a deadline (`context.WithTimeout`).
- No call is made while holding a mutex (the original gateway froze
  entirely when the Driver service was slow).
- Clients come from `chassis.Dial` (round_robin + retry on UNAVAILABLE).
- Retries only for idempotent operations, or the operation is made
  idempotent. Note: retrying a booking could double-book.
- What does the caller do when the callee is down? Fallback, cached data,
  or a clear "unavailable" (see `errMatchingUnavailable`).

### 3. Concurrency
- Shared maps/slices behind a mutex; run `go test -race ./...`.
- No pointer into a slice kept across unlock/relock (append may reallocate).
- I/O (Postgres, push notifications) happens after unlocking.

### 4. Resilience & deployment (ch. 11-12)
- Health: services use the chassis (grpc.health.v1); probes in `k8s/` match.
- Liveness checks only the process itself, never downstream services.
- Graceful shutdown drains in-flight work.
- State survives a pod restart, or the system rebuilds it (see the
  gateway's driver re-registration in `refreshDrivers`).

### 5. Scalability
- Is the service stateless? If not, it cannot scale past one replica.
- gRPC behind Kubernetes: multi-replica targets need a headless Service +
  `dns:///` + round_robin, or all traffic pins to one pod.

### 6. Observability (ch. 11)
- Structured logs with `service` and `instance`; no PII (emails,
  passwords, tokens) in logs.
- Responses that matter say who served them (`served_by`, `matchedBy`).

### 7. Security
- Secrets in env/Secrets, never in `k8s/*.yaml` or the repo.
- *Known gap:* the gateway does not verify access tokens; any client can
  act as any driverId. Access Token pattern (ch. 11) is the fix.
- Inputs validated at the edge and in each service.

## Known debts (don't re-report as new; do mention if a change makes them worse)

1. Gateway owns riders/trips/plans (distributed monolith smell).
2. No auth on gateway endpoints.
3. Booking flow is a chain of in-memory steps, not a saga; a gateway crash
   mid-booking loses it (ch. 4, Sagas).
4. No distributed tracing; correlate by tripId/driverId in logs.
5. In-memory stores everywhere except optional Postgres in the gateway.

## Output format

Group by severity (breaks correctness / breaks under failure / scalability /
hygiene). Lead with the single most important finding.

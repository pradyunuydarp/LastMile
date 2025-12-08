# LastMile Architecture (frontend + gRPC edge)

## Services
- **Driver / Rider / Station / Trip / Location / Notification / Matching**: gRPC microservices defined in `api/*.proto`, implemented under `cmd/<service>` + `internal/<service>`.
- **User**: gRPC microservice for rider/driver profiles (`api/user.proto`).
- **Gateway**: new edge service (`cmd/gateway`) that aggregates in-memory state, exposes gRPC (`GatewayService`) and keeps a small HTTP compatibility surface for the mobile client (`/aggregates/snapshot`, `/matching/match`).
- **Mobile app**: Expo/React Native in `mobile/`, calls the HTTP endpoints by default; can be pointed at the gRPC edge if you add a gRPC-capable client.

## Gateway contract (gRPC)
- **Proto**: `api/gateway.proto` → `lastmile.gen.go.gateway`.
- **RPCs**:
  - `GetSnapshot(SnapshotRequest) -> BackendSnapshot` – aggregated drivers, riders, trips, stations, metrics, highlight trip, last_updated.
  - `TriggerMatch(TriggerMatchRequest) -> GatewayTrip` – creates a pending trip for a driver/station pair and marks the rider as matched.
- Messages are shaped for the mobile UI: `GatewayDriver` (seats/status/ETA/route), `GatewayRider` (arrival_time ISO), `GatewayTrip` (station/destination/ETA), `GatewayStation` (load_factor), plus `BackendMetrics`.

## Deployment / ports
- **Gateway gRPC**: `:50060` (override `GATEWAY_GRPC_ADDR`).
- **Gateway HTTP**: `:8082` (override `GATEWAY_HTTP_ADDR`), same paths the mobile app already uses.
- Other services default to separate ports: driver `:50051`, user `:50052`, matching `:50053`, location `:50054`, rider `:50055`, station `:50056`, trip `:50057`, notification `:50058`. Adjust via `*_GRPC_ADDR` env vars or front them with Envoy/Ingress.

## Local runs
```bash
# gRPC + HTTP edge
go run ./cmd/gateway
# Mobile (points at HTTP)
EXPO_PUBLIC_API_URL=http://localhost:8082 pnpm start --filter mobile

# Regenerate Go stubs (module-aware output to gen/go)
protoc -I . \
  --go_out=module=lastmile:. \
  --go-grpc_out=module=lastmile:. \
  api/driver.proto api/rider.proto api/matching.proto api/trip.proto api/station.proto api/location.proto api/notification.proto api/gateway.proto
```
Add `-grpc` or k8s service annotations as needed for cluster ingress. The gateway uses in-memory state for now; swap the data layer with real service calls as they come online.

## Kubernetes + scaling notes
- Package each `cmd/<service>` as a Deployment with its own Service; use HPA or `kubectl scale` on Matching (`cmd/matching`) from 1→5 replicas to satisfy the scaling demo.
- Deploy the Gateway (gRPC + HTTP) behind an Ingress/Envoy that also routes to the mobile HTTP surface.
- Prove resilience by killing a service pod (e.g., `kubectl delete pod <matching-pod>`) while Gateway continues serving cached/in-memory data; Location service proximity triggers can be turned on with `MATCHING_ADDR=<matching-host:port>` to auto-call matching when a driver enters a station radius.

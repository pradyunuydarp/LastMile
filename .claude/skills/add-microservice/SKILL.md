---
name: add-microservice
description: Add a new microservice to LastMile end-to-end - proto, server, main on the chassis, tests, Docker, compose, Kubernetes, gateway wiring. Use when asked to create, split out, or extract a service (e.g. moving trip lifecycle out of the gateway).
---

# Adding a microservice

Every service follows the same shape. Copy it exactly; the consistency is
what makes eight services manageable.

```
api/<name>.proto                 contract
gen/go/<name>/                   generated (make proto)
internal/<name>/server.go        gRPC handlers, state behind a mutex
internal/<name>/server_test.go   table-driven tests
cmd/<name>/main.go               ~15 lines on the chassis
```

## 1. Contract

Write `api/<name>.proto` (see the `grpc-contract-change` skill), then `make proto`.

## 2. Server

- Embed `pb.Unimplemented<Name>ServiceServer`.
- In-memory state is fine for the course, but guard it with `sync.RWMutex`
  and `proto.Clone` what you store and return. gRPC calls handlers
  concurrently; `go test -race` will catch you if you forget.
- Decide up front: **stateless** (can scale to N replicas freely, like
  Matching) or **stateful** (one replica unless you add a database).

## 3. Main on the chassis

```go
func main() {
	svc := chassis.New("<name>")
	pb.Register<Name>ServiceServer(svc.GRPC, <name>.NewServer(svc.Logger))
	if err := svc.Run(chassis.Getenv("<NAME>_GRPC_ADDR", ":<port>")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
```

The chassis gives you health checks, reflection (`grpcurl -plaintext localhost:<port> list`),
graceful shutdown and load-balancing-friendly keepalives for free.

Ports in use: driver 50051, user 50052, matching 50053, location 50054,
rider 50055, station 50056, trip 50057, notification 50058, gateway 8082/50060.

## 4. Callers

Dial with `chassis.Dial(target)` (round_robin + retries). Every call gets a
`context.WithTimeout`. Never call another service while holding a mutex.

## 5. Deploy

- `Makefile`: add to `SERVICES`.
- `docker-compose.yml`: add a service block (copy `trip`).
- `k8s/lastmile.yaml`: Deployment + Service with gRPC probes (copy `trip`);
  headless Service only if it will run multiple replicas behind gRPC clients.
- `scripts/run-local.sh`: add a `start` line.
- Gateway: add the target to the `targets` map in `cmd/gateway/main.go` so
  it shows up in `/system/status`.

## 6. Verify

`make race lint`, then the `run-lastmile` skill (`/system/status` should list
the new service as SERVING).

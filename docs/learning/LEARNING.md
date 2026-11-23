# LastMile Learning & Build Guide

Steps for learning Go and building the LastMile microservices project.
## How to Learn by Doing (Go + Project)
1) Do → Read → Do again: implement a small slice, skim the concept, then refactor.  
2) Keep `gofmt` and tests on repeat; they teach idioms fast.  
3) Ask “what’s the contract?” first: proto/handler/test before plumbing.

## Minimal Environment Setup
- Go already installed (`go1.25.x`). Ensure tools on PATH: `export PATH="$HOME/go/bin:$PATH"`.  
- Local caches to avoid macOS restrictions: `GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache`.
- Proto toolchain (done): `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`.

## Daily Dev Loop
```bash
export PATH="$HOME/go/bin:$PATH"
export GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache
go test ./...
go run ./cmd/station   # or another service
```
Format/lint: `gofmt -w <files>` (run before commits).

## Project Walkthrough (stepwise)
1) **APIs first**: define proto in `api/<service>.proto`; generate with `protoc ... --go_out --go-grpc_out`.  
2) **Service skeleton**: `cmd/<service>/main.go` starts gRPC server, loads config, registers handlers.  
3) **Core logic**: in `internal/<service>/`; keep state behind interfaces for easy fakes.  
4) **Tests**: table-driven `*_test.go` beside code; focus on logic before transport.  
5) **Run & probe**: use `grpcurl` or curl; add health endpoints early.  
6) **Iterate**: add Driver, Rider, Matching, Trip, Notification, Location services following the Station pattern.  
7) **Kubernetes demo**: add manifests/Helm; readiness/liveness probes; scale Matching 1→5; kill a pod to show resilience.

## What to Learn When
- Day 0: Go syntax, modules, `context`, errors, slices/maps, `net/http`.  
- Day 1: gRPC basics, proto conventions (snake_case fields, verb RPCs), codegen.  
- Day 2: Testing patterns, dependency injection via interfaces, concurrency only where needed (location streams).  
- Later: Observability (structured logging), retries/timeouts, Kubernetes basics (Service, Deployment, HPA).

## Reference Commands
- Generate stubs:  
  `protoc --go_out=gen/go --go_opt=paths=source_relative --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative api/<file>.proto`
- Station service example call:  
  `grpcurl -plaintext -d '{"station":{"id":"stn-1","name":"Central","metro_line":"Blue","nearby_areas":["Downtown"]}}' localhost:50051 station.v1.StationService/UpsertStation`

## Workflow Tips
- Keep commits small, imperative subjects.  
- Prefer pure functions and tiny interfaces; return errors instead of panics.  
- Copy request structs before storing to avoid external mutation.  
- Use context deadlines in clients; propagate `ctx` as first param.  
- Document environment variables in service `main.go` defaults.

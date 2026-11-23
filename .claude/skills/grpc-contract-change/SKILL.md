---
name: grpc-contract-change
description: Change a LastMile gRPC API safely - edit api/*.proto, regenerate gen/go, update the server, the gateway adapter and tests. Use whenever a .proto file, a message field, or an RPC is added, renamed or removed.
---

# Changing a gRPC contract

The `.proto` files in `api/` are the contracts between services. Treat them
like a public API: other services (and old pods during a rolling update) are
still speaking the previous version.

## Rules

- Messages `UpperCamelCase`, fields `snake_case`, RPCs are verbs
  (`RankDrivers`, `ListStations`).
- **Never reuse or renumber a field.** To remove one, delete it and add
  `reserved <n>;` so the number can't come back with a new meaning.
- Adding a field is safe (old clients ignore it, new code must handle the
  zero value). Changing a field's type or meaning is not: add a new field.
- Validate inputs in the handler and return `codes.InvalidArgument`. Never
  dereference a nested message without a nil check (two services used to
  panic on empty payloads).

## Steps

1. Edit `api/<service>.proto`.
2. Regenerate:
   ```bash
   make proto
   ```
   Needs `protoc` plus `protoc-gen-go` v1.36.x and `protoc-gen-go-grpc` v1.5.1:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
   ```
   No `protoc` on the machine? `pip install grpcio-tools` in a venv and use
   `python -m grpc_tools.protoc` with the same flags as the Makefile.
   Only commit regenerated files for protos you changed (others differ only
   in the version header).
3. Implement in `internal/<service>/server.go`. Keep business rules in pure
   functions (see `internal/matching/rules.go`) and the gRPC layer thin.
4. Update callers. The gateway adapts between HTTP JSON and gRPC in
   `internal/api/` (e.g. `matching_client.go`).
5. Test the contract, not a mock: `internal/api/gateway_test.go` shows how to
   start the real server over `bufconn` (`startGRPC`).
6. `make race lint`.

## Checklist before you're done

- [ ] `go build ./... && make race lint` pass
- [ ] Field numbers unchanged for existing fields
- [ ] Handler validates required fields
- [ ] Gateway and loadgen (`cmd/loadgen`) still compile and send the new shape
- [ ] `docs/` or the slide that shows this RPC updated if the shape changed

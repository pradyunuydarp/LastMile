@AGENTS.md

## Claude Code specifics

Project skills live in `.claude/skills/` and load when relevant:

| Skill | Use it for |
|---|---|
| `run-lastmile` | starting the system and proving a change works end-to-end |
| `k8s-demo` | minikube deploy, resilience and autoscaling demo, cluster debugging |
| `grpc-contract-change` | any edit to `api/*.proto` |
| `add-microservice` | creating or extracting a service |
| `microservices-review` | reviewing a change against Richardson's patterns |

Before saying a change works: `make race lint`, and for anything touching a
flow, `scripts/smoke.sh` against a running stack. Report the actual output.

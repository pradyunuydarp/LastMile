# How LastMile was built with AI coding agents

This project was built by a student working with several AI coding agents
(Codex via `AGENTS.md`, Gemini CLI via `GEMINI.md`, and later Claude Code).
This page is the honest version of that story: what worked, what quietly
broke, and the workflow that fixed it. It is part of the class demo.

## Phase 1: prompting features into existence

The early workflow was "describe the next feature, let the agent build it".
The prompts are preserved in `docs/history/`:

- `next-prompt.md` — a real prompt: *"great! it works. Now use sockets and
  libraries like socket.io to do matchmaking… divide it into phases: DB,
  backend, frontend(ios), frontend(web)."*
- `todo-tasks.md` — a backlog handed to an agent in one go.
- `walkthrough.md` — the agent's own report of what it built, ending in
  "All Go tests passed."

This was fast. In a few sessions the project had eight services, a web app, an
iOS app, socket.io trip rooms and live maps. The demo *looked* finished.

## Phase 2: a review found what the demo hid

A full review with Claude Code (reading every service, then running things
instead of trusting them) found:

| Finding | How it was proven |
|---|---|
| The **Matching service was a stub** returning a hard-coded `rider-456`. All real matching lived in the gateway, so "scale Matching 1→5" scaled nothing. | Read `internal/matching/server.go` |
| **Data races**: every service stored state in plain maps; gRPC handlers run concurrently. | A 50-goroutine test + `go test -race` → `concurrent map writes` |
| **Scaling wouldn't spread load**: gRPC over a normal ClusterIP pins every call to one pod. | `TestDefaultClientPinsOnePod`: 300/300 calls hit one replica |
| The gateway **held its global lock across un-timed gRPC calls**: one slow service froze every request. | Code reading; fixed with deadlines + lock discipline |
| Location used **San Francisco coordinates** for Bengaluru stations, so the proximity trigger could never fire. | Copy-paste from a template, never exercised |
| Trips **"completed" the instant the rider got in** (drop-off checked against the pickup station). | Simulated a ride end-to-end |
| Live bookings **stalled**: the rider's approval prompt was dismissed a millisecond after it appeared. | Two-browser Playwright test |
| Two gateway tests **failing on main**; all User-service tests **silently skipped**. | `go test ./...` |
| A 16 MB macOS binary committed; a Supabase key in a k8s manifest; a web Dockerfile that couldn't build (Node 18 vs Vite 7). | `file`, `grep`, reading Dockerfiles |

None of these showed up in a happy-path click-through. Each agent had done
what it was asked; nobody had asked "does the whole thing actually satisfy
the spec under failure?"

## Phase 3: the workflow that works

1. **Give the agent the spec, not just the next feature.** `AGENTS.md` now
   holds the architecture, the rules (lock discipline, deadlines, race-free
   state), and the course requirements. `CLAUDE.md` and `GEMINI.md` import it,
   so every agent reads the same thing.
2. **Encode the spec as tests.** Matching rules are table-driven tests, one row
   per rule. Load balancing and failover are unit tests with three real gRPC
   servers. The gateway is tested against real services over `bufconn`.
3. **Make the agent prove it.** `go test -race`, `scripts/smoke.sh` (a full
   ride through real processes), Docker with `--scale matching=4` and a
   replica killed mid-traffic, and a two-browser Playwright run. "It compiles"
   is not evidence.
4. **Capture repeatable workflows as skills** (`.claude/skills/`):
   `run-lastmile`, `k8s-demo`, `grpc-contract-change`, `add-microservice`,
   `microservices-review`. The next session starts from the team's playbook
   instead of from scratch.
5. **Keep the human in charge of permissions.** The agent proposed a hook and
   an allowlist (below) but does not install its own permissions; that's a
   decision for the developer.

## Optional: guardrails you can install

Add to `.claude/settings.json` yourself if you want them. The hook runs
`gofmt` on every Go file Claude edits (deterministic, unlike an instruction);
the allowlist skips prompts for read-only/test commands.

```json
{
  "permissions": {
    "allow": ["Bash(go test:*)", "Bash(go vet:*)", "Bash(go build:*)",
              "Bash(make race)", "Bash(make lint)", "Bash(scripts/smoke.sh)"]
  },
  "hooks": {
    "PostToolUse": [{
      "matcher": "Edit|Write",
      "hooks": [{"type": "command",
                 "command": "f=$(sed -n 's/.*\"file_path\" *: *\"\\([^\"]*\\)\".*/\\1/p' | head -1); case \"$f\" in *.go) gofmt -w \"$f\";; esac; exit 0"}]
    }]
  }
}
```

## Takeaways for students

- AI agents are excellent at *volume*: scaffolding, UI, glue code.
- They are weak at *system properties nobody tested*: concurrency, failure
  behaviour, load distribution, "does the stub ever get replaced?".
- Architecture knowledge is what lets you ask the right question. Every bug
  above maps to a pattern in Richardson's *Microservices Patterns*.

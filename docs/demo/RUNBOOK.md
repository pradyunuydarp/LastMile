# LastMile class demo: runbook

A 45-minute session for *Software Architecture & Design Patterns*: explain
microservices with a real system, break it live, and show how it was built
with an AI coding agent.

| You need | Where |
|---|---|
| Slide deck (24 slides, speaker notes on every slide) | Your claude.ai artifacts: “LastMile: Microservices, Live” (private until you share it) |
| Interactive control room (works offline, no install) | `docs/demo/index.html`: open it in any browser |
| Fallback evidence (real outputs + screenshots) | `docs/demo/evidence/` |
| The story of building it with AI | `docs/ai-workflow.md` |

---

## The day before

- [ ] `git pull` on this branch; `make race` passes.
- [ ] `minikube start --cpus=4 --memory=8g`, then `make k8s-up`. First build takes a few minutes; do it now, not in class.
- [ ] `kubectl -n lastmile get pods`: everything `Running`, `1/1 READY`.
- [ ] `kubectl -n lastmile get hpa`: shows a percentage, not `<unknown>`. If unknown, wait 60 s for metrics-server.
- [ ] Do one full rehearsal of all three demos below. Then `make load-stop` and `make scale-down`.
- [ ] Open `docs/demo/index.html` once to check it renders on the classroom laptop.
- [ ] Optional cloud backup: open <https://lastmile-web-ihr9.onrender.com> and run `GATEWAY=https://lastmile-stack.onrender.com scripts/smoke.sh`. It is the free Render tier, so it sleeps after 15 idle minutes; open it again 5 minutes before class. It can't do the kill-a-pod or scaling demos (one container, see `docs/deploy.md`).
- [ ] Fill in `[Your name] · [Course code] · [Date]` on the cover slide.

## 15 minutes before

Arrange four windows (big font, dark theme):

1. **Slides**, presenting.
2. **Terminal A**: `kubectl -n lastmile get pods -w` (leave running).
3. **Terminal B**: commands. First run
   `kubectl -n lastmile port-forward svc/gateway 8082:80` in a background tab,
   then `curl -s localhost:8082/system/status | python3 -m json.tool` to confirm all eight services say `SERVING`.
4. **Two browser windows side by side**: `minikube service web -n lastmile --url`, open it twice
   (one normal, one private window). Log in as **Ravi · driver** (`driver2@lastmile.demo`)
   and **Kabir · rider** (`rider2@lastmile.demo`); password `lastmile`.

Restart the gateway if you rehearsed (resets the bot riders):
`kubectl -n lastmile rollout restart deploy/gateway`.

---

## Timeline

| Min | Slides | What happens |
|---|---|---|
| 0–3 | 1–3 | Hook: the 2 km walk from Electronic City metro. Route for the session. |
| 3–12 | 4–8 | Architecture: monolith vs microservices, the eight services, one ride step by step (click to reveal), the approach-zone trigger, Matching as a pure function. |
| 12–17 | 9–11 | Patterns table, the API gateway (and its confession), the chassis. |
| 17–30 | 12–16 | **Live demo** (below). Slide 16 explains the scaling bug after they've seen scaling work. |
| 30–38 | 17–21 | Built with AI: the real prompt, what the review found, proof, workflow, skills. |
| 38–45 | 22–24 | What we'd fix next, discussion questions in pairs, thanks + repo. |

If you're short on time, skip slides 10–11 and the Driver-kill in demo 2.

---

## Demo 1: a real match (≈ 4 min) · slide 13

1. **Driver window** → Trips → tap *Konappana Agrahara* and *Huskur Road* → **Save Route** → **Simulated Trip**.
2. **Rider window** → Book → *Konappana Agrahara Metro Station* → *Siemens Campus* → **Book Ride**.
   Point at the purple **“Decided by matching-…”** badge: *“that's the pod that made the decision.”*
3. **Driver window** → Riders tab → **Live socket offers** → **Accept**.
4. **Rider window** → **Approve ride**. Both windows open the trip room.
5. While the car drives: the driver gets **“Approaching Konappana”**: *“that's the Location service's 1.5 km trigger.”*
   Seats go 3 → 2 at pickup and back to 3 at Siemens Campus.

Say: *“The browser only ever talked to the gateway. Behind it, User signed us in, Driver holds the route, Location watched the car, and Matching made both decisions, all over gRPC.”*

## Demo 2: kill things (≈ 4 min) · slide 14

```bash
make chaos                                   # deletes a random Matching pod
```

Terminal A shows the pod `Terminating` and a new one `ContainerCreating` → `Running`.
Keep booking in the rider window during this: it keeps working.

Then the scarier one (optional):

```bash
kubectl -n lastmile delete pod -l app=driver
curl -s localhost:8082/system/status | python3 -m json.tool   # driver DOWN, driverDataFresh false
```

The web app keeps showing drivers (gateway cache). When Driver is back, gateway logs
show it re-registering them: `kubectl -n lastmile logs deploy/gateway | grep re-register`.

Ask the room: *“What if I kill the gateway?”* (It's the single point of failure. Good discussion.)

## Demo 3: scale 1 → 5 (≈ 5 min) · slides 15–16

```bash
make load-start
kubectl -n lastmile get hpa -w               # REPLICAS climbs 1 → 2 → 4 → 5 over ~1–2 min
kubectl -n lastmile logs -f deploy/loadgen   # served_by spreads across pod names
```

While it climbs, book another ride in the rider window: the “Decided by” pod name changes.
Then `make load-stop`. Scale-down takes about a minute.

**If the HPA is slow** (metrics-server not ready): `make scale-up` to go to 5 by hand. Same lesson.

Then go to slide 16 and explain why this almost didn't work (headless Service, round robin,
30-second reconnect). Optional: in the control room page, press **Show the scaling bug**.

---

## Optional: Claude Code live (≈ 5 min, slides 20–21)

Open Claude Code in the repo and type:

> Using the microservices-review skill, check what happens if two drivers accept the same rider at the same moment. Don't change any code.

It's the same as discussion question 1, so the answer sets up the pair work.
Point out that it loaded the skill on its own, and that it cites files and line numbers.

---

## If something breaks

| Problem | Do this |
|---|---|
| minikube won't start / laptop too slow | `docker compose up --build -d --scale matching=3`, web on `localhost:8080`, kill with `docker kill lastmile-matching-2`, scale with `docker compose up -d --scale matching=4 matching`, load with `docker compose --profile load up -d loadgen` and `docker compose logs -f loadgen`. |
| No Docker at all | `scripts/run-local.sh` + `cd web && pnpm dev`. Demo 1 works; for demos 2–3 use the control room page. |
| Web app won't load | `scripts/smoke.sh` runs a whole ride in the terminal, with narration. |
| Projector Wi-Fi is down | Everything above works offline. The map tiles go grey; the demo doesn't need them. |
| Live demo dies completely | `docs/demo/index.html` (the control room) + `docs/demo/evidence/`: real screenshots, the race-detector output, and the loadgen log showing 50/50 → 25/25/25/25. |
| Rider never gets the Approve button | The rider must be logged in when the driver accepts (approval times out after 25 s). Book again. |
| “No drivers available” | The driver must press **Simulated Trip** first (rule 1: trip started), and the rider's station must be on the route and not yet passed. The booking card lists why each driver was ruled out. |

---

## Answers for the discussion questions (slide 23)

1. **Two drivers accept the same rider.** Today it works by accident: one gateway, one mutex.
   With two gateway replicas both could win. Put the decision in one place with an atomic
   check (the Trip service doing a conditional write, or a seat reservation keyed by rider),
   and make accept idempotent.
2. **Location can't run five replicas** because live subscribers are Go channels inside one
   process; a driver's pings and a rider's subscription could land on different pods. Fix with
   pub-sub (Redis, NATS) between Location replicas. Proximity state (“inside zone”) also has to be shared or sticky.
3. **Next service to extract**: Trip is the natural one. What breaks first: every place the
   gateway reads or writes `g.trips` directly (pickup, drop-off, snapshot). That's why the
   strangler pattern goes one responsibility at a time.

## Likely questions

- **Why gRPC and not REST between services?** Typed contracts (`api/*.proto`), generated
  clients, streaming for location. Browsers can't speak gRPC easily, which is why there's a gateway.
- **Is in-memory state cheating?** The spec allows it for the demo. It's also why Driver
  restarting empty is interesting: the gateway has to heal it.
- **Why is Matching stateless but Location isn't?** Matching is a pure decision; Location has
  to remember last positions and who is subscribed.
- **Did the AI write all of this?** It wrote most of the code. The review, the tests that
  encode the spec, and deciding what “done” means were the human part, and they caught
  everything on slide 18.

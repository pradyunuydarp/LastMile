---
name: k8s-demo
description: Deploy LastMile to Kubernetes (minikube) and run the course demo - all services up, matching works, kill a pod and keep working, scale Matching 1 to 5 with the HPA. Use for anything about the cluster, manifests in k8s/, autoscaling, resilience demos, or "it works locally but not in minikube".
---

# Kubernetes demo

Course requirements this demo must show (from `docs/spec/overall-spec.md`):
1. every service deployed on Kubernetes,
2. riders matched with drivers,
3. the app keeps working when a service fails,
4. Matching scales from 1 to 5 replicas and keeps matching.

The full live script with timings is `docs/demo/RUNBOOK.md`. This skill is
the operator's view.

## Deploy

```bash
minikube start --cpus=4 --memory=8g
make k8s-up            # builds images inside minikube, enables metrics-server, applies k8s/
kubectl -n lastmile get pods -w
```

Web: `minikube service web -n lastmile`. Gateway for scripts:
`kubectl -n lastmile port-forward svc/gateway 8082:80` then `scripts/smoke.sh`.

## The three demo beats

| Beat | Do | Show |
|---|---|---|
| Match | `scripts/smoke.sh` (or the web two-person flow) | trip goes pending → in_progress → completed |
| Resilience | `make chaos` (kills a random Matching pod) or `kubectl -n lastmile delete pod -l app=driver` | `curl localhost:8082/system/status`; bookings keep succeeding; Driver comes back and the gateway re-registers drivers |
| Scale | `make load-start`, then `kubectl -n lastmile get hpa -w` | REPLICAS climbs 1 → 5; `kubectl -n lastmile logs -f deploy/loadgen` shows `served_by` spreading across pods |

Manual scaling (if metrics-server is slow): `make scale-up` / `make scale-down`.

## Why it works (explain these, they are the lesson)

- **Headless Service for Matching** (`clusterIP: None` in `k8s/lastmile.yaml`):
  DNS returns every pod IP. With a normal ClusterIP, gRPC opens one HTTP/2
  connection to one pod and scaling does nothing. Proven in
  `internal/pkg/chassis/chassis_test.go` (`TestDefaultClientPinsOnePod`).
- **round_robin + retries** in `chassis.Dial`: spreads calls and retries
  `UNAVAILABLE` on another replica when one dies.
- **MaxConnectionAge = 30s** in the chassis: clients reconnect and re-resolve
  DNS, so new pods get traffic within ~30s of scaling up.
- **gRPC health probes**: readiness flips to NOT_SERVING during graceful
  shutdown, so Kubernetes stops routing before the pod exits.
- **Stateless Matching**: nothing to lose when a pod dies.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `ErrImagePull` | Images weren't built inside minikube. Run `eval $(minikube docker-env)` in that shell, then `make images web-image`. `imagePullPolicy: IfNotPresent` is set. |
| HPA shows `<unknown>/60%` | metrics-server not ready: `minikube addons enable metrics-server`, wait ~60s. |
| Probe errors mentioning `grpc` | Native gRPC probes need Kubernetes ≥ 1.24. `minikube start --kubernetes-version=stable`. |
| All load on one Matching pod | Someone changed the Service back to ClusterIP or the target lacks `dns:///`. |
| Web loads but API 502s | `web` nginx proxies to `gateway.lastmile.svc.cluster.local:80`; check `kubectl -n lastmile get endpoints gateway`. |
| Supabase auth wanted | `kubectl -n lastmile create secret generic lastmile-supabase --from-literal=SUPABASE_URL=... --from-literal=SUPABASE_KEY=...` then `kubectl -n lastmile rollout restart deploy/user`. |

Never commit keys into `k8s/*.yaml`.

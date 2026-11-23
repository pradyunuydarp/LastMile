# Kubernetes deployment

The live demo script is [`docs/demo/RUNBOOK.md`](demo/RUNBOOK.md); the
operator notes and troubleshooting table are in the `k8s-demo` skill
(`.claude/skills/k8s-demo/SKILL.md`). This page is the reference.

## Deploy to minikube

```bash
minikube start --cpus=4 --memory=8g
make k8s-up     # builds all images inside minikube, enables metrics-server, applies k8s/
kubectl -n lastmile get pods
minikube service web -n lastmile
```

`make k8s-up` builds with `eval $(minikube docker-env)`, so no registry is
needed. To push to Docker Hub instead: `make images` then `docker push` each
`pradyunuydarp/lastmile-<svc>:latest`.

## What's in `k8s/`

| File | Contents |
|---|---|
| `lastmile.yaml` | namespace, 8 services + gateway (Deployment + Service each), Matching HPA |
| `web.yaml` | React app behind nginx (proxies API + socket.io to the gateway) |
| `loadgen.yaml` | load generator for the autoscaling demo |

Every Deployment has native gRPC readiness/liveness probes (Kubernetes ≥ 1.24),
CPU/memory requests and limits, and `POD_NAME` from the downward API.
Matching's Service is **headless** (`clusterIP: None`); see
[`architecture.md`](architecture.md) for why.

## Optional: Supabase

```bash
kubectl -n lastmile create secret generic lastmile-supabase \
  --from-literal=SUPABASE_URL=https://<project>.supabase.co \
  --from-literal=SUPABASE_KEY=<anon-key>
kubectl -n lastmile rollout restart deploy/user
```

Without the Secret the User service uses in-memory demo accounts.

## Cleanup

```bash
make k8s-down   # deletes the lastmile namespace
```

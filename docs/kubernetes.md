# Kubernetes Deployment Guide

This guide explains how to deploy the LastMile microservices and gateway on a Kubernetes cluster using the manifests under `k8s/`.

## Prerequisites
- Docker (logged into Docker Hub as `pradyunuydarp`)
- `kubectl` configured to talk to your cluster
- Metrics server installed (`kubectl top nodes` should work) so the Matching HPA can function

## 1. Spin Up a Free Local Cluster (Minikube)
For a zero-cost, course-friendly Kubernetes environment we recommend [Minikube](https://minikube.sigs.k8s.io/). It runs a single-node cluster on your laptop while still exposing all the primitives (HPA, Services, pod killing, etc.) required by the LastMile demo.

```bash
brew install minikube kubectl            # once
minikube start --kubernetes-version=v1.30.0 \
  --cpus=4 --memory=8g --driver=hyperkit # adjust resources as needed

# Enable addons we rely on
minikube addons enable metrics-server    # required for matching HPA
minikube addons enable ingress           # optional but handy for future routes

# Point kubectl at the Minikube context
kubectl config use-context minikube
```

You can either push images to Docker Hub (see next section) or build directly inside Minikube’s Docker daemon:

```bash
eval "$(minikube docker-env)"            # temporarily use the in-cluster Docker
make build-all                           # or make push-all if you prefer Docker Hub
```

If you built inside Minikube, skip `make push-all`; the images are already present in the VM.

## 2. Build & Push Images
```bash
# From the repo root
make push-all                 # builds every cmd/<service> and pushes pradyunuydarp/lastmile-*:latest

# Optional: override the tag or registry prefix
TAG=v0.1.0 IMAGE_PREFIX=pradyunuydarp/lastmile make push-all
```

Each binary is produced by the multi-stage `Dockerfile` and uploaded as:
- `pradyunuydarp/lastmile-driver:latest`
- `pradyunuydarp/lastmile-rider:latest`
- `pradyunuydarp/lastmile-station:latest`
- `pradyunuydarp/lastmile-trip:latest`
- `pradyunuydarp/lastmile-notification:latest`
- `pradyunuydarp/lastmile-location:latest`
- `pradyunuydarp/lastmile-matching:latest`
- `pradyunuydarp/lastmile-user:latest`
- `pradyunuydarp/lastmile-gateway:latest`

## 3. Deploy to Kubernetes
```bash
# Apply the namespace + every Deployment/Service/HPA in one go
kubectl apply -f k8s/lastmile.yaml

# Or deploy individual services if needed
kubectl apply -f k8s/driver.yaml
kubectl apply -f k8s/rider.yaml
# ...and so on for station, trip, notification, location, matching, user, gateway
```

Verify everything came up:
```bash
kubectl get pods -n lastmile
kubectl get svc -n lastmile
kubectl get hpa matching -n lastmile
```

If you are using Minikube and want public LoadBalancer-style access, run `minikube tunnel` in a separate terminal so Kubernetes Services of type `LoadBalancer` receive an external IP. Otherwise, use port-forwarding as shown below.

## 4. Access the Gateway
The gateway exposes gRPC (`:50060`) and HTTP (`:8082`) inside the cluster. For local testing:
```bash
kubectl port-forward -n lastmile svc/gateway 8082:80 50060:50060

# Mobile app expects HTTP; point Expo at the forwarded URL
EXPO_PUBLIC_API_URL=http://localhost:8082 pnpm start --filter mobile
```

Alternatively, if you prefer to use the Kubernetes Service URL that Minikube exposes:
```bash
minikube service -n lastmile gateway --url
```
Copy the printed HTTPS/HTTP URL into the mobile/web `.env` files.

## 5. Scaling & Resilience Demos
- Matching service auto-scales between 1 and 5 replicas via `autoscaling/v2` HPA (CPU target 60%). Generate load or scale manually:
  ```bash
  kubectl scale deploy/matching -n lastmile --replicas=5
  ```
- Demonstrate failure tolerance by deleting a pod (e.g., `kubectl delete pod -n lastmile <matching-pod>`). Kubernetes will recreate it; the gateway keeps serving cached state meanwhile.
- Location service has `MATCHING_ADDR` preset (`matching.lastmile.svc.cluster.local:50053`) so proximity updates trigger matching without extra wiring.

## 6. Cleanup
```bash
kubectl delete -f k8s/lastmile.yaml
# If you were using Minikube and want to reclaim disk:
minikube delete
```

These manifests stick to the MicroserviceAppSpec requirements: each service is isolated, the matching deployment can be scaled 1→5 replicas, and the gateway exposes both HTTP and gRPC entry points for the React Native client and future consumers.

# LastMile Kubernetes Demo Tasks

This document guides you through running the LastMile demo on Minikube, including deployment, autoscaling (HPA), residence testing, and client interactions.

## 1. Prerequisites
Ensure you have the following installed and running:
- **Minikube** (`minikube start`)
- **Docker**
- **Kubectl**
- **Node.js / pnpm** (for local client builds if needed)
- **CocoaPods** (for iOS)

## 2. Deployment
We have automated the deployment process. This script enables metrics-server, builds images, and deploys services.

```bash
./deploy_demo.sh
```
*Note: This script will also start the load generator by default. To stop it, see below.*

## 3. Kubernetes Dashboard
To view the cluster state visually:

```bash
minikube dashboard
```
- Switch Namespace: Select **`lastmile`** from the dropdown.
- Navigate to **Deployments** or **Pods** to see your workloads.

## 4. Autoscaling (HPA) Demo
The `matching` service is configured to autoscale based on CPU usage.

### Start Load
The `deploy_demo.sh` script deploys a load generator automatically. If you stopped it, restart it:
```bash
kubectl apply -f k8s/loadgen.yaml
```

### Observe Scaling
Open a new terminal to watch the Horizontal Pod Autoscaler:
```bash
kubectl get hpa -n lastmile -w
```
*Watch the `REPLICAS` count increase from 1 to 5 as `TARGET` CPU usage exceeds 60%.*

To see the pods being created:
```bash
kubectl get pods -n lastmile -w
```

### Stop Load
```bash
kubectl delete -f k8s/loadgen.yaml
```
*After stopping load, the HPA will eventually scale the pods back down (cooldown period is usually ~5 mins).*

## 5. Resilience Demo
Demonstrate that the system recovers from failures.

### Delete a Pod
Find a running pod (e.g., driver service) and delete it:
```bash
# Get pod name
kubectl get pods -n lastmile -l app=driver

# Delete it (replace with actual pod name)
kubectl delete pod driver-xxxxxxxxx-xxxxx -n lastmile
```

### Observe Recreation
Watch the pods immediately:
```bash
kubectl get pods -n lastmile -w
```
*You will see the old pod `Terminating` and a new pod `ContainerCreating` / `Running` almost instantly due to the Deployment controller.*

## 6. Web Client
The web client is deployed as a Service in Kubernetes.

### Access Web UI
```bash
minikube service web -n lastmile
```
This command opens the React application in your default browser.

## 7. iOS Client
The iOS client must be run locally but needs to talk to the Minikube cluster.

### Connect Mobile to Minikube
Since Minikube runs in a VM/Container, you need to expose the Gateway service to your host machine's localhost.

**Option 1: Minikube Tunnel (Recommended)**
```bash
minikube tunnel
```
*Keep this running. It assigns an external IP to LoadBalancer services.*

**Option 2: Port Forwarding**
Forward the Gateway service ports (HTTP and gRPC):
```bash
# Forward HTTP (8082) and gRPC (50060)
kubectl port-forward svc/gateway -n lastmile 8082:80 50060:50060
```

### Build & Run iOS App
Navigate to the mobile directory:
```bash
cd mobile
```

Install dependencies (if not done):
```bash
pnpm install
cd ios && pod install --repo-update && cd ..
```

Run the app:
```bash
pnpm ios
```

*Note: Ensure `.env.development` in `mobile/` points to your localhost gateway (e.g., `http://localhost:8082` if port forwarding).*

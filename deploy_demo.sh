#!/bin/bash
set -e

echo "🚀 Starting LastMile Kubernetes Demo Setup..."

# Check if minikube is running
if ! minikube status | grep -q "Running"; then
    echo "❌ Minikube is not running. Please start it with 'minikube start'"
    exit 1
fi

# Enable metrics-server for HPA
echo "📊 Enabling metrics-server..."
minikube addons enable metrics-server

# Point docker to minikube
echo "🐳 Configuring Docker environment..."
eval $(minikube docker-env)

# Services to build
SERVICES=("station" "driver" "rider" "location" "matching" "trip" "notification" "user" "gateway" "loadgen")

echo "🏗️  Building Docker images..."
for SERVICE in "${SERVICES[@]}"; do
    echo "   - Building ${SERVICE}..."
    docker build -t pradyunuydarp/lastmile-${SERVICE}:latest --build-arg SERVICE=${SERVICE} .
done

# Build Web
echo "   - Building web..."
docker build -t pradyunuydarp/lastmile-web:latest web/

# Apply manifests
echo "📄 Applying Kubernetes manifests..."
kubectl apply -f k8s/lastmile.yaml
kubectl apply -f k8s/web.yaml
kubectl rollout status deployment/web -n lastmile

# Wait for rollout of matching service

# Wait for rollout of matching service
echo "⏳ Waiting for Matching Service to be ready..."
kubectl rollout status deployment/matching -n lastmile

# Deploy load generator
echo "🔥 Deploying Load Generator..."
kubectl apply -f k8s/loadgen.yaml

echo "✅ Demo environment ready!"
echo "---------------------------------------------------"
echo "To observe autoscaling, run the following command in a separate terminal:"
echo "kubectl get hpa -n lastmile -w"
echo ""
echo "To observe pods scaling up:"
echo "kubectl get pods -n lastmile -w"
echo ""
echo "To stop load generation:"
echo "kubectl delete -f k8s/loadgen.yaml"
echo "To access the Web UI:"
echo "minikube service web -n lastmile"
echo "---------------------------------------------------"


#!/usr/bin/env bash
# Kept for muscle memory: the demo deploy now lives in the Makefile.
#   make k8s-up        build images inside minikube + deploy everything
#   make load-start    start load for the autoscaling demo
# See docs/demo/RUNBOOK.md for the full live-demo script.
set -euo pipefail
cd "$(dirname "$0")"
minikube status >/dev/null 2>&1 || { echo "minikube is not running: minikube start --cpus=4 --memory=8g"; exit 1; }
make k8s-up
echo
echo "Next: make load-start   then   kubectl -n lastmile get hpa -w"

#!/usr/bin/env bash
set -e

source "$(dirname "$0")/../helpers.sh"

kubectl create namespace redis || true

# Apply Redis manifests
kubectl apply -n redis -f "$(dirname "$0")/serviceaccount.yaml" --wait=true
kubectl apply -n redis -f "$(dirname "$0")/statefulset.yaml" --wait=true
kubectl apply -n redis -f "$(dirname "$0")/service.yaml" --wait=true

# Check Redis pod status
echo "Waiting for Redis StatefulSet to start..."
kubectl rollout status statefulset/redis -n redis --timeout=2m || true

# Display pod and PVC status
echo "Current pod status:"
kubectl get pods -n redis
echo "PVC status:"
kubectl get pvc -n redis

#!/usr/bin/env bash
set -e

source "$(dirname "$0")/../helpers.sh"

kubectl create namespace rabbitmq || true

kubectl create secret generic secrets \
  -n rabbitmq \
  "$(gen_secret rabbitmqpw)" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

# Apply RabbitMQ manifests
kubectl apply -n rabbitmq -f "$(dirname "$0")/serviceaccount.yaml" --wait=true
kubectl apply -n rabbitmq -f "$(dirname "$0")/statefulset.yaml" --wait=true
kubectl apply -n rabbitmq -f "$(dirname "$0")/service.yaml" --wait=true

# Check RabbitMQ pod status
echo "Waiting for RabbitMQ StatefulSet to start..."
kubectl rollout status statefulset/rabbitmq -n rabbitmq --timeout=2m || true

# Display pod and PVC status
echo "Current pod status:"
kubectl get pods -n rabbitmq
echo "PVC status:"
kubectl get pvc -n rabbitmq

echo ""
echo "RabbitMQ UI will be available at: http://your-cluster-ip:32347"
echo "Default credentials: admin / admin"

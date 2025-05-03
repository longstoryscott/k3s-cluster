#!/bin/bash

set -e

source "${1}/../helpers.sh"

# Create namespace
kubectl create namespace mysql || true

# Create a secret for MySQL credentials
# Create a secret for MySQL credentials if .secrets/mysqlpw doesn't exist
if [ ! -f ".secrets/mysqlpw" ]; then
  echo "Creating MySQL credentials secret..."
  kubectl create secret generic secrets \
    -n mysql \
    "$(gen_secret mysqlpw)" \
    --dry-run=client -o yaml | kubectl apply -f - --wait=true
else
  echo "MySQL credentials secret already exists, skipping creation."
fi

# Apply Kubernetes resources
echo "Applying MySQL resources..."
kubectl apply -f "${1}/serviceaccount.yaml" --wait=true
kubectl apply -f "${1}/service.yaml" --wait=true
kubectl apply -f "${1}/dbs.yaml" --wait=true
kubectl apply -f "${1}/statefulset.yaml" --wait=true
kubectl rollout restart statefulset/mysql -n mysql || true
kubectl rollout status statefulset/mysql -n mysql --timeout=5m || true

echo "Waiting for MySQL deployment to be ready..."
kubectl wait --for=condition=ready pod -n mysql -l app=mysql --timeout=180s || true

echo "MySQL deployment completed."

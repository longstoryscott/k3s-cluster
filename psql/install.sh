#!/usr/bin/env bash
set -e

source "${1}/../helpers.sh"

kubectl create namespace psql || true

kubectl create secret generic secrets \
    -n psql \
    "$(gen_secret psqlpw)" \
    --dry-run=client -o yaml | kubectl apply -f - --wait=true

# First cleanup any existing resources to ensure a fresh start
echo "Cleaning up any existing PostgreSQL resources..."
kubectl delete statefulset/postgres -n psql --ignore-not-found=true
kubectl delete pod postgres-0 -n psql --ignore-not-found=true --grace-period=0 --force

# kubectl apply -n psql -f "${1}/pvc.yaml" --wait=true
kubectl apply -n psql -f "${1}/serviceaccount.yaml" --wait=true
kubectl apply -n psql -f "${1}/dbs.yaml" --wait=true
kubectl apply -n psql -f "${1}/statefulset.yaml" --wait=true
kubectl apply -n psql -f "${1}/service.yaml" --wait=true

echo "Waiting for PostgreSQL StatefulSet to start..."
kubectl rollout status statefulset/postgres -n psql --timeout=2m || true

# Check pod status and add troubleshooting info
echo "Current pod status:"
kubectl get pods -n psql
echo "Pod description:"
kubectl describe pod postgres-0 -n psql | grep -A 10 "Events:"
echo "PVC status:"
kubectl get pvc -n psql

# Try to force the pod to restart if it's stuck
echo "Attempting to fix pod if stuck..."
if kubectl get pod postgres-0 -n psql -o jsonpath='{.status.phase}' | grep -q "Pending"; then
    echo "Pod is still pending. Attempting to recover..."
    kubectl delete pod postgres-0 -n psql --grace-period=0 --force || true
    echo "Waiting for pod to be recreated..."
    sleep 10
    kubectl get pods -n psql
fi

echo "PostgreSQL setup completed - checking database connectivity..."
# Wait for pod to be ready and try to connect to the database
timeout=120
counter=0
ready=false
while [ $counter -lt $timeout ] && [ "$ready" = false ]; do
    if kubectl exec -it postgres-0 -n psql -- pg_isready -U lsm 2>/dev/null; then
        ready=true
        echo "PostgreSQL is ready!"
    else
        counter=$((counter + 5))
        echo "Waiting for PostgreSQL to be ready... ($counter/$timeout seconds)"
        sleep 5
    fi
done

if [ "$ready" = false ]; then
    echo "PostgreSQL failed to become ready within timeout period. Check logs for more details:"
    kubectl logs postgres-0 -n psql || echo "Could not retrieve logs"
else
    echo "Checking databases:"
    kubectl exec -it postgres-0 -n psql -- psql -U lsm -c "\l" || echo "Could not list databases"
fi

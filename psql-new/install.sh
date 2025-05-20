#!/usr/bin/env bash
set -e

source "$(dirname "$0")/../helpers.sh" "$(dirname "$0")/../psql"

echo "Setting up new PostgreSQL instance..."

# Ensure the psql namespace exists
kubectl create namespace psql || true

# Create or update the secret
kubectl create secret generic secrets \
  -n psql \
  "$(gen_secret psqlpw)" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

# Apply the new PostgreSQL resources
echo "Deploying new PostgreSQL instance..."
# kubectl apply -n psql -f "$(dirname "$0")/pvc.yaml" --wait=true
kubectl apply -n psql -f "$(dirname "$0")/dbs.yaml" --wait=true
kubectl apply -n psql -f "$(dirname "$0")/statefulset.yaml" --wait=true

echo "Waiting for new PostgreSQL StatefulSet to start..."
kubectl rollout status statefulset/psql -n psql --timeout=2m || true

# Try to force the pod to restart if it's stuck
echo "Checking if pod is stuck in Pending state..."
if kubectl get pod psql-0 -n psql -o jsonpath='{.status.phase}' 2>/dev/null | grep -q "Pending"; then
  echo "Pod is still pending. Attempting to recover..."
  kubectl delete pod psql-0 -n psql --grace-period=0 --force || true
  echo "Waiting for pod to be recreated..."
  sleep 10
  kubectl get pods -n psql
fi

echo "New PostgreSQL setup completed - checking database connectivity..."
# Wait for pod to be ready and try to connect to the database
timeout=120
counter=0
ready=false
while [ $counter -lt $timeout ] && [ "$ready" = false ]; do
  if kubectl exec -it psql-0 -n psql -- pg_isready -U lsm 2>/dev/null; then
    ready=true
    echo "New PostgreSQL instance is ready!"
  else
    counter=$((counter + 5))
    echo "Waiting for new PostgreSQL to be ready... ($counter/$timeout seconds)"
    sleep 5
  fi
done

if [ "$ready" = false ]; then
  echo "New PostgreSQL failed to become ready within timeout period. Check logs for more details:"
  kubectl logs psql-0 -n psql || echo "Could not retrieve logs"
else
  echo "Checking databases on new instance:"
  kubectl exec -it psql-0 -n psql -- psql -U lsm -d postgres -c "\l" || echo "Could not list databases"
  echo ""
  echo "New PostgreSQL instance is ready for data migration."
  echo "Run the migrate-data.sh script next to copy your data from the old instance."
fi

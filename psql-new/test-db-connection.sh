#!/bin/bash
set -e

# Get the Nextcloud pod name
NC_POD=$(kubectl get pods -n nextcloud -l app=nextcloud -o jsonpath='{.items[0].metadata.name}')

# Extract database credentials from environment variables
DB_HOST=$(kubectl exec -it $NC_POD -n nextcloud -- printenv POSTGRES_HOST | tr -d '\r')
DB_USER=$(kubectl exec -it $NC_POD -n nextcloud -- printenv POSTGRES_USER | tr -d '\r')
DB_PASSWORD=$(kubectl exec -it $NC_POD -n nextcloud -- printenv POSTGRES_PASSWORD | tr -d '\r')
DB_NAME=$(kubectl exec -it $NC_POD -n nextcloud -- printenv POSTGRES_DB | tr -d '\r')

# Test database connection from the Nextcloud pod
echo "Testing database connection from Nextcloud pod..."
kubectl exec -it $NC_POD -n nextcloud -- bash -c "PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c 'SELECT 1;'" || echo "Database connection failed."

# Get PostgreSQL pod name
PSQL_POD=$(kubectl get pods -n psql -l app=psql -o jsonpath='{.items[0].metadata.name}')

# Gather PostgreSQL logs
echo "Gathering PostgreSQL logs..."
kubectl logs $PSQL_POD -n psql > postgres-logs.txt

echo "PostgreSQL logs saved to postgres-logs.txt. Please review the logs for any authentication errors or other issues."
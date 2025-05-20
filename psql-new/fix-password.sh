#!/bin/bash
set -e

echo "Fixing PostgreSQL user password to match what Nextcloud expects..."

# Extract the password that Nextcloud is using
NEXTCLOUD_DB_PASSWORD=$(kubectl get secret nextcloud-db -n nextcloud -o jsonpath='{.data.db-password}' | base64 --decode)

# Get the current PostgreSQL password
CURRENT_PSQL_PASSWORD=$(kubectl get secret secrets -n psql -o jsonpath='{.data.psqlpw}' | base64 --decode)

echo "Current situation:"
echo "- Nextcloud is using a specific password for database connection"
echo "- Your new PostgreSQL instance (psql-0) has a different password"

echo "Updating the PostgreSQL 'lsm' user password to match what Nextcloud expects..."
kubectl exec -it psql-0 -n psql -- bash -c "export PGPASSWORD='${CURRENT_PSQL_PASSWORD}' && psql -U postgres -d postgres -c \"ALTER USER lsm WITH PASSWORD '${NEXTCLOUD_DB_PASSWORD}';\""

echo "Password updated successfully!"
echo ""
echo "Testing the connection with the new password..."
kubectl exec -it psql-0 -n psql -- bash -c "export PGPASSWORD='${NEXTCLOUD_DB_PASSWORD}' && psql -U lsm -d postgres -c \"SELECT 'Connection successful!';\""

echo "Now restart your Nextcloud pod to try connecting with the updated password:"
echo "kubectl delete pod \$(kubectl get pod -n nextcloud -l app=nextcloud -o jsonpath='{.items[0].metadata.name}') -n nextcloud"

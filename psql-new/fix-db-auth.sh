#!/bin/bash
set -e

echo "========== PostgreSQL Database Authentication Fix Script =========="
echo "This script will extract the password from Nextcloud's config and update PostgreSQL"

# Get the Nextcloud pod name
NC_POD=$(kubectl get pods -n nextcloud -l app=nextcloud -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$NC_POD" ]; then
  echo "Error: Could not find Nextcloud pod. Is it running?"
  exit 1
fi

echo "Found Nextcloud pod: $NC_POD"

# Extract the password directly from Nextcloud's config.php (most reliable source)
echo "Extracting database password from Nextcloud config..."
NC_DB_PASSWORD=$(kubectl exec -it "$NC_POD" -n nextcloud -- bash -c 'grep -A1 dbpassword /var/www/html/config/config.php | tail -1 | sed -e "s/.*=> .//; s/.,//"' | sed -e "s/'//g" | tr -d '[:space:]')

if [ -z "$NC_DB_PASSWORD" ]; then
  echo "Failed to extract password from Nextcloud config. Using password from secret..."
  # Try to get password from Kubernetes secret
  NC_DB_PASSWORD=$(kubectl get secret nextcloud-db -n nextcloud -o jsonpath='{.data.db-password}' 2>/dev/null | base64 --decode || echo "")

  if [ -z "$NC_DB_PASSWORD" ]; then
    echo "Error: Could not determine Nextcloud database password"
    exit 1
  fi
fi

# For security, mask the password when displaying
MASKED_PASSWORD="${NC_DB_PASSWORD:0:3}****${NC_DB_PASSWORD: -3}"
echo "Retrieved Nextcloud database password (masked): $MASKED_PASSWORD"

# Get PostgreSQL admin credentials to connect to the database
echo "Getting PostgreSQL admin credentials..."
PSQL_ADMIN_PASSWORD=$(kubectl get secret secrets -n psql -o jsonpath='{.data.psqlpw}' | base64 --decode)

if [ -z "$PSQL_ADMIN_PASSWORD" ]; then
  # Try the password from the .secrets file
  if [ -f "/Users/scott.long/workspace/k3s-cluster/psql-new/.secrets/psqlpw" ]; then
    PSQL_ADMIN_PASSWORD=$(cat /Users/scott.long/workspace/k3s-cluster/psql-new/.secrets/psqlpw)
    echo "Using password from psql-new/.secrets/psqlpw"
  else
    echo "Error: Could not determine PostgreSQL admin password"
    exit 1
  fi
fi

# Update the PostgreSQL 'lsm' user password
echo "Updating PostgreSQL 'lsm' user password..."

# First try with postgres user (superuser)
echo "Attempting to update password using postgres superuser..."
if kubectl exec -it psql-0 -n psql -- bash -c "PGPASSWORD='$PSQL_ADMIN_PASSWORD' psql -U postgres -d postgres -c \"ALTER USER lsm WITH PASSWORD '$NC_DB_PASSWORD';\"" 2>/dev/null; then
  echo "Successfully updated password with postgres superuser"
else
  # Try with the lsm user itself
  echo "Attempting to update password using lsm user..."
  if kubectl exec -it psql-0 -n psql -- bash -c "PGPASSWORD='$PSQL_ADMIN_PASSWORD' psql -U lsm -d postgres -c \"ALTER USER lsm WITH PASSWORD '$NC_DB_PASSWORD';\"" 2>/dev/null; then
    echo "Successfully updated password with lsm user"
  else
    # Last resort - try as the default user
    echo "Attempting with default credentials..."
    kubectl exec -it psql-0 -n psql -- psql -c "ALTER USER lsm WITH PASSWORD '$NC_DB_PASSWORD';" 2>/dev/null || echo "Failed to update password"
  fi
fi

# Verify the connection with the new password
echo "Testing connection with the new password..."
if kubectl exec -it psql-0 -n psql -- bash -c "PGPASSWORD='$NC_DB_PASSWORD' psql -U lsm -d postgres -c \"SELECT 'Connection successful as lsm user!'\"" &>/dev/null; then
  echo "✅ Connection test successful! The lsm user can now authenticate with the password."
else
  echo "❌ Connection test failed. The password might not be correctly set."
fi

echo "Restarting Nextcloud pod to apply changes..."
kubectl delete pod "$NC_POD" -n nextcloud

echo "Done! Nextcloud will restart and should now connect successfully to PostgreSQL."
echo "Wait a few moments for the pod to restart, then check if Nextcloud is accessible."
echo "If issues persist, you may need to check the PostgreSQL logs with:"
echo "kubectl logs psql-0 -n psql"

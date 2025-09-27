#!/bin/bash
set -e

echo "========== PostgreSQL & Nextcloud Password Diagnostic =========="

# Get pod names
NC_POD=$(kubectl get pods -n nextcloud -l app=nextcloud -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
PSQL_POD="psql-0"

if [ -z "$NC_POD" ]; then
  echo "Error: Could not find Nextcloud pod."
  exit 1
fi

echo "Found pods:"
echo "- Nextcloud: $NC_POD"
echo "- PostgreSQL: $PSQL_POD"

# Extract password hash from Nextcloud config
echo -e "\n1. Checking Nextcloud database configuration..."
NC_CONFIG=$(kubectl exec -it "$NC_POD" -n nextcloud -- bash -c 'cat /var/www/html/config/config.php | grep -A5 dbhost' 2>/dev/null || echo "Failed to get config")
echo "$NC_CONFIG" | grep -v dbpassword

# Generate an MD5 hash of the password (for comparison only, not revealing the actual password)
echo -e "\n2. Generating password hash for comparison..."
NC_PW_HASH=$(kubectl exec -it "$NC_POD" -n nextcloud -- bash -c 'grep -A1 dbpassword /var/www/html/config/config.php | tail -1 | md5sum | cut -d " " -f1' 2>/dev/null || echo "Failed")
echo "Nextcloud password hash: $NC_PW_HASH"

# Check if PostgreSQL service is resolving correctly
echo -e "\n3. Checking DNS resolution from Nextcloud pod..."
NC_DNS_CHECK=$(kubectl exec -it "$NC_POD" -n nextcloud -- bash -c 'getent hosts postgres.psql.svc.cluster.local || echo "DNS resolution failed"')
echo "$NC_DNS_CHECK"

# Get PostgreSQL user info
echo -e "\n4. Checking PostgreSQL users..."
USERS=$(kubectl exec -it $PSQL_POD -n psql -- bash -c "psql -U postgres -tc \"SELECT usename FROM pg_catalog.pg_user;\"" 2>/dev/null || echo "Failed to list users")
echo "PostgreSQL users: $USERS"

# Try a direct network connection test
echo -e "\n5. Testing network connectivity from Nextcloud to PostgreSQL..."
NC_CONN_TEST=$(kubectl exec -it $NC_POD -n nextcloud -- bash -c "timeout 5 bash -c '</dev/tcp/postgres.psql.svc.cluster.local/5432' && echo 'Connection successful' || echo 'Connection failed'")
echo "$NC_CONN_TEST"

# Check which service is being used
echo -e "\n6. Checking current PostgreSQL service endpoints..."
echo "Services in psql namespace:"
kubectl get services -n psql -o wide

# Get database environment variables from the Nextcloud pod
echo -e "\n7. Checking Nextcloud environment variables..."
NC_ENV=$(kubectl exec -it $NC_POD -n nextcloud -- bash -c "env | grep -i postgres")
echo "$NC_ENV"

echo -e "\n8. Creating test user in PostgreSQL to verify password authentication..."
TEST_PW=$(date +%s | sha256sum | base64 | head -c 16)
TEST_USER="nc_test_$(date +%s | sha256sum | base64 | head -c 5)"

# Create a test user with a known password
kubectl exec -it $PSQL_POD -n psql -- bash -c "psql -U postgres -c \"CREATE USER $TEST_USER WITH PASSWORD '$TEST_PW';\""
echo "Created test user: $TEST_USER"

# Try to connect from Nextcloud pod to PostgreSQL using the test user
echo -e "\n9. Testing connection from Nextcloud to PostgreSQL with test user..."
NC_TEST_CONN=$(kubectl exec -it $NC_POD -n nextcloud -- bash -c "PGPASSWORD='$TEST_PW' psql -h postgres.psql.svc.cluster.local -U $TEST_USER -c 'SELECT 1;' || echo 'Connection failed'")
echo "$NC_TEST_CONN"

# Check if any other PostgreSQL pods exist that might be serving the connection
echo -e "\n10. Checking for all PostgreSQL pods in the cluster..."
kubectl get pods --all-namespaces | grep -i -E 'postgres|psql'

# Clean up test user
echo -e "\n11. Cleaning up test user..."
kubectl exec -it $PSQL_POD -n psql -- bash -c "psql -U postgres -c \"DROP USER $TEST_USER;\""

echo -e "\nDiagnostic completed. Please examine the results to identify the authentication issue."

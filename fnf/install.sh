#!/bin/bash

set -e

source "${1}/../helpers.sh"

# Check if dedicated MySQL service exists
if ! kubectl get service -n mysql mysql &>/dev/null; then
  echo "ERROR: The dedicated MySQL service was not found in the mysql namespace."
  echo "Please deploy MySQL first using: make mysql"
  exit 1
fi

# Create namespace and required resources
kubectl create namespace fnf || true

# Create a secret for WordPress database credentials
# Retrieve the MySQL password from MySQL namespace
echo "Creating MySQL credentials secret..."
MYSQL_PW=$(kubectl get secret -n mysql secrets -o jsonpath="{.data.mysqlpw}")
MYSQL_USER=$(echo -n "lsm" | base64 -i -)

# Check if we got the MySQL password
if [ -z "$MYSQL_PW" ]; then
  echo "ERROR: Could not retrieve MySQL password from mysql namespace"
  exit 1
fi

# Create the secret with the same data format
cat >"${1}/.secrets/secrets.yaml" <<EOF_SECRET
apiVersion: v1
kind: Secret
metadata:
  name: fnf-db
  namespace: fnf
type: Opaque
data:
  db-username: ${MYSQL_USER}
  db-password: ${MYSQL_PW}
EOF_SECRET

kubectl apply -f "${1}/.secrets/secrets.yaml" --wait=true

# Create the ConfigMap with our configuration files
kubectl create configmap fnf-config -n fnf --from-file=wp-config.php="${1}/wp-config.php" || true

# Apply Kubernetes resources
echo "Applying WordPress resources..."
kubectl apply -f "${1}/serviceaccount.yaml" --wait=true
kubectl apply -f "${1}/pvc.yaml" --wait=true || true
kubectl apply -f "${1}/deployment.yaml" --wait=true
kubectl apply -f "${1}/service.yaml" --wait=true
kubectl apply -f "${1}/referencegrant.yaml"

echo "Waiting for WordPress deployment to be available..."
kubectl rollout status deployment fnf -n fnf --timeout=5m || true

echo "WordPress deployment process completed."
echo "You can access it at http://192.168.0.71:8082/"
echo "To complete the installation, visit http://192.168.0.71:8082/wp-admin/install.php"

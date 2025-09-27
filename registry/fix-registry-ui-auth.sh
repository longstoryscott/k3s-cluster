#!/usr/bin/env bash
set -e

echo "Fixing Registry UI authentication..."

# Get the credentials from your existing setup
SECRETS_DIR="$(dirname "$0")/.secrets"
USER_SECRET_FILE="${SECRETS_DIR}/registryuser"
PW_SECRET_FILE="${SECRETS_DIR}/registrypw"

if [[ -f "$USER_SECRET_FILE" && -f "$PW_SECRET_FILE" ]]; then
    REGISTRY_USER=$(cat "$USER_SECRET_FILE")
    REGISTRY_PW=$(cat "$PW_SECRET_FILE")
    echo "Found existing credentials: ${REGISTRY_USER}"
else
    # Use the credentials from k3s-config.yaml as fallback
    REGISTRY_USER="dc4c537fa4a4acf4b2906b94"
    REGISTRY_PW="613a048331b0928c6ae3b0aa"
    echo "Using k3s-config credentials: ${REGISTRY_USER}"
fi

# Create base64 encoded credentials for Basic auth
AUTH_STRING=$(echo -n "${REGISTRY_USER}:${REGISTRY_PW}" | base64)
echo "Generated auth string: Basic ${AUTH_STRING}"

# Add the authentication environment variable to the existing deployment
kubectl patch deployment registry-ui -n registry --type='json' -p='[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {
      "name": "NGINX_PROXY_HEADER_Authorization",
      "value": "Basic '${AUTH_STRING}'"
    }
  }
]'

echo "Deployment patched. Waiting for rollout..."
kubectl rollout status deployment/registry-ui -n registry --timeout=60s

echo "Registry UI should now be able to connect to the registry!"
echo "Try accessing the UI at: http://$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}'):31501"
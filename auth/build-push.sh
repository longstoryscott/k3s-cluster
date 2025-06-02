#!/bin/bash
set -e

# Use registry.local instead of NODE_IP:PORT
REGISTRY_HOME="$(dirname "$0")/../registry"
REGISTRY_URL="192.168.0.71:31500"

# Registry credentials
USER_SECRET_FILE="${REGISTRY_HOME}/.secrets/registryuser"
PW_SECRET_FILE="${REGISTRY_HOME}/.secrets/registrypw"

if [[ -f "$USER_SECRET_FILE" && -f "$PW_SECRET_FILE" ]]; then
  REGISTRY_USER=$(cat "$USER_SECRET_FILE")
  REGISTRY_PW=$(cat "$PW_SECRET_FILE")
else
  echo "Registry credentials not found. Please run registry-mgmt.sh install first."
  exit 1
fi

# Login to registry
echo "Logging into private registry at ${REGISTRY_URL}..."
echo "${REGISTRY_PW}" | docker login --username "${REGISTRY_USER}" --password-stdin "${REGISTRY_URL}"

# Build the image
echo "Building usrmgr image..."
docker build -t "${REGISTRY_URL}/usrmgr:latest" "$(dirname "$0")"

# Push to registry
echo "Pushing image to registry..."
docker push "${REGISTRY_URL}/usrmgr:latest"

echo "✅ Image built and pushed to ${REGISTRY_URL}/usrmgr:latest"

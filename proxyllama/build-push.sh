#!/bin/bash
set -e

# Use registry.local instead of NODE_IP:PORT
REGISTRY_HOME="${HOME}/.registry"
REGISTRY_URL="registry.local"

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
echo "${REGISTRY_PW}" | docker login --username "${REGISTRY_USER}" --password-stdin "https://${REGISTRY_URL}"

# Build the image
echo "Building proxyllama image..."
docker build -t "${REGISTRY_URL}/proxyllama:latest" .

# Push to registry
echo "Pushing image to registry..."
docker push "${REGISTRY_URL}/proxyllama:latest"

echo "✅ Image built and pushed to ${REGISTRY_URL}/proxyllama:latest"

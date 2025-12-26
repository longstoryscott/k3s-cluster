#!/bin/bash
set -e

# Configuration
HOST_NODE="lsnode-3"
HOST_USER="${HOST_USER:-root}"
REMOTE_BUILD_DIR="/opt/builds/steam-desktop"
IMAGE_NAME="steam-desktop"
REGISTRY="${REGISTRY:-192.168.0.71:31500}"
TAG="${TAG:-latest}"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"

echo "=== Steam Desktop Deployment Script ==="
echo "Building on: ${HOST_USER}@${HOST_NODE}"
echo "Image: ${FULL_IMAGE}"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ ! -f "${PROJECT_ROOT}/registry/.secrets/registryuser" ]] || [[ ! -f "${PROJECT_ROOT}/registry/.secrets/registrypw" ]]; then
  echo "ERROR: Registry credentials not found"
  exit 1
fi

echo "Step 1: Preparing build directory on ${HOST_NODE}..."
ssh "${HOST_USER}@${HOST_NODE}" "mkdir -p ${REMOTE_BUILD_DIR}"

echo "Step 2: Copying files..."
scp "${SCRIPT_DIR}/Dockerfile" "${HOST_USER}@${HOST_NODE}:${REMOTE_BUILD_DIR}/"
scp "${PROJECT_ROOT}/registry/.secrets/registryuser" "${HOST_USER}@${HOST_NODE}:${REMOTE_BUILD_DIR}/"
scp "${PROJECT_ROOT}/registry/.secrets/registrypw" "${HOST_USER}@${HOST_NODE}:${REMOTE_BUILD_DIR}/"

echo "Step 3: Building..."
ssh "${HOST_USER}@${HOST_NODE}" "cd ${REMOTE_BUILD_DIR} && sudo docker build -t ${FULL_IMAGE} ."

echo "Step 4: Login..."
ssh "${HOST_USER}@${HOST_NODE}" "cd ${REMOTE_BUILD_DIR} && cat registrypw | sudo docker login http://${REGISTRY} -u \$(cat registryuser) --password-stdin"

echo "Step 5: Pushing..."
ssh "${HOST_USER}@${HOST_NODE}" "sudo docker push ${FULL_IMAGE}"

echo "Step 6: Cleanup secrets..."
ssh "${HOST_USER}@${HOST_NODE}" "rm -f ${REMOTE_BUILD_DIR}/registryuser ${REMOTE_BUILD_DIR}/registrypw"

echo ""
echo "Step 7: Applying manifests..."
cd "${SCRIPT_DIR}"
kubectl apply -f selkies-service-official.yaml
kubectl apply -f selkies-desktop-official.yaml
kubectl rollout restart statefulset/selkies-desktop-official -n steam

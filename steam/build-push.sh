#!/bin/bash
set -e

# Configuration
IMAGE_NAME="steam-desktop"
REGISTRY="registry.local:31500"
TAG="${TAG:-latest}"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"

echo "=== Building Steam Desktop Image ==="
echo "Image: ${FULL_IMAGE}"
echo ""

# Build the image
echo "Building Docker image..."
docker build -t "${FULL_IMAGE}" .

echo ""
echo "=== Build complete! ==="
echo ""
echo "To push to registry:"
echo "  docker push ${FULL_IMAGE}"
echo ""
echo "Or run with TAG environment variable:"
echo "  TAG=v1.0.0 ./build-push.sh"
echo ""

# Ask if user wants to push
read -p "Push to registry now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Pushing image to registry..."
    docker push "${FULL_IMAGE}"
    echo ""
    echo "✓ Image pushed successfully!"
    echo ""
    echo "Update your StatefulSet to use:"
    echo "  image: ${FULL_IMAGE}"
fi

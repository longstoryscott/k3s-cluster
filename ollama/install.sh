#!/bin/bash

set -e

source "${1}/../helpers.sh"

# Create the namespace if it doesn't exist
kubectl create namespace ai || true

echo "Applying PVC..."
kubectl apply -f "${1}/pvc.yaml" -n ai --wait=true

echo "Applying init script ConfigMap..."
kubectl apply -f "${1}/init-script.yaml" -n ai --wait=true

echo "Applying Ollama deployment..."
kubectl apply -f "${1}/deployment.yaml" -n ai --wait=true

echo "Applying Ollama service..."
kubectl apply -f "${1}/service.yaml" -n ai --wait=true

kubectl apply -f "${1}/referencegrant.yaml"

echo "Ollama deployment complete! Service is available at ollama.ai.svc.cluster.local:11434"
echo "Wait a few minutes for the models to be loaded and configured."

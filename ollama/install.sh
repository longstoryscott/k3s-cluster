#!/bin/bash

set -e

source "${1}/../helpers.sh"

# Create the namespace if it doesn't exist
kubectl create namespace ollama || true

kubectl apply \
  --server-side=true \
  -f https://raw.githubusercontent.com/nekomeowww/ollama-operator/v0.10.1/dist/install.yaml

kubectl wait \
  -n ollama-operator-system \
  --for=jsonpath='{.status.readyReplicas}'=1 deployment/ollama-operator-controller-manager

# kubectl apply -f "${1}/pvc.yaml" -n ollama --wait=true
kubectl apply -f "${1}/models" -n ollama --wait=true

# Wait for the model pod to be ready
# echo "Waiting for models to be ready..."
# kubectl wait --for=condition=ready pod -l app=ollama-model-llama3-8b -n ollama --timeout=300s
# kubectl wait --for=condition=ready pod -l app=ollama-model-phi3.5 -n ollama --timeout=300s

# # Get the pod name
# LLAMA38B=$(kubectl get pods -n ollama -l app=ollama-model-llama3-8b -o jsonpath="{.items[0].metadata.name}")
# # PHI35=$(kubectl get pods -n ollama -l app=ollama-model-phi3.5 -o jsonpath="{.items[0].metadata.name}")

# kubectl apply -f "${1}/configmap.yaml" -n ollama --wait=true

# # Copy the modelfile into the pod
# kubectl cp -n ollama llama3-8b-modelfile/Modelfile $LLAMA38B:/tmp/Modelfile

# # Apply the modelfile
# kubectl exec -n ollama $LLAMA38B -- ollama create llama3-8b -f /tmp/Modelfile

# echo "Quantized model llama3-8b created successfully!"

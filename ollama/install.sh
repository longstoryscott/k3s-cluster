#!/bin/bash

set -e

source "${1}/../helpers.sh"

kubectl delete namespace ollama --ignore-not-found=true

# Create the namespace if it doesn't exist
kubectl create namespace ollama || true

echo "Applying PVC..."
kubectl apply -f "${1}/pvc.yaml" -n ollama --wait=true

echo "Applying init script ConfigMap..."
kubectl apply -f "${1}/init-script.yaml" -n ollama --wait=true

echo "Applying Ollama deployment..."
kubectl apply -f "${1}/deployment.yaml" -n ollama --wait=true

echo "Applying Ollama service..."
kubectl apply -f "${1}/service.yaml" -n ollama --wait=true

# Wait for the Ollama pod to be running
echo "Waiting for Ollama pod to be running..."
sleep 5
OLLAMA_POD=$(kubectl get pods -n ollama -l name=ollama -o jsonpath="{.items[0].metadata.name}")
kubectl wait --for=condition=ready pod/$OLLAMA_POD -n ollama --timeout=120s

echo "Copying model files to the PVC..."
# Create the modelfiles directory in the pod if it doesn't exist
kubectl exec -n ollama $OLLAMA_POD -- mkdir -p /root/.ollama/modelfiles

# Copy each modelfile to the pod
for file in "${1}"/models/*.modelfile; do
  if [ -f "$file" ]; then
    MODEL_NAME=$(basename $file)
    echo "Copying $MODEL_NAME to the pod..."
    kubectl cp -n ollama "$file" $OLLAMA_POD:/root/.ollama/modelfiles/$MODEL_NAME
  fi
done

echo "Restarting Ollama pod to load new models..."
kubectl delete pod $OLLAMA_POD -n ollama

kubectl apply -f "${1}/referencegrant.yaml"

echo "Ollama deployment complete! Service is available at ollama.ollama.svc.cluster.local:11434"
echo "Wait a few minutes for the models to be loaded and configured."

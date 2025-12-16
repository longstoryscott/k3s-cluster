#!/bin/bash

echo "Installing Official Selkies Desktop..."

# Create namespace
kubectl create namespace steam || true

# Deploy StatefulSet and Service
kubectl apply -f selkies-desktop-official.yaml --wait=true
kubectl apply -f selkies-service-official.yaml --wait=true

# Wait for pod to be ready
echo "Waiting for Selkies desktop to be ready..."
kubectl wait --for=condition=ready pod -l app=selkies-desktop-official -n steam --timeout=300s

# Show status
echo "Deployment status:"
kubectl get pods -n steam -l app=selkies-desktop-official
kubectl get svc -n steam -l app=selkies-desktop-official

echo "Selkies Desktop should be available at: http://192.168.0.71:8087"
echo "Container logs can be viewed with: kubectl logs -f -n steam statefulset/selkies-desktop-official"
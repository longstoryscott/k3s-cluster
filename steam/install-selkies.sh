#!/bin/bash

set -e

echo "🎮 Installing Selkies Gaming Desktop..."

# Create namespace
kubectl create namespace steam || true

# Apply all manifests with wait for sequential deployment
echo "📦 Applying Selkies desktop configuration..."
kubectl apply -f selkies-desktop-statefulset.yaml --wait=true
kubectl apply -f selkies-service.yaml --wait=true
kubectl apply -f selkies-referencegrant.yaml --wait=true

# Wait for deployment to be ready
echo "⏳ Waiting for Selkies desktop to be ready..."
kubectl rollout status statefulset/selkies-desktop -n steam --timeout=600s

# Get connection info
echo "✅ Selkies desktop deployed successfully!"
echo ""
echo "🎯 Connection Info:"
echo "   Web Interface: Access through NGINX Gateway on port 8087"
echo "   Username: user"
echo "   Password: mypasswd"
echo ""
echo "🎮 Gaming Features:"
echo ""
echo "📚 Usage:"
echo "   1. Access the web interface through your gateway"
echo "   2. Login with user/mypasswd"
echo "   3. Run '/home/user/install-steam.sh' to set up Steam"
echo "   4. Launch games from Steam with full GPU acceleration"
echo ""
echo "🔧 To check status:"
echo "   kubectl get pods -n steam"
echo "   kubectl logs -f statefulset/selkies-desktop -n steam"
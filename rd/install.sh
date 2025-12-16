#!/bin/bash

# Steam Desktop Install Script
set -e

echo "Installing Linux Desktop for Steam..."

# Create namespace if it doesn't exist
kubectl create namespace steam --dry-run=client -o yaml | kubectl apply -f -

# Check if StatefulSet exists and delete it if needed (for volume template updates)
if kubectl get statefulset linux-desktop -n steam &>/dev/null; then
    echo "Existing StatefulSet found. Deleting to update volume configuration..."
    kubectl delete statefulset linux-desktop -n steam --wait=true
    echo "Waiting for pods to terminate..."
    sleep 5
fi

# Apply all desktop components
echo "Deploying desktop StatefulSet..."
kubectl apply -f /Users/lons7862/workspace/k3s-cluster/steam/desktop-statefulset.yaml

echo "Deploying desktop service..."
kubectl apply -f /Users/lons7862/workspace/k3s-cluster/steam/desktop-service.yaml

echo "Creating reference grant..."
kubectl apply -f /Users/lons7862/workspace/k3s-cluster/steam/desktop-referencegrant.yaml

echo "Updating gateway routes..."
kubectl apply -f /Users/lons7862/workspace/k3s-cluster/router/routes.yaml

echo "Waiting for desktop to be ready..."
kubectl wait --for=condition=Ready pod/linux-desktop-0 -n steam --timeout=300s

echo ""
echo "✅ Desktop deployed successfully!"
echo ""
echo "Access your Linux desktop at: http://192.168.0.71:8087/"
echo "VNC Password: mypassword"
echo ""
echo "🔧 Package Installation Helper:"
echo "Copy steam/install-packages.sh to the desktop for easy software installation"
echo ""
echo "📦 To install software:"
echo "1. Upload install-packages.sh to /home/ubuntu in the desktop"
echo "2. Run: chmod +x install-packages.sh && ./install-packages.sh"
echo "3. Or use individual commands like: ./install-packages.sh steam"
echo ""
echo "💾 Persistent Storage Locations:"
echo "• /home/ubuntu (100GB) - Your files and settings"  
echo "• /opt (100GB) - Applications like Steam"
echo "• /usr/local (50GB) - User-installed software"
echo "• Package database and repositories are also persistent"
echo ""
echo "🎮 GPU acceleration is enabled with NVIDIA drivers."
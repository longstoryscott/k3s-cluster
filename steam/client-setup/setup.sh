#!/bin/bash
# Automated setup script for SteamLink client on Raspberry Pi
# Run this script as the user (not root)

set -e

echo "=== SteamLink Client Setup ==="
echo "This script will configure a Raspberry Pi to auto-start SteamLink on boot"
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then 
    echo "ERROR: Please run as regular user, not root"
    exit 1
fi

USER_NAME=$(whoami)
echo "Setting up for user: $USER_NAME"
echo ""

# 1. Install kubectl
echo "Step 1: Installing kubectl..."
if ! command -v kubectl &> /dev/null; then
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
    echo "kubectl installed"
else
    echo "kubectl already installed"
fi

# 2. Copy kubectl config
echo ""
echo "Step 2: Setting up kubectl config..."
echo "Please provide the hostname of a k3s node (e.g., lsnode-3):"
read -r K3S_NODE

mkdir -p ~/.kube
echo "Copying kubeconfig from $K3S_NODE..."
ssh "$USER_NAME@$K3S_NODE" "cat ~/.kube/config" > ~/.kube/config
chmod 600 ~/.kube/config
echo "kubectl configured"

# Test connection
echo "Testing cluster connection..."
kubectl get nodes

# 3. Install scripts
echo ""
echo "Step 3: Installing auto-start scripts..."
cp start-steam-in-pod.sh ~/
chmod +x ~/start-steam-in-pod.sh

sudo cp bt-autoconnect.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/bt-autoconnect.sh

# 4. Configure auto-login
echo ""
echo "Step 4: Setting up auto-login..."
sudo mkdir -p /etc/systemd/system/getty@tty1.service.d
sudo cp autologin.conf /etc/systemd/system/getty@tty1.service.d/
sudo systemctl daemon-reload

# 5. Configure Bluetooth auto-connect
echo ""
echo "Step 5: Setting up Bluetooth auto-connect..."
sudo cp bt-autoconnect.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable bt-autoconnect.service

# 6. Add bashrc modifications
echo ""
echo "Step 6: Adding SteamLink auto-start to .bashrc..."
if ! grep -q "Auto-start SteamLink on tty1" ~/.bashrc 2>/dev/null; then
    cat bashrc-additions.sh >> ~/.bashrc
    echo "bashrc updated"
else
    echo "bashrc already configured"
fi

# 7. Configure Bluetooth
echo ""
echo "Step 7: Configuring Bluetooth..."
sudo rfkill unblock bluetooth
sudo systemctl restart bluetooth

# Add bluetooth config to boot
if ! grep -q "dtparam=bluetooth=on" /boot/firmware/config.txt 2>/dev/null; then
    echo "dtparam=bluetooth=on" | sudo tee -a /boot/firmware/config.txt
fi

echo ""
echo "=== Setup Complete! ==="
echo ""
echo "Next steps:"
echo "1. Pair your Bluetooth controller:"
echo "   bluetoothctl"
echo "   > agent on"
echo "   > pairable on"
echo "   > scan on"
echo "   > pair <MAC_ADDRESS>"
echo "   > trust <MAC_ADDRESS>"
echo "   > connect <MAC_ADDRESS>"
echo ""
echo "2. Update the controller MAC address in /usr/local/bin/bt-autoconnect.sh"
echo ""
echo "3. Reboot to test:"
echo "   sudo reboot"
echo ""
echo "After reboot, SteamLink should auto-start with Steam in the k8s pod!"

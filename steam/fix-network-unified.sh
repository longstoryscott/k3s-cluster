#!/bin/bash
# Unified Network Management Fix for lsnode-3
# Resolves: NetworkManager vs netplan conflict, missing WiFi/Ethernet drivers

set -e

echo "=== Unified Network Management Fix ==="
echo ""

# Step 1: Install missing network drivers
echo "[1/5] Installing network drivers..."
sudo apt update
sudo apt install -y \
    linux-modules-extra-$(uname -r) \
    firmware-linux \
    firmware-linux-nonfree \
    iwlwifi-dkms \
    r8169-dkms 2>/dev/null || true

# Step 2: Configure NetworkManager to manage all devices
echo "[2/5] Configuring NetworkManager..."
sudo tee /etc/NetworkManager/NetworkManager.conf <<'EOF'
[main]
plugins=ifupdown,keyfile

[ifupdown]
managed=true

[keyfile]
unmanaged-devices=interface-name:docker0;interface-name:cni0;interface-name:flannel*;interface-name:veth*

[device]
wifi.scan-rand-mac-address=no
wifi.mac-address-randomization=none
EOF

# Step 3: Remove conflicting netplan configs
echo "[3/5] Cleaning up netplan configs..."
sudo rm -f /etc/netplan/90-NM-*.yaml

# Step 4: Create unified netplan config that delegates to NetworkManager
echo "[4/5] Creating unified netplan config..."
sudo tee /etc/netplan/01-netcfg.yaml <<'EOF'
network:
  version: 2
  renderer: NetworkManager
  ethernets:
    # Managed by NetworkManager
    enp7s0:
      match:
        macaddress: 3C:7C:3F:1E:CB:15
      dhcp4: true
      dhcp6: true
    enp6s0:
      match:
        driver: r8169
      dhcp4: true
      dhcp6: true
  wifis:
    wlp5s0:
      match:
        driver: iwlwifi
      dhcp4: true
      dhcp6: true
  # Exclude container networking from NetworkManager
  links:
    docker0:
      required-for: default
    cni0:
      required-for: default
    flannel.1:
      required-for: default
EOF

sudo netplan apply

# Step 5: Reload modules and restart NetworkManager
echo "[5/5] Loading modules and restarting NetworkManager..."

# Load WiFi modules
sudo modprobe -r iwlwifi 2>/dev/null || true
sudo modprobe -r iwldvm 2>/dev/null || true
sudo modprobe -r mac80211 2>/dev/null || true
sudo modprobe iwlwifi
sudo modprobe iwldvm 2>/dev/null || true

# Load Realtek module
sudo modprobe r8169 2>/dev/null || echo "r8169 module not available (may need DKMS build)"

# Restart NetworkManager
sudo systemctl restart NetworkManager

echo ""
echo "=== Verification ==="
echo ""
echo "Network interfaces:"
ip link show | grep -E '^[0-9]+:' | head -10

echo ""
echo "NetworkManager status:"
nmcli device status

echo ""
echo "WiFi devices:"
nmcli radio wifi && nmcli device wifi list 2>/dev/null | head -5 || echo "No WiFi detected"

echo ""
echo "=== Done ==="
echo ""
echo "To connect to WiFi via GUI or CLI:"
echo "  nmcli device wifi connect \"SSID\" password \"PASSWORD\""
echo ""
echo "Sunshine should now be accessible at:"
echo "  https://127.0.0.1:47990 (local)"
echo "  https://192.168.0.234:47990 (network)"

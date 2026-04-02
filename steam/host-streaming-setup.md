# Host Streaming Setup for lsnode-3

This document details the step-by-step commands to transform lsnode-3 into a streaming gaming server while maintaining K8s GPU access.

## Prerequisites

- SSH access to lsnode-3 with sudo privileges
- Physical/console access as fallback
- Existing K8s cluster with GPU Operator running

---

## Phase 1: Pre-Installation Backup

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Backup K8s config
cp ~/.kube/config ~/.kube/config.backup

# Document current GPU Operator settings
kubectl get clusterpolicy cluster-policy -n gpu-operator -o yaml > ~/gpu-operator-clusterpolicy.yaml

# Backup containerd config
sudo cp /var/lib/rancher/k3s/agent/etc/containerd/config.toml ~/containerd-config.toml.backup

# Verify current GPU state
nvidia-smi 2>/dev/null || echo "No native nvidia-smi - expected"
kubectl get nodes -l nvidia.com/gpu.present=true -o wide
```

---

## Phase 2: Desktop Installation

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Update package lists
sudo apt update

# Install minimal KDE Plasma desktop (no extra apps)
# This will prompt for display manager selection - choose LightDM
sudo apt install -y kubuntu-desktop-minimal

# If prompted during install, select LightDM as default display manager
# If already installed, configure manually:
sudo dpkg-reconfigure lightdm

# Configure LightDM to NOT auto-start on boot
sudo systemctl disable lightdm

# Verify LightDM is installed but not running
systemctl status lightdm  # Should show "disabled"
```

---

## Phase 3: NVIDIA Driver Installation

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Check available NVIDIA drivers
ubuntu-drivers devices

# Auto-install recommended proprietary drivers
sudo ubuntu-drivers autoinstall

# The command will install the recommended driver version
# Review the output to note which version was installed

# Reboot to load the new drivers
sudo reboot

# After reboot, verify drivers are loaded
nvidia-smi

# Check device files exist
ls -la /dev/nvidia*

# Verify all 3 GPUs are visible
nvidia-smi -L
```

---

## Phase 4: K8s GPU Operator Reconfiguration

```bash
# From your local machine (not lsnode-3)

# Get current ClusterPolicy
kubectl get clusterpolicy cluster-policy -n gpu-operator -o yaml > ~/gpu-operator-backup.yaml

# Edit the ClusterPolicy to disable driver daemonset
kubectl edit clusterpolicy cluster-policy -n gpu-operator

# In the editor, find and change:
#   driver:
#     enabled: true   ->   enabled: false

# Alternatively, apply this patched YAML:
cat <<EOF | kubectl apply -f -
apiVersion: nvidia.com/v1
kind: ClusterPolicy
metadata:
  name: cluster-policy
  namespace: gpu-operator
spec:
  driver:
    enabled: false
  # Keep other settings as-is (devicePlugin, containerToolkit, etc.)
EOF

# Wait for driver daemonset pods to terminate
kubectl get pods -n gpu-operator -l app=nvidia-driver-daemonset

# Verify device-plugin and container-toolkit are still running
kubectl get pods -n gpu-operator -l app=nvidia-device-plugin-daemonset
kubectl get pods -n gpu-operator -l app=nvidia-container-toolkit-daemonset

# Test GPU access from a pod
kubectl run gpu-test --rm -it --restart=Never \
  --image=nvcr.io/nvidia/k8s-device-plugin:v0.18.1 \
  --runtime-class=nvidia \
  --overrides='{"spec":{"containers":[{"name":"gpu-test","image":"nvcr.io/nvidia/k8s-device-plugin:v0.18.1","command":["/bin/sh","-c","nvidia-smi"]}]}}' \
  -- nvidia-smi
```

---

## Phase 5: Streaming Stack Installation

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Add Sunshine PPA and install
sudo add-apt-repository ppa:briando/sunshine
sudo apt update
sudo apt install -y sunshine

# Configure Sunshine to start manually (not as a service)
sudo systemctl disable sunshine
sudo systemctl stop sunshine

# Install Steam
sudo apt install -y steam-installer

# Install additional dependencies for Steam controller support
sudo apt install -y steam-devices xinput

# Install x11-xserver-utils for display management
sudo apt install -y x11-xserver-utils

# Install plasma-desktop for full Plasma features
sudo apt install -y plasma-desktop

# Install utilities for headless operation
sudo apt install -y xorg xserver-xorg-video-dummy
```

---

## Phase 6: On-Demand Streaming Service Configuration

### Create systemd target for streaming

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Create streaming.target
sudo tee /etc/systemd/system/streaming.target <<'EOF'
[Unit]
Description=Streaming Gaming Session
After=network.target display-manager.service
Requires=display-manager.service

[Install]
WantedBy=multi-user.target
EOF

# Create streaming-session.service - starts Plasma session with Sunshine
sudo tee /etc/systemd/system/streaming-session.service <<'EOF'
[Unit]
Description=Streaming Session (Plasma + Sunshine + Steam)
After=display-manager.service
Wants=display-manager.service

[Service]
Type=simple
User=%h
Environment=DISPLAY=:0
Environment=XAUTHORITY=/home/%h/.Xauthority
ExecStart=/bin/bash -c 'echo "Starting streaming session..." && \
    sunshine --config /etc/sunshine/sunshine.conf & \
    sleep 5 && \
    steam -silent -bigpicture &'
Restart=on-failure
RestartSec=10

[Install]
WantedBy=streaming.target
EOF

# Create start-streaming script
sudo tee /usr/local/bin/start-streaming <<'EOF'
#!/bin/bash
# Start streaming session on lsnode-3

USER=$(whoami)
DISPLAY=:0

echo "Starting streaming session..."

# Start LightDM if not running
if ! systemctl is-active --quiet lightdm; then
    echo "Starting LightDM..."
    sudo systemctl start lightdm
    sleep 3
fi

# Wait for display to be ready
until xset -display $DISPLAY q >/dev/null 2>&1; do
    echo "Waiting for display..."
    sleep 2
done

# Start Sunshine
sudo systemctl start sunshine 2>/dev/null || sunshine --daemon &

# Start Steam in Big Picture mode
su $USER -c "DISPLAY=$DISPLAY steam -silent -bigpicture &"

echo "Streaming session started. Connect via Moonlight."
EOF
sudo chmod +x /usr/local/bin/start-streaming

# Create stop-streaming script
sudo tee /usr/local/bin/stop-streaming <<'EOF'
#!/bin/bash
# Stop streaming session on lsnode-3

echo "Stopping streaming session..."

# Stop Steam
pkill -u $USER steam 2>/dev/null

# Stop Sunshine
sudo systemctl stop sunshine 2>/dev/null
pkill sunshine 2>/dev/null

# Optionally stop LightDM (comment out if you want to keep display running)
# sudo systemctl stop lightdm

echo "Streaming session stopped."
EOF
sudo chmod +x /usr/local/bin/stop-streaming

# Create enable-streaming-on-boot script
sudo tee /usr/local/bin/enable-streaming-on-boot <<'EOF'
#!/bin/bash
# Enable streaming to start automatically on boot

sudo systemctl enable streaming.target
echo "Streaming will start automatically on boot."
echo "Use 'disable-streaming-on-boot' to disable."
EOF
sudo chmod +x /usr/local/bin/enable-streaming-on-boot

# Create disable-streaming-on-boot script
sudo tee /usr/local/bin/disable-streaming-on-boot <<'EOF'
#!/bin/bash
# Disable streaming auto-start on boot

sudo systemctl disable streaming.target
echo "Streaming will NOT start automatically on boot."
EOF
sudo chmod +x /usr/local/bin/disable-streaming-on-boot

# Reload systemd daemon
sudo systemctl daemon-reload
```

### Configure Sunshine

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Create Sunshine config directory
mkdir -p ~/.config/sunshine

# Create basic Sunshine config
cat > ~/.config/sunshine/sunshine.conf <<'EOF'
# Sunshine configuration for headless streaming
log_level = info
web_enabled = true
port = 47989
https_port = 47990
game_port = 47991

# Video settings
width = 1920
height = 1080
fps = 60
bitrate = 20000

# Encoder settings
encoder = auto
hwaccel = true

# Input settings
input_source = auto

# Streaming settings
multicast = false
recovery_level = 150
EOF

# Generate self-signed certificate for Sunshine (first time only)
sunshine --create-cert

# Configure Sunshine to use the correct display
sudo systemctl edit sunshine --force <<'EOF'
[Service]
Environment=DISPLAY=:0
EOF
```

### Configure Steam for Big Picture auto-start

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Create Steam autostart config
mkdir -p ~/.config/autostart

# Create Steam Big Picture autostart entry
cat > ~/.config/autostart/steam-bigpicture.desktop <<'EOF'
[Desktop Entry]
Version=1.0
Type=Application
Name=Steam Big Picture
Exec=steam -silent -bigpicture
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Comment=Start Steam in Big Picture mode for gaming
EOF

# Enable Steam controller support
steam --ui bigpicture
```

---

## Verification Steps

### 1. Test Streaming Session Start

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Start streaming session
start-streaming

# Wait 30 seconds for everything to initialize
sleep 30

# Verify processes are running
ps aux | grep -E 'sunshine|steam|X'

# Check Sunshine is listening
sudo ss -tlnp | grep sunshine

# From your client device, try connecting via Moonlight
# (Moonlight should discover lsnode-3 on the network)
```

### 2. Test GPU Access on Host

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Verify all 3 GPUs are visible
nvidia-smi

# Check GPU utilization during gaming
watch -n 1 nvidia-smi
```

### 3. Test K8s GPU Access

```bash
# From your local machine
kubectl run gpu-test --rm -it --restart=Never \
  --image=nvcr.io/nvidia/k8s-device-plugin:v0.18.1 \
  --runtime-class=nvidia \
  --overrides='{"spec":{"containers":[{"name":"gpu-test","image":"nvcr.io/nvidia/k8s-device-plugin:v0.18.1","command":["/bin/sh","-c","nvidia-smi"]}]}}' \
  -- nvidia-smi

# Deploy existing llmmll workload and verify it runs
kubectl get pods -n llmmll
```

### 4. Test Controller Support

```bash
# While streaming via Moonlight
# Connect your controller
# Verify Steam detects it:
#   Steam -> Settings -> Controller -> General Controller Settings

# Check xinput sees the controller
xinput list
```

---

## Convenience Commands Summary

| Command | Description |
|---------|-------------|
| `start-streaming` | Start Plasma + Sunshine + Steam session |
| `stop-streaming` | Stop streaming session |
| `enable-streaming-on-boot` | Auto-start streaming on boot |
| `disable-streaming-on-boot` | Disable auto-start on boot |
| `nvidia-smi` | Check GPU status on host |
| `systemctl status lightdm` | Check if display manager is running |

---

## Troubleshooting

### Steam doesn't start in Big Picture mode

```bash
# Check if Steam is installed correctly
steam --version

# Try starting Steam manually
DISPLAY=:0 steam -silent -bigpicture

# Check Steam logs
~/.steam/debian-install-error.log
```

### Sunshine not discovered by Moonlight

```bash
# Check Sunshine is running
sudo systemctl status sunshine

# Check Sunshine is listening on correct ports
sudo ss -tlnp | grep -E '47989|47990|47991'

# Check firewall allows Sunshine ports
sudo ufw status | grep -E '47989|47990|47991'

# If needed, open ports
sudo ufw allow 47989/tcp
sudo ufw allow 47990/tcp
sudo ufw allow 47991/tcp
```

### GPU not accessible in containers after driver install

```bash
# Check NVIDIA device files exist
ls -la /dev/nvidia*

# Check NVIDIA kernel modules loaded
lsmod | grep nvidia

# Restart GPU Operator device plugin
kubectl rollout restart daemonset nvidia-device-plugin-daemonset -n gpu-operator

# Check CDI annotations
kubectl get node lsnode-3 -o json | jq '.metadata.labels' | grep nvidia
```

---

## Rollback Plan

If you need to revert to the original headless setup:

```bash
# SSH into lsnode-3
ssh lsnode-3.local

# Stop streaming services
stop-streaming
sudo systemctl disable lightdm

# Remove desktop packages (optional - keeps system minimal)
sudo apt remove -y kubuntu-desktop-minimal plasma-desktop

# Re-enable GPU Operator driver daemonset
kubectl edit clusterpolicy cluster-policy -n gpu-operator
# Change: enabled: false -> enabled: true

# Reboot
sudo reboot
```

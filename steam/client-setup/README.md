# SteamLink Client Setup for Raspberry Pi

Complete setup guide for configuring a Raspberry Pi 4 as a dedicated SteamLink client that automatically connects to your Steam gaming pod running in a k3s cluster.

## Overview

This setup configures a Raspberry Pi to:
- Auto-login on boot
- Auto-start SteamLink
- Auto-start Steam in the k8s gaming pod
- Auto-connect a Bluetooth game controller

## Prerequisites

- Raspberry Pi 4 with Raspberry Pi OS Lite installed
- SSH access configured with public key authentication
- SteamLink installed (`sudo apt install steamlink`)
- A k3s cluster with the Steam gaming pod running
- Bluetooth game controller (PS4/PS5 controller recommended)

## Quick Setup

### Automated Installation

1. Copy this directory to your Raspberry Pi:
   ```bash
   scp -r client-setup/ user@rpi:~/
   ```

2. SSH into your Pi and run the setup script:
   ```bash
   cd ~/client-setup
   chmod +x setup.sh
   ./setup.sh
   ```

3. Follow the prompts to configure kubectl and other settings.

4. Pair your Bluetooth controller:
   ```bash
   bluetoothctl
   agent on
   pairable on
   scan on
   # Wait for your controller to appear
   pair <MAC_ADDRESS>
   trust <MAC_ADDRESS>
   connect <MAC_ADDRESS>
   exit
   ```

5. Update the MAC address in `/usr/local/bin/bt-autoconnect.sh`:
   ```bash
   sudo nano /usr/local/bin/bt-autoconnect.sh
   # Change MAC="..." to your controller's MAC address
   ```

6. Reboot:
   ```bash
   sudo reboot
   ```

## Manual Setup

If you prefer manual configuration:

### 1. Install kubectl

```bash
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/
```

### 2. Configure kubectl

Copy your k3s cluster config:
```bash
mkdir -p ~/.kube
scp user@k3s-node:~/.kube/config ~/.kube/config
chmod 600 ~/.kube/config
```

Test connection:
```bash
kubectl get pods -n steam
```

### 3. Install Steam Auto-Start Script

```bash
cp start-steam-in-pod.sh ~/
chmod +x ~/start-steam-in-pod.sh
```

### 4. Configure Auto-Login

```bash
sudo mkdir -p /etc/systemd/system/getty@tty1.service.d
sudo cp autologin.conf /etc/systemd/system/getty@tty1.service.d/
sudo systemctl daemon-reload
```

### 5. Configure SteamLink Auto-Start

Add to the end of `~/.bashrc`:
```bash
cat bashrc-additions.sh >> ~/.bashrc
```

### 6. Setup Bluetooth

Install required packages:
```bash
sudo apt update
sudo apt install -y pi-bluetooth bluez
```

Enable Bluetooth in boot config:
```bash
echo "dtparam=bluetooth=on" | sudo tee -a /boot/firmware/config.txt
```

Unblock Bluetooth:
```bash
sudo rfkill unblock bluetooth
sudo systemctl restart bluetooth
```

### 7. Configure Bluetooth Auto-Connect

Install the auto-connect script:
```bash
sudo cp bt-autoconnect.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/bt-autoconnect.sh
```

Update the MAC address in `/usr/local/bin/bt-autoconnect.sh` with your controller's address.

Install the systemd service:
```bash
sudo cp bt-autoconnect.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable bt-autoconnect.service
```

### 8. Pair Bluetooth Controller

```bash
bluetoothctl
```

In bluetoothctl:
```
agent on
pairable on
scan on
# Hold PS + Share buttons on your controller until it flashes
pair <MAC_ADDRESS>
trust <MAC_ADDRESS>
connect <MAC_ADDRESS>
exit
```

The light bar should turn solid blue when properly connected.

## Boot Sequence

When the Raspberry Pi boots:

1. **Auto-login** - User logs in automatically on tty1
2. **Bluetooth** - Controller auto-connects (solid blue light)
3. **Steam Pod** - Script launches Steam in the k8s pod (runs in background for 15s)
4. **SteamLink** - Launches on screen and connects to the Steam pod

Total boot-to-ready time: ~30-40 seconds

## File Descriptions

### Scripts

- **`start-steam-in-pod.sh`** - Launches Steam in the k8s pod via supervisord
  - Location: `~/start-steam-in-pod.sh`
  - Called automatically from `.bashrc` on boot
  - Logs to `/tmp/steam-start.log`

- **`bt-autoconnect.sh`** - Auto-connects Bluetooth controller
  - Location: `/usr/local/bin/bt-autoconnect.sh`
  - Called by systemd service on boot
  - Edit to update controller MAC address

### Configuration Files

- **`autologin.conf`** - Getty auto-login configuration
  - Location: `/etc/systemd/system/getty@tty1.service.d/autologin.conf`
  - Change `lsm` to your username if different

- **`bt-autoconnect.service`** - Systemd service for Bluetooth auto-connect
  - Location: `/etc/systemd/system/bt-autoconnect.service`

- **`bashrc-additions.sh`** - Shell configuration for auto-starting SteamLink
  - Append to `~/.bashrc`

### Utility

- **`setup.sh`** - Automated setup script
  - Installs and configures everything automatically

## Troubleshooting

### SteamLink doesn't start on boot

Check logs:
```bash
cat /tmp/steamlink-starting.log
journalctl -u getty@tty1.service
```

Verify bashrc:
```bash
tail -15 ~/.bashrc
```

### Steam doesn't start in pod

Check logs:
```bash
cat /tmp/steam-start.log
kubectl logs -n steam <pod-name>
kubectl exec -n steam <pod-name> -- cat /tmp/steam-supervisor.log
```

Verify kubectl config:
```bash
kubectl get pods -n steam
```

### Bluetooth controller won't connect

Check Bluetooth status:
```bash
sudo systemctl status bluetooth
hciconfig
```

Unblock if needed:
```bash
sudo rfkill list
sudo rfkill unblock bluetooth
```

Re-pair controller:
```bash
bluetoothctl
remove <MAC_ADDRESS>
scan on
# Then pair again
```

### Boot hangs or loops

Access via SSH and check:
```bash
sudo systemctl status getty@tty1.service
ps aux | grep steamlink
```

Disable auto-start temporarily:
```bash
# Comment out the auto-start block in ~/.bashrc
nano ~/.bashrc
```

## Log Files

- `/tmp/steamlink-starting.log` - SteamLink startup
- `/tmp/steam-start.log` - Steam pod startup script
- `/tmp/steam-supervisor.log` - Steam process in pod
- `journalctl -u getty@tty1.service` - Auto-login service
- `journalctl -u bt-autoconnect.service` - Bluetooth auto-connect

## Customization

### Change auto-login user

Edit `/etc/systemd/system/getty@tty1.service.d/autologin.conf`:
```bash
sudo nano /etc/systemd/system/getty@tty1.service.d/autologin.conf
```
Change `lsm` to your username.

### Update controller MAC address

Edit `/usr/local/bin/bt-autoconnect.sh`:
```bash
sudo nano /usr/local/bin/bt-autoconnect.sh
```
Change the `MAC=` line.

### Change Steam pod namespace/label

Edit `~/start-steam-in-pod.sh`:
```bash
nano ~/start-steam-in-pod.sh
```
Modify the `kubectl get pod` command.

### Adjust boot delays

Edit `~/start-steam-in-pod.sh`:
```bash
nano ~/start-steam-in-pod.sh
```
Change the `sleep 15` value (seconds to wait before starting Steam).

## Uninstall

To remove the auto-start configuration:

```bash
# Remove bashrc additions
nano ~/.bashrc  # Delete the auto-start block

# Disable services
sudo systemctl disable bt-autoconnect.service
sudo rm /etc/systemd/system/bt-autoconnect.service

# Remove auto-login
sudo rm /etc/systemd/system/getty@tty1.service.d/autologin.conf
sudo systemctl daemon-reload

# Remove scripts
rm ~/start-steam-in-pod.sh
sudo rm /usr/local/bin/bt-autoconnect.sh
```

## Important: Container Image Requirement

**The Steam container image must have `steamdeps` replaced with a no-op script** to prevent interactive dependency check popups. This is included in the Dockerfile:

```dockerfile
# Replace steamdeps with a no-op script to skip interactive dependency checks
RUN mv /usr/bin/steamdeps /usr/bin/steamdeps.real && \
  echo '#!/bin/bash\nexit 0' > /usr/bin/steamdeps && \
  chmod +x /usr/bin/steamdeps
```

Without this fix, Steam will launch an interactive Konsole window asking to evaluate dependencies, which blocks auto-start.

## Notes

- The Pi should be on the same network as your k3s cluster
- Display resolution is controlled by the Steam pod configuration (1920x1080 by default)
- Audio routing is handled by the SteamLink protocol
- Controller input has ~10-20ms latency depending on network conditions
- For best performance, use a wired Ethernet connection

## Support

For issues specific to:
- **k3s cluster setup** - See main k3s-cluster documentation
- **Steam pod configuration** - See `steam/README.md`
- **SteamLink client** - Check SteamLink official documentation
- **Bluetooth pairing** - Consult Raspberry Pi Bluetooth guides

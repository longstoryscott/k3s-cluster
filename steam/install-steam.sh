#!/bin/bash

echo "=== Steam Installer for Selkies GLX Desktop ==="
echo "Installing Steam from official Valve package..."

# Remove container detection files
rm -f /run/systemd/container /run/host/container-manager

# Verify user namespaces are available
cat /proc/sys/kernel/unprivileged_userns_clone
cat /proc/sys/kernel/apparmor_restrict_unprivileged_userns

# Update package lists
apt update
apt full-upgrade -y

apt install -y libgl1-mesa-dri:i386 libgl1:amd64 libgl1:i386 libgbm1:amd64 libgbm1:i386 steam-libs-amd64:amd64 steam-libs-i386:i386 xdg-desktop-portal xdg-desktop-portal-kde

# Download Steam directly from Valve
cd /home/ubuntu
wget -O steam.deb "https://cdn.akamai.steamstatic.com/client/installer/steam.deb"

# Install Steam and its dependencies
apt install -y ./steam.deb

# Ensure ubuntu user owns the home directory
chown -R ubuntu:ubuntu /home/ubuntu

echo ""
echo "Steam installed successfully!"
echo "Launch Steam from the desktop or run: steam"
echo ""
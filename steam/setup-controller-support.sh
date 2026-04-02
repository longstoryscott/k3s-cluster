#!/bin/bash
# Setup controller support inside the Selkies desktop container
# Run this after mounting /dev/input from the host

set -e

echo "=== Setting up controller support in Selkies container ==="

# Install steam-devices for udev rules (Ubuntu/Debian)
echo "Installing steam-devices package..."
apt-get update
apt-get install -y steam-devices

# Reload udev rules
echo "Reloading udev rules..."
udevadm control --reload-rules
udevadm trigger

# Add ubuntu user to input group (if not already)
echo "Adding ubuntu user to input group..."
usermod -a -G input ubuntu

# Create uinput device rule if needed
echo "Setting up uinput access..."
echo 'KERNEL="uinput", GROUP="input", MODE="0664"' > /etc/udev/rules.d/99-uinput.rules
udevadm control --reload-rules

# Load uinput module if available
modprobe uinput 2>/dev/null || echo "uinput module not available (expected in container)"

echo ""
echo "=== Controller setup complete ==="
echo ""
echo "To verify controller detection inside the container:"
echo "  1. Restart the container: kubectl rollout restart statefulset/selkies-desktop-official -n steam"
echo "  2. Exec into the container:"
echo "     kubectl exec -it -n steam \$(kubectl get pods -n steam -o jsonpath='{.items[0].metadata.name}') -- bash"
echo "  3. Check controller devices:"
echo "     ls -la /dev/input/"
echo "  4. Test with jstest (if installed):"
echo "     apt-get install -y joystick && jstest /dev/input/js0"
echo ""
echo "Then restart Steam and check controller settings:"
echo "  Steam -> Settings -> Controller -> General Controller Settings"
echo "  Ensure 'Steam Controller Support' is enabled"

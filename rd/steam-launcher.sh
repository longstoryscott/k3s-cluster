#!/bin/bash
# Steam Launcher Script - Run Steam as ubuntu user from root terminal
# This script solves the "cannot run as root" issue

set -e

echo "🎮 Steam Launcher for Ubuntu Desktop"
echo "===================================="

# Ensure ubuntu user exists
if ! id ubuntu &>/dev/null; then
    echo "Creating ubuntu user..."
    useradd -m -s /bin/bash ubuntu
    usermod -aG sudo ubuntu
    echo "ubuntu ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/ubuntu
fi

# Check if Steam is installed
if ! command -v steam &>/dev/null; then
    echo "Steam not found. Installing Steam..."
    
    # Enable multiverse repository and add 32-bit architecture
    apt update
    add-apt-repository -y multiverse
    dpkg --add-architecture i386
    apt update
    
    # Install Steam from multiverse
    apt install -y steam-installer
    
    echo "✅ Steam installed!"
fi

# Create steam launcher script for ubuntu user
cat > /home/ubuntu/launch-steam.sh << 'EOF'
#!/bin/bash
export HOME=/home/ubuntu
export USER=ubuntu
export DISPLAY=:1

# Set up Steam directories
mkdir -p /home/ubuntu/.steam
mkdir -p /home/ubuntu/.local/share/Steam

# Launch Steam
echo "🎮 Starting Steam as ubuntu user..."
steam "$@"
EOF

chmod +x /home/ubuntu/launch-steam.sh
chown ubuntu:ubuntu /home/ubuntu/launch-steam.sh

echo ""
echo "🎯 Steam Setup Complete!"
echo ""
echo "To launch Steam:"
echo "1. Open terminal in the desktop"
echo "2. Run: su - ubuntu"
echo "3. Run: ./launch-steam.sh"
echo ""
echo "Or run directly as root:"
echo "su - ubuntu -c '/home/ubuntu/launch-steam.sh'"
echo ""
echo "Steam will now run as the ubuntu user and work correctly!"
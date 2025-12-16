#!/bin/bash
# Helper script for installing packages in the persistent desktop environment
# Run this script inside the desktop environment via terminal

set -e

echo "🔧 Package Installation Helper for Steam Desktop"
echo "================================================"

# Check if running as root or with sudo
if [[ $EUID -eq 0 ]]; then
    SUDO=""
elif command -v sudo &> /dev/null; then
    SUDO="sudo"
    echo "ℹ️  Running with sudo privileges"
else
    echo "❌ Error: Need root privileges or sudo access"
    exit 1
fi

# Function to install .deb packages
install_deb() {
    local deb_file="$1"
    echo "📦 Installing .deb package: $deb_file"
    
    if [[ ! -f "$deb_file" ]]; then
        echo "❌ Error: File $deb_file not found"
        return 1
    fi
    
    # Ensure gdebi-core is installed
    if ! command -v gdebi &>/dev/null; then
        echo "📥 Installing gdebi-core..."
        $SUDO apt update --allow-unauthenticated || true
        $SUDO apt install -y gdebi-core
    fi
    
    # Use gdebi for better dependency handling
    $SUDO gdebi -n "$deb_file"
    echo "✅ Successfully installed $deb_file"
}

# Function to install Steam
install_steam() {
    echo "🎮 Installing Steam..."
    
    # Clean up any problematic Steam repository
    $SUDO rm -f /etc/apt/sources.list.d/steam*.list 2>/dev/null || true
    
    # Add multiverse repository for 32-bit libraries
    $SUDO add-apt-repository multiverse -y
    $SUDO dpkg --add-architecture i386
    $SUDO apt update --allow-unauthenticated || $SUDO apt update
    
    # Try different installation methods
    if $SUDO apt install -y steam-installer; then
        echo "✅ Steam installed via apt!"
    elif [[ -f "/home/ubuntu/steam_latest.deb" ]]; then
        echo "📦 Installing Steam from .deb file..."
        install_deb "/home/ubuntu/steam_latest.deb"
    else
        echo "📥 Downloading Steam .deb..."
        cd /tmp
        wget -O steam.deb "https://cdn.akamai.steamstatic.com/client/installer/steam.deb"
        install_deb "steam.deb"
    fi
    
    # Set up Steam for ubuntu user
    setup_steam_for_ubuntu
    
    echo "✅ Steam installation complete!"
    echo "🎯 Run: su - ubuntu -c 'steam' to launch Steam as ubuntu user"
}

# Function to install common development tools
install_dev_tools() {
    echo "🛠️  Installing development tools..."
    
    # Clean up problematic repositories
    $SUDO rm -f /etc/apt/sources.list.d/steam*.list 2>/dev/null || true
    
    $SUDO apt update --allow-unauthenticated || $SUDO apt update || true
    $SUDO apt install -y \
        build-essential \
        git \
        vim \
        nano \
        htop \
        firefox \
        libreoffice \
        vlc \
        gimp \
        gdebi-core \
        software-properties-common
        
    echo "✅ Development tools installed!"
}

# Function to install Google Chrome
install_chrome() {
    echo "🌐 Installing Google Chrome..."
    
    # Download and install Chrome
    cd /tmp
    wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | $SUDO apt-key add -
    echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" | $SUDO tee /etc/apt/sources.list.d/google-chrome.list
    $SUDO apt update
    $SUDO apt install -y google-chrome-stable
    
    echo "✅ Google Chrome installed!"
}

# Function to install VS Code
install_vscode() {
    echo "💻 Installing Visual Studio Code..."
    
    # Download and install VS Code
    wget -qO- https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > packages.microsoft.gpg
    $SUDO install -o root -g root -m 644 packages.microsoft.gpg /etc/apt/trusted.gpg.d/
    $SUDO sh -c 'echo "deb [arch=amd64,arm64,armhf signed-by=/etc/apt/trusted.gpg.d/packages.microsoft.gpg] https://packages.microsoft.com/repos/code stable main" > /etc/apt/sources.list.d/vscode.list'
    $SUDO apt update
    $SUDO apt install -y code
    
    echo "✅ VS Code installed!"
}

# Function to set up Steam for ubuntu user
setup_steam_for_ubuntu() {
    echo "🔧 Setting up Steam for ubuntu user..."
    
    # Ensure ubuntu user exists
    if ! id ubuntu &>/dev/null; then
        $SUDO useradd -m -s /bin/bash ubuntu
        $SUDO usermod -aG sudo ubuntu
        echo "ubuntu ALL=(ALL) NOPASSWD:ALL" | $SUDO tee /etc/sudoers.d/ubuntu
    fi
    
    # Create Steam launcher script for ubuntu user
    $SUDO tee /home/ubuntu/launch-steam.sh > /dev/null << 'EOF'
#!/bin/bash
export HOME=/home/ubuntu
export USER=ubuntu
export DISPLAY=:1

# Set up Steam directories in persistent storage
mkdir -p /home/ubuntu/.steam
mkdir -p /home/ubuntu/.local/share/Steam
mkdir -p /opt/steam-games  # Games in persistent /opt

# Link games directory to persistent storage
if [[ ! -L /home/ubuntu/.steam/steam/steamapps ]]; then
    mkdir -p /home/ubuntu/.steam/steam
    ln -sf /opt/steam-games /home/ubuntu/.steam/steam/steamapps
fi

echo "🎮 Starting Steam as ubuntu user..."
steam "$@"
EOF
    
    $SUDO chmod +x /home/ubuntu/launch-steam.sh
    $SUDO chown ubuntu:ubuntu /home/ubuntu/launch-steam.sh
    
    # Create convenience script for root to launch Steam
    $SUDO tee /home/ubuntu/start-steam.sh > /dev/null << 'EOF'
#!/bin/bash
echo "🎮 Launching Steam as ubuntu user..."
su - ubuntu -c '/home/ubuntu/launch-steam.sh'
EOF
    
    $SUDO chmod +x /home/ubuntu/start-steam.sh
}

# Function to show persistent storage info
show_storage_info() {
    echo ""
    echo "💾 Persistent Storage Information"
    echo "================================="
    echo "✅ /home/ubuntu               - Your home directory (100GB)"
    echo "✅ /opt                       - Applications like Steam (100GB)"  
    echo "✅ /usr/local/bin-custom      - Custom executables (10GB)"
    echo "✅ /etc/apt/sources.list.d    - Additional repositories (5GB)"
    echo ""
    echo "These directories persist across pod restarts!"
    echo "Install applications to /opt for persistence, or use 'apt install' for system-wide installation."
    echo "Note: Package database is managed by the system for best compatibility."
}

# Function to check disk usage
check_disk_usage() {
    echo ""
    echo "📊 Current Disk Usage"
    echo "===================="
    df -h /home/ubuntu /opt /usr/local/bin-custom /etc/apt/sources.list.d 2>/dev/null || true
}

# Function to initialize system
init_system() {
    echo "🔧 Initializing system..."
    
    # Remove problematic repositories
    $SUDO rm -f /etc/apt/sources.list.d/steam*.list 2>/dev/null || true
    
    # Update package lists (ignore signature errors)
    $SUDO apt update --allow-unauthenticated || $SUDO apt update || true
    
    # Install essential packages
    $SUDO apt install -y gdebi-core software-properties-common wget curl
    
    # Set up ubuntu user
    if ! id ubuntu &>/dev/null; then
        $SUDO useradd -m -s /bin/bash ubuntu
        $SUDO usermod -aG sudo ubuntu
        echo "ubuntu ALL=(ALL) NOPASSWD:ALL" | $SUDO tee /etc/sudoers.d/ubuntu
    fi
    
    echo "✅ System initialized!"
}

# Main menu
show_menu() {
    echo ""
    echo "Available options:"
    echo "1) Install Steam (with ubuntu user setup)"
    echo "2) Install development tools bundle" 
    echo "3) Install Google Chrome"
    echo "4) Install VS Code"
    echo "5) Install .deb package (specify path)"
    echo "6) Set up ubuntu user for Steam"
    echo "7) Show storage information"
    echo "8) Check disk usage"
    echo "9) Initialize system (install gdebi, fix repos)"
    echo "10) Exit"
    echo ""
}

# Main script logic
if [[ $# -eq 0 ]]; then
    # Interactive mode
    while true; do
        show_menu
        read -p "Select option (1-10): " choice
        
        case $choice in
            1) install_steam ;;
            2) install_dev_tools ;;
            3) install_chrome ;;
            4) install_vscode ;;
            5) 
                read -p "Enter path to .deb file: " deb_path
                install_deb "$deb_path"
                ;;
            6) setup_steam_for_ubuntu ;;
            7) show_storage_info ;;
            8) check_disk_usage ;;
            9) init_system ;;
            10) echo "Goodbye!"; exit 0 ;;
            *) echo "❌ Invalid option. Please try again." ;;
        esac
        echo ""
        read -p "Press Enter to continue..."
    done
else
    # Command line mode
    case "$1" in
        "steam") install_steam ;;
        "dev-tools") install_dev_tools ;;
        "chrome") install_chrome ;;
        "vscode") install_vscode ;;
        "init") init_system ;;
        "setup-steam") setup_steam_for_ubuntu ;;
        "deb") 
            if [[ -z "$2" ]]; then
                echo "❌ Usage: $0 deb <path-to-deb-file>"
                exit 1
            fi
            install_deb "$2"
            ;;
        "info") show_storage_info ;;
        "usage") check_disk_usage ;;
        *)
            echo "❌ Usage: $0 [steam|dev-tools|chrome|vscode|init|setup-steam|deb <file>|info|usage]"
            echo "   Or run without arguments for interactive mode"
            exit 1
            ;;
    esac
fi
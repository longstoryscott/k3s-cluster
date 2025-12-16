# Steam Desktop Environment

A GPU-accelerated Linux desktop environment with persistent application storage and root privileges for manual software installation.

## Files

- `desktop-statefulset.yaml` - Ubuntu LXDE desktop with GPU acceleration and comprehensive persistent storage
- `desktop-service.yaml` - Kubernetes service for desktop access  
- `desktop-referencegrant.yaml` - Gateway API reference grant for cross-namespace access
- `install.sh` - Installation script for the desktop environment
- `install-packages.sh` - Helper script for installing applications and .deb packages

## Usage

1. **Deploy the desktop:**

   ```bash
   ./install.sh
   ```

2. **Access the desktop:**
   - URL: `http://192.168.0.71:8087/`
   - Password: `mypassword` (for VNC access)
   - Uses noVNC web interface

3. **Install software (multiple options):**

   **Option A - Using the helper script (Recommended):**
   ```bash
   # Copy the helper script to the desktop
   # In the desktop terminal, run:
   chmod +x install-packages.sh
   ./install-packages.sh
   # Or run specific commands:
   ./install-packages.sh steam
   ./install-packages.sh chrome
   ./install-packages.sh deb /path/to/package.deb
   ```

   **Option B - Manual installation:**
   ```bash
   # In the desktop terminal:
   sudo apt update
   sudo apt install steam-installer
   # Or for .deb packages:
   sudo gdebi /path/to/package.deb
   ```

## Features

### Persistent Storage (New!)
- **User Home**: `/home/ubuntu` (100GB) - Your personal files and settings
- **Applications**: `/opt` (100GB) - Installed applications persist across restarts
- **Custom Binaries**: `/usr/local/bin-custom` (10GB) - User-installed executables 
- **Repositories**: `/etc/apt/sources.list.d` (5GB) - Custom software sources

**Note**: Install applications to `/opt` for persistence, or use standard package management. Package database is managed by the system for optimal compatibility.

### Other Features
- **GPU Acceleration**: NVIDIA RTX 2060 available for gaming
- **Root Access**: Full sudo privileges for software installation
- **Web Access**: Browser-based desktop via noVNC
- **Ubuntu LXDE**: Lightweight desktop environment at 1920x1080 resolution
- **Enhanced Security**: Runs as root with proper privilege management

## Architecture

- **Runtime**: nvidia runtime class for GPU access
- **Node**: lsnode-3 (GPU node)
- **Port**: 6080 (noVNC) via gateway on port 8087
- **Storage**: local-path persistent volumes for comprehensive data retention

## Troubleshooting

### Can't Install .deb Packages?
1. Ensure you're using `sudo` or running as root
2. Use `gdebi` instead of `dpkg` for better dependency resolution:
   ```bash
   sudo gdebi package.deb
   ```
3. Try the helper script: `./install-packages.sh deb /path/to/package.deb`

### Applications Don't Persist?
- Install to `/opt` or `/usr/local` directories
- Use the package manager (`apt`) which automatically uses persistent locations
- Check storage with: `./install-packages.sh usage`

### Need More Storage?
Edit the `volumeClaimTemplates` in `desktop-statefulset.yaml` to increase storage sizes.

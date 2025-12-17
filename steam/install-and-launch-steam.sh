#!/bin/bash

echo "=== Complete Steam Setup for Selkies GLX Desktop ==="

# Remove container detection files
echo "Removing container detection files..."
rm -f /run/systemd/container /run/host/container-manager

# Verify user namespaces
echo "Checking user namespace settings..."
echo "  unprivileged_userns_clone: $(cat /proc/sys/kernel/unprivileged_userns_clone)"
echo "  apparmor_restrict_unprivileged_userns: $(cat /proc/sys/kernel/apparmor_restrict_unprivileged_userns)"

# Check if Steam is already installed
if ! command -v steam &> /dev/null; then
    echo ""
    echo "Installing Steam..."
    
    # Update and install dependencies
    apt update
    apt install -y \
        libgl1-mesa-dri:i386 \
        libgl1:amd64 \
        libgl1:i386 \
        libgbm1:amd64 \
        libgbm1:i386 \
        steam-libs-amd64:amd64 \
        steam-libs-i386:i386 \
        xdg-desktop-portal \
        xdg-desktop-portal-kde \
        zenity
    
    # Download and install Steam
    cd /tmp
    wget -O steam.deb "https://cdn.akamai.steamstatic.com/client/installer/steam.deb"
    apt install -y ./steam.deb
    rm steam.deb
    
    echo "Steam installed successfully!"
else
    echo "Steam is already installed."
fi

# Create the namespaceless launcher
echo ""
echo "Setting up namespaceless launcher..."

STEAM_DIR="${HOME}/.local/share/Steam"
CUSTOM_ENTRYPOINT="${HOME}/.steam-bypass"

mkdir -p "${CUSTOM_ENTRYPOINT}"
cat > "${CUSTOM_ENTRYPOINT}/_v2-entry-point" << 'BYPASS_EOF'
#!/bin/sh
# Bypass pressure-vessel completely - run commands directly
exec "$@"
BYPASS_EOF
chmod +x "${CUSTOM_ENTRYPOINT}/_v2-entry-point"

# Create desktop launcher script
cat > "${HOME}/launch-steam.sh" << 'LAUNCHER_EOF'
#!/bin/bash

STEAM_DIR="${HOME}/.local/share/Steam"
CUSTOM_ENTRYPOINT="${HOME}/.steam-bypass"

patch_steam_runtime() {
    local runtime_path="$1"
    local custom_entrypoint="${CUSTOM_ENTRYPOINT}/_v2-entry-point"
    local temp_entrypoint="/tmp/_v2-entry-point.padded"
    
    if [[ -f "$runtime_path" ]]; then
        local original_size=$(stat -c %s "$runtime_path" 2>/dev/null)
        if [[ -n "$original_size" ]] && [[ "$original_size" -gt 0 ]]; then
            cp "$custom_entrypoint" "$temp_entrypoint" 2>/dev/null
            truncate -s "$original_size" "$temp_entrypoint" 2>/dev/null
            cp "$temp_entrypoint" "$runtime_path" 2>/dev/null
        fi
    fi
}

# Start background patcher
(
    while true; do
        sudo rm -f /run/systemd/container /run/host/container-manager 2>/dev/null
        patch_steam_runtime "${STEAM_DIR}/ubuntu12_64/steam-runtime-sniper/_v2-entry-point"
        patch_steam_runtime "${STEAM_DIR}/steamapps/common/SteamLinuxRuntime_sniper/_v2-entry-point"
        patch_steam_runtime "${STEAM_DIR}/steamapps/common/SteamLinuxRuntime_soldier/_v2-entry-point"
        sleep 0.1
    done
) &
PATCHER_PID=$!

cleanup() {
    kill $PATCHER_PID 2>/dev/null || true
    exit 0
}

trap cleanup EXIT INT TERM

steam
LAUNCHER_EOF

chmod +x "${HOME}/launch-steam.sh"

echo ""
echo "✓ Setup complete!"
echo ""
echo "To launch Steam, run:"
echo "  ~/launch-steam.sh"
echo ""
echo "Or add it to your desktop as a shortcut."

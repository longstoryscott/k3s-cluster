#!/bin/bash

# Steam Namespaceless Launcher
# Based on solution from: https://github.com/selkies-project/docker-selkies-glx-desktop/issues/47

set -e

STEAM_DIR="${HOME}/.local/share/Steam"
CUSTOM_ENTRYPOINT="${HOME}/.steam-bypass"

echo "=== Steam Namespaceless Launcher ==="
echo "This script bypasses Steam's bubblewrap/namespace requirements"
echo ""

# Create custom entry point that bypasses bubblewrap
mkdir -p "${CUSTOM_ENTRYPOINT}"
cat > "${CUSTOM_ENTRYPOINT}/_v2-entry-point" << 'EOF'
#!/bin/sh
# Bypass pressure-vessel completely
exec "$@"
EOF
chmod +x "${CUSTOM_ENTRYPOINT}/_v2-entry-point"

echo "Created bypass entrypoint at ${CUSTOM_ENTRYPOINT}/_v2-entry-point"

# Function to patch Steam runtime files
patch_steam_runtime() {
    local runtime_path="$1"
    local custom_entrypoint="${CUSTOM_ENTRYPOINT}/_v2-entry-point"
    local temp_entrypoint="/tmp/_v2-entry-point.padded"
    
    if [[ -f "$runtime_path" ]]; then
        # Get original file size
        local original_size
        original_size=$(stat -c %s "$runtime_path" 2>/dev/null)
        
        if [[ -n "$original_size" ]] && [[ "$original_size" -gt 0 ]]; then
            # Copy custom entrypoint and pad to match original size
            cp "$custom_entrypoint" "$temp_entrypoint" 2>/dev/null
            truncate -s "$original_size" "$temp_entrypoint" 2>/dev/null
            cp "$temp_entrypoint" "$runtime_path" 2>/dev/null && echo "  ✓ Patched: $runtime_path"
        fi
    fi
}

# Start background patcher
echo "Starting background patcher..."
(
    while true; do
        # Remove container detection files continuously
        sudo rm -f /run/systemd/container /run/host/container-manager 2>/dev/null
        
        # Patch all known Steam runtime entry points
        patch_steam_runtime "${STEAM_DIR}/ubuntu12_64/steam-runtime-sniper/_v2-entry-point"
        patch_steam_runtime "${STEAM_DIR}/steamapps/common/SteamLinuxRuntime_sniper/_v2-entry-point"
        patch_steam_runtime "${STEAM_DIR}/steamapps/common/SteamLinuxRuntime_soldier/_v2-entry-point"
        
        sleep 0.1
    done
) &
PATCHER_PID=$!

echo "Background patcher started (PID: $PATCHER_PID)"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Stopping background patcher..."
    kill $PATCHER_PID 2>/dev/null || true
    exit 0
}

trap cleanup EXIT INT TERM

# Launch Steam
echo "Launching Steam..."
echo "Press Ctrl+C to stop"
echo ""

steam

#!/bin/bash
# Custom entrypoint that disables the Selkies joystick interposer
# This allows games to detect controllers through Steam's native input handling
# without the interposer creating fake device nodes

set -e

# Skip the joystick interposer setup
# Original interposer would set:
#   export SELKIES_INTERPOSER='/usr/$LIB/selkies_joystick_interposer.so'
#   export LD_PRELOAD="${SELKIES_INTERPOSER}${LD_PRELOAD:+:${LD_PRELOAD}}"
#   export SDL_JOYSTICK_DEVICE=/dev/input/js0

# Remove any fake joystick device nodes that may exist
rm -f /dev/input/js0 /dev/input/js1 /dev/input/js2 /dev/input/js3 2>/dev/null || true

# Set display environment
export DISPLAY=:0

# Start supervisord to manage desktop services
exec /usr/bin/supervisord -c /etc/supervisor/supervisord.conf

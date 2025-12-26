# Add this to the end of ~/.bashrc on the client

# Auto-start SteamLink on tty1 and start Steam in pod
if [ -z "$DISPLAY" ] && [ "$(tty)" = "/dev/tty1" ]; then
    echo "Starting SteamLink..." | tee /tmp/steamlink-starting.log
    # Start Steam in pod in background before exec
    ~/start-steam-in-pod.sh &
    # Now exec steamlink (replaces this shell)
    exec steamlink
fi

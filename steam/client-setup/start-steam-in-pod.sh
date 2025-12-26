#!/bin/bash
# Script to auto-start Steam in the k8s pod via supervisord
# This ensures Steam starts cleanly without interactive prompts

# Wait for cluster and pod to be fully ready
sleep 15

# Get the pod name
POD_NAME=$(kubectl get pod -n steam -l app=selkies-desktop-official -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -z "$POD_NAME" ]; then
    echo "Steam pod not found" | tee -a /tmp/steam-start.log
    exit 1
fi

echo "Starting Steam in pod $POD_NAME at $(date)" | tee -a /tmp/steam-start.log

# Create a systemd-style autostart by adding to supervisord config
kubectl exec -n steam "$POD_NAME" -- bash -c 'cat > /etc/supervisor/conf.d/steam.conf << '\''STEAMCONF'\''
[program:steam]
command=/bin/bash -c "sleep 10 && DISPLAY=:20 steam"
autostart=true
autorestart=false
stdout_logfile=/tmp/steam-supervisor.log
stderr_logfile=/tmp/steam-supervisor.log
user=ubuntu
STEAMCONF

supervisorctl reread
supervisorctl update
supervisorctl start steam' 2>&1 | tee -a /tmp/steam-start.log

echo "Steam supervisord config added" | tee -a /tmp/steam-start.log

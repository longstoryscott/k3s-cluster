#!/bin/bash

NODES=(lsnode-1 lsnode-2 lsnode-3)

for node in "${NODES[@]}"; do
    ssh "${node}.local" '/usr/local/bin/k3s-killall.sh && /usr/local/bin/k3s-agent-uninstall.sh'
done

/usr/local/bin/k3s-killall.sh && /usr/local/bin/k3s-uninstall.sh

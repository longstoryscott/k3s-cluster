#!/bin/bash
# Test controller locally to isolate if issue is local or remote

echo "Testing controller on local machine..."
echo "Press Ctrl+C to stop"
echo ""
echo "Joystick device: /dev/input/js0"
echo "Checking for stuck axes..."

# Monitor for 10 seconds and show if any axis values are stuck
timeout 10 cat /dev/input/js0 | od -An -tx1 -w16 -v | while read line; do
    echo "$line"
done

echo ""
echo "If you see repeating identical lines, the controller is sticking locally"
echo "If values change smoothly, the issue is on the remote desktop side"

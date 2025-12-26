#!/bin/bash
# Auto-connect Bluetooth controller on boot
MAC="84:17:66:D5:E8:B9"
echo "power on" | bluetoothctl
sleep 1
echo "connect $MAC" | bluetoothctl

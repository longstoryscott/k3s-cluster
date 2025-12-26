#!/bin/bash
# Auto-connect Bluetooth controller on boot
MAC="84:17:66:D5:E8:B9"

# Ensure Bluetooth is powered on
bluetoothctl power on
sleep 2

# Trust the device (required for auto-reconnect)
bluetoothctl trust "$MAC"

# Connect
bluetoothctl connect "$MAC"

# Verify connection and bonding status
sleep 2
bluetoothctl info "$MAC" | grep -E "Connected|Bonded|Paired"

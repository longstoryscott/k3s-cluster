#!/bin/bash

DEV_NAME=${1:-eth0}

IP=$(ip a show "${DEV_NAME}" | grep "inet[[:space:]]" | tr -s ' ' | cut -d' ' -f 3 | cut -d'/' -f 1)
NAME=$(nmcli con show | grep "${DEV_NAME}" | sed -E 's|\s\s|:|' | cut -d':' -f 1)

sudo nmcli con mod "${NAME}" ipv4.addresses "${IP}/24" >/dev/null 2>&1
sudo nmcli con mod "${NAME}" ipv4.gateway "192.168.0.1" >/dev/null 2>&1
sudo nmcli con mod "${NAME}" ipv4.method manual >/dev/null 2>&1
sudo nmcli con up "${NAME}" >/dev/null 2>&1

printf '%s' "${IP}"

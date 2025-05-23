#!/bin/bash

NODE_NAME=${1:-$(hostname)}
DEV_NAME=${2:-eth0}
MASTER_IP="192.168.0.71"
TOKEN=sKrGps5Hfv5wB87obv5MHkhhM2fbFaPMSRrJc
HOSTS_FILE="/etc/hosts"
BACKUP_EXT=".bak"

IP=$($(dirname $0)/set-static-ip.sh "${DEV_NAME}")

mkdir -p ~/.kube
touch ~/.kube/config
curl -sfL https://get.k3s.io | K3S_URL=https://${MASTER_IP}:6443 K3S_NODE_NAME=\"${NODE_NAME}\" K3S_TOKEN=${TOKEN} K3S_KUBECONFIG_MODE='600' sh -s -

CONF=$(sudo cat /etc/rancher/k3s/k3s.yaml | sed "s|127.0.0.1|${MASTER_IP}|g")

echo "${CONF}" >~/.kube/config
sudo chmod 600 ~/.kube/config

sudo hostnamectl set-hostname "${NODE_NAME}" --static

# --- Script Logic ---

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run this script with sudo."
  exit 1
fi

# Check if the hosts file exists
if [ ! -f "$HOSTS_FILE" ]; then
  echo "Error: $HOSTS_FILE not found!"
  exit 1
fi

# Create a backup of the original file
if [ -f "${HOSTS_FILE}${BACKUP_EXT}" ]; then
  echo "Backup file ${HOSTS_FILE}${BACKUP_EXT} already exists. Overwriting."
fi
sudo cp "$HOSTS_FILE" "${HOSTS_FILE}${BACKUP_EXT}"
if [ $? -eq 0 ]; then
  echo "Backup created at ${HOSTS_FILE}${BACKUP_EXT}"
else
  echo "Error: Failed to create backup. Exiting."
  exit 1
fi

echo "Updating hostname in $HOSTS_FILE..."

# Use sed to find the line starting with 127.0.1.1
# and replace the first hostname field after the IP and whitespace.
# Explanation of the sed command:
# /^127\.0\.1\.1/  - Match lines that start with "127.0.1.1" (escaped dots)
# s/             - Start substitution command on the matched line
# (\s+)          - Capture one or more whitespace characters (Group 1)
# \S+            - Match one or more non-whitespace characters (the current hostname)
# /              - Separator
# \1             - Insert the captured whitespace (from Group 1)
# ${NEW_HOSTNAME} - Insert the new hostname
# /              - Separator
# The command replaces the whitespace followed by the old hostname with the same whitespace followed by the new hostname.

sudo sed -i "$BACKUP_EXT" "/^127\.0\.1\.1/ s/(\s+)\S+/\1${NODE_NAME}/" "$HOSTS_FILE"

# Check if sed command was successful (basic check)
if [ $? -eq 0 ]; then
  echo "Successfully updated $HOSTS_FILE. The old line (likely starting with 127.0.1.1) should now use '$NODE_NAME'."
  echo "Displaying the updated line(s) for 127.0.1.1:"
  grep "^127\.0\.1\.1" "$HOSTS_FILE"
else
  echo "Error: An error occurred during sed execution."
  echo "The original file has been preserved in ${HOSTS_FILE}${BACKUP_EXT}"
  exit 1
fi

echo "Generating SSH key..."
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N "" -C "${NODE_NAME}" >/dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "Error: Failed to generate SSH key."
  exit 1
fi
echo "SSH key generated at ~/.ssh/id_rsa"
echo "Copying SSH key to master node..."
ssh-copy-id -i ~/.ssh/id_rsa.pub lsm@${MASTER_IP} >/dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "Error: Failed to copy SSH key to master node."
  exit 1
fi
echo "SSH key copied to master node ${MASTER_IP}."

echo "New node ${NODE_NAME} (${IP}) added to cluster!"

exit

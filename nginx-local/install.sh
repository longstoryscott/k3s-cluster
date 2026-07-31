#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ── Configuration ──
TARGET_HOST="${TARGET_HOST:-192.168.0.71}"
TARGET_USER="${TARGET_USER:-lsm}"
NGINX_CONF="${SCRIPT_DIR}/nginx.conf"
DNSMASQ_CONF="${SCRIPT_DIR}/dnsmasq.conf"
NGINX_DIR="/opt/nginx-local"
SYSTEMD_UNIT="/etc/systemd/system/nginx-local.service"

# Router config (load from ../.env if available)
if [[ -f "${PROJECT_ROOT}/.env" ]]; then
  source "${PROJECT_ROOT}/.env"
fi
ROUTER_HOST="${ROUTER_HOST:-192.168.0.1}"
ROUTER_USER="${ROUTER_USER:-root}"

echo "=== nginx-local: Central LAN Reverse Proxy ==="
echo "Target: ${TARGET_USER}@${TARGET_HOST}"
echo "Router: ${ROUTER_USER}@${ROUTER_HOST}"
echo ""

# ── Step 1: Create directories ──
echo "Step 1: Creating remote directories..."
ssh "${TARGET_USER}@${TARGET_HOST}" "sudo mkdir -p ${NGINX_DIR} /var/log/nginx-local /run/nginx-local"

# ── Step 2: Copy nginx config ──
echo "Step 2: Copying nginx configuration..."
scp "${NGINX_CONF}" "${TARGET_USER}@${TARGET_HOST}:~/nginx-local.conf"
ssh "${TARGET_USER}@${TARGET_HOST}" "sudo mv /home/${TARGET_USER}/nginx-local.conf ${NGINX_DIR}/nginx.conf && sudo chmod 644 ${NGINX_DIR}/nginx.conf"

# ── Step 3: Install nginx if not present ──
echo "Step 3: Ensuring nginx is installed..."
ssh "${TARGET_USER}@${TARGET_HOST}" "
if sudo command -v nginx &>/dev/null; then
  echo 'nginx already installed.'
else
  echo 'Installing nginx...'
  sudo dpkg --purge nginx-common 2>/dev/null || true
  sudo apt-get update -qq 2>/dev/null || true
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq nginx
  echo 'nginx installed.'
fi
"

# ── Step 4: Create systemd unit ──
echo "Step 4: Creating systemd unit..."
cat > /tmp/nginx-local.service <<'UNIT'
[Unit]
Description=nginx-local — Central LAN Reverse Proxy
After=network-online.target
Wants=network-online.target

[Service]
Type=forking
PIDFile=/run/nginx-local/nginx.pid
ExecStartPre=/usr/sbin/nginx -t -c /opt/nginx-local/nginx.conf
ExecStart=/usr/sbin/nginx -c /opt/nginx-local/nginx.conf
ExecReload=/bin/kill -HUP $MAINPID
ExecStop=/sbin/start-stop-daemon --quiet --stop --retry QUIT/5 --pidfile /run/nginx-local/nginx.pid
TimeoutSec=5
KillMode=mixed
KillSignal=SIGTERM
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
UNIT
scp /tmp/nginx-local.service "${TARGET_USER}@${TARGET_HOST}:~/nginx-local.service"
ssh "${TARGET_USER}@${TARGET_HOST}" "sudo mv /home/${TARGET_USER}/nginx-local.service ${SYSTEMD_UNIT}"
rm -f /tmp/nginx-local.service

# ── Step 5: Fix DNS if needed ──
echo "Step 5: Fixing DNS resolution..."
ssh "${TARGET_USER}@${TARGET_HOST}" "
# Add IPv4 nameservers if missing
if ! grep -q 'nameserver 192.168.0.1' /etc/resolv.conf; then
  echo 'Adding IPv4 DNS nameservers...'
  sudo bash -c 'echo \"nameserver 192.168.0.1\" >> /etc/resolv.conf && echo \"nameserver 8.8.8.8\" >> /etc/resolv.conf'
fi
# Make persistent via NetworkManager
sudo bash -c 'mkdir -p /etc/NetworkManager/conf.d && cat > /etc/NetworkManager/conf.d/ipv4-dns.conf << EOF
[device]
dns=192.168.0.1,8.8.8.8
EOF'
# Fix NSS so DNS is tried before mDNS (mDNS intercepts .local)
if grep -q 'mdns4_minimal \[NOTFOUND=return\] dns' /etc/nsswitch.conf; then
  echo 'Fixing NSS order (DNS before mDNS)...'
  sudo sed -i 's/hosts:.*files mdns4_minimal \[NOTFOUND=return\] dns/hosts:          files dns mdns4_minimal [NOTFOUND=return]/' /etc/nsswitch.conf
fi
"

# ── Step 6: Install and configure dnsmasq ──
echo "Step 6: Setting up dnsmasq for hostname resolution..."
ssh "${TARGET_USER}@${TARGET_HOST}" "
if ! sudo command -v dnsmasq &>/dev/null; then
  echo 'Installing dnsmasq...'
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq dnsmasq
fi
"
scp "${DNSMASQ_CONF}" "${TARGET_USER}@${TARGET_HOST}:~/dnsmasq-local.conf"
ssh "${TARGET_USER}@${TARGET_HOST}" "
sudo mv /home/${TARGET_USER}/dnsmasq-local.conf /etc/dnsmasq.d/local-services
# Ensure resolv.conf has 127.0.0.1 first for dnsmasq
if ! head -1 /etc/resolv.conf | grep -q 'nameserver 127.0.0.1'; then
  sudo bash -c 'echo \"nameserver 127.0.0.1\" | cat - /etc/resolv.conf > /tmp/resolv.new && mv /tmp/resolv.new /etc/resolv.conf'
fi
sudo systemctl restart dnsmasq
sudo systemctl enable dnsmasq
echo 'dnsmasq configured.'
"

# ── Step 7: Configure router DNS ──
echo "Step 7: Configuring router DNS for LAN-wide resolution..."
# Create persistent dnsmasq config on router's overlay filesystem
ROUTER_DNSMASQ_CONF=$(mktemp)
cat > "${ROUTER_DNSMASQ_CONF}" <<'EOF'
# nginx-local reverse proxy services on lsnode-0
address=/nc.lan/192.168.0.71
address=/ha.lan/192.168.0.71
address=/auth.lan/192.168.0.71
address=/usrmgr.lan/192.168.0.71
address=/search.lan/192.168.0.71
address=/steam.lan/192.168.0.71
address=/pgadmin.lan/192.168.0.71
address=/registry.lan/192.168.0.71
address=/regui.lan/192.168.0.71
address=/fnf.lan/192.168.0.71
address=/llmll.lan/192.168.0.71
address=/lsnode-0.lan/192.168.0.71
EOF

scp "${ROUTER_DNSMASQ_CONF}" "${ROUTER_USER}@${ROUTER_HOST}:/tmp/local-services.conf"
ssh "${ROUTER_USER}@${ROUTER_HOST}" "
mkdir -p /etc/dnsmasq.d
cp /tmp/local-services.conf /etc/dnsmasq.d/local-services.conf
uci set dhcp.@dnsmasq[0].confdir='/etc/dnsmasq.d'
uci commit dhcp
/etc/init.d/dnsmasq restart
echo 'Router DNS configured (persistent on overlay filesystem).'
"
rm -f "${ROUTER_DNSMASQ_CONF}"

# ── Step 8: Stop default nginx ──
echo "Step 8: Stopping default nginx if running..."
ssh "${TARGET_USER}@${TARGET_HOST}" "
if sudo systemctl is-active --quiet nginx; then
  echo 'Stopping default nginx service...'
  sudo systemctl stop nginx
  sudo systemctl disable nginx
  echo 'Default nginx stopped and disabled.'
else
  echo 'Default nginx not running.'
fi
# Kill anything else on port 80
if sudo ss -tlnp 2>/dev/null | grep -q ':80 '; then
  echo 'Something is on port 80, killing...'
  sudo fuser -k 80/tcp 2>/dev/null || true
  sleep 1
fi
"

# ── Step 9: Start nginx-local ──
echo "Step 9: Starting nginx-local..."
ssh "${TARGET_USER}@${TARGET_HOST}" "
sudo systemctl daemon-reload
sudo systemctl stop nginx-local 2>/dev/null || true
sudo systemctl start nginx-local
sudo systemctl enable nginx-local
"

# ── Step 10: Verify ──
echo ""
echo "Step 10: Verifying..."
sleep 2
REMOTE_STATUS=$(ssh "${TARGET_USER}@${TARGET_HOST}" "sudo systemctl is-active nginx-local" 2>&1)
echo "nginx-local status: ${REMOTE_STATUS}"

if [[ "$REMOTE_STATUS" == "active" ]]; then
  echo ""
  echo "=== nginx-local is running on ${TARGET_HOST} ==="
  echo ""
  echo "Service addresses (from any LAN device):"
  echo "  http://nc.lan        Nextcloud"
  echo "  http://ha.lan        Home Assistant"
  echo "  http://auth.lan      Auth (Dex)"
  echo "  http://usrmgr.lan    User Manager"
  echo "  http://search.lan    SearXNG Search"
  echo "  http://steam.lan     Steam Desktop"
  echo "  http://pgadmin.lan   pgAdmin"
  echo "  http://registry.lan  Container Registry"
  echo "  http://regui.lan     Registry UI"
  echo "  http://fnf.lan       FNF"
  echo "  http://llmll.lan     LLMll Server"
  echo ""
  echo "Service directory: http://lsnode-0.lan  (or http://${TARGET_HOST})"
  echo ""
  echo "DNS is handled by the router (${ROUTER_HOST})."
  echo "All LAN devices resolve .lan hostnames automatically."
  echo "Ad blocking and parental controls are preserved."
  echo ""
else
  echo "ERROR: nginx-local failed to start. Check logs:"
  ssh "${TARGET_USER}@${TARGET_HOST}" "sudo journalctl -u nginx-local --no-pager -n 30"
  exit 1
fi

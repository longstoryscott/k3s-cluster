# nginx-local — Central LAN Reverse Proxy

A single nginx instance on `lsnode-0` that proxies all k3s services to short, memorable hostnames.

## What It Does

```
Browser → http://nc.lan  →  lsnode-0:80  →  nginx  →  gateway:3000  →  nextcloud pod
Browser → http://ha.lan  →  lsnode-0:80  →  nginx  →  gateway:8123  →  home-assistant pod
...
```

nginx sits in front of the in-cluster NGINX Gateway and routes based on the `Host` header. DNS is handled by dnsmasq on the router (192.168.0.1).

## Service Address Map

| Address | Service | Gateway Port |
|---|---|---|
| `http://nc.lan` | Nextcloud | 3000 |
| `http://ha.lan` | Home Assistant | 8123 |
| `http://auth.lan` | Dex OIDC | 9091 |
| `http://usrmgr.lan` | User Manager | 3333 |
| `http://registry.lan` | Container Registry | 5000 |
| `http://regui.lan` | Registry UI | 8085 |
| `http://fnf.lan` | FNF | 8082 |
| `http://pgadmin.lan` | pgAdmin | 8084 |
| `http://search.lan` | SearXNG | 8086 |
| `http://steam.lan` | Steam Desktop (WebSocket) | 4567 |
| `http://llmll.lan` | LLMll Server | 8001 |

Hitting `http://lsnode-0.lan` (or the raw IP) shows a service directory with links to everything.
Bare hostnames (e.g. `http://nc`) also work for convenience.

## Install

```bash
cd k3s-cluster/nginx-local
chmod +x install.sh
./install.sh
```

The script:
1. Installs nginx on lsnode-0
2. Deploys the reverse proxy config
3. Installs dnsmasq for DNS resolution
4. Fixes NSS so DNS resolves before mDNS
5. Creates a systemd service (`nginx-local`)
6. Configures router DNS for LAN-wide resolution

## DNS on All LAN Devices

DNS resolution is handled by the router's dnsmasq (GL.iNet Flint 3 / OpenWrt).

**Router config** (persistent, survives reboot):
- Config file: `/etc/dnsmasq.d/local-services.conf` (on overlay filesystem)
- UCI setting: `dhcp.@dnsmasq[0].confdir='/etc/dnsmasq.d'`
- The install script configures this automatically using router credentials from `../.env`

**DNS chain preserved**:
```
Clients → Router dnsmasq → AdGuard Home (port 3053) → NextDNS (75.75.75.75)
              ↑
    .lan hostnames resolved here
    (all other queries forwarded as usual)
```

This means:
- Ad blocking continues to work
- Parental controls continue to work
- Local services are resolved at the first hop (fastest possible)
- No device configuration needed — router handles DHCP DNS for all clients

## Adding a New Service

1. **Add a gateway listener** in `k3s-cluster/router/routes.yaml` (pick an unused port).
2. **Add an upstream** in `nginx.conf`:
   ```
   upstream my_svc { server 127.0.0.1:<port>; }
   ```
3. **Add a server block** in `nginx.conf`:
   ```
   server {
       listen 80;
       server_name myname.lan myname;
       location / { proxy_pass http://my_svc; }
   }
   ```
4. **Add a dnsmasq entry** in `dnsmasq.conf`:
   ```
   address=/myname.lan/192.168.0.71
   ```
5. **Update router DNS** (or run `./install.sh` to sync automatically):
   ```bash
   ssh root@192.168.0.1 "echo 'address=/myname.lan/192.168.0.71' >> /etc/dnsmasq.d/local-services.conf"
   ssh root@192.168.0.1 "/etc/init.d/dnsmasq restart"
   ```
6. **Reload nginx**:
   ```bash
   ssh lsm@192.168.0.71 "sudo systemctl reload nginx-local"
   ```

## Operations

### View logs
```bash
ssh lsm@192.168.0.71 "sudo journalctl -u nginx-local -f --no-pager"
```

### Reload config (no downtime)
```bash
ssh lsm@192.168.0.71 "sudo systemctl reload nginx-local"
```

### Check status
```bash
ssh lsm@192.168.0.71 "sudo systemctl status nginx-local dnsmasq"
```

### Uninstall
```bash
ssh lsm@192.168.0.71 "sudo systemctl disable --now nginx-local; sudo rm /etc/systemd/system/nginx-local.service /opt/nginx-local/nginx.conf /etc/dnsmasq.d/local-services"
```

## Architecture Notes

- **Systemd service** on lsnode-0, not in k3s — avoids image pull issues and keeps port 80 stable.
- **No TLS** — LAN-only. For external access, use the edge reverse proxy.
- **WebSocket support** — `steam.lan` has full WebSocket upgrade headers for the noVNC desktop.
- **CORS** — `auth.lan` and `usrmgr.lan` include CORS headers matching the existing gateway config.
- **Service directory** — hitting `http://lsnode-0.lan` returns a styled HTML page linking to all services.
- **Bare hostname fallback** — nginx `server_name` includes both `.lan` and bare forms (e.g. `nc.lan nc`).
- **Router DNS** — all `.lan` entries configured on the router's dnsmasq via persistent overlay filesystem.
- **DNS chain preserved** — lsnode-0 is NOT the DNS server; the router handles DNS, preserving AdGuard Home and parental controls.

## File Reference

| File | Purpose |
|------|---------|
| `nginx.conf` | nginx reverse proxy config |
| `dnsmasq.conf` | dnsmasq hostname→IP mappings (for lsnode-0) |
| `install.sh` | One-command deployment |
| `README.md` | This file |

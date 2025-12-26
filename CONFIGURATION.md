# Configuration Guide

## Initial Setup

1. **Copy the configuration template:**
   ```bash
   cp config.env.example config.env
   ```

2. **Edit `config.env` with your values:**
   ```bash
   # Required settings
   DOMAIN_BASE="yourdomain.com"
   MASTER_NODE_IP="192.168.x.x"
   
   # Optional: Customize ports, trusted IPs, etc.
   ```

3. **Create service-specific secrets:**

   The installation scripts will automatically generate passwords for services that need them:
   - `monitoring/grafana-password.txt` - Grafana admin password
   - `rd/vnc-password.txt` - VNC desktop password
   - `registry/.secrets/` - Registry credentials
   - `.secrets/` - Other service secrets

4. **For LDAP/Auth services (optional):**
   ```bash
   # Copy and customize LDIF files
   cp auth/ldap-users.ldif.example auth/ldap-users.ldif
   cp auth/ldap-groups.ldif.example auth/ldap-groups.ldif
   
   # Generate password hashes
   slappasswd -h {SSHA}
   ```

## What Gets Generated vs What You Provide

### Auto-Generated (by install scripts)
- All service passwords (Grafana, VNC, etc.)
- Registry credentials
- Database passwords
- Secret keys and tokens

### You Must Provide
- Domain names in `config.env`
- Network IP addresses in `config.env`
- LDAP user data in `auth/ldap-*.ldif` (if using auth service)

## Security Notes

**Never commit these files:**
- `config.env` - Your actual configuration
- `*.ldif` (except `.example` files) - LDAP user data
- `**/grafana-password.txt` - Service passwords
- `**/vnc-password.txt` - Service passwords
- `.secrets/**` - Generated secrets
- `*.sql` - Database dumps

These are all in `.gitignore` to prevent accidental commits.

## Finding Your Passwords

After running install scripts, credentials are stored in:

```bash
# Grafana
cat monitoring/grafana-password.txt

# VNC Desktop
cat rd/vnc-password.txt

# Registry
cat registry/.secrets/registryuser
cat registry/.secrets/registrypw
```

## Configuration File Locations

| Service | Config File | Purpose |
|---------|-------------|---------|
| Global | `config.env` | Domain names, IPs, ports |
| Auth | `auth/ldap-users.ldif` | LDAP user definitions |
| Auth | `auth/ldap-groups.ldif` | LDAP group definitions |
| Monitoring | `monitoring/grafana-password.txt` | Grafana admin password |
| Desktop | `rd/vnc-password.txt` | VNC access password |
| Registry | `registry/.secrets/` | Registry credentials |

## Example Workflow

```bash
# 1. Set up configuration
cp config.env.example config.env
nano config.env  # Edit with your values

# 2. Deploy services (passwords auto-generated)
make monitoring
make registry
make ollama

# 3. Check your credentials
cat monitoring/grafana-password.txt
cat registry/.secrets/registryuser

# 4. Access services using the credentials
```

## Troubleshooting

**Problem:** Service can't find password file
- **Solution:** Run the install script for that service - it generates the password

**Problem:** Need to reset a password
- **Solution:** Delete the password file and re-run the install script:
  ```bash
  rm monitoring/grafana-password.txt
  make monitoring
  ```

**Problem:** Domain names not working
- **Solution:** Check `config.env` has correct values and restart affected services

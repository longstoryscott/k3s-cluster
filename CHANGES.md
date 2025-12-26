# Security and Configuration Updates

This document tracks the refactoring of sensitive data handling in the k3s-cluster project.

## Changes Made

### 1. Configuration System
- ✅ Created `config.env.example` template for domain names, IPs, and ports
- ✅ Updated `.gitignore` to exclude `config.env`, `*.sql`, `*.ldif`, and password files
- ✅ Added helper functions in `helpers.sh` for config loading and password generation

### 2. Password Management
- ✅ **Grafana**: Moved from hardcoded `6u!tar00!QAZ` to auto-generated `monitoring/grafana-password.txt`
  - Updated `monitoring/install.sh` to generate password
  - Updated `monitoring/grafana-values.yaml` to remove hardcoded value
  - Updated `monitoring/import-dashboards.sh` to read from file
  
- ✅ **VNC Desktop**: Moved from hardcoded `mypassword` to auto-generated `rd/vnc-password.txt`
  - Updated `rd/install.sh` to generate password and create K8s secret
  - Updated `rd/desktop-statefulset.yaml` to use secret reference
  - Updated `rd/README.md` to remove hardcoded password

### 3. LDAP/Auth Data
- ✅ Moved `auth/users.ldif` and `auth/groups.ldif` to gitignored names
- ✅ Created `auth/ldap-users.ldif.example` and `auth/ldap-groups.ldif.example` templates

### 4. Database Dumps
- ✅ Removed all `*.sql` files from repository
- ✅ Added `*.sql` to `.gitignore`

### 5. Documentation
- ✅ Created `CONFIGURATION.md` - Setup guide for new users
- ✅ Created `SECURITY.md` - Security checklist and migration guide

## Still Using Placeholders

Some files still contain real values and should be updated to use `config.env`:

### Domain Names
Files with `auth.longstorymedia.com`, `lsnet.longstorymedia.com`:
- `router/routes.yaml` - Gateway routes
- `nc/configmap.yaml` - NextCloud trusted domains
- `nc/deployment.yaml` - NextCloud environment variables
- `auth/config.yaml` - Dex OIDC issuer
- `auth/nginx.conf` - Server names

### IP Addresses  
Files with `192.168.0.71` and other private IPs:
- Multiple service deployments reference `192.168.0.71:31500` for registry
- Network configurations with trusted proxy IPs

## Recommended Next Steps

1. **Template remaining configs** - Replace hardcoded domains/IPs with placeholders:
   ```yaml
   # Instead of:
   server_name: auth.longstorymedia.com
   
   # Use:
   server_name: {{AUTH_DOMAIN}}
   ```

2. **Create substitution step** - Add config substitution to install scripts:
   ```bash
   source helpers.sh
   load_config
   envsubst < template.yaml > applied.yaml
   ```

3. **Clean git history** - If previously committed secrets, use `git-filter-repo`

4. **Add pre-commit hook** - Prevent accidental secret commits:
   ```bash
   # .git/hooks/pre-commit
   if git diff --cached | grep -i "password.*=.*[^{]"; then
     echo "Error: Found hardcoded password!"
     exit 1
   fi
   ```

## Migration Guide for Existing Deployments

If you've already deployed services with hardcoded values:

1. Generate new passwords:
   ```bash
   ./monitoring/install.sh monitoring
   ./rd/install.sh
   ```

2. Update existing secrets:
   ```bash
   kubectl delete secret vnc-password -n steam
   # Re-run install script to recreate with new password
   ```

3. Verify no hardcoded values remain:
   ```bash
   git grep -i "password.*=" -- '*.yaml' '*.sh'
   ```

## Testing

Before pushing to public repo:

```bash
# 1. Clone to a new directory (simulates fresh clone)
cd /tmp
git clone /path/to/k3s-cluster test-clone
cd test-clone

# 2. Verify sensitive files don't exist
ls *.sql                    # Should fail
ls config.env              # Should fail  
ls auth/users.ldif         # Should fail

# 3. Verify templates exist
ls config.env.example      # Should succeed
ls auth/*.example          # Should succeed

# 4. Follow CONFIGURATION.md to set up
cp config.env.example config.env
# Edit config.env...

# 5. Run install and verify it works
make monitoring
```

## Files Modified

- `.gitignore`
- `helpers.sh`
- `monitoring/install.sh`
- `monitoring/grafana-values.yaml`
- `monitoring/import-dashboards.sh`
- `rd/install.sh`
- `rd/desktop-statefulset.yaml`
- `rd/README.md`

## Files Created

- `config.env.example`
- `CONFIGURATION.md`
- `SECURITY.md`
- `CHANGES.md` (this file)
- `substitute-config.sh`
- `auth/ldap-users.ldif.example`
- `auth/ldap-groups.ldif.example`

## Files Deleted

- `llmmll.2025.06.05.sql`
- `llmmll.2025.07.14.sql`
- `nc.2025.05.27.sql`
- `nextcloud_dump.sql`

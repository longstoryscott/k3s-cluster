# Making Your k3s Cluster Public-Safe

This project has been refactored to separate sensitive data from configuration files.

## ⚠️ Before You Push

**Run this checklist:**

```bash
# 1. Ensure you have config.env (not tracked)
[ -f config.env ] && echo "✓ config.env exists" || echo "✗ Create config.env from template"

# 2. Check for SQL dumps
ls *.sql 2>/dev/null && echo "✗ Remove SQL dumps!" || echo "✓ No SQL dumps"

# 3. Check for sensitive LDIF files
ls auth/*.ldif 2>/dev/null | grep -v example && echo "✗ LDIF files found!" || echo "✓ No tracked LDIF"

# 4. Verify gitignore is working
git status --ignored | grep -E "(config.env|\.sql|password\.txt)" && echo "✓ Sensitive files ignored" || echo "⚠️  Check .gitignore"

# 5. Check git history for secrets
git log --all --full-history -- "*.sql" ".secrets/*" "*.ldif" | head -5
```

## What's Safe to Commit

✅ **Safe:**
- `config.env.example` - Template with placeholders
- `*.example` files - Example configurations
- All deployment YAML files (now use secrets/configmaps)
- Install scripts (now generate passwords dynamically)
- Documentation

❌ **Never Commit:**
- `config.env` - Your actual configuration
- `*.sql` files - Database dumps
- `*.ldif` (except `.example`) - LDAP user data  
- `*password.txt` - Generated passwords
- `.secrets/` directory - Generated secrets



## Configuration Files Created

| File | Purpose | Tracked? |
|------|---------|----------|
| `config.env.example` | Template | ✅ Yes |
| `config.env` | Your values | ❌ No |
| `monitoring/grafana-password.txt` | Auto-generated | ❌ No |
| `rd/vnc-password.txt` | Auto-generated | ❌ No |
| `auth/ldap-*.ldif.example` | Templates | ✅ Yes |
| `auth/ldap-*.ldif` | Your LDAP data | ❌ No |
| `.secrets/` | Service secrets | ❌ No |

## If You've Already Committed Secrets

If your git history contains secrets, you need to clean it:

```bash
# Option 1: Use git-filter-repo (recommended)
pip install git-filter-repo
git filter-repo --path '*.sql' --invert-paths
git filter-repo --path 'auth/users.ldif' --invert-paths
git filter-repo --path 'auth/groups.ldif' --invert-paths

# Option 2: Start fresh (nuclear option)
# Create new empty repo and copy only safe files
```

## Setup for New Users

Anyone cloning your public repo will:

1. Copy `config.env.example` to `config.env`
2. Edit with their values
3. Run `make install` or individual service installs
4. Scripts automatically generate all passwords

See [CONFIGURATION.md](CONFIGURATION.md) for detailed setup instructions.

## Quick Security Audit

```bash
# Search for potential secrets in tracked files
git grep -i "password.*=" -- '*.yaml' '*.sh'
git grep -E "\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}" -- '*.yaml'
git grep -E "\.com|\.org" -- '*.yaml' | grep -v example
```

If these return matches with real values, they need to be templatized.

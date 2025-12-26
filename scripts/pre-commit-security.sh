#!/bin/bash
set -euo pipefail

# Pre-commit security check
# Install: ln -s ../../scripts/pre-commit-security.sh .git/hooks/pre-commit

echo "🔒 Running security checks..."

# Check for hardcoded passwords (only in added lines, excluding comments/examples)
# Look for actual secret assignments like: password: "realvalue" or PASSWORD="realvalue"
# Exclude template vars ({{, ${), example placeholders, and comment lines
SUSPICIOUS=""
current_file=""
current_line=0

while IFS= read -r line; do
    # Track filename from diff header
    if [[ "$line" =~ ^\+\+\+\ b/(.*) ]]; then
        current_file="${BASH_REMATCH[1]}"
        continue
    fi
    # Track line number from hunk header (@@ -X,Y +LINE,COUNT @@)
    if [[ "$line" =~ ^@@.*\+([0-9]+) ]]; then
        current_line="${BASH_REMATCH[1]}"
        continue
    fi
    # Check added lines (starting with + but not +++)
    if [[ "$line" =~ ^\+[^\+] ]] || [[ "$line" == "+" ]]; then
        content="${line:1}"  # Strip leading +
        # Check if line contains potential secrets
        if echo "$content" | grep -qiE '(password|secret|api_?key|auth_?token)["\047]?\s*[:=]\s*["\047]'; then
            # Exclude safe patterns:
            # - example/placeholder values
            # - template vars: {{VAR}}, ${VAR}
            # - shell vars in quotes: ="$VAR", ='$VAR' (value is a variable reference)
            if ! echo "$content" | grep -qE '(example|placeholder|changeme|xxx|your[-_]|<.*>|\{\{|\$\{|[:=]\s*["\047]\$)'; then
                # Exclude comment lines
                if ! echo "$content" | grep -qE '^\s*#'; then
                    SUSPICIOUS+="  $current_file:$current_line: $content"$'\n'
                fi
            fi
        fi
        ((current_line++))
    fi
done < <(git diff --cached --diff-filter=ACM -U0)

if [ -n "$SUSPICIOUS" ]; then
    echo "❌ Error: Found potential hardcoded secret in staged files!"
    echo "   Secrets should be in gitignored files or use placeholders like {{PASSWORD}}"
    echo "   Suspicious lines:"
    echo "$SUSPICIOUS"
    exit 1
fi

# Check for SQL dump files (large data exports, not schema/migrations)
# Block: dump, backup, export files. "data" excluded - too many false positives (metadata, userdata, etc)
SQL_DUMPS=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.sql$' | grep -iE '(dump|backup|export)' || true)
if [ -n "$SQL_DUMPS" ]; then
    echo "❌ Error: SQL dump file in staged changes!"
    echo "   Database dumps should not be committed. Add to .gitignore."
    echo "   Files: $SQL_DUMPS"
    exit 1
fi

# Check for LDIF files (except examples)
LDIF_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.ldif$' | grep -v example || true)
if [ -n "$LDIF_FILES" ]; then
    echo "❌ Error: LDIF file in staged changes!"
    echo "   LDIF files may contain sensitive user data. Only commit .example files."
    echo "   Files: $LDIF_FILES"
    exit 1
fi

# Check for config.env and other env files (except examples)
ENV_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.env$' | grep -vE '(\.example$|\.env\.example)' || true)
if [ -n "$ENV_FILES" ]; then
    echo "❌ Error: Environment file in staged changes!"
    echo "   Env files may contain secrets. Only commit .example versions."
    echo "   Files: $ENV_FILES"
    exit 1
fi

# Check for private key files
KEY_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -iE '\.(pem|key|p12|pfx)$' || true)
if [ -n "$KEY_FILES" ]; then
    echo "❌ Error: Private key file in staged changes!"
    echo "   Private keys should never be committed. Add to .gitignore."
    echo "   Files: $KEY_FILES"
    exit 1
fi

# Check for password files
SECRET_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -iE 'password\.txt|\.secrets/' || true)
if [ -n "$SECRET_FILES" ]; then
    echo "❌ Error: Password/secret file in staged changes!"
    echo "   These files should be in .gitignore"
    echo "   Files: $SECRET_FILES"
    exit 1
fi

echo "✅ Security checks passed"
exit 0

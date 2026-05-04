# Thunder Interactive Resource Setup Script

> **Compatibility:** Thunder IDP 0.35.0+

## Overview

`setup.sh` is an interactive Bash script that bootstraps the Thunder Identity & Access Management (IAM) platform with sample resources. It creates organization units, user types, groups, roles, users, and applications through a series of guided API calls.

The script is **POSIX-compatible** and runs on macOS Bash 3.x+, newer Bash (4.x+), and standard Linux shells. It supports both **interactive** mode (with per-step confirmations) and **auto-run** mode (continuous with error prompts).

---

## Prerequisites

- **Bash** 3.x or later (macOS default or Linux standard)
- **curl** with TLS support
- **Python 3** (optional, for pretty-printed JSON output)
- **Thunder IDP instance** running and accessible at `THUNDER_BASE_URL`
- **Valid access token** with system/admin privileges

---

## Configuration

### Environment Variables

Create a `.env` file in the same directory as `setup.sh` (or copy from `.env.example` as a template), or export variables before running:

```bash
# Required
THUNDER_BASE_URL=https://localhost:8090          # Base URL of Thunder IDP
THUNDER_ACCESS_TOKEN=<jwt-token>                 # Admin/system access token

# Optional: User credentials (defaults to "1234" if not set)
THUNDER_SAMPLE_USER_PASSWORD=mypassword
THUNDER_SAMPLE_USER123_PASSWORD=user123pwd
THUNDER_SAMPLE_USER456_PASSWORD=user456pwd
THUNDER_SAMPLE_USER789_PASSWORD=user789pwd
THUNDER_SAMPLE_NPQS_USER_PASSWORD=npqspwd
THUNDER_SAMPLE_FCAU_USER_PASSWORD=fcaupwd
THUNDER_SAMPLE_IRD_USER_PASSWORD=irdpwd
THUNDER_SAMPLE_CDA_USER_PASSWORD=cdapwd

# Optional: M2M client secrets (defaults to "1234" if not set)
THUNDER_M2M_CLIENT_SECRET=m2msecret
THUNDER_M2M_NPQS_SECRET=npqsm2msecret
THUNDER_M2M_FCAU_SECRET=fcaum2msecret
THUNDER_M2M_IRD_SECRET=irdm2msecret
THUNDER_M2M_CDA_SECRET=cdam2msecret

# Optional: TLS verification (set to 1 to skip on self-signed certs)
THUNDER_INSECURE=0
```

#### Getting Started with `.env`

Copy the provided `.env.example` template:

```bash
cp .env.example .env
# Edit .env and fill in your Thunder instance details
```

Sample `.env`:
```bash
THUNDER_BASE_URL=https://localhost:8090
THUNDER_ACCESS_TOKEN=eyJhbGci...
THUNDER_INSECURE=1
THUNDER_SAMPLE_USER_PASSWORD=SecurePassword123
```

---

## Usage

### Interactive Mode (Step-by-Step)

Run the script without `--auto` to prompt for each step:

```bash
./setup.sh
```

**Output:**
```
Step 1: Create Private Sector OU
  POST https://localhost:8090/organization-units
  Body: {...}

  [R]un  [S]kip  [Q]uit  > _
```

At each prompt, enter:
- `r` or `R` or press Enter → Run this step
- `s` or `S` → Skip this step
- `q` or `Q` → Abort script

### Auto-Run Mode (Continuous)

Run with `--auto` to execute all steps automatically; script will prompt only on errors:

```bash
./setup.sh --auto
```

**On error:**
```
Step 1: Create Private Sector OU — failed (HTTP 400).
  Step failed. [R]etry  [S]kip  [Q]uit  > _
```

Choose:
- `r` or `R` → Retry the failed step
- `s` or `S` → Skip and continue
- `q` or `Q` → Abort

### Help

```bash
./setup.sh --help
```

---

## TLS and HTTPS

By default, the script validates TLS certificates. If your Thunder instance uses a **self-signed certificate**, you must either:

### Option 1: Skip verification (development only)

```bash
THUNDER_INSECURE=1 ./setup.sh --auto
```

Or add to `.env`:
```
THUNDER_INSECURE=1
```

### Option 2: Trust the certificate on macOS

1. Export the server certificate:
   ```bash
   openssl s_client -connect localhost:8090 -showcerts < /dev/null | \
     sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > /tmp/thunder.crt
   ```

2. Add to macOS keychain:
   ```bash
   security add-certificates -k ~/Library/Keychains/login.keychain /tmp/thunder.crt
   ```

---

## Resources Created

The script creates the following Thunder IAM resources:

### Organization Units (OUs)

- **private-sector** (root) → **abcd-traders** (child)
- **government-organization** (root) → **npqs**, **fcau**, **ird**, **cda** (children)

### User Types (Schemas)

- **Private_User** (email, phone, username, password)
- **Government_User** (email, phone, username, password)

### Groups

- **Traders** (ABCD Traders OU)
- **CHA** (ABCD Traders OU)

### Roles

- **Trader** (Private Sector OU)
- **CHA** (Private Sector OU)

### Users

| Username | Type | OU | Groups | Password Var |
|----------|------|----|------------|------|
| user123 | Private_User | ABCD Traders | Traders, CHA | `THUNDER_SAMPLE_USER123_PASSWORD` |
| user456 | Private_User | ABCD Traders | CHA | `THUNDER_SAMPLE_USER456_PASSWORD` |
| user789 | Private_User | ABCD Traders | Traders | `THUNDER_SAMPLE_USER789_PASSWORD` |
| npqs_user | Government_User | NPQS | — | `THUNDER_SAMPLE_NPQS_USER_PASSWORD` |
| fcau_user | Government_User | FCAU | — | `THUNDER_SAMPLE_FCAU_USER_PASSWORD` |
| ird_user | Government_User | IRD | — | `THUNDER_SAMPLE_IRD_USER_PASSWORD` |
| cda_user | Government_User | CDA | — | `THUNDER_SAMPLE_CDA_USER_PASSWORD` |

### Applications

#### SPA Applications (OAuth 2.0 / PKCE)

- **TraderApp** (Private User, port 5173)
- **NPQSPortalApp** (Government User, port 5174)
- **FCAUPortalApp** (Government User, port 5175)
- **IRDPortalApp** (Government User, port 5176)
- **CDAPortalApp** (Government User, port 5177)

#### M2M Applications (Client Credentials)

- **NPQS_TO_NSW_M2M** (NPQS → NSW integration)
- **FCAU_TO_NSW_M2M** (FCAU → NSW integration)
- **IRD_TO_NSW_M2M** (IRD → NSW integration)
- **CDA_TO_NSW_M2M** (CDA → NSW integration)

---

## Examples

### Example 1: Quick Setup (Auto-Run with Self-Signed Cert)

```bash
export THUNDER_BASE_URL=https://localhost:8090
export THUNDER_ACCESS_TOKEN="<your-jwt-token>"
export THUNDER_INSECURE=1
export THUNDER_SAMPLE_USER_PASSWORD="MySecurePass123"

./setup.sh --auto
```

### Example 2: Interactive Setup with .env File

```bash
# Create .env
cat > .env <<EOF
THUNDER_BASE_URL=https://thunderidp.example.com
THUNDER_ACCESS_TOKEN=eyJhbGciOiJSUzI1NiIs...
THUNDER_SAMPLE_USER_PASSWORD=Secure123!@#
THUNDER_INSECURE=0
EOF

# Run interactively
./setup.sh
```

### Example 3: Dry-Run (Review All Steps Without Creating)

```bash
export THUNDER_INSECURE=1
./setup.sh    # Select [S]kip for every step to see all without creating
```

---

## Troubleshooting

### ❌ Curl Exit 60: Certificate Verification Failed

**Cause:** TLS certificate validation failed (self-signed cert on localhost).

**Solution:**
```bash
THUNDER_INSECURE=1 ./setup.sh --auto
```

Or set in `.env`:
```
THUNDER_INSECURE=1
```

### ❌ HTTP 401 Unauthorized

**Cause:** Invalid or expired `THUNDER_ACCESS_TOKEN`.

**Solution:** Obtain a new admin token from your Thunder instance:
```bash
curl https://localhost:8090/oauth2/token -X POST \
  -d 'grant_type=password&username=admin&password=...&client_id=CONSOLE&client_secret=...'
```

### ❌ HTTP 409 Conflict: Resource Already Exists

**Cause:** The resource (e.g., organization unit) has already been created.

**Solution:** The script automatically skips and fetches the existing resource ID via fallback GET. No action needed; continue running.

### ❌ HTTP 400 Bad Request

**Cause:** Malformed payload or validation error. Check the response body logged by the script.

**Solution:** Review the JSON body in the step output; verify field types (e.g., UUID format for parent OUs).

### ❌ Bash: Command Not Found: declare -A

**Cause:** Running under an old Bash that doesn't support associative arrays (pre-Bash 4).

**Solution:** This script is refactored to use POSIX-compatible indexed arrays. Upgrade Bash or use your system's default Bash.

---

## Output & Logging

### Summary Report

After all steps, the script prints:

```
━━━  Summary  ━━━
  Passed : 42
  Skipped: 3
  Failed : 0
  Total  : 45

Resolved IDs:
  ABCD_TRADERS_OU_ID              019df0c8-e1f7-cbdd-bff3-53e676c2d72c
  CHA_GROUP_ID                    019df0c9-2f3a-7c4d-8e5f-8a9b7c6d5e4f
  ...
```

### Debug Mode

To see verbose curl output, modify the `thunder_call()` function to remove `-s` (silent) flag.

---

## Architecture Notes

- **Variable Storage:** Script uses POSIX-compatible indexed arrays with sanitized variable names (no Bash 4+ associative arrays).
- **Error Handling:** `set -euo pipefail` enables strict error mode; curl failures are captured and logged.
- **JSON Processing:** Uses `grep`, `cut`, `sed` for parsing (no external dependencies like `jq`).
- **Retry Logic:** Failed HTTP calls can be retried, skipped, or aborted per step.

---

## Advanced: Customizing Resource Creation

To modify which resources are created, edit the `run_step` calls in the MAIN section (line ~340+).

Example: Skip creating M2M apps by commenting out:
```bash
# Commented out: don't create M2M apps
# run_step "Create NPQS_TO_NSW M2M app" POST "/applications" \
#     "$(m2m_body ...)"
```

---

## Support & Contributing

For issues or improvements, refer to the inline comments in `setup.sh` or contact your Thunder IAM administrator.

---

## License

See LICENSE file in the repository root.

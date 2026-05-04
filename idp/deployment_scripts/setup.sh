#!/usr/bin/env bash
set -euo pipefail

# ============================================================================
# Thunder Interactive Resource Setup Script
# Compatibility: Thunder IDP 0.35.0+
# 
# Reads THUNDER_BASE_URL and THUNDER_ACCESS_TOKEN from .env
# Runs each API call step-by-step with confirmation prompts
# ============================================================================
#
# Usage:
#   Interactive mode:   ./setup.sh
#   Auto-run mode:      ./setup.sh --auto
#   Help:              ./setup.sh --help
#
# See README.md for detailed documentation.
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

# CLI flags
AUTO_RUN=0
while [[ $# -gt 0 ]]; do
    case "$1" in
        --auto|-a) AUTO_RUN=1; shift ;;
        --help|-h) echo "Usage: $0 [--auto]"; exit 0 ;;
        *) shift ;;
    esac
done

# ── Colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${RESET}  $*"; }
log_success() { echo -e "${GREEN}[OK]${RESET}    $*"; }
log_warning() { echo -e "${YELLOW}[WARN]${RESET}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${RESET} $*" >&2; }
log_step()    { echo -e "\n${BOLD}${CYAN}━━━  $*  ━━━${RESET}"; }

# ── Load .env ─────────────────────────────────────────────────────────────────
ENV_FILE="${SCRIPT_DIR}/.env"
if [[ -f "$ENV_FILE" ]]; then
    set -a; source "$ENV_FILE"; set +a
    log_info "Loaded .env from ${ENV_FILE}"
else
    log_warning ".env file not found at ${ENV_FILE}; relying on exported environment variables"
fi

THUNDER_BASE_URL="${THUNDER_BASE_URL:-}"
THUNDER_ACCESS_TOKEN="${THUNDER_ACCESS_TOKEN:-}"

if [[ -z "$THUNDER_BASE_URL" ]]; then
    log_error "THUNDER_BASE_URL is not set. Add it to .env or export it."
    exit 1
fi
if [[ -z "$THUNDER_ACCESS_TOKEN" ]]; then
    log_error "THUNDER_ACCESS_TOKEN is not set. Add it to .env or export it."
    exit 1
fi

THUNDER_BASE_URL="${THUNDER_BASE_URL%/}"   # strip trailing slash

# ── Passwords / secrets (with defaults) ──────────────────────────────────────
SAMPLE_PW="${THUNDER_SAMPLE_USER_PASSWORD:-1234}"
USER123_PASSWORD="${THUNDER_SAMPLE_USER123_PASSWORD:-${SAMPLE_PW}}"
USER456_PASSWORD="${THUNDER_SAMPLE_USER456_PASSWORD:-${SAMPLE_PW}}"
USER789_PASSWORD="${THUNDER_SAMPLE_USER789_PASSWORD:-${SAMPLE_PW}}"
NPQS_USER_PASSWORD="${THUNDER_SAMPLE_NPQS_USER_PASSWORD:-${SAMPLE_PW}}"
FCAU_USER_PASSWORD="${THUNDER_SAMPLE_FCAU_USER_PASSWORD:-${SAMPLE_PW}}"
IRD_USER_PASSWORD="${THUNDER_SAMPLE_IRD_USER_PASSWORD:-${SAMPLE_PW}}"
CDA_USER_PASSWORD="${THUNDER_SAMPLE_CDA_USER_PASSWORD:-${SAMPLE_PW}}"
M2M_SECRET="${THUNDER_M2M_CLIENT_SECRET:-1234}"
NPQS_M2M_SECRET="${THUNDER_M2M_NPQS_SECRET:-${M2M_SECRET}}"
FCAU_M2M_SECRET="${THUNDER_M2M_FCAU_SECRET:-${M2M_SECRET}}"
IRD_M2M_SECRET="${THUNDER_M2M_IRD_SECRET:-${M2M_SECRET}}"
CDA_M2M_SECRET="${THUNDER_M2M_CDA_SECRET:-${M2M_SECRET}}"

# ── Runtime variable store (POSIX-friendly) ──────────────────────────────────
# Use an indexed array of keys plus per-key named variables so the script
# works on macOS's older bash (no associative arrays) and on newer bash.
declare -a VAR_KEYS=()   # ordered list of keys
STEP_INDEX=0
STEP_PASS=0
STEP_SKIP=0
STEP_FAIL=0

# Sanitize a key into a safe variable suffix (alnum -> keep, others -> _)
sanitize_varname() { echo "$1" | tr -c '[:alnum:]' '_' ; }

set_var() {
    local key="$1" val="$2" varname
    varname="VARS_$(sanitize_varname "$key")"
    eval "$varname=\"\$val\""
    for existing in "${VAR_KEYS[@]:-}"; do
        [[ "$existing" == "$key" ]] && return 0
    done
    VAR_KEYS+=("$key")
}

get_var() {
    local key="$1" varname
    varname="VARS_$(sanitize_varname "$key")"
    eval 'echo "${'"$varname"':-}"'
}

# ── Core helpers ──────────────────────────────────────────────────────────────
extract_first_id() {
    echo "$1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

# Pretty-print JSON if python3 available, else raw
pretty_json() {
    if command -v python3 &>/dev/null; then
        echo "$1" | python3 -m json.tool 2>/dev/null || echo "$1"
    else
        echo "$1"
    fi
}

# ── Prompt helper ─────────────────────────────────────────────────────────────
# Usage: confirm_step "Step title" "METHOD" "/path" '{"json":"body"}'
# Returns 0 to proceed, 1 to skip
confirm_step() {
    local TITLE="$1" METHOD="$2" PATH_="$3" BODY="${4:-}"
    (( STEP_INDEX++ )) || true

    echo ""
    echo -e "${BOLD}Step ${STEP_INDEX}:${RESET} ${TITLE}"
    echo -e "  ${CYAN}${METHOD}${RESET} ${THUNDER_BASE_URL}${PATH_}"
    if [[ -n "$BODY" ]]; then
        echo -e "  ${YELLOW}Body:${RESET}"
        pretty_json "$BODY" | sed 's/^/    /'
    fi
    echo ""
    echo ""

    if [[ "$AUTO_RUN" == "1" ]]; then
        log_info "Auto-run enabled; running step."
        return 0
    fi

    while true; do
        read -r -p "  [R]un  [S]kip  [Q]uit  > " CHOICE
        CHOICE_LOWER=$(echo "${CHOICE}" | tr '[:upper:]' '[:lower:]')
        case "$CHOICE_LOWER" in
            r|run|"")  return 0 ;;
            s|skip)    log_warning "Skipped."; (( STEP_SKIP++ )) || true; return 1 ;;
            q|quit)    echo ""; log_info "Aborted by user."; exit 0 ;;
            *)         echo "  Please enter r, s, or q." ;;
        esac
    done
}

# ── API call ──────────────────────────────────────────────────────────────────
# Outputs HTTP status on last line, body on all prior lines
thunder_call() {
    local METHOD="$1" PATH_="$2" BODY="${3:-}"
    local URL="${THUNDER_BASE_URL}${PATH_}"
    local ARGS=(-s -w "\n%{http_code}" -X "$METHOD" \
        -H "Authorization: Bearer ${THUNDER_ACCESS_TOKEN}" \
        -H "Content-Type: application/json")
    # Allow skipping TLS verification when using self-signed certs (set THUNDER_INSECURE=1)
    if [[ "${THUNDER_INSECURE:-}" =~ ^(1|true|yes)$ ]]; then
        ARGS+=(-k)
    fi
    [[ -n "$BODY" ]] && ARGS+=(-d "$BODY")
    curl "${ARGS[@]}" "$URL"
}

parse_response() {
    # $1 = raw curl output (body\nHTTP_CODE)
    HTTP_CODE="${1##*$'\n'}"
    BODY="${1%$'\n'*}"
}

show_response() {
    local STATUS="$1" BODY="$2"
    if (( STATUS >= 200 && STATUS < 300 )); then
        echo -e "  ${GREEN}HTTP ${STATUS}${RESET}"
    elif (( STATUS == 409 )); then
        echo -e "  ${YELLOW}HTTP ${STATUS} (already exists)${RESET}"
    else
        echo -e "  ${RED}HTTP ${STATUS}${RESET}"
    fi
    pretty_json "$BODY" | sed 's/^/  /'
}

# ── Step runner ───────────────────────────────────────────────────────────────
# run_step TITLE METHOD PATH BODY [EXTRACT_VAR]
# Handles 201/200/409 logic; optionally extracts id into VARS[EXTRACT_VAR]
run_step() {
    local TITLE="$1" METHOD="$2" PATH_="$3" BODY="${4:-}" EXTRACT_VAR="${5:-}"

    local RAW HTTP_CODE BODY_OUT attempt=0
    while true; do
        ((attempt++))
        confirm_step "$TITLE" "$METHOD" "$PATH_" "$BODY" || return 0

        # Call thunder_call but capture curl failures without exiting the script
        set +e
        RAW=$(thunder_call "$METHOD" "$PATH_" "$BODY")
        CURL_EXIT=$?
        set -e

        if [[ $CURL_EXIT -ne 0 ]]; then
            log_error "HTTP call failed (curl exit $CURL_EXIT). Raw output:" 
            echo "$RAW"
            HTTP_CODE=""
            BODY_OUT="$RAW"
        else
            parse_response "$RAW"
            HTTP_CODE="${RAW##*$'\n'}"
            BODY_OUT="${RAW%$'\n'*}"
        fi

        show_response "$HTTP_CODE" "$BODY_OUT"

        if (( HTTP_CODE == 201 || HTTP_CODE == 200 || HTTP_CODE == 204 || HTTP_CODE == 202 )); then
            log_success "${TITLE} — done."
            (( STEP_PASS++ )) || true
            if [[ -n "$EXTRACT_VAR" ]]; then
                local ID
                ID=$(extract_first_id "$BODY_OUT")
                if [[ -n "$ID" ]]; then
                    set_var "$EXTRACT_VAR" "$ID"
                    log_info "Stored ${EXTRACT_VAR} = ${ID}"
                fi
            fi
            return 0
        elif (( HTTP_CODE == 409 )); then
            log_warning "${TITLE} — already exists, skipping."
            (( STEP_SKIP++ )) || true
            return 0
        elif [[ "$HTTP_CODE" == "400" ]] && echo "$BODY_OUT" | grep -q "APP-1022"; then
            log_warning "${TITLE} — application already exists, skipping."
            (( STEP_SKIP++ )) || true
            return 0
        else
            log_error "${TITLE} — failed (HTTP ${HTTP_CODE})."
            (( STEP_FAIL++ )) || true

            if [[ "$AUTO_RUN" == "1" ]]; then
                # In auto-run mode, ask user to retry, skip, or quit
                while true; do
                    read -r -p "  Step failed. [R]etry  [S]kip  [Q]uit  > " CHOICE
                    CHOICE_LOWER=$(echo "${CHOICE}" | tr '[:upper:]' '[:lower:]')
                    case "$CHOICE_LOWER" in
                        r|retry) log_info "Retrying..."; break ;;
                        s|skip)  log_warning "Skipping step."; (( STEP_SKIP++ )) || true; return 0 ;;
                        q|quit)  log_info "Aborted."; exit 1 ;;
                        *)       echo "  Please enter r, s, or q." ;;
                    esac
                done
                # loop will retry
            else
                read -r -p "  Step failed. [C]ontinue or [Q]uit? > " CHOICE
                CHOICE_LOWER=$(echo "${CHOICE}" | tr '[:upper:]' '[:lower:]')
                case "$CHOICE_LOWER" in
                    q|quit) log_info "Aborted."; exit 1 ;;
                    *) log_warning "Continuing despite error." ; return 0 ;;
                esac
            fi
        fi
    done
}

# Variant: run_step_fetch — GET that fetches an ID into a var (no confirm needed for fallback)
fetch_id_by_path() {
    local PATH_="$1" EXTRACT_VAR="$2"
    local RAW HTTP_CODE BODY_OUT
    RAW=$(thunder_call GET "$PATH_")
    HTTP_CODE="${RAW##*$'\n'}"
    BODY_OUT="${RAW%$'\n'*}"
    if (( HTTP_CODE == 200 )); then
        local ID
        ID=$(extract_first_id "$BODY_OUT")
        if [[ -n "$ID" ]]; then
            set_var "$EXTRACT_VAR" "$ID"
            log_info "Fetched ${EXTRACT_VAR} = ${ID}"
        fi
    fi
}

# ── Build JSON bodies ─────────────────────────────────────────────────────────
spa_body() {
    local NAME="$1" DESC="$2" CLIENT_ID="$3" PORT="$4" USER_TYPE="$5" OU_ID="$6"
    cat <<JSON
{
    "name": "${NAME}",
    "description": "${DESC}",
    "ouId": "${OU_ID}",
    "isRegistrationFlowEnabled": false,
    "template": "react",
    "logoUrl": "https://ssl.gstatic.com/docs/common/profile/kiwi_lg.png",
    "assertion": { "validityPeriod": 3600 },
    "inboundAuthConfig": [{
        "type": "oauth2",
        "config": {
            "clientId": "${CLIENT_ID}",
            "redirectUris": ["http://localhost:${PORT}", "https://localhost:${PORT}"],
            "grantTypes": ["authorization_code", "refresh_token"],
            "responseTypes": ["code"],
            "tokenEndpointAuthMethod": "none",
            "pkceRequired": true,
            "publicClient": true,
            "token": {
                "accessToken": {
                    "validityPeriod": 3600,
                    "userAttributes": ["email","phone_number","family_name","given_name","groups","roles","ouHandle","ouId","ouName","username"]
                },
                "idToken": {
                    "validityPeriod": 3600,
                    "userAttributes": ["email","family_name","given_name","groups","roles","ouHandle","ouId","ouName","username"]
                }
            },
            "scopes": ["openid","profile","email","group","role"],
            "userInfo": { "userAttributes": ["family_name","given_name","email"] },
            "scopeClaims": {
                "profile": ["name","given_name","family_name"],
                "email": ["email"],
                "phone": ["phone_number"],
                "group": ["groups"],
                "ou": ["ouId"],
                "role": ["roles"]
            }
        }
    }],
    "userAttributes": ["given_name","family_name","email","groups","ouId","ouHandle","ouName","username"],
    "allowedUserTypes": ["${USER_TYPE}"]
}
JSON
}

m2m_body() {
    local NAME="$1" DESC="$2" CLIENT_ID="$3" SECRET="$4" OU_ID="$5"
    cat <<JSON
{
    "name": "${NAME}",
    "description": "${DESC}",
    "ouId": "${OU_ID}",
    "isRegistrationFlowEnabled": false,
    "assertion": { "validityPeriod": 3600 },
    "inboundAuthConfig": [{
        "type": "oauth2",
        "config": {
            "clientId": "${CLIENT_ID}",
            "clientSecret": "${SECRET}",
            "grantTypes": ["client_credentials"],
            "tokenEndpointAuthMethod": "client_secret_basic",
            "pkceRequired": false,
            "publicClient": false,
            "token": { "accessToken": { "validityPeriod": 3600 } }
        }
    }],
    "allowedUserTypes": []
}
JSON
}

# ═════════════════════════════════════════════════════════════════════════════
# MAIN
# ═════════════════════════════════════════════════════════════════════════════

clear
echo -e "${BOLD}${CYAN}"
echo "╔══════════════════════════════════════════════════════╗"
echo "║      Thunder Interactive Resource Setup              ║"
echo "╚══════════════════════════════════════════════════════╝"
echo -e "${RESET}"
echo -e "  Base URL : ${THUNDER_BASE_URL}"
echo -e "  Token    : ${THUNDER_ACCESS_TOKEN:0:8}…"
echo ""
echo -e "  At each step: ${CYAN}[R]un${RESET}  ${YELLOW}[S]kip${RESET}  ${RED}[Q]uit${RESET}"
echo ""

# ── 1. Organization Units ────────────────────────────────────────────────────
log_step "Organization Units"

run_step "Create Private Sector OU" POST "/organization-units" \
    '{"handle":"private-sector","name":"Private Sector","description":"Organization unit for private sector entities"}' \
    PRIVATE_SECTOR_OU_ID

# Fallback fetch if skipped/already existed
[[ -z "$(get_var PRIVATE_SECTOR_OU_ID)" ]] && fetch_id_by_path "/organization-units/tree/private-sector" PRIVATE_SECTOR_OU_ID

run_step "Create ABCD Traders child OU" POST "/organization-units" \
    "{\"handle\":\"abcd-traders\",\"name\":\"ABCD Traders\",\"description\":\"Child organization unit for ABCD Traders\",\"parent\":\"$(get_var PRIVATE_SECTOR_OU_ID)\"}" \
    ABCD_TRADERS_OU_ID

[[ -z "$(get_var ABCD_TRADERS_OU_ID)" ]] && fetch_id_by_path "/organization-units/tree/private-sector/abcd-traders" ABCD_TRADERS_OU_ID

run_step "Create Government Organization root OU" POST "/organization-units" \
    '{"handle":"government-organization","name":"Government Organization","description":"Root organization unit for government entities"}' \
    GOVERNMENT_ORG_OU_ID

[[ -z "$(get_var GOVERNMENT_ORG_OU_ID)" ]] && fetch_id_by_path "/organization-units/tree/government-organization" GOVERNMENT_ORG_OU_ID

run_step "Create NPQS child OU" POST "/organization-units" \
    "{\"handle\":\"npqs\",\"name\":\"NPQS\",\"description\":\"National Plant Quarantine Service\",\"parent\":\"$(get_var GOVERNMENT_ORG_OU_ID)\"}" \
    NPQS_OU_ID

[[ -z "$(get_var NPQS_OU_ID)" ]] && fetch_id_by_path "/organization-units/tree/government-organization/npqs" NPQS_OU_ID

run_step "Create FCAU child OU" POST "/organization-units" \
    "{\"handle\":\"fcau\",\"name\":\"FCAU\",\"description\":\"Food Control Administration Unit\",\"parent\":\"$(get_var GOVERNMENT_ORG_OU_ID)\"}" \
    FCAU_OU_ID

[[ -z "$(get_var FCAU_OU_ID)" ]] && fetch_id_by_path "/organization-units/tree/government-organization/fcau" FCAU_OU_ID

run_step "Create IRD child OU" POST "/organization-units" \
    "{\"handle\":\"ird\",\"name\":\"IRD\",\"description\":\"Inland Revenue Department\",\"parent\":\"$(get_var GOVERNMENT_ORG_OU_ID)\"}" \
    IRD_OU_ID

[[ -z "$(get_var IRD_OU_ID)" ]] && fetch_id_by_path "/organization-units/tree/government-organization/ird" IRD_OU_ID

run_step "Create CDA child OU" POST "/organization-units" \
    "{\"handle\":\"cda\",\"name\":\"CDA\",\"description\":\"Coconut Development Authority\",\"parent\":\"$(get_var GOVERNMENT_ORG_OU_ID)\"}" \
    CDA_OU_ID

[[ -z "$(get_var CDA_OU_ID)" ]] && fetch_id_by_path "/organization-units/tree/government-organization/cda" CDA_OU_ID

# Fetch default OU ID
fetch_id_by_path "/organization-units/tree/default" DEFAULT_OU_ID

# ── 2. User Types ─────────────────────────────────────────────────────────────
log_step "User Types"

run_step "Create Private_User user type" POST "/user-schemas" \
    "{\"name\":\"Private_User\",\"ouId\":\"$(get_var PRIVATE_SECTOR_OU_ID)\",\"allowSelfRegistration\":false,\"schema\":{\"username\":{\"type\":\"string\",\"required\":true,\"unique\":true},\"password\":{\"type\":\"string\",\"required\":true,\"credential\":true},\"email\":{\"type\":\"string\",\"required\":true,\"unique\":true,\"regex\":\"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\\\.[a-zA-Z]{2,}\$\"},\"phone_number\":{\"type\":\"string\",\"required\":false,\"regex\":\"^\\\\+?[1-9]\\\\d{1,14}\$\"},\"given_name\":{\"type\":\"string\",\"required\":false},\"family_name\":{\"type\":\"string\",\"required\":false}},\"systemAttributes\":{\"display\":\"username\"}}"

run_step "Create Government_User user type" POST "/user-schemas" \
    "{\"name\":\"Government_User\",\"ouId\":\"$(get_var GOVERNMENT_ORG_OU_ID)\",\"allowSelfRegistration\":false,\"schema\":{\"username\":{\"type\":\"string\",\"required\":true,\"unique\":true},\"password\":{\"type\":\"string\",\"required\":true,\"credential\":true},\"email\":{\"type\":\"string\",\"required\":true,\"unique\":true,\"regex\":\"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\\\.[a-zA-Z]{2,}\$\"},\"phone_number\":{\"type\":\"string\",\"required\":false},\"given_name\":{\"type\":\"string\",\"required\":false},\"family_name\":{\"type\":\"string\",\"required\":false}},\"systemAttributes\":{\"display\":\"username\"}}"

# ── 3. Groups ─────────────────────────────────────────────────────────────────
log_step "Groups"

run_step "Create Traders group" POST "/groups" \
    "{\"name\":\"Traders\",\"description\":\"Trader members group\",\"ouId\":\"$(get_var ABCD_TRADERS_OU_ID)\"}" \
    TRADERS_GROUP_ID

[[ -z "$(get_var TRADERS_GROUP_ID)" ]] && {
    RAW=$(thunder_call GET "/groups?limit=100&offset=0")
    BODY_OUT="${RAW%$'\n'*}"
    ID=$(echo "$BODY_OUT" | sed 's/},{/}\n{/g' | grep '"name":"Traders"' | grep "\"ouId\":\"$(get_var ABCD_TRADERS_OU_ID)\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [[ -n "$ID" ]] && set_var TRADERS_GROUP_ID "$ID" && log_info "Fetched TRADERS_GROUP_ID = $ID"
}

run_step "Create CHA group" POST "/groups" \
    "{\"name\":\"CHA\",\"description\":\"CHA members group\",\"ouId\":\"$(get_var ABCD_TRADERS_OU_ID)\"}" \
    CHA_GROUP_ID

[[ -z "$(get_var CHA_GROUP_ID)" ]] && {
    RAW=$(thunder_call GET "/groups?limit=100&offset=0")
    BODY_OUT="${RAW%$'\n'*}"
    ID=$(echo "$BODY_OUT" | sed 's/},{/}\n{/g' | grep '"name":"CHA"' | grep "\"ouId\":\"$(get_var ABCD_TRADERS_OU_ID)\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [[ -n "$ID" ]] && set_var CHA_GROUP_ID "$ID" && log_info "Fetched CHA_GROUP_ID = $ID"
}

# ── 4. Roles ──────────────────────────────────────────────────────────────────
log_step "Roles"

run_step "Create Trader role" POST "/roles" \
    "{\"name\":\"Trader\",\"description\":\"Role for trader operations\",\"ouId\":\"$(get_var PRIVATE_SECTOR_OU_ID)\",\"permissions\":[]}" \
    TRADER_ROLE_ID

[[ -z "$(get_var TRADER_ROLE_ID)" ]] && {
    RAW=$(thunder_call GET "/roles?limit=100&offset=0")
    BODY_OUT="${RAW%$'\n'*}"
    ID=$(echo "$BODY_OUT" | sed 's/},{/}\n{/g' | grep '"name":"Trader"' | grep "\"ouId\":\"$(get_var PRIVATE_SECTOR_OU_ID)\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [[ -n "$ID" ]] && set_var TRADER_ROLE_ID "$ID" && log_info "Fetched TRADER_ROLE_ID = $ID"
}

run_step "Create CHA role" POST "/roles" \
    "{\"name\":\"CHA\",\"description\":\"Role for CHA operations\",\"ouId\":\"$(get_var PRIVATE_SECTOR_OU_ID)\",\"permissions\":[]}" \
    CHA_ROLE_ID

[[ -z "$(get_var CHA_ROLE_ID)" ]] && {
    RAW=$(thunder_call GET "/roles?limit=100&offset=0")
    BODY_OUT="${RAW%$'\n'*}"
    ID=$(echo "$BODY_OUT" | sed 's/},{/}\n{/g' | grep '"name":"CHA"' | grep "\"ouId\":\"$(get_var PRIVATE_SECTOR_OU_ID)\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [[ -n "$ID" ]] && set_var CHA_ROLE_ID "$ID" && log_info "Fetched CHA_ROLE_ID = $ID"
}

# ── 5. Role → Group assignments ───────────────────────────────────────────────
log_step "Role Assignments"

run_step "Assign Trader role → Traders group" POST "/roles/$(get_var TRADER_ROLE_ID)/assignments/add" \
    "{\"assignments\":[{\"id\":\"$(get_var TRADERS_GROUP_ID)\",\"type\":\"group\"}]}"

run_step "Assign CHA role → CHA group" POST "/roles/$(get_var CHA_ROLE_ID)/assignments/add" \
    "{\"assignments\":[{\"id\":\"$(get_var CHA_GROUP_ID)\",\"type\":\"group\"}]}"

# ── 6. Users ──────────────────────────────────────────────────────────────────
log_step "Users"

run_step "Create user123 (both roles)" POST "/users" \
    "{\"type\":\"Private_User\",\"ouId\":\"$(get_var ABCD_TRADERS_OU_ID)\",\"attributes\":{\"username\":\"user123\",\"password\":\"${USER123_PASSWORD}\",\"email\":\"user123@abcd-traders.private-sector.dev\",\"given_name\":\"Both\",\"family_name\":\"Roles\",\"phone_number\":\"+94771234567\"}}" \
    USER_123_ID

run_step "Create user456 (CHA only)" POST "/users" \
    "{\"type\":\"Private_User\",\"ouId\":\"$(get_var ABCD_TRADERS_OU_ID)\",\"attributes\":{\"username\":\"user456\",\"password\":\"${USER456_PASSWORD}\",\"email\":\"user456@abcd-traders.private-sector.dev\",\"given_name\":\"CHA\",\"family_name\":\"Only\",\"phone_number\":\"+94771234568\"}}" \
    USER_456_ID

run_step "Create user789 (Trader only)" POST "/users" \
    "{\"type\":\"Private_User\",\"ouId\":\"$(get_var ABCD_TRADERS_OU_ID)\",\"attributes\":{\"username\":\"user789\",\"password\":\"${USER789_PASSWORD}\",\"email\":\"user789@abcd-traders.private-sector.dev\",\"given_name\":\"Trader\",\"family_name\":\"Only\",\"phone_number\":\"+94771234569\"}}" \
    USER_789_ID

run_step "Create npqs_user" POST "/users" \
    "{\"type\":\"Government_User\",\"ouId\":\"$(get_var NPQS_OU_ID)\",\"attributes\":{\"username\":\"npqs_user\",\"password\":\"${NPQS_USER_PASSWORD}\",\"email\":\"npqs_user@government.dev\",\"given_name\":\"NPQS\",\"family_name\":\"User\",\"phone_number\":\"+94771234560\"}}" \
    NPQS_USER_ID

run_step "Create fcau_user" POST "/users" \
    "{\"type\":\"Government_User\",\"ouId\":\"$(get_var FCAU_OU_ID)\",\"attributes\":{\"username\":\"fcau_user\",\"password\":\"${FCAU_USER_PASSWORD}\",\"email\":\"fcau_user@government.dev\",\"given_name\":\"FCAU\",\"family_name\":\"User\",\"phone_number\":\"+94771234561\"}}" \
    FCAU_USER_ID

run_step "Create ird_user" POST "/users" \
    "{\"type\":\"Government_User\",\"ouId\":\"$(get_var IRD_OU_ID)\",\"attributes\":{\"username\":\"ird_user\",\"password\":\"${IRD_USER_PASSWORD}\",\"email\":\"ird_user@government.dev\",\"given_name\":\"IRD\",\"family_name\":\"User\",\"phone_number\":\"+94771234562\"}}" \
    IRD_USER_ID

run_step "Create cda_user" POST "/users" \
    "{\"type\":\"Government_User\",\"ouId\":\"$(get_var CDA_OU_ID)\",\"attributes\":{\"username\":\"cda_user\",\"password\":\"${CDA_USER_PASSWORD}\",\"email\":\"cda_user@government.dev\",\"given_name\":\"CDA\",\"family_name\":\"User\",\"phone_number\":\"+94771234563\"}}" \
    CDA_USER_ID

# ── 7. Group membership ───────────────────────────────────────────────────────
log_step "Group Membership"

run_step "Add user123 → Traders group" POST "/groups/$(get_var TRADERS_GROUP_ID)/members/add" \
    "{\"members\":[{\"id\":\"$(get_var USER_123_ID)\",\"type\":\"user\"}]}"

run_step "Add user123 → CHA group" POST "/groups/$(get_var CHA_GROUP_ID)/members/add" \
    "{\"members\":[{\"id\":\"$(get_var USER_123_ID)\",\"type\":\"user\"}]}"

run_step "Add user456 → CHA group" POST "/groups/$(get_var CHA_GROUP_ID)/members/add" \
    "{\"members\":[{\"id\":\"$(get_var USER_456_ID)\",\"type\":\"user\"}]}"

run_step "Add user789 → Traders group" POST "/groups/$(get_var TRADERS_GROUP_ID)/members/add" \
    "{\"members\":[{\"id\":\"$(get_var USER_789_ID)\",\"type\":\"user\"}]}"

# ── 8. SPA Applications ───────────────────────────────────────────────────────
log_step "SPA Applications"

run_step "Create TraderApp SPA" POST "/applications" \
    "$(spa_body "TraderApp" "Application for trader portal built with React" "TRADER_PORTAL_APP" "5173" "Private_User" "$(get_var DEFAULT_OU_ID)")"

run_step "Create NPQSPortalApp SPA" POST "/applications" \
    "$(spa_body "NPQSPortalApp" "Application for NPQS portal built with React" "OGA_PORTAL_APP_NPQS" "5174" "Government_User" "$(get_var NPQS_OU_ID)")"

run_step "Create FCAUPortalApp SPA" POST "/applications" \
    "$(spa_body "FCAUPortalApp" "Application for FCAU portal built with React" "OGA_PORTAL_APP_FCAU" "5175" "Government_User" "$(get_var FCAU_OU_ID)")"

run_step "Create IRDPortalApp SPA" POST "/applications" \
    "$(spa_body "IRDPortalApp" "Application for IRD portal built with React" "OGA_PORTAL_APP_IRD" "5176" "Government_User" "$(get_var IRD_OU_ID)")"

run_step "Create CDAPortalApp SPA" POST "/applications" \
    "$(spa_body "CDAPortalApp" "Application for CDA portal built with React" "OGA_PORTAL_APP_CDA" "5177" "Government_User" "$(get_var CDA_OU_ID)")"

# ── 9. M2M Applications ───────────────────────────────────────────────────────
log_step "M2M Applications"

run_step "Create NPQS_TO_NSW M2M app" POST "/applications" \
    "$(m2m_body "NPQS_TO_NSW_M2M" "Machine-to-machine integration for NPQS to NSW" "NPQS_TO_NSW" "$NPQS_M2M_SECRET" "$(get_var DEFAULT_OU_ID)")"

run_step "Create FCAU_TO_NSW M2M app" POST "/applications" \
    "$(m2m_body "FCAU_TO_NSW_M2M" "Machine-to-machine integration for FCAU to NSW" "FCAU_TO_NSW" "$FCAU_M2M_SECRET" "$(get_var DEFAULT_OU_ID)")"

run_step "Create IRD_TO_NSW M2M app" POST "/applications" \
    "$(m2m_body "IRD_TO_NSW_M2M" "Machine-to-machine integration for IRD to NSW" "IRD_TO_NSW" "$IRD_M2M_SECRET" "$(get_var DEFAULT_OU_ID)")"

run_step "Create CDA_TO_NSW M2M app" POST "/applications" \
    "$(m2m_body "CDA_TO_NSW_M2M" "Machine-to-machine integration for CDA to NSW" "CDA_TO_NSW" "$CDA_M2M_SECRET" "$(get_var DEFAULT_OU_ID)")"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}${CYAN}━━━  Summary  ━━━${RESET}"
echo -e "  ${GREEN}Passed : ${STEP_PASS}${RESET}"
echo -e "  ${YELLOW}Skipped: ${STEP_SKIP}${RESET}"
echo -e "  ${RED}Failed : ${STEP_FAIL}${RESET}"
echo -e "  Total  : ${STEP_INDEX}"
echo ""
echo -e "${BOLD}Resolved IDs:${RESET}"
for KEY in "${VAR_KEYS[@]:-}"; do
    printf "  %-30s %s\n" "${KEY}" "$(get_var "$KEY")"
done | sort
echo ""

if (( STEP_FAIL == 0 )); then
    log_success "All steps completed successfully!"
else
    log_warning "Some steps failed. Review the output above."
fi
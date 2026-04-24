set -e

# Source common functions from the same directory as this script
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]:-$0}")"
source "${SCRIPT_DIR}/common.sh"

# Load .env values when available (useful for local execution).
ENV_FILE="${SCRIPT_DIR}/.env"
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

SAMPLE_USER_PASSWORD="${THUNDER_SAMPLE_USER_PASSWORD:-1234}"
USER123_PASSWORD="${THUNDER_SAMPLE_USER123_PASSWORD:-${SAMPLE_USER_PASSWORD}}"
USER456_PASSWORD="${THUNDER_SAMPLE_USER456_PASSWORD:-${SAMPLE_USER_PASSWORD}}"
USER789_PASSWORD="${THUNDER_SAMPLE_USER789_PASSWORD:-${SAMPLE_USER_PASSWORD}}"
NPQS_USER_PASSWORD="${THUNDER_SAMPLE_NPQS_USER_PASSWORD:-${SAMPLE_USER_PASSWORD}}"
FCAU_USER_PASSWORD="${THUNDER_SAMPLE_FCAU_USER_PASSWORD:-${SAMPLE_USER_PASSWORD}}"
IRD_USER_PASSWORD="${THUNDER_SAMPLE_IRD_USER_PASSWORD:-${SAMPLE_USER_PASSWORD}}"
M2M_CLIENT_SECRET="${THUNDER_M2M_CLIENT_SECRET:-1234}"
NPQS_M2M_CLIENT_SECRET="${THUNDER_M2M_NPQS_SECRET:-${M2M_CLIENT_SECRET}}"
FCAU_M2M_CLIENT_SECRET="${THUNDER_M2M_FCAU_SECRET:-${M2M_CLIENT_SECRET}}"
IRD_M2M_CLIENT_SECRET="${THUNDER_M2M_IRD_SECRET:-${M2M_CLIENT_SECRET}}"

log_info "Creating sample Thunder resources..."
echo ""

# ============================================================================
# Helpers
# ============================================================================

extract_first_id() {
    echo "$1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

get_user_id_by_username() {
    local USERNAME="$1"
    local RESPONSE HTTP_CODE BODY
    RESPONSE=$(thunder_api_call GET "/users?limit=100&offset=0")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" != "200" ]]; then
        echo ""
        return
    fi

    # Parse one user object per line and locate matching username inside attributes.
    echo "$BODY" | sed 's/},{/}\n{/g' | grep "\"username\":\"${USERNAME}\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

get_group_id_by_name() {
    local GROUP_NAME="$1"
    local OU_ID="$2"
    local RESPONSE HTTP_CODE BODY
    RESPONSE=$(thunder_api_call GET "/groups?limit=100&offset=0")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" != "200" ]]; then
        echo ""
        return
    fi

    echo "$BODY" | sed 's/},{/}\n{/g' | grep "\"name\":\"${GROUP_NAME}\"" | grep "\"ouId\":\"${OU_ID}\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

get_role_id_by_name() {
    local ROLE_NAME="$1"
    local OU_ID="$2"
    local RESPONSE HTTP_CODE BODY
    RESPONSE=$(thunder_api_call GET "/roles?limit=100&offset=0")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" != "200" ]]; then
        echo ""
        return
    fi

    echo "$BODY" | sed 's/},{/}\n{/g' | grep "\"name\":\"${ROLE_NAME}\"" | grep "\"ouId\":\"${OU_ID}\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

get_flow_id_by_handle() {
    local FLOW_TYPE="$1"
    local FLOW_HANDLE="$2"
    local RESPONSE HTTP_CODE BODY
    RESPONSE=$(thunder_api_call GET "/flows?limit=30&offset=0&flowType=${FLOW_TYPE}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" != "200" ]]; then
            echo ""
            return
    fi

    echo "$BODY" | grep -o '{[^}]*"handle":"'"${FLOW_HANDLE}"'"[^}]*}' | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

get_application_id_by_client_id() {
    local CLIENT_ID="$1"
    local RESPONSE HTTP_CODE BODY
    RESPONSE=$(thunder_api_call GET "/applications?limit=200&offset=0")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" != "200" ]]; then
        echo ""
        return
    fi

    echo "$BODY" | sed 's/},{/}\n{/g' | grep "\"client_id\":\"${CLIENT_ID}\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

create_user_in_ou() {
        local USER_TYPE="$1"
        local OU_ID="$2"
        local USERNAME="$3"
        local EMAIL="$4"
        local GIVEN_NAME="$5"
        local FAMILY_NAME="$6"
    local PASSWORD="$7"
        local RESPONSE HTTP_CODE BODY USER_ID

        read -r -d '' USER_PAYLOAD <<JSON || true
{
"type": "${USER_TYPE}",
"ouId": "${OU_ID}",
"attributes": {
    "username": "${USERNAME}",
    "password": "${PASSWORD}",
    "email": "${EMAIL}",
    "given_name": "${GIVEN_NAME}",
    "family_name": "${FAMILY_NAME}"
}
}
JSON

        RESPONSE=$(thunder_api_call POST "/users" "${USER_PAYLOAD}")
        HTTP_CODE="${RESPONSE: -3}"
        BODY="${RESPONSE%???}"

        if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
                log_success "User ${USERNAME} created successfully"
                USER_ID=$(extract_first_id "$BODY")
        elif [[ "$HTTP_CODE" == "409" ]]; then
                log_warning "User ${USERNAME} already exists, retrieving ID..."
                USER_ID=$(get_user_id_by_username "$USERNAME")
        else
                log_error "Failed to create user ${USERNAME} (HTTP $HTTP_CODE)"
                echo "Response: $BODY"
                exit 1
        fi

        if [[ -z "$USER_ID" ]]; then
                log_error "Could not determine user ID for ${USERNAME}"
                exit 1
        fi

        log_info "${USERNAME} user ID: $USER_ID"
        CREATED_USER_ID="$USER_ID"
}

create_spa_application() {
        local APP_NAME="$1"
        local APP_DESCRIPTION="$2"
        local CLIENT_ID="$3"
        local PORT="$4"
        local ALLOWED_USER_TYPE="$5"
        local RESPONSE HTTP_CODE BODY
        local APP_ID APP_CLIENT_ID

        log_info "Creating ${APP_NAME} application..."

        ADDITIONAL_FIELDS=""
        if [[ -n "$CLASSIC_THEME_ID" ]]; then
                ADDITIONAL_FIELDS="${ADDITIONAL_FIELDS}
    \"theme_id\": \"${CLASSIC_THEME_ID}\"," 
        fi
        if [[ -n "$AUTH_FLOW_ID" ]]; then
                ADDITIONAL_FIELDS="${ADDITIONAL_FIELDS}
    \"auth_flow_id\": \"${AUTH_FLOW_ID}\"," 
        fi
        if [[ -n "$REG_FLOW_ID" ]]; then
                ADDITIONAL_FIELDS="${ADDITIONAL_FIELDS}
    \"registration_flow_id\": \"${REG_FLOW_ID}\"," 
        fi

        read -r -d '' APP_PAYLOAD <<JSON || true
{
    "name": "${APP_NAME}",
    "description": "${APP_DESCRIPTION}",${ADDITIONAL_FIELDS}
    "is_registration_flow_enabled": false,
    "template": "react",
    "logo_url": "https://ssl.gstatic.com/docs/common/profile/kiwi_lg.png",
    "assertion": {
        "validity_period": 3600
    },
    "certificate": {
        "type": "NONE"
    },
    "inbound_auth_config": [
        {
            "type": "oauth2",
            "config": {
                "client_id": "${CLIENT_ID}",
                "redirect_uris": [
                    "http://localhost:${PORT}",
                    "https://localhost:${PORT}"
                ],
                "grant_types": [
                    "authorization_code",
                    "refresh_token"
                ],
                "response_types": [
                    "code"
                ],
                "token_endpoint_auth_method": "none",
                "pkce_required": true,
                "public_client": true,
                "token": {
                    "access_token": {
                        "validity_period": 600,
                        "user_attributes": [
                            "email",
                            "family_name",
                            "given_name",
                            "groups",
                            "roles",
                            "ouHandle",
                            "ouId",
                            "ouName",
                            "username"
                        ]
                    },
                    "id_token": {
                        "validity_period": 600,
                        "user_attributes": [
                            "email",
                            "family_name",
                            "given_name",
                            "groups",
                            "roles",
                            "ouHandle",
                            "ouId",
                            "ouName",
                            "username"
                        ]
                    }
                },
                "scopes": [
                    "openid",
                    "profile",
                    "email",
                    "group",
                    "role"
                ],
                "user_info": {
                    "user_attributes": [
                        "family_name",
                        "given_name",
                        "email"
                    ]
                },
                "scope_claims": {
                    "group": [
                        "groups"
                    ],
                    "role": [
                        "roles"
                    ]
                }
            }
        }
    ],
    "allowed_user_types": [
        "${ALLOWED_USER_TYPE}"
    ]
}
JSON

        RESPONSE=$(thunder_api_call POST "/applications" "${APP_PAYLOAD}")
        HTTP_CODE="${RESPONSE: -3}"
        BODY="${RESPONSE%???}"

        if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]] || [[ "$HTTP_CODE" == "202" ]]; then
                log_success "${APP_NAME} application created successfully"
                APP_ID=$(extract_first_id "$BODY")
                APP_CLIENT_ID=$(echo "$BODY" | grep -o '"client_id":"[^"]*"' | head -1 | cut -d'"' -f4)
                if [[ -n "$APP_ID" ]]; then
                        log_info "${APP_NAME} app ID: ${APP_ID}"
                fi
                if [[ -n "$APP_CLIENT_ID" ]]; then
                        log_info "${APP_NAME} client ID: ${APP_CLIENT_ID}"
                fi
        elif [[ "$HTTP_CODE" == "409" ]] || ([[ "$HTTP_CODE" == "400" ]] && [[ "$BODY" =~ (Application\ already\ exists|APP-1022) ]]); then
                log_warning "${APP_NAME} application already exists, skipping"
        else
                log_error "Failed to create ${APP_NAME} application (HTTP $HTTP_CODE)"
                echo "Response: $BODY"
                exit 1
        fi
}

create_m2m_application() {
    local APP_NAME="$1"
    local APP_DESCRIPTION="$2"
    local CLIENT_ID="$3"
    local CLIENT_SECRET="$4"
    local RESPONSE HTTP_CODE BODY
    local APP_ID APP_CLIENT_ID

    log_info "Creating ${APP_NAME} M2M application..."

    read -r -d '' APP_PAYLOAD <<JSON || true
{
    "name": "${APP_NAME}",
    "description": "${APP_DESCRIPTION}",
    "is_registration_flow_enabled": false,
    "assertion": {
        "validity_period": 3600
    },
    "certificate": {
        "type": "NONE"
    },
    "inbound_auth_config": [
        {
            "type": "oauth2",
            "config": {
                "client_id": "${CLIENT_ID}",
                "client_secret": "${CLIENT_SECRET}",
                "grant_types": [
                    "client_credentials"
                ],
                "token_endpoint_auth_method": "client_secret_basic",
                "pkce_required": false,
                "public_client": false,
                "token": {
                    "access_token": {
                        "validity_period": 3600
                    }
                }
            }
        }
    ],
    "allowed_user_types": []
}
JSON

    RESPONSE=$(thunder_api_call POST "/applications" "${APP_PAYLOAD}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]] || [[ "$HTTP_CODE" == "202" ]]; then
        log_success "${APP_NAME} M2M application created successfully"
        APP_ID=$(extract_first_id "$BODY")
        APP_CLIENT_ID=$(echo "$BODY" | grep -o '"client_id":"[^"]*"' | head -1 | cut -d'"' -f4)
    elif [[ "$HTTP_CODE" == "409" ]] || ([[ "$HTTP_CODE" == "400" ]] && [[ "$BODY" =~ (Application\ already\ exists|APP-1022) ]]); then
        log_warning "${APP_NAME} M2M application already exists, retrieving ID..."
        APP_ID=$(get_application_id_by_client_id "$CLIENT_ID")
        APP_CLIENT_ID="$CLIENT_ID"
    else
        log_error "Failed to create ${APP_NAME} M2M application (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi

    if [[ -n "$APP_ID" ]]; then
        log_info "${APP_NAME} M2M app ID: ${APP_ID}"
    fi
    if [[ -n "$APP_CLIENT_ID" ]]; then
        log_info "${APP_NAME} M2M client ID: ${APP_CLIENT_ID}"
    fi

    CREATED_M2M_APP_ID="$APP_ID"
}

ensure_user_in_group() {
    local GROUP_ID="$1"
    local USER_ID="$2"
    local GROUP_NAME="$3"
    local USERNAME="$4"
    local RESPONSE HTTP_CODE BODY

    read -r -d '' MEMBERS_ADD_PAYLOAD <<JSON || true
{
  "members": [
    {
      "id": "${USER_ID}",
      "type": "user"
    }
  ]
}
JSON

    RESPONSE=$(thunder_api_call POST "/groups/${GROUP_ID}/members/add" "${MEMBERS_ADD_PAYLOAD}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]] || [[ "$HTTP_CODE" == "204" ]]; then
        log_success "Added ${USERNAME} to group ${GROUP_NAME}"
    elif [[ "$HTTP_CODE" == "409" ]]; then
        log_warning "${USERNAME} is already a member of group ${GROUP_NAME}, skipping"
    else
        log_error "Failed to add ${USERNAME} to group ${GROUP_NAME} (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
}

assign_role_to_group() {
    local ROLE_ID="$1"
    local GROUP_ID="$2"
    local ROLE_NAME="$3"
    local GROUP_NAME="$4"
    local RESPONSE HTTP_CODE BODY

    read -r -d '' ROLE_ASSIGNMENT_PAYLOAD <<JSON || true
{
  "assignments": [
    {
      "id": "${GROUP_ID}",
      "type": "group"
    }
  ]
}
JSON

    RESPONSE=$(thunder_api_call POST "/roles/${ROLE_ID}/assignments/add" "${ROLE_ASSIGNMENT_PAYLOAD}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]] || [[ "$HTTP_CODE" == "204" ]]; then
        log_success "Assigned role ${ROLE_NAME} to group ${GROUP_NAME}"
    elif [[ "$HTTP_CODE" == "409" ]]; then
        log_warning "Role ${ROLE_NAME} is already assigned to group ${GROUP_NAME}, skipping"
    else
        log_error "Failed to assign role ${ROLE_NAME} to group ${GROUP_NAME} (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
}

# ============================================================================
# Create Private Sector Organization Unit
# ============================================================================

PRIVATE_SECTOR_OU_HANDLE="private-sector"

log_info "Creating Private Sector organization unit..."

read -r -d '' PRIVATE_SECTOR_OU_PAYLOAD <<JSON || true
{
  "handle": "${PRIVATE_SECTOR_OU_HANDLE}",
  "name": "Private Sector",
  "description": "Organization unit for private sector entities"
}
JSON

RESPONSE=$(thunder_api_call POST "/organization-units" "${PRIVATE_SECTOR_OU_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "Private Sector organization unit created successfully"
    PRIVATE_SECTOR_OU_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "Private Sector organization unit already exists, retrieving ID..."
    RESPONSE=$(thunder_api_call GET "/organization-units/tree/${PRIVATE_SECTOR_OU_HANDLE}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]]; then
        PRIVATE_SECTOR_OU_ID=$(extract_first_id "$BODY")
    else
        log_error "Failed to fetch organization unit by handle '${PRIVATE_SECTOR_OU_HANDLE}' (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
else
    log_error "Failed to create Private Sector organization unit (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$PRIVATE_SECTOR_OU_ID" ]]; then
    log_error "Could not determine Private Sector organization unit ID"
    exit 1
fi

log_info "Private Sector OU ID: $PRIVATE_SECTOR_OU_ID"

echo ""

# ============================================================================
# Create ABCD Traders Child Organization Unit
# ============================================================================

ABCD_TRADERS_OU_HANDLE="abcd-traders"
ABCD_TRADERS_OU_PATH="${PRIVATE_SECTOR_OU_HANDLE}/${ABCD_TRADERS_OU_HANDLE}"

log_info "Creating ABCD Traders child organization unit..."

read -r -d '' ABCD_TRADERS_OU_PAYLOAD <<JSON || true
{
  "handle": "${ABCD_TRADERS_OU_HANDLE}",
  "name": "ABCD Traders",
  "description": "Child organization unit for ABCD Traders",
  "parent": "${PRIVATE_SECTOR_OU_ID}"
}
JSON

RESPONSE=$(thunder_api_call POST "/organization-units" "${ABCD_TRADERS_OU_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "ABCD Traders organization unit created successfully"
    ABCD_TRADERS_OU_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "ABCD Traders organization unit already exists, retrieving ID..."
    RESPONSE=$(thunder_api_call GET "/organization-units/tree/${ABCD_TRADERS_OU_PATH}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]]; then
        ABCD_TRADERS_OU_ID=$(extract_first_id "$BODY")
    else
        log_error "Failed to fetch organization unit by path '${ABCD_TRADERS_OU_PATH}' (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
else
    log_error "Failed to create ABCD Traders organization unit (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$ABCD_TRADERS_OU_ID" ]]; then
    log_error "Could not determine ABCD Traders organization unit ID"
    exit 1
fi

log_info "ABCD Traders OU ID: $ABCD_TRADERS_OU_ID"

echo ""

# ============================================================================
# Create Government Organization and Child OUs
# ============================================================================

GOVERNMENT_ORG_OU_HANDLE="government-organization"
NPQS_OU_HANDLE="npqs"
FCAU_OU_HANDLE="fcau"
IRD_OU_HANDLE="ird"

log_info "Creating Government Organization root organization unit..."

read -r -d '' GOVERNMENT_ORG_OU_PAYLOAD <<JSON || true
{
  "handle": "${GOVERNMENT_ORG_OU_HANDLE}",
  "name": "Government Organization",
  "description": "Root organization unit for government entities"
}
JSON

RESPONSE=$(thunder_api_call POST "/organization-units" "${GOVERNMENT_ORG_OU_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "Government Organization created successfully"
    GOVERNMENT_ORG_OU_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "Government Organization already exists, retrieving ID..."
    RESPONSE=$(thunder_api_call GET "/organization-units/tree/${GOVERNMENT_ORG_OU_HANDLE}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]]; then
        GOVERNMENT_ORG_OU_ID=$(extract_first_id "$BODY")
    else
        log_error "Failed to fetch organization unit by path '${GOVERNMENT_ORG_OU_HANDLE}' (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
else
    log_error "Failed to create Government Organization (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$GOVERNMENT_ORG_OU_ID" ]]; then
    log_error "Could not determine Government Organization ID"
    exit 1
fi

log_info "Government Organization OU ID: $GOVERNMENT_ORG_OU_ID"

echo ""
log_info "Creating NPQS organization unit..."

read -r -d '' NPQS_OU_PAYLOAD <<JSON || true
{
  "handle": "${NPQS_OU_HANDLE}",
  "name": "NPQS",
  "description": "National Plant Quarantine Service",
  "parent": "${GOVERNMENT_ORG_OU_ID}"
}
JSON

RESPONSE=$(thunder_api_call POST "/organization-units" "${NPQS_OU_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "NPQS organization unit created successfully"
    NPQS_OU_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "NPQS organization unit already exists, retrieving ID..."
    RESPONSE=$(thunder_api_call GET "/organization-units/tree/${GOVERNMENT_ORG_OU_HANDLE}/${NPQS_OU_HANDLE}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]]; then
        NPQS_OU_ID=$(extract_first_id "$BODY")
    else
        log_error "Failed to fetch NPQS OU (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
else
    log_error "Failed to create NPQS organization unit (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$NPQS_OU_ID" ]]; then
    log_error "Could not determine NPQS organization unit ID"
    exit 1
fi

log_info "NPQS OU ID: $NPQS_OU_ID"

echo ""
log_info "Creating FCAU organization unit..."

read -r -d '' FCAU_OU_PAYLOAD <<JSON || true
{
  "handle": "${FCAU_OU_HANDLE}",
  "name": "FCAU",
  "description": "Food Control Administration Unit",
  "parent": "${GOVERNMENT_ORG_OU_ID}"
}
JSON

RESPONSE=$(thunder_api_call POST "/organization-units" "${FCAU_OU_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "FCAU organization unit created successfully"
    FCAU_OU_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "FCAU organization unit already exists, retrieving ID..."
    RESPONSE=$(thunder_api_call GET "/organization-units/tree/${GOVERNMENT_ORG_OU_HANDLE}/${FCAU_OU_HANDLE}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]]; then
        FCAU_OU_ID=$(extract_first_id "$BODY")
    else
        log_error "Failed to fetch FCAU OU (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
else
    log_error "Failed to create FCAU organization unit (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$FCAU_OU_ID" ]]; then
    log_error "Could not determine FCAU organization unit ID"
    exit 1
fi

log_info "FCAU OU ID: $FCAU_OU_ID"

echo ""
log_info "Creating IRD organization unit..."

read -r -d '' IRD_OU_PAYLOAD <<JSON || true
{
  "handle": "${IRD_OU_HANDLE}",
  "name": "IRD",
  "description": "Inland Revenue Department",
  "parent": "${GOVERNMENT_ORG_OU_ID}"
}
JSON

RESPONSE=$(thunder_api_call POST "/organization-units" "${IRD_OU_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "IRD organization unit created successfully"
    IRD_OU_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "IRD organization unit already exists, retrieving ID..."
    RESPONSE=$(thunder_api_call GET "/organization-units/tree/${GOVERNMENT_ORG_OU_HANDLE}/${IRD_OU_HANDLE}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]]; then
        IRD_OU_ID=$(extract_first_id "$BODY")
    else
        log_error "Failed to fetch IRD OU (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
else
    log_error "Failed to create IRD organization unit (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$IRD_OU_ID" ]]; then
    log_error "Could not determine IRD organization unit ID"
    exit 1
fi

log_info "IRD OU ID: $IRD_OU_ID"

echo ""

# ============================================================================
# Create Private_User User Type
# ============================================================================

log_info "Creating Private_User user type..."

read -r -d '' PRIVATE_USER_TYPE_PAYLOAD <<JSON || true
{
  "name": "Private_User",
  "ouId": "${PRIVATE_SECTOR_OU_ID}",
  "allowSelfRegistration": false,
  "schema": {
    "username": {
      "type": "string",
      "required": true,
      "unique": true
    },
    "password": {
      "type": "string",
      "required": true,
      "credential": true
    },
    "email": {
      "type": "string",
      "required": true,
      "unique": true,
      "regex": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\\\.[a-zA-Z]{2,}$"
    },
    "given_name": {
      "type": "string",
      "required": false
    },
    "family_name": {
      "type": "string",
      "required": false
    }
  },
  "systemAttributes": {
    "display": "username"
  }
}
JSON

RESPONSE=$(thunder_api_call POST "/user-schemas" "${PRIVATE_USER_TYPE_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
        log_success "Private_User user type created successfully"
elif [[ "$HTTP_CODE" == "409" ]]; then
        log_warning "Private_User user type already exists, skipping"
else
        log_error "Failed to create Private_User user type (HTTP $HTTP_CODE)"
    exit 1
fi

echo ""

# ============================================================================
# Create Government_User User Type
# ============================================================================

log_info "Creating Government_User user type..."

read -r -d '' GOVERNMENT_USER_TYPE_PAYLOAD <<JSON || true
{
    "name": "Government_User",
    "ouId": "${GOVERNMENT_ORG_OU_ID}",
    "allowSelfRegistration": false,
    "schema": {
        "username": {
            "type": "string",
            "required": true,
            "unique": true
        },
        "password": {
            "type": "string",
            "required": true,
            "credential": true
        },
        "email": {
            "type": "string",
            "required": true,
            "unique": true,
            "regex": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\\\.[a-zA-Z]{2,}$"
        },
        "given_name": {
            "type": "string",
            "required": false
        },
        "family_name": {
            "type": "string",
            "required": false
        }
    },
    "systemAttributes": {
        "display": "username"
    }
}
JSON

RESPONSE=$(thunder_api_call POST "/user-schemas" "${GOVERNMENT_USER_TYPE_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
        log_success "Government_User user type created successfully"
elif [[ "$HTTP_CODE" == "409" ]]; then
        log_warning "Government_User user type already exists, skipping"
else
        log_error "Failed to create Government_User user type (HTTP $HTTP_CODE)"
        exit 1
fi

echo ""

# ============================================================================
# Create Groups (Traders, CHA)
# ============================================================================

log_info "Creating Traders group..."

read -r -d '' TRADERS_GROUP_PAYLOAD <<JSON || true
{
  "name": "Traders",
  "description": "Trader members group",
  "ouId": "${ABCD_TRADERS_OU_ID}"
}
JSON

RESPONSE=$(thunder_api_call POST "/groups" "${TRADERS_GROUP_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "Traders group created successfully"
    TRADERS_GROUP_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "Traders group already exists, retrieving ID..."
    TRADERS_GROUP_ID=$(get_group_id_by_name "Traders" "$ABCD_TRADERS_OU_ID")
else
    log_error "Failed to create Traders group (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$TRADERS_GROUP_ID" ]]; then
    log_error "Could not determine Traders group ID"
    exit 1
fi

log_info "Traders group ID: $TRADERS_GROUP_ID"

echo ""

# ============================================================================
# Create CHA Group
# ============================================================================

log_info "Creating CHA group..."

read -r -d '' CHA_GROUP_PAYLOAD <<JSON || true
{
  "name": "CHA",
  "description": "CHA members group",
  "ouId": "${ABCD_TRADERS_OU_ID}"
}
JSON

RESPONSE=$(thunder_api_call POST "/groups" "${CHA_GROUP_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "CHA group created successfully"
    CHA_GROUP_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "CHA group already exists, retrieving ID..."
    CHA_GROUP_ID=$(get_group_id_by_name "CHA" "$ABCD_TRADERS_OU_ID")
else
    log_error "Failed to create CHA group (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$CHA_GROUP_ID" ]]; then
    log_error "Could not determine CHA group ID"
    exit 1
fi

log_info "CHA group ID: $CHA_GROUP_ID"

echo ""

# ============================================================================
# Create Roles (Trader, CHA)
# ============================================================================

log_info "Creating Trader role..."

read -r -d '' TRADER_ROLE_PAYLOAD <<JSON || true
{
  "name": "Trader",
  "description": "Role for trader operations",
  "ouId": "${PRIVATE_SECTOR_OU_ID}",
  "permissions": []
}
JSON

RESPONSE=$(thunder_api_call POST "/roles" "${TRADER_ROLE_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "Trader role created successfully"
    TRADER_ROLE_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "Trader role already exists, retrieving ID..."
    TRADER_ROLE_ID=$(get_role_id_by_name "Trader" "$PRIVATE_SECTOR_OU_ID")
else
    log_error "Failed to create Trader role (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$TRADER_ROLE_ID" ]]; then
    log_error "Could not determine Trader role ID"
    exit 1
fi

log_info "Trader role ID: $TRADER_ROLE_ID"

echo ""
log_info "Creating CHA role..."

read -r -d '' CHA_ROLE_PAYLOAD <<JSON || true
{
  "name": "CHA",
  "description": "Role for CHA operations",
  "ouId": "${PRIVATE_SECTOR_OU_ID}",
  "permissions": []
}
JSON

RESPONSE=$(thunder_api_call POST "/roles" "${CHA_ROLE_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "CHA role created successfully"
    CHA_ROLE_ID=$(extract_first_id "$BODY")
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "CHA role already exists, retrieving ID..."
    CHA_ROLE_ID=$(get_role_id_by_name "CHA" "$PRIVATE_SECTOR_OU_ID")
else
    log_error "Failed to create CHA role (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$CHA_ROLE_ID" ]]; then
    log_error "Could not determine CHA role ID"
    exit 1
fi

log_info "CHA role ID: $CHA_ROLE_ID"

echo ""

# ============================================================================
# Assign Roles to Groups
# ============================================================================

log_info "Assigning roles to groups..."
assign_role_to_group "$TRADER_ROLE_ID" "$TRADERS_GROUP_ID" "Trader" "Traders"
assign_role_to_group "$CHA_ROLE_ID" "$CHA_GROUP_ID" "CHA" "CHA"

echo ""

# ============================================================================
# Create Users in ABCD Traders OU
# ============================================================================

log_info "Creating sample users..."

create_user_in_ou "Private_User" "$ABCD_TRADERS_OU_ID" "user123" "user123@abcd-traders.private-sector.dev" "Both" "Roles" "$USER123_PASSWORD"
USER_123="$CREATED_USER_ID"

create_user_in_ou "Private_User" "$ABCD_TRADERS_OU_ID" "user456" "user456@abcd-traders.private-sector.dev" "CHA" "Only" "$USER456_PASSWORD"
USER_456="$CREATED_USER_ID"

create_user_in_ou "Private_User" "$ABCD_TRADERS_OU_ID" "user789" "user789@abcd-traders.private-sector.dev" "Trader" "Only" "$USER789_PASSWORD"
USER_789="$CREATED_USER_ID"

create_user_in_ou "Government_User" "$NPQS_OU_ID" "npqs_user" "npqs_user@government.dev" "NPQS" "User" "$NPQS_USER_PASSWORD"
USER_NPQS_ID="$CREATED_USER_ID"

create_user_in_ou "Government_User" "$FCAU_OU_ID" "fcau_user" "fcau_user@government.dev" "FCAU" "User" "$FCAU_USER_PASSWORD"
USER_FCAU_ID="$CREATED_USER_ID"

create_user_in_ou "Government_User" "$IRD_OU_ID" "ird_user" "ird_user@government.dev" "IRD" "User" "$IRD_USER_PASSWORD"
USER_IRD_ID="$CREATED_USER_ID"

echo ""

# ============================================================================
# Assign Users to Groups (Role inheritance is group-based)
# ============================================================================

log_info "Assigning users to groups..."

ensure_user_in_group "$TRADERS_GROUP_ID" "$USER_123" "Traders" "both_roles_user"
ensure_user_in_group "$CHA_GROUP_ID" "$USER_123" "CHA" "both_roles_user"
ensure_user_in_group "$CHA_GROUP_ID" "$USER_456" "CHA" "cha_only_user"
ensure_user_in_group "$TRADERS_GROUP_ID" "$USER_789" "Traders" "trader_only_user"

echo ""

# ============================================================================
# Fetch Theme and Flow IDs (optional)
# ============================================================================

log_info "Fetching Classic theme and default flows..."

CLASSIC_THEME_ID=""
RESPONSE=$(thunder_api_call GET "/design/themes")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "200" ]]; then
    CLASSIC_THEME_ID=$(echo "$BODY" | grep -o '{[^}]*"displayName":"Classic"[^}]*}' | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [[ -n "$CLASSIC_THEME_ID" ]]; then
        log_success "Found Classic theme ID: $CLASSIC_THEME_ID"
    else
        log_warning "Classic theme not found; app creation will continue without theme_id"
    fi
else
    log_warning "Failed to fetch themes (HTTP $HTTP_CODE); app creation will continue without theme_id"
fi

AUTH_FLOW_ID=$(get_flow_id_by_handle "AUTHENTICATION" "default-basic-flow")
REG_FLOW_ID=$(get_flow_id_by_handle "REGISTRATION" "default-basic-flow")

if [[ -n "$AUTH_FLOW_ID" ]]; then
    log_success "Found default authentication flow ID: $AUTH_FLOW_ID"
else
    log_warning "Default authentication flow not found; app creation will continue without auth_flow_id"
fi

if [[ -n "$REG_FLOW_ID" ]]; then
    log_success "Found default registration flow ID: $REG_FLOW_ID"
else
    log_warning "Default registration flow not found; app creation will continue without registration_flow_id"
fi

echo ""

# ============================================================================
# Create 4 SPA Applications
# ============================================================================

create_spa_application "TraderApp" "Application for trader portal built with React" "TRADER_PORTAL_APP" "5173" "Private_User"
create_spa_application "NPQSPortalApp" "Application for NPQS portal built with React" "OGA_PORTAL_APP_NPQS" "5174" "Government_User"
create_spa_application "FCAUPortalApp" "Application for FCAU portal built with React" "OGA_PORTAL_APP_FCAU" "5175" "Government_User"
create_spa_application "IRDPortalApp" "Application for IRD portal built with React" "OGA_PORTAL_APP_IRD" "5176" "Government_User"

echo ""

# ============================================================================
# Resolve Default Organization Unit
# ============================================================================

DEFAULT_OU_HANDLE="default"
log_info "Resolving Default organization unit..."

RESPONSE=$(thunder_api_call GET "/organization-units/tree/${DEFAULT_OU_HANDLE}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "200" ]]; then
    DEFAULT_OU_ID=$(extract_first_id "$BODY")
else
    log_error "Failed to resolve Default organization unit (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$DEFAULT_OU_ID" ]]; then
    log_error "Could not determine Default organization unit ID"
    exit 1
fi

log_info "Default organization unit ID: ${DEFAULT_OU_ID}"

echo ""

# ============================================================================
# Create 3 M2M Applications
# ============================================================================

create_m2m_application "NPQS_TO_NSW_M2M" "Machine-to-machine integration for NPQS to NSW" "NPQS_TO_NSW" "${NPQS_M2M_CLIENT_SECRET}"
NPQS_TO_NSW_M2M_APP_ID="$CREATED_M2M_APP_ID"

create_m2m_application "FCAU_TO_NSW_M2M" "Machine-to-machine integration for FCAU to NSW" "FCAU_TO_NSW" "${FCAU_M2M_CLIENT_SECRET}"
FCAU_TO_NSW_M2M_APP_ID="$CREATED_M2M_APP_ID"

create_m2m_application "IRD_TO_NSW_M2M" "Machine-to-machine integration for IRD to NSW" "IRD_TO_NSW" "${IRD_M2M_CLIENT_SECRET}"
IRD_TO_NSW_M2M_APP_ID="$CREATED_M2M_APP_ID"

echo ""

# ============================================================================
# Summary
# ============================================================================

log_success "Sample resources setup completed successfully!"
log_info "Private Sector OU path: ${PRIVATE_SECTOR_OU_HANDLE}"
log_info "ABCD Traders OU path: ${ABCD_TRADERS_OU_PATH}"
log_info "Government Organization OU path: ${GOVERNMENT_ORG_OU_HANDLE}"
log_info "Government child OUs: ${NPQS_OU_HANDLE}, ${FCAU_OU_HANDLE}, ${IRD_OU_HANDLE}"
log_info "Private user type: Private_User"
log_info "Government user type: Government_User"
log_info "Traders group -> Trader role"
log_info "CHA group -> CHA role"
log_info "both_roles_user in groups: Traders, CHA"
log_info "cha_only_user in groups: CHA"
log_info "trader_only_user in groups: Traders"
log_info "Government users: npqs_user, fcau_user, ird_user"
log_info "App client IDs: TRADER_PORTAL_APP, OGA_PORTAL_APP_NPQS, OGA_PORTAL_APP_FCAU, OGA_PORTAL_APP_IRD"
log_info "M2M client IDs: NPQS_TO_NSW, FCAU_TO_NSW, IRD_TO_NSW"
log_info "M2M auth method: client_secret_basic"
echo ""
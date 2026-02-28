set -e

# Source common functions from the same directory as this script
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]:-$0}")"
source "${SCRIPT_DIR}/common.sh"

log_info "Creating sample Thunder resources..."
echo ""

# ============================================================================
# Create Traders Organization Unit
# ============================================================================

TRADER_OU_HANDLE="traders"

log_info "Creating Traders organization unit..."

read -r -d '' TRADERS_OU_PAYLOAD <<JSON || true
{
  "handle": "${TRADER_OU_HANDLE}",
  "name": "Traders",
  "description": "Organization unit for trader accounts"
}
JSON

RESPONSE=$(thunder_api_call POST "/organization-units" "${TRADERS_OU_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "Traders organization unit created successfully"
    TRADER_OU_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "Traders organization unit already exists, retrieving ID..."
    # Get existing OU ID by handle to ensure we get the correct "traders" OU
    RESPONSE=$(thunder_api_call GET "/organization-units/tree/${TRADER_OU_HANDLE}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]]; then
        TRADER_OU_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    else
        log_error "Failed to fetch organization unit by handle '${TRADER_OU_HANDLE}' (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi
else
    log_error "Failed to create Traders organization unit (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

if [[ -z "$TRADER_OU_ID" ]]; then
    log_error "Could not determine Traders organization unit ID"
    exit 1
fi

log_info "Traders OU ID: $TRADER_OU_ID"

echo ""

create_oga_resources() {
    local OGA_CODE="$1"
    local OGA_OU_HANDLE="$2"
    local OGA_OU_NAME="$3"
    local OGA_USER_TYPE="$4"
    local OGA_APP_NAME="$5"
    local OGA_APP_CLIENT_ID="$6"
    local OGA_APP_PORT="$7"
    local OGA_SAMPLE_USERNAME="$8"
    local OGA_SAMPLE_PASSWORD="$9"
    local OGA_SAMPLE_EMAIL="${10}"
    local OGA_SAMPLE_GIVEN_NAME="${11}"
    local OGA_SAMPLE_FAMILY_NAME="${12}"

    log_info "Creating ${OGA_CODE} organization unit..."

    local OGA_OU_ID=""
    read -r -d '' OGA_OU_PAYLOAD <<JSON || true
{
  "handle": "${OGA_OU_HANDLE}",
  "name": "${OGA_OU_NAME}",
  "description": "Organization unit for ${OGA_OU_NAME}"
}
JSON

    RESPONSE=$(thunder_api_call POST "/organization-units" "${OGA_OU_PAYLOAD}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
        log_success "${OGA_CODE} organization unit created successfully"
        OGA_OU_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    elif [[ "$HTTP_CODE" == "409" ]]; then
        log_warning "${OGA_CODE} organization unit already exists, retrieving ID..."
        RESPONSE=$(thunder_api_call GET "/organization-units/tree/${OGA_OU_HANDLE}")
        HTTP_CODE="${RESPONSE: -3}"
        BODY="${RESPONSE%???}"

        if [[ "$HTTP_CODE" == "200" ]]; then
            OGA_OU_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        else
            log_error "Failed to fetch organization unit by handle '${OGA_OU_HANDLE}' (HTTP $HTTP_CODE)"
            echo "Response: $BODY"
            exit 1
        fi
    else
        log_error "Failed to create ${OGA_CODE} organization unit (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi

    if [[ -z "$OGA_OU_ID" ]]; then
        log_error "Could not determine ${OGA_CODE} organization unit ID"
        exit 1
    fi

    log_info "${OGA_CODE} OU ID: $OGA_OU_ID"
    echo ""

    log_info "Creating ${OGA_CODE} user type..."

    read -r -d '' OGA_USER_TYPE_PAYLOAD <<JSON || true
{
  "name": "${OGA_USER_TYPE}",
  "ouId": "${OGA_OU_ID}",
  "allowSelfRegistration": false,
  "schema": {
    "username": {
      "type": "string",
      "required": true,
      "unique": true
    },
    "email": {
      "type": "string",
      "required": true,
      "unique": true
    },
    "given_name": {
      "type": "string",
      "required": false
    },
    "family_name": {
      "type": "string",
      "required": false
    }
  }
}
JSON

    RESPONSE=$(thunder_api_call POST "/user-schemas" "${OGA_USER_TYPE_PAYLOAD}")
    HTTP_CODE="${RESPONSE: -3}"

    if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
        log_success "${OGA_CODE} user type created successfully"
    elif [[ "$HTTP_CODE" == "409" ]]; then
        log_warning "${OGA_CODE} user type already exists, skipping"
    else
        log_error "Failed to create ${OGA_CODE} user type (HTTP $HTTP_CODE)"
        exit 1
    fi

    echo ""
    log_info "Creating ${OGA_CODE} portal application..."

    read -r -d '' OGA_PORTAL_APP_PAYLOAD <<JSON || true
{
    "name": "${OGA_APP_NAME}",
    "description": "Application for ${OGA_OU_NAME} portal built with React",
    ${THEME_ID_FIELD}
    ${AUTH_FLOW_FIELD}
    ${REG_FLOW_FIELD}
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
                "client_id": "${OGA_APP_CLIENT_ID}",
                "redirect_uris": [
                    "http://localhost:${OGA_APP_PORT}",
                    "https://localhost:${OGA_APP_PORT}"
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
                        "validity_period": 3600,
                        "user_attributes": [
                            "email",
                            "family_name",
                            "given_name",
                            "ouHandle",
                            "ouId",
                            "ouName",
                            "username"
                        ]
                    },
                    "id_token": {
                        "validity_period": 3600,
                        "user_attributes": [
                            "family_name",
                            "given_name",
                            "email"
                        ]
                    }
                },
                "scopes": [
                    "openid",
                    "profile",
                    "email"
                ],
                "user_info": {
                    "user_attributes": [
                        "family_name",
                        "given_name",
                        "email"
                    ]
                }
            }
        }
    ],
    "allowed_user_types": [
        "${OGA_USER_TYPE}"
    ]
}
JSON

    RESPONSE=$(thunder_api_call POST "/applications" "${OGA_PORTAL_APP_PAYLOAD}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]] || [[ "$HTTP_CODE" == "202" ]]; then
        log_success "${OGA_CODE} portal application created successfully"
        local OGA_APP_ID
        local OGA_CLIENT_ID
        OGA_APP_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        OGA_CLIENT_ID=$(echo "$BODY" | grep -o '"client_id":"[^"]*"' | head -1 | cut -d'"' -f4)

        if [[ -n "$OGA_APP_ID" ]]; then
            log_info "${OGA_CODE} portal app ID: $OGA_APP_ID"
        else
            log_warning "Could not extract ${OGA_CODE} portal app ID from response"
        fi

        if [[ -n "$OGA_CLIENT_ID" ]]; then
            log_info "${OGA_CODE} portal client ID: $OGA_CLIENT_ID"
        else
            log_warning "Could not extract ${OGA_CODE} portal client ID from response"
        fi
    elif [[ "$HTTP_CODE" == "409" ]]; then
        log_warning "${OGA_CODE} portal application already exists, skipping"
    elif [[ "$HTTP_CODE" == "400" ]] && [[ "$BODY" =~ (Application already exists|APP-1022) ]]; then
        log_warning "${OGA_CODE} portal application already exists, skipping"
    else
        log_error "Failed to create ${OGA_CODE} portal application (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi

    echo ""
    log_info "Creating sample ${OGA_CODE} user..."

    read -r -d '' OGA_SAMPLE_USER_PAYLOAD <<JSON || true
{
    "type": "${OGA_USER_TYPE}",
    "organizationUnit": "${OGA_OU_ID}",
    "attributes": {
        "username": "${OGA_SAMPLE_USERNAME}",
        "password": "${OGA_SAMPLE_PASSWORD}",
        "sub": "${OGA_SAMPLE_USERNAME}",
        "email": "${OGA_SAMPLE_EMAIL}",
        "email_verified": true,
        "name": "Sample ${OGA_OU_NAME} Officer",
        "given_name": "${OGA_SAMPLE_GIVEN_NAME}",
        "family_name": "${OGA_SAMPLE_FAMILY_NAME}"
    }
}
JSON

    RESPONSE=$(thunder_api_call POST "/users" "${OGA_SAMPLE_USER_PAYLOAD}")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
        log_success "Sample ${OGA_CODE} user created successfully"
        log_info "Username: ${OGA_SAMPLE_USERNAME}"
        log_info "Password: ${OGA_SAMPLE_PASSWORD}"
    elif [[ "$HTTP_CODE" == "409" ]]; then
        log_warning "Sample ${OGA_CODE} user already exists, skipping"
    else
        log_error "Failed to create sample ${OGA_CODE} user (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        exit 1
    fi

    echo ""
}

# ============================================================================
# Create Trader User Type
# ============================================================================

log_info "Creating Trader user type..."

read -r -d '' TRADER_USER_TYPE_PAYLOAD <<JSON || true
{
  "name": "Trader",
  "ouId": "${TRADER_OU_ID}",
  "allowSelfRegistration": true,
  "schema": {
    "username": {
      "type": "string",
      "required": true,
      "unique": true
    },
    "email": {
      "type": "string",
      "required": true,
      "unique": true
    },
    "given_name": {
      "type": "string",
      "required": false
    },
    "family_name": {
      "type": "string",
      "required": false
    }
  }
}
JSON

RESPONSE=$(thunder_api_call POST "/user-schemas" "${TRADER_USER_TYPE_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "Trader user type created successfully"
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "Trader user type already exists, skipping"
else
    log_error "Failed to create Trader user type (HTTP $HTTP_CODE)"
    exit 1
fi

echo ""

# ============================================================================
# Fetch Classic Theme ID
# ============================================================================
log_info "Fetching Classic theme..."

CLASSIC_THEME_ID=""
RESPONSE=$(thunder_api_call GET "/design/themes")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "200" ]]; then
    # Extract theme ID for "Classic" theme by displayName
    # Parse JSON to find theme with displayName "Classic"
    CLASSIC_THEME_ID=$(echo "$BODY" | grep -o '{[^}]*"displayName":"Classic"[^}]*}' | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [[ -n "$CLASSIC_THEME_ID" ]]; then
        log_success "Found Classic theme with ID: $CLASSIC_THEME_ID"
    else
        log_warning "Classic theme not found, will use default theme"
    fi
else
    log_warning "Failed to fetch themes (HTTP $HTTP_CODE), will use default theme"
fi

echo ""

# ============================================================================
# Fetch Default Authentication and Registration Flows
# ============================================================================
log_info "Fetching default authentication and registration flows..."

AUTH_FLOW_ID=""
REG_FLOW_ID=""

# Fetch authentication flow (default-basic-flow)
RESPONSE=$(thunder_api_call GET "/flows?limit=30&offset=0&flowType=AUTHENTICATION")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "200" ]]; then
    # Extract flow ID for "default-basic-flow" by handle
    AUTH_FLOW_ID=$(echo "$BODY" | grep -o '{[^}]*"handle":"default-basic-flow"[^}]*}' | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [[ -n "$AUTH_FLOW_ID" ]]; then
        log_success "Found default authentication flow with ID: $AUTH_FLOW_ID"
    else
        log_warning "Default authentication flow not found"
    fi
else
    log_warning "Failed to fetch authentication flows (HTTP $HTTP_CODE)"
fi

# Fetch registration flow (default-basic-flow)
RESPONSE=$(thunder_api_call GET "/flows?limit=30&offset=0&flowType=REGISTRATION")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "200" ]]; then
    # Extract flow ID for "default-basic-flow" by handle
    REG_FLOW_ID=$(echo "$BODY" | grep -o '{[^}]*"handle":"default-basic-flow"[^}]*}' | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [[ -n "$REG_FLOW_ID" ]]; then
        log_success "Found default registration flow with ID: $REG_FLOW_ID"
    else
        log_warning "Default registration flow not found"
    fi
else
    log_warning "Failed to fetch registration flows (HTTP $HTTP_CODE)"
fi

echo ""

# ============================================================================
# Create Trader Portal React Application
# ============================================================================
log_info "Creating Trader Portal React App application..."

# Build theme_id field conditionally
THEME_ID_FIELD=""
if [[ -n "$CLASSIC_THEME_ID" ]]; then
    THEME_ID_FIELD="\"theme_id\": \"${CLASSIC_THEME_ID}\","
fi

# Build auth_flow_id and registration_flow_id fields conditionally
AUTH_FLOW_FIELD=""
if [[ -n "$AUTH_FLOW_ID" ]]; then
    AUTH_FLOW_FIELD="\"auth_flow_id\": \"${AUTH_FLOW_ID}\","
fi

REG_FLOW_FIELD=""
if [[ -n "$REG_FLOW_ID" ]]; then
    REG_FLOW_FIELD="\"registration_flow_id\": \"${REG_FLOW_ID}\","
fi

read -r -d '' TRADER_PORTAL_APP_PAYLOAD <<JSON || true
{
    "name": "TraderApp",
    "description": "Application for trader portal built with React",
    ${THEME_ID_FIELD}
    ${AUTH_FLOW_FIELD}
    ${REG_FLOW_FIELD}
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
                "client_id": "TRADER_PORTAL_APP",
                "redirect_uris": [
                    "http://localhost:5173",
                    "https://localhost:5173"
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
                        "validity_period": 3600,
                        "user_attributes": [
                            "email",
                            "family_name",
                            "given_name"
                        ]
                    },
                    "id_token": {
                        "validity_period": 3600,
                        "user_attributes": [
                            "family_name",
                            "given_name",
                            "email"
                        ]
                    }
                },
                "scopes": [
                    "openid",
                    "profile",
                    "email"
                ],
                "user_info": {
                    "user_attributes": [
                        "family_name",
                        "given_name",
                        "email"
                    ]
                }
            }
        }
    ],
    "allowed_user_types": [
        "Trader"
    ]
}
JSON

RESPONSE=$(thunder_api_call POST "/applications" "${TRADER_PORTAL_APP_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"

if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]] || [[ "$HTTP_CODE" == "202" ]]; then
    log_success "Trader Portal React App created successfully"
    TRADER_PORTAL_APP_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    # Extract client_id from the OAuth2 config in the response
    TRADER_PORTAL_CLIENT_ID=$(echo "$BODY" | grep -o '"client_id":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [[ -n "$TRADER_PORTAL_APP_ID" ]]; then
        log_info "Trader Portal React App ID: $TRADER_PORTAL_APP_ID"
    else
        log_warning "Could not extract Trader Portal React App ID from response"
    fi
    
    if [[ -n "$TRADER_PORTAL_CLIENT_ID" ]]; then
        log_info "Trader Portal Client ID: $TRADER_PORTAL_CLIENT_ID"
        log_info "Update your trader-app .env file with: VITE_CLIENT_ID=$TRADER_PORTAL_CLIENT_ID"
    else
        log_warning "Could not extract Trader Portal Client ID from response"
    fi
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "Trader Portal React App already exists, skipping"
elif [[ "$HTTP_CODE" == "400" ]] && [[ "$BODY" =~ (Application already exists|APP-1022) ]]; then
    log_warning "Trader Portal React App already exists, skipping"
else
    log_error "Failed to create Trader Portal React App (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

echo ""
# ============================================================================
# Add a sample user of user type Trader
# ============================================================================
log_info "Creating sample trader user..."
read -r -d '' SAMPLE_USER_PAYLOAD <<JSON || true
{
    "type": "Trader",
    "organizationUnit": "${TRADER_OU_ID}",
    "attributes": {
        "username": "user123",
        "password": "1234",
        "sub": "user123",
        "email": "user123@trader.dev",
        "email_verified": true,
        "name": "Sample Trader",
        "given_name": "User",
        "family_name": "Trader"
    }
}
JSON

RESPONSE=$(thunder_api_call POST "/users" "${SAMPLE_USER_PAYLOAD}")
HTTP_CODE="${RESPONSE: -3}"
BODY="${RESPONSE%???}"
if [[ "$HTTP_CODE" == "201" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log_success "Sample trader user created successfully"
    log_info "Username: user123"
    log_info "Password: 1234"

    SAMPLE_TRADER_USER_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [[ -z "$SAMPLE_TRADER_USER_ID" ]]; then
        log_warning "Could not extract sample trader user ID from response"
    else
        log_info "Sample trader user ID: $SAMPLE_TRADER_USER_ID"
    fi
elif [[ "$HTTP_CODE" == "409" ]]; then
    log_warning "Sample trader user already exists, retrieving user ID..."

    RESPONSE=$(thunder_api_call GET "/users")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"

    if [[ "$HTTP_CODE" == "200" ]]; then
        SAMPLE_TRADER_USER_ID=$(echo "$BODY" | grep -o '"id":"[^"]*","[^"]*":"[^"]*","attributes":{[^}]*"username":"user123"' | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

        if [[ -z "$SAMPLE_TRADER_USER_ID" ]]; then
            SAMPLE_TRADER_USER_ID=$(echo "$BODY" | sed 's/},{/}\n{/g' | grep '"username":"user123"' | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        fi

        if [[ -n "$SAMPLE_TRADER_USER_ID" ]]; then
            log_success "Found sample trader user ID: $SAMPLE_TRADER_USER_ID"
        else
            log_error "Could not find sample trader user in response"
            exit 1
        fi
    else
        log_error "Failed to fetch users (HTTP $HTTP_CODE)"
        exit 1
    fi
else
    log_error "Failed to create sample trader user (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi

echo ""

# ============================================================================
# Create NPQS and FCAU resources
# ============================================================================

create_oga_resources \
    "NPQS" \
    "npqs" \
    "National Plant Quarantine Service" \
    "NPQSOfficer" \
    "NPQSPortalApp" \
    "OGA_PORTAL_APP_NPQS" \
    "5174" \
    "npqs_officer" \
    "1234" \
    "npqs_officer@oga.dev" \
    "NPQS" \
    "Officer"

create_oga_resources \
    "FCAU" \
    "fcau" \
    "Food Control Administration Unit" \
    "FCAUOfficer" \
    "FCAUPortalApp" \
    "OGA_PORTAL_APP_FCAU" \
    "5175" \
    "fcau_officer" \
    "1234" \
    "fcau_officer@oga.dev" \
    "FCAU" \
    "Officer"

# ============================================================================
# Summary
# ============================================================================

log_success "Sample resources setup completed successfully!"
log_info "Trader app client ID: TRADER_PORTAL_APP"
log_info "NPQS app client ID: OGA_PORTAL_APP_NPQS"
log_info "FCAU app client ID: OGA_PORTAL_APP_FCAU"
echo ""

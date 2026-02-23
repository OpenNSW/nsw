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
# Create Trader Portal React Application
# ============================================================================
log_info "Creating Trader Portal React App application..."

# Build theme_id field conditionally
THEME_ID_FIELD=""
if [[ -n "$CLASSIC_THEME_ID" ]]; then
    THEME_ID_FIELD="\"theme_id\": \"${CLASSIC_THEME_ID}\","
fi

read -r -d '' TRADER_PORTAL_APP_PAYLOAD <<JSON || true
{
    "name": "TraderApp",
    "description": "Application for trader portal built with React",
    ${THEME_ID_FIELD}
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
                    "http://localhost:5173"
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
# Summary
# ============================================================================

log_success "Sample resources setup completed successfully!"
echo ""

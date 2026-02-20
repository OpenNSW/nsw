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
# Create Trader Portal React Application
# ============================================================================
log_info "Creating Trader Portal React App application..."

read -r -d '' TRADER_PORTAL_APP_PAYLOAD <<JSON || true
{
    "name": "Trader Portal",
    "description": "React application for trader portal",
    "logo_url": "https://ssl.gstatic.com/docs/common/profile/tiger_lg.png",
    "user_attributes": [
        "given_name",
        "family_name",
        "email",
        "groups"
    ],
    "is_registration_flow_enabled": true,
    "allowed_user_types": [
        "Trader"
    ],
    "template": "react",
    "inbound_auth_config": [
        {
            "type": "oauth2",
            "config": {
                "client_id": "TRADER_PORTAL_APP",
                "public_client": true,
                "pkce_required": true,
                "grant_types": [
                    "authorization_code",
                    "refresh_token"
                ],
                "response_types": [
                    "code"
                ],
                "redirect_uris": [
                    "http://localhost:5173"
                ],
                "token_endpoint_auth_method": "none",
                "scopes": [
                    "openid",
                    "profile",
                    "email"
                ]
            }
        }
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
# Summary
# ============================================================================

log_success "Sample resources setup completed successfully!"
echo ""

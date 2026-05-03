#!/bin/bash

# ============================================================================
# Identity Provider (Thunder) Common Utility Functions
# ============================================================================

THUNDER_URL="${THUNDER_URL:-https://localhost:8090}"

# --- Logging Utilities ---

log_info() {
    echo -e "[\033[34mINFO\033[0m] $1" >&2
}

log_success() {
    echo -e "[\033[32mSUCCESS\033[0m] $1" >&2
}

log_error() {
    echo -e "[\033[31mERROR\033[0m] $1" >&2
}

log_warning() {
    echo -e "[\033[33mWARNING\033[0m] $1" >&2
}

# --- API Utilities ---

thunder_api_call() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    local url="${THUNDER_URL}${endpoint}"

    if [[ -z "$data" ]]; then
        curl -k -s -w "\n%{http_code}" -X "$method" "$url" \
            -H "Content-Type: application/json"
    else
        curl -k -s -w "\n%{http_code}" -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data"
    fi
}

extract_id() {
    echo "$1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

# --- Flow Management ---

create_flow() {
    local flow_file="$1"
    if [[ ! -f "$flow_file" ]]; then
        log_error "Flow file not found: $flow_file"
        return 1
    fi

    local flow_payload=$(cat "$flow_file")
    local flow_handle=$(echo "$flow_payload" | grep -o '"handle"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | cut -d'"' -f4)
    local flow_name=$(echo "$flow_payload" | grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | cut -d'"' -f4)

    log_info "Creating flow: ${flow_name} (${flow_handle})"
    
    local response=$(thunder_api_call POST "/flows" "$flow_payload")
    local http_code="${response: -3}"
    local body="${response%???}"

    if [[ "$http_code" == "201" ]] || [[ "$http_code" == "200" ]]; then
        local flow_id=$(extract_id "$body")
        log_success "Flow created successfully (ID: $flow_id)"
        echo "$flow_id"
        return 0
    elif [[ "$http_code" == "409" ]]; then
        log_warning "Flow '${flow_handle}' already exists"
        return 2
    else
        log_error "Failed to create flow (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

update_flow() {
    local flow_id="$1"
    local flow_file="$2"
    if [[ ! -f "$flow_file" ]]; then
        log_error "Flow file not found: $flow_file"
        return 1
    fi

    local flow_payload=$(cat "$flow_file")
    local response=$(thunder_api_call PUT "/flows/${flow_id}" "$flow_payload")
    local http_code="${response: -3}"
    local body="${response%???}"

    if [[ "$http_code" == "200" ]]; then
        log_success "Flow updated successfully"
        return 0
    else
        log_error "Failed to update flow (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

get_flow_id_by_handle() {
    local flow_type="$1"
    local flow_handle="$2"
    
    local response=$(thunder_api_call GET "/flows?limit=200&flowType=${flow_type}")
    local http_code="${response: -3}"
    local body="${response%???}"

    if [[ "$http_code" == "200" ]]; then
        # Parse JSON manually to find matching handle
        echo "$body" | sed 's/},{/}\n{/g' | grep "\"handle\":\"${flow_handle}\"" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
    else
        return 1
    fi
}

#!/bin/sh
set -e
cat > /usr/share/nginx/html/config.js << EOF
window.__APP_CONFIG__ = {
  VITE_API_BASE_URL: '${VITE_API_BASE_URL:-}',
  VITE_IDP_CLIENT_ID: '${VITE_IDP_CLIENT_ID:-}',
  VITE_IDP_BASE_URL: '${VITE_IDP_BASE_URL:-}',
  VITE_IDP_PLATFORM: '${VITE_IDP_PLATFORM:-}',
  VITE_IDP_SCOPES: '${VITE_IDP_SCOPES:-}',
  VITE_APP_URL: '${VITE_APP_URL:-}',
  VITE_SHOW_AUTOFILL_BUTTON: '${VITE_SHOW_AUTOFILL_BUTTON:-}'
};
EOF
exec nginx -g 'daemon off;'

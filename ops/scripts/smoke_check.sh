#!/bin/zsh
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:8080}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

require_cmd curl
require_cmd jq

json_get() {
  local url="$1"
  shift || true
  command curl -fsS "$url" "$@"
}

echo "[smoke] checking runtime config"
RUNTIME_JSON="$(json_get "$API_BASE/api/v1/runtime/config")"
printf '%s\n' "$RUNTIME_JSON" | jq '{app_name: .data.app_name, run_mode: .data.run_mode, frontend: .data.frontend.mode, health: .data.health.status}'

echo "[smoke] checking frontend entry scripts"
BOOT_JS_STATUS="$(command curl -fsS -o /dev/null -w '%{http_code}' "$API_BASE/src/boot/main.js")"
DELIVERY_JS_STATUS="$(command curl -fsS -o /dev/null -w '%{http_code}' "$API_BASE/src/domains/delivery/index.js")"
MACHINES_JS_STATUS="$(command curl -fsS -o /dev/null -w '%{http_code}' "$API_BASE/src/domains/machines/index.js")"
if [[ "$BOOT_JS_STATUS" != "200" || "$DELIVERY_JS_STATUS" != "200" || "$MACHINES_JS_STATUS" != "200" ]]; then
  echo "frontend script check failed: boot=$BOOT_JS_STATUS delivery=$DELIVERY_JS_STATUS machines=$MACHINES_JS_STATUS" >&2
  exit 1
fi

echo "[smoke] logging in"
LOGIN_JSON="$(command curl -fsS -X POST "$API_BASE/api/v1/auth/login/local" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123456"}')"
TOKEN="$(printf '%s\n' "$LOGIN_JSON" | jq -r '.data.token')"
if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "login failed: token missing" >&2
  exit 1
fi

auth_get() {
  local api_path="$1"
  json_get "$API_BASE$api_path" -H "Authorization: Bearer $TOKEN"
}

echo "[smoke] checking system module"
USERS_JSON="$(auth_get "/api/v1/users")"
ROLES_JSON="$(auth_get "/api/v1/roles")"
SETTINGS_JSON="$(auth_get "/api/v1/system-settings")"

echo "[smoke] checking overview bootstrap"
OVERVIEW_USERS_COUNT="$(printf '%s\n' "$USERS_JSON" | jq -r '(.data // []) | length')"
if [[ "$OVERVIEW_USERS_COUNT" == "0" ]]; then
  echo "overview bootstrap failed: users list empty" >&2
  exit 1
fi

echo "[smoke] checking cloud module"
CLOUD_SUMMARY_JSON="$(auth_get "/api/v1/cloud/summary")"
CLOUD_ACCOUNTS_JSON="$(auth_get "/api/v1/cloud/accounts")"
CLOUD_NETWORKS_JSON="$(auth_get "/api/v1/cloud/network-plans")"
CLOUD_BLUEPRINTS_JSON="$(auth_get "/api/v1/cloud/blueprints")"
CLOUD_JOBS_JSON="$(auth_get "/api/v1/cloud/jobs")"

echo "[smoke] checking k8s module"
CLUSTERS_JSON="$(auth_get "/api/v1/k8s/clusters")"
NAMESPACES_JSON="$(auth_get "/api/v1/k8s/namespaces")"
WORKLOADS_JSON="$(auth_get "/api/v1/k8s/workloads")"

echo "[smoke] checking machine module"
MACHINE_SUMMARY_JSON="$(auth_get "/api/v1/machines/summary")"
MACHINE_GROUPS_JSON="$(auth_get "/api/v1/machines/groups")"
MACHINE_ASSETS_JSON="$(auth_get "/api/v1/machines/assets?page=1&page_size=10")"

echo "[smoke] summary"
jq -n \
  --arg users "$(printf '%s\n' "$USERS_JSON" | jq -r '(.data // []) | length')" \
  --arg roles "$(printf '%s\n' "$ROLES_JSON" | jq -r '(.data // []) | length')" \
  --arg settings "$(printf '%s\n' "$SETTINGS_JSON" | jq -r '(.data // []) | length')" \
  --arg cloud_accounts "$(printf '%s\n' "$CLOUD_ACCOUNTS_JSON" | jq -r '(.data // []) | length')" \
  --arg cloud_networks "$(printf '%s\n' "$CLOUD_NETWORKS_JSON" | jq -r '(.data // []) | length')" \
  --arg cloud_blueprints "$(printf '%s\n' "$CLOUD_BLUEPRINTS_JSON" | jq -r '(.data // []) | length')" \
  --arg cloud_jobs "$(printf '%s\n' "$CLOUD_JOBS_JSON" | jq -r '(.data // []) | length')" \
  --arg cloud_queued_jobs "$(printf '%s\n' "$CLOUD_SUMMARY_JSON" | jq -r '.data.queued_jobs // 0')" \
  --arg frontend_boot_js "$BOOT_JS_STATUS" \
  --arg frontend_delivery_js "$DELIVERY_JS_STATUS" \
  --arg frontend_machines_js "$MACHINES_JS_STATUS" \
  --arg clusters "$(printf '%s\n' "$CLUSTERS_JSON" | jq -r '(.data // []) | length')" \
  --arg namespaces "$(printf '%s\n' "$NAMESPACES_JSON" | jq -r '(.data // []) | length')" \
  --arg workloads "$(printf '%s\n' "$WORKLOADS_JSON" | jq -r '(.data // []) | length')" \
  --arg machine_groups "$(printf '%s\n' "$MACHINE_GROUPS_JSON" | jq -r '(.data // []) | length')" \
  --arg machine_assets "$(printf '%s\n' "$MACHINE_ASSETS_JSON" | jq -r '(.data.items // []) | length')" \
  --arg machine_total_assets "$(printf '%s\n' "$MACHINE_SUMMARY_JSON" | jq -r '.data.total_assets // 0')" \
  '{
    ok: true,
    system: {
      users: ($users | tonumber),
      roles: ($roles | tonumber),
      settings: ($settings | tonumber)
    },
    cloud: {
      accounts: ($cloud_accounts | tonumber),
      networks: ($cloud_networks | tonumber),
      blueprints: ($cloud_blueprints | tonumber),
      jobs: ($cloud_jobs | tonumber),
      queued_jobs: ($cloud_queued_jobs | tonumber)
    },
    frontend: {
      boot_js: ($frontend_boot_js | tonumber),
      delivery_js: ($frontend_delivery_js | tonumber),
      machines_js: ($frontend_machines_js | tonumber)
    },
    k8s: {
      clusters: ($clusters | tonumber),
      namespaces: ($namespaces | tonumber),
      workloads: ($workloads | tonumber)
    },
    machine: {
      groups: ($machine_groups | tonumber),
      list_page_items: ($machine_assets | tonumber),
      total_assets: ($machine_total_assets | tonumber)
    }
  }'

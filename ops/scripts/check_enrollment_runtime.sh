#!/usr/bin/env bash
set -euo pipefail

SCRIPT_PATH="$0"
ROOT_DIR="$(cd "$(dirname "$SCRIPT_PATH")/.." && pwd)"
BACKEND_BASE_URL="${BACKEND_API_BASE_URL:-http://127.0.0.1:18080}"
BACKEND_TOKEN="${BACKEND_API_TOKEN:-runner-dev-token}"

pass() { printf '[PASS] %s\n' "$1"; }
warn() { printf '[WARN] %s\n' "$1"; }
fail() { printf '[FAIL] %s\n' "$1"; }

check_command() {
  local name="$1"
  if command -v "$name" >/dev/null 2>&1; then
    pass "found command: $name"
    return 0
  fi
  warn "missing command: $name"
  return 1
}

check_env() {
  local key="$1"
  local value
  value="$(printenv "$key" 2>/dev/null || true)"
  if [[ -n "$value" ]]; then
    pass "env present: $key"
    return 0
  fi
  warn "env missing: $key"
  return 1
}

check_file() {
  local path="$1"
  if [[ -f "$path" ]]; then
    pass "file present: $path"
    return 0
  fi
  warn "file missing: $path"
  return 1
}

printf '== Platform Center Enrollment Runtime Check ==\n'
printf 'root: %s\n' "$ROOT_DIR"
printf 'backend: %s\n' "$BACKEND_BASE_URL"
printf '\n'

printf '1. Preferred runtime\n'
pass "P0 real provider enrollment should run cluster-enrollment-worker on host process first"
warn "compose container exists, but current image does not bundle aws/aliyun CLI; use it only after provider runtime is prepared"
printf '\n'

printf '2. Backend reachability\n'
if curl -fsS "$BACKEND_BASE_URL/healthz" >/dev/null 2>&1; then
  pass "backend healthz reachable"
else
  fail "backend healthz unreachable at $BACKEND_BASE_URL"
fi
if [[ -n "$BACKEND_TOKEN" ]]; then
  pass "backend internal token configured"
else
  fail "backend internal token missing"
fi
printf '\n'

printf '3. Worker config files\n'
check_file "$ROOT_DIR/cluster-enrollment-worker/config/cluster-enrollment-worker.yaml" || true
check_file "$ROOT_DIR/machine-enrollment-worker/config/machine-enrollment-worker.yaml" || true
printf '\n'

printf '4. AWS provider prerequisites\n'
aws_ok=0
check_command aws && aws_ok=1 || true
check_env AWS_REGION || true
check_env AWS_DEFAULT_REGION || true
check_env AWS_ACCESS_KEY_ID || true
check_env AWS_SECRET_ACCESS_KEY || true
check_file "${HOME}/.aws/credentials" || true
if [[ "$aws_ok" -eq 1 ]]; then
  warn "run: aws sts get-caller-identity"
  warn "run: aws eks describe-cluster --name <cluster> --region <region>"
fi
printf '\n'

printf '5. Alicloud provider prerequisites\n'
aliyun_ok=0
check_command aliyun && aliyun_ok=1 || true
check_env ALICLOUD_REGION_ID || true
check_env ALIBABA_CLOUD_ACCESS_KEY_ID || true
check_env ALIBABA_CLOUD_ACCESS_KEY_SECRET || true
check_file "${HOME}/.aliyun/config.json" || true
if [[ "$aliyun_ok" -eq 1 ]]; then
  warn "run: aliyun sts GetCallerIdentity"
  warn "run: aliyun cs GET /clusters/<cluster-id> --region <region> --output json"
fi
printf '\n'

printf '6. Launch commands\n'
cat <<'EOF'
Host runtime:
cd /Users/alex/ops/platform-center/apps/cluster-enrollment-worker
  BACKEND_API_BASE_URL=http://127.0.0.1:18080 \
  BACKEND_API_TOKEN=runner-dev-token \
  go run ./cmd/cluster-enrollment-worker

Compose runtime:
  cd /Users/alex/ops/platform-center
  docker compose up -d --build cluster-enrollment-worker machine-enrollment-worker
EOF

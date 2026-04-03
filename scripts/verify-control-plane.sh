#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"
ADMIN_TOKEN="${ADMIN_TOKEN:-change-me}"
READONLY_ADMIN_TOKEN="${READONLY_ADMIN_TOKEN:-readonly-change-me}"
HTTP_PORT="${HTTP_PORT:-8899}"
SOCKS5_PORT="${SOCKS5_PORT:-1080}"
ADMIN_PORT="${ADMIN_PORT:-8088}"
TEST_USER="e2e_$(date +%s)"
TEST_PASS="p_$(date +%s)"
TEST_ADMIN="adm_$(date +%s)"
TEST_ADMIN_TOKEN="adm_tok_$(date +%s)"

log() {
  printf '%s\n' "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "missing command: $1"
    exit 1
  fi
}

check_http_status_contains_ok() {
  payload="$1"
  echo "$payload" | grep '"ok":true' >/dev/null 2>&1
}

log "[1/8] Checking required commands"
require_cmd docker
require_cmd curl

log "[2/8] Checking Docker daemon"
if ! docker version >/dev/null 2>&1; then
  log "docker daemon is not ready"
  exit 1
fi

log "[3/8] Starting stack"
docker compose -f "$COMPOSE_FILE" up -d --build

log "[4/8] Waiting admin health"
i=0
while [ "$i" -lt 30 ]; do
  body="$(curl -sS "http://127.0.0.1:${ADMIN_PORT}/api/admin/healthz" || true)"
  if check_http_status_contains_ok "$body"; then
    log "admin health ready"
    break
  fi
  i=$((i + 1))
  sleep 1
  if [ "$i" -eq 30 ]; then
    log "admin health check timeout"
    exit 1
  fi
done

log "[5/8] Creating test user: ${TEST_USER}"
create_resp="$(curl -sS -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/users" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${TEST_USER}\",\"password\":\"${TEST_PASS}\",\"max_devices\":1,\"quota_bytes\":10485760,\"expires_at\":0}" )"

echo "$create_resp" | grep '"ok":true' >/dev/null 2>&1 || {
  log "create user failed: $create_resp"
  exit 1
}

log "[6/8] Verifying user exists in list"
list_resp="$(curl -sS "http://127.0.0.1:${ADMIN_PORT}/api/admin/users" -H "Authorization: Bearer ${ADMIN_TOKEN}")"
echo "$list_resp" | grep "\"username\":\"${TEST_USER}\"" >/dev/null 2>&1 || {
  log "user not found in list"
  exit 1
}

log "[7/8] Testing HTTP and SOCKS5 proxies"
http_code="$(curl -sS -o /dev/null -w '%{http_code}' -x "http://${TEST_USER}:${TEST_PASS}@127.0.0.1:${HTTP_PORT}" http://example.com || true)"
if [ "$http_code" -lt 200 ] || [ "$http_code" -ge 500 ]; then
  log "http proxy check failed, status=$http_code"
  exit 1
fi

socks_code="$(curl -sS -o /dev/null -w '%{http_code}' --socks5 "${TEST_USER}:${TEST_PASS}@127.0.0.1:${SOCKS5_PORT}" http://example.com || true)"
if [ "$socks_code" -lt 200 ] || [ "$socks_code" -ge 500 ]; then
  log "socks5 proxy check failed, status=$socks_code"
  exit 1
fi

log "[8/8] Testing disable policy"
disable_resp="$(curl -sS -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/users/${TEST_USER}/disable" -H "Authorization: Bearer ${ADMIN_TOKEN}")"
echo "$disable_resp" | grep '"ok":true' >/dev/null 2>&1 || {
  log "disable user failed: $disable_resp"
  exit 1
}

blocked_code="$(curl -sS -o /dev/null -w '%{http_code}' -x "http://${TEST_USER}:${TEST_PASS}@127.0.0.1:${HTTP_PORT}" http://example.com || true)"
if [ "$blocked_code" -ne 407 ]; then
  log "expected disabled user to be blocked with 407, got=$blocked_code"
  exit 1
fi

log "[extra] Verifying readonly role behavior"
ro_get_code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${ADMIN_PORT}/api/admin/users" \
  -H "Authorization: Bearer ${READONLY_ADMIN_TOKEN}" || true)"
if [ "$ro_get_code" -ne 200 ]; then
  log "readonly GET users failed, status=$ro_get_code"
  exit 1
fi

ro_post_code="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/users" \
  -H "Authorization: Bearer ${READONLY_ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"username":"ro_fail","password":"x","max_devices":1,"quota_bytes":1,"expires_at":0}' || true)"
if [ "$ro_post_code" -ne 403 ]; then
  log "readonly POST users should be 403, got=$ro_post_code"
  exit 1
fi

log "[extra] Verifying admin management endpoints"
admin_create_code="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/admins" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${TEST_ADMIN}\",\"token\":\"${TEST_ADMIN_TOKEN}\",\"role\":\"readonly\"}" || true)"
if [ "$admin_create_code" -ne 201 ]; then
  log "create admin failed, status=$admin_create_code"
  exit 1
fi

admin_me_code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${ADMIN_PORT}/api/admin/me" \
  -H "Authorization: Bearer ${TEST_ADMIN_TOKEN}" || true)"
if [ "$admin_me_code" -ne 200 ]; then
  log "new admin token not accepted, status=$admin_me_code"
  exit 1
fi

admin_rotate_code="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/admins/${TEST_ADMIN}/rotate-token" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"${TEST_ADMIN_TOKEN}_new\"}" || true)"
if [ "$admin_rotate_code" -ne 200 ]; then
  log "rotate admin token failed, status=$admin_rotate_code"
  exit 1
fi

admin_new_me_code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${ADMIN_PORT}/api/admin/me" \
  -H "Authorization: Bearer ${TEST_ADMIN_TOKEN}_new" || true)"
if [ "$admin_new_me_code" -ne 200 ]; then
  log "rotated admin token not accepted, status=$admin_new_me_code"
  exit 1
fi

admin_old_me_code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${ADMIN_PORT}/api/admin/me" \
  -H "Authorization: Bearer ${TEST_ADMIN_TOKEN}" || true)"
if [ "$admin_old_me_code" -ne 401 ]; then
  log "old admin token should be rejected, got status=$admin_old_me_code"
  exit 1
fi

log "All verification checks passed"

#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"
HTTP_PORT="${HTTP_PORT:-8899}"
SOCKS5_PORT="${SOCKS5_PORT:-1080}"
ADMIN_PORT="${ADMIN_PORT:-8088}"
BOOTSTRAP_ADMIN_USER="${BOOTSTRAP_ADMIN_USER:-admin}"
BOOTSTRAP_ADMIN_PASS="${BOOTSTRAP_ADMIN_PASS:-admin123}"
BOOTSTRAP_READONLY="${BOOTSTRAP_READONLY:-ops}"
BOOTSTRAP_READONLY_PASS="${BOOTSTRAP_READONLY_PASS:-ops123456}"
TEST_USER="e2e_$(date +%s)"
TEST_PASS="Pass_$(date +%s)"
TEST_ADMIN="adm_$(date +%s)"
TEST_ADMIN_PASS="AdmPass_$(date +%s)"
ADMIN_COOKIE="${TMPDIR:-/tmp}/proxy_admin_cookie_$$.txt"
RO_COOKIE="${TMPDIR:-/tmp}/proxy_ro_cookie_$$.txt"
TEST_ADMIN_COOKIE="${TMPDIR:-/tmp}/proxy_test_admin_cookie_$$.txt"

cleanup() {
  rm -f "$ADMIN_COOKIE" "$RO_COOKIE" "$TEST_ADMIN_COOKIE"
}
trap cleanup EXIT INT TERM

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

login_cookie() {
  cookie_file="$1"
  username="$2"
  password="$3"
  code="$(curl -sS -o /dev/null -w '%{http_code}' -c "$cookie_file" -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"${username}\",\"password\":\"${password}\"}" || true)"
  [ "$code" -eq 200 ]
}

log "[1/10] Checking required commands"
require_cmd docker
require_cmd curl

log "[2/10] Checking Docker daemon"
if ! docker version >/dev/null 2>&1; then
  log "docker daemon is not ready"
  exit 1
fi

log "[3/10] Starting stack"
docker compose -f "$COMPOSE_FILE" up -d --build

log "[4/10] Waiting admin health"
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

log "[5/10] Logging in as super admin"
if ! login_cookie "$ADMIN_COOKIE" "$BOOTSTRAP_ADMIN_USER" "$BOOTSTRAP_ADMIN_PASS"; then
  log "super admin login failed"
  exit 1
fi

log "[6/10] Creating test user: ${TEST_USER}"
create_resp="$(curl -sS -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/users" \
  -b "$ADMIN_COOKIE" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${TEST_USER}\",\"password\":\"${TEST_PASS}\",\"max_devices\":1,\"quota_bytes\":10485760,\"expires_at\":0}" )"

echo "$create_resp" | grep '"ok":true' >/dev/null 2>&1 || {
  log "create user failed: $create_resp"
  exit 1
}

log "[7/10] Verifying user exists in list"
list_resp="$(curl -sS "http://127.0.0.1:${ADMIN_PORT}/api/admin/users?offset=0&limit=20" -b "$ADMIN_COOKIE")"
echo "$list_resp" | grep "\"username\":\"${TEST_USER}\"" >/dev/null 2>&1 || {
  log "user not found in list"
  exit 1
}

log "[8/10] Testing HTTP and SOCKS5 proxies"
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

log "[9/10] Testing disable policy"
disable_resp="$(curl -sS -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/users/${TEST_USER}/disable" -b "$ADMIN_COOKIE")"
echo "$disable_resp" | grep '"ok":true' >/dev/null 2>&1 || {
  log "disable user failed: $disable_resp"
  exit 1
}

blocked_code="$(curl -sS -o /dev/null -w '%{http_code}' -x "http://${TEST_USER}:${TEST_PASS}@127.0.0.1:${HTTP_PORT}" http://example.com || true)"
if [ "$blocked_code" -ne 407 ]; then
  log "expected disabled user to be blocked with 407, got=$blocked_code"
  exit 1
fi

log "[10/10] Verifying readonly role and admin management"
if ! login_cookie "$RO_COOKIE" "$BOOTSTRAP_READONLY" "$BOOTSTRAP_READONLY_PASS"; then
  log "readonly admin login failed"
  exit 1
fi

ro_get_code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${ADMIN_PORT}/api/admin/users" -b "$RO_COOKIE" || true)"
if [ "$ro_get_code" -ne 200 ]; then
  log "readonly GET users failed, status=$ro_get_code"
  exit 1
fi

ro_post_code="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/users" \
  -b "$RO_COOKIE" \
  -H 'Content-Type: application/json' \
  -d '{"username":"ro_fail","password":"password88","max_devices":1,"quota_bytes":1,"expires_at":0}' || true)"
if [ "$ro_post_code" -ne 403 ]; then
  log "readonly POST users should be 403, got=$ro_post_code"
  exit 1
fi

admin_create_code="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${ADMIN_PORT}/api/admin/admins" \
  -b "$ADMIN_COOKIE" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${TEST_ADMIN}\",\"password\":\"${TEST_ADMIN_PASS}\",\"role\":\"readonly\"}" || true)"
if [ "$admin_create_code" -ne 201 ]; then
  log "create admin failed, status=$admin_create_code"
  exit 1
fi

if ! login_cookie "$TEST_ADMIN_COOKIE" "$TEST_ADMIN" "$TEST_ADMIN_PASS"; then
  log "new admin password login failed"
  exit 1
fi

admin_me_code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${ADMIN_PORT}/api/admin/me" -b "$TEST_ADMIN_COOKIE" || true)"
if [ "$admin_me_code" -ne 200 ]; then
  log "new admin session not accepted, status=$admin_me_code"
  exit 1
fi

log "All verification checks passed"

#!/usr/bin/env bash
# FixPhone production smoke test (curl + jq).
#
#   SMOKE_BASE_URL=https://fixphone.example.com \
#   SMOKE_EMAIL=owner@toko.example.com \
#   SMOKE_PASSWORD='...' \
#   ./scripts/smoke-prod.sh
#
# Optional: SMOKE_INSECURE=1 to skip TLS verification (self-signed staging).
set -euo pipefail

: "${SMOKE_BASE_URL:?SMOKE_BASE_URL wajib diisi}"
: "${SMOKE_EMAIL:?SMOKE_EMAIL wajib diisi}"
: "${SMOKE_PASSWORD:?SMOKE_PASSWORD wajib diisi}"

BASE_URL="${SMOKE_BASE_URL%/}"
CURL=(curl -sS --max-time 20)
if [ "${SMOKE_INSECURE:-0}" = "1" ]; then
    CURL+=(-k)
fi

TMP="$(mktemp -d)"
COOKIE_JAR="$TMP/cookies.txt"
BODY="$TMP/body"
FAILURES=0
TOKEN=""

trap 'rm -rf "$TMP"' EXIT

log() { printf '%s\n' "$*"; }
ok() { log "[ OK ] $1"; }
bad() { log "[FAIL] $1"; FAILURES=$((FAILURES + 1)); }

# check NAME EXPECTED METHOD PATH [JSON_BODY] [BEARER]
check() {
    local name="$1" expected="$2" method="$3" path="$4" data="${5:-}" token="${6:-}"
    local args=(-X "$method" -o "$BODY" -w '%{http_code}' -b "$COOKIE_JAR" -c "$COOKIE_JAR")
    if [ -n "$token" ]; then args+=(-H "Authorization: Bearer $token"); fi
    if [ -n "$data" ]; then args+=(-H 'Content-Type: application/json' --data "$data"); fi

    local code
    code="$("${CURL[@]}" "${args[@]}" "$BASE_URL$path" || true)"
    code="${code:0:3}"
    if [ "$code" = "$expected" ]; then
        ok "$name ($code)"
    else
        bad "$name - status $code, expected $expected"
        sed -n '1,5p' "$BODY" | sed 's/^/       /'
    fi
}

log "FixPhone smoke test -> $BASE_URL"
check 'liveness /healthz' 200 GET /healthz
# The Nginx edge answers /healthz with plain "ok" while the API returns JSON.
# Accept either shape, but reject the SPA fallback (HTML).
if jq -e '.status == "ok"' "$BODY" >/dev/null 2>&1 || [ "$(tr -d '[:space:]' < "$BODY")" = "ok" ]; then
    ok 'liveness response valid'
else
    bad '/healthz bukan JSON status ok atau teks ok (SPA fallback?)'
fi
check 'readiness /readyz' 200 GET /readyz
if jq -e '.status == "ready" and .database == "up"' "$BODY" >/dev/null 2>&1; then
    ok 'readiness reports database up'
else
    bad '/readyz bukan JSON siap dari API (SPA fallback?)'
fi
check 'build metadata /version' 200 GET /version
if jq -e '.version' "$BODY" >/dev/null 2>&1; then
    ok "versi API $(jq -r '.version' "$BODY")"
else
    bad '/version bukan metadata build dari API (SPA fallback?)'
fi

if [[ "$BASE_URL" == https://* ]]; then
    if "${CURL[@]}" -sSI "$BASE_URL/healthz" | grep -qi '^strict-transport-security:'; then
        ok 'HSTS header present'
    else
        bad 'HSTS header missing on HTTPS response'
    fi
fi

check 'anonymous request rejected' 401 GET /api/v1/customers
check 'unknown tracking ticket fails closed' 404 GET \
    '/api/v1/public/track?ticket=NONEXISTENT&phone_last4=0000'

LOGIN_BODY="$(printf '{"email":"%s","password":"%s"}' "$SMOKE_EMAIL" "$SMOKE_PASSWORD")"
check 'login' 200 POST /api/v1/auth/login "$LOGIN_BODY"
TOKEN="$(jq -r '.data.access_token // empty' "$BODY" 2>/dev/null || true)"
if [ -z "$TOKEN" ]; then
    bad 'respons login tidak memuat access_token'
    log "Smoke test GAGAL: $FAILURES pemeriksaan."
    exit 1
fi

check 'profile /auth/me' 200 GET /api/v1/auth/me '' "$TOKEN"
log "       role=$(jq -r '.data.user.role // "-"' "$BODY" 2>/dev/null || echo '-')"
check 'dashboard summary' 200 GET /api/v1/dashboard/summary '' "$TOKEN"
check 'service order list' 200 GET '/api/v1/service-orders?per_page=1' '' "$TOKEN"
check 'customer list' 200 GET '/api/v1/customers?per_page=1' '' "$TOKEN"

check 'refresh cookie rotation' 200 POST /api/v1/auth/refresh '{}'
NEW_TOKEN="$(jq -r '.data.access_token // empty' "$BODY" 2>/dev/null || true)"
if [ -n "$NEW_TOKEN" ]; then
    TOKEN="$NEW_TOKEN"
    ok 'access token diperbarui'
else
    bad 'refresh tidak mengembalikan access_token'
fi

check 'logout' 200 POST /api/v1/auth/logout '{}' "$TOKEN"
check 'refresh cookie revoked after logout' 401 POST /api/v1/auth/refresh '{}'

if [ "$FAILURES" -gt 0 ]; then
    log "Smoke test GAGAL: $FAILURES pemeriksaan."
    exit 1
fi
log 'Smoke test LULUS.'

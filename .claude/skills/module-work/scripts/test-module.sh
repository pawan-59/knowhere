#!/usr/bin/env bash
#
# test-module.sh — verification harness for a single Central-Devtron module.
#
# Brings up Postgres (a dedicated *test* database, never your real data),
# builds + vets the backend, starts it on a test port with throwaway
# credentials, then runs auth-gating and module-specific endpoint checks.
#
# Usage:  bash test-module.sh <module>
#   <module> ∈ zoho | devtron | onboarding | license | auth | all
#
# Env toggles:
#   SKIP_FRONTEND=1   skip the `npm run build` check (use when only backend changed)
#
# Exit code 0 = all checks passed. Non-zero = a check failed (details printed).

set -uo pipefail

MODULE="${1:-all}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"   # project root
BACKEND="$ROOT/backend"
FRONTEND="$ROOT/frontend"

PORT=8090
BASE="http://localhost:$PORT"
DBFILE="$(mktemp -u -t cd-harness-XXXXXX).db"   # isolated throwaway SQLite file
ADMIN_EMAIL="admin@devtron.ai"
ADMIN_PASSWORD="testpassword123"
SECRET="module-work-harness-secret-not-for-production"

COOKIES="$(mktemp)"
LOG="$(mktemp)"
BACKEND_PID=""

pass()  { echo "  ✅ $*"; }
info()  { echo "→ $*"; }
fail()  { echo "  ❌ FAIL: $*"; echo "----- backend log (tail) -----"; tail -n 25 "$LOG" 2>/dev/null; finish 1; }
finish(){ [ -n "$BACKEND_PID" ] && kill "$BACKEND_PID" 2>/dev/null; rm -f "$COOKIES" "$LOG" "$DBFILE" "$DBFILE"-shm "$DBFILE"-wal; exit "${1:-0}"; }
trap 'finish 1' INT TERM

case "$MODULE" in zoho|devtron|onboarding|license|auth|all) ;; *)
  echo "Unknown module '$MODULE'. Use: zoho | devtron | onboarding | license | auth | all"; exit 2;; esac

echo "=== Central-Devtron module harness: $MODULE ==="

# ── 1. Build + vet ──────────────────────────────────────────────────────────
info "go build ./..."
( cd "$BACKEND" && go build ./... ) || fail "go build failed — fix compile errors"
pass "build clean"

info "go vet ./..."
( cd "$BACKEND" && go vet ./... ) || fail "go vet reported issues"
pass "vet clean"

# ── 2. Isolated test database ────────────────────────────────────────────────
# Embedded SQLite → no external services. Each run gets a fresh throwaway file,
# so the harness never touches real license/onboarding data.
info "using throwaway SQLite db: $DBFILE"
pass "test DB isolated (SQLite)"

# ── 3. Start backend fresh (resets in-memory rate limiter each run) ──────────
info "starting backend on :$PORT"
lsof -ti:"$PORT" 2>/dev/null | xargs kill 2>/dev/null
sleep 1
( cd "$BACKEND" && PORT="$PORT" DB_PATH="$DBFILE" AUTH_SECRET="$SECRET" \
    ADMIN_EMAIL="$ADMIN_EMAIL" ADMIN_PASSWORD="$ADMIN_PASSWORD" \
    go run ./cmd/server > "$LOG" 2>&1 ) &
BACKEND_PID=$!
for i in $(seq 1 20); do
  [ "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/health" 2>/dev/null)" = "200" ] && break
  sleep 1
done
[ "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/health" 2>/dev/null)" = "200" ] || fail "backend did not come up"
pass "backend healthy"

# ── helpers ──────────────────────────────────────────────────────────────────
code_noauth() { curl -s -o /dev/null -w '%{http_code}' "$BASE$1"; }
code_auth()   { curl -s -b "$COOKIES" -o /dev/null -w '%{http_code}' "$BASE$1"; }
login() { # $1 password ; stores cookie ; echoes http code
  curl -s -c "$COOKIES" -o /dev/null -w '%{http_code}' -X POST "$BASE/api/auth/login" \
    -H 'Content-Type: application/json' -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$1\"}"
}

# Assert an endpoint blocks unauthenticated access — the security backbone.
gate() { local c; c=$(code_noauth "$1"); [ "$c" = "401" ] && pass "auth gate: $1 → 401 (no session)" || fail "$1 → $c without auth (expected 401 — auth gate broken!)"; }

# ── 4. Establish an authenticated session for data-module checks ─────────────
if [ "$MODULE" != "auth" ]; then
  [ "$(login "$ADMIN_PASSWORD")" = "200" ] || fail "login failed with seeded admin credentials"
  pass "logged in (session cookie stored)"
fi

# ── 5. Module checks ──────────────────────────────────────────────────────────
check_zoho() {
  gate "/api/zoho/summary"
  local c; c=$(code_auth "/api/zoho/summary")
  case "$c" in
    200) pass "zoho summary → 200 (configured, live data)";;
    503) pass "zoho summary → 503 (wired correctly, credentials not set — OK)";;
    *)   fail "zoho summary → $c (expected 200 or 503; 500/other = broken)";;
  esac
}

check_devtron() {
  gate "/api/devtron/summary"
  local c; c=$(code_auth "/api/devtron/summary")
  case "$c" in
    200) pass "devtron summary → 200 (configured, live data)";;
    503) pass "devtron summary → 503 (wired correctly, credentials not set — OK)";;
    *)   fail "devtron summary → $c (expected 200 or 503; 500/other = broken)";;
  esac
}

check_onboarding() {
  gate "/api/onboarding"
  local body code id
  # Use a blocked stage with a reason + an RFC3339 targetDate so the store's
  # date bind/scan and the blocked_reason column round-trips are both exercised.
  body=$(curl -s -b "$COOKIES" -X POST "$BASE/api/onboarding" -H 'Content-Type: application/json' \
    -d '{"customer":"__harness_onboarding__","stage":"Blocked On Devtron","status":"blocked","progress":10,"targetDate":"2026-01-15T00:00:00Z","blockedReason":"awaiting Devtron support ticket"}')
  echo "$body" | grep -q '"id"' || fail "onboarding create did not return a record: $body"
  echo "$body" | grep -q '"targetDate":"2026-01-15T00:00:00Z"' || fail "onboarding targetDate not round-tripped: $body"
  echo "$body" | grep -q '"blockedReason":"awaiting Devtron support ticket"' || fail "onboarding blockedReason not round-tripped: $body"
  id=$(echo "$body" | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
  pass "onboarding create → id=$id (targetDate + blockedReason round-trip)"
  [ "$(code_auth "/api/onboarding")" = "200" ]         || fail "onboarding list → not 200"
  [ "$(code_auth "/api/onboarding/summary")" = "200" ] || fail "onboarding summary → not 200"
  code=$(curl -s -b "$COOKIES" -o /dev/null -w '%{http_code}' -X DELETE "$BASE/api/onboarding/$id")
  [ "$code" = "204" ] || fail "onboarding delete → $code (expected 204)"
  pass "onboarding CRUD round-trip OK (list, summary, delete)"
}

check_license() {
  gate "/api/licenses"
  local body code id
  body=$(curl -s -b "$COOKIES" -X POST "$BASE/api/licenses" -H 'Content-Type: application/json' \
    -d '{"customer":"__harness_license__","installation":"harness-inst","seats":10,"seatsUsed":3,"status":"active"}')
  echo "$body" | grep -q '"id"' || fail "license create did not return a record: $body"
  id=$(echo "$body" | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
  pass "license create → id=$id"
  [ "$(code_auth "/api/licenses")" = "200" ]         || fail "license list → not 200"
  [ "$(code_auth "/api/licenses/summary")" = "200" ] || fail "license summary → not 200"
  code=$(curl -s -b "$COOKIES" -o /dev/null -w '%{http_code}' -X DELETE "$BASE/api/licenses/$id")
  [ "$code" = "204" ] || fail "license delete → $code (expected 204)"
  pass "license CRUD round-trip OK (list, summary, delete)"
}

check_auth() {
  # Data routes must reject unauthenticated access.
  gate "/api/licenses"
  gate "/api/onboarding"
  # Bad password → 401 (generic).
  [ "$(login "definitely-wrong")" = "401" ] || fail "login with wrong password did not return 401"
  pass "bad credentials → 401 (generic, no enumeration)"
  # Good password → 200 + HttpOnly cookie.
  [ "$(login "$ADMIN_PASSWORD")" = "200" ] || fail "login with correct password did not return 200"
  curl -s -D - -o /dev/null -X POST "$BASE/api/auth/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" | grep -iq 'set-cookie:.*httponly' \
    || fail "session cookie is not HttpOnly (XSS protection lost)"
  pass "good credentials → 200 + HttpOnly session cookie"
  # Session works, then logout invalidates it.
  [ "$(code_auth "/api/auth/me")" = "200" ] || fail "/api/auth/me → not 200 with valid session"
  curl -s -b "$COOKIES" -c "$COOKIES" -o /dev/null -X POST "$BASE/api/auth/logout"
  [ "$(code_auth "/api/auth/me")" = "401" ] || fail "/api/auth/me still authorized after logout"
  pass "session lifecycle OK (me→200, logout, me→401)"
}

echo "--- endpoint checks ---"
case "$MODULE" in
  zoho)       check_zoho ;;
  devtron)    check_devtron ;;
  onboarding) check_onboarding ;;
  license)    check_license ;;
  auth)       check_auth ;;
  all)        check_auth; login "$ADMIN_PASSWORD" >/dev/null; check_zoho; check_devtron; check_onboarding; check_license ;;
esac

# ── 6. Frontend build ─────────────────────────────────────────────────────────
if [ "${SKIP_FRONTEND:-0}" != "1" ]; then
  echo "--- frontend build ---"
  [ -d "$FRONTEND/node_modules" ] || ( info "installing frontend deps"; cd "$FRONTEND" && npm install >/dev/null 2>&1 )
  ( cd "$FRONTEND" && npm run build >/dev/null 2>&1 ) && pass "frontend build clean" || fail "npm run build failed"
else
  info "SKIP_FRONTEND=1 — skipping frontend build"
fi

echo "=== ✅ ALL CHECKS PASSED for module: $MODULE ==="
finish 0

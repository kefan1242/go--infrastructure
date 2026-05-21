#!/usr/bin/env bash
# End-to-end smoke test: boots all three kris-* services, hits each known
# endpoint, asserts the right code / body, tears down.
#
# Used by CI .github/workflows/ci.yml (e2e job) and runnable locally:
#   bash scripts/smoke-test.sh
#
# Exits non-zero on any failed assertion.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

ALPHA_HTTP=28080
ALPHA_OTHER=28081
BETA_HTTP=28082
BETA_OTHER=28083
GAMMA_OTHER=28085

cleanup() {
  echo ">>> cleanup"
  make demo-stop >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo ">>> building + starting demo stack"
make demo >/dev/null

# Give Kratos boot a beat — make demo already sleeps 1s but the gRPC servers
# can still be settling. Poll alpha sidecar to confirm.
for i in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${ALPHA_OTHER}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done

fail=0
assert_eq() {
  local label="$1" got="$2" want="$3"
  if [ "$got" = "$want" ]; then
    echo "  ok  $label  ($got)"
  else
    echo "  FAIL $label  want=$want got=$got"
    fail=$((fail+1))
  fi
}

assert_contains() {
  local label="$1" haystack="$2" needle="$3"
  if echo "$haystack" | grep -qF "$needle"; then
    echo "  ok  $label  (contains $needle)"
  else
    echo "  FAIL $label  body did not contain $needle"
    echo "       body: $haystack"
    fail=$((fail+1))
  fi
}

http_code() {
  curl -s -o /dev/null -w '%{http_code}' "$@"
}

http_body() {
  curl -s "$@"
}

echo ">>> alpha biz"
assert_eq "alpha GET / status"           "$(http_code "http://127.0.0.1:${ALPHA_HTTP}/")"       "200"
assert_eq "alpha GET / body"             "$(http_body "http://127.0.0.1:${ALPHA_HTTP}/")"       "kris-alpha ok"

echo ">>> alpha sidecar"
assert_eq "alpha healthz status"         "$(http_code "http://127.0.0.1:${ALPHA_OTHER}/healthz")" "200"
assert_eq "alpha readyz status (live)"   "$(http_code "http://127.0.0.1:${ALPHA_OTHER}/readyz")"  "200"
assert_contains "alpha /version json" \
  "$(http_body "http://127.0.0.1:${ALPHA_OTHER}/version")" "kris-alpha"
assert_contains "alpha /metrics has kris_*" \
  "$(http_body "http://127.0.0.1:${ALPHA_OTHER}/metrics")" "kris_requests_total"

echo ">>> beta public"
assert_eq "beta GET / status"            "$(http_code "http://127.0.0.1:${BETA_HTTP}/")"        "200"
assert_eq "beta GET / body"              "$(http_body "http://127.0.0.1:${BETA_HTTP}/")"        "kris-beta public ok"

echo ">>> beta authed"
assert_eq "beta /whoami unauthed"        "$(http_code "http://127.0.0.1:${BETA_HTTP}/whoami")"  "401"
assert_eq "beta /whoami valid-token" \
  "$(http_code -H 'Authorization: Bearer demo-alice' "http://127.0.0.1:${BETA_HTTP}/whoami")" "200"
assert_contains "beta /whoami body" \
  "$(http_body -H 'Authorization: Bearer demo-alice' "http://127.0.0.1:${BETA_HTTP}/whoami")" "alice"

echo ">>> beta CORS preflight"
assert_eq "beta OPTIONS preflight status" \
  "$(http_code -X OPTIONS -H 'Origin: https://x.example.com' -H 'Access-Control-Request-Method: GET' "http://127.0.0.1:${BETA_HTTP}/")" "204"

echo ">>> gamma readyz (probes upstream alpha)"
assert_contains "gamma /readyz body upstream:ok" \
  "$(http_body "http://127.0.0.1:${GAMMA_OTHER}/readyz")" '"upstream":"ok"'

echo
if [ "$fail" -ne 0 ]; then
  echo "smoke-test: $fail assertion(s) failed"
  exit 1
fi
echo "smoke-test: all green"

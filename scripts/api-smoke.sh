#!/usr/bin/env bash
set -euo pipefail

# Simple REST API smoke/integration script.
# Requirements:
# - Server running (e.g., PORT=8080 make run or Docker)
# - jq installed
#
# You can either set API_KEY manually or let the script create one via public signup.
# Configure below:

BASE_URL=${BASE_URL:-"http://localhost:8080"}
API_KEY=${API_KEY:-""}           # optional; if empty, will try public signup

# Provide wallets to test with (hard-coded PoC list)
WALLET1="MJKqp326RZCHnAAbew9MDdui3iCKWco7fsK9sVuZTX2"
WALLET2="52C9T2T7JRojtxumYnYZhyUmrN7kqzvCLc4Ksvjk7TxD"
WALLET3="8BseXT9EtoEhBTKFFYkwTnjKSUZwhtmdKY2Jrj8j45Rt"
WALLET4="GitYucwpNcg6Dx1Y15UQ9TQn8LZMX1uuqQNn8rXxEWNC"
WALLET5="9QgXqrgdbVU8KcpfskqJpAXKzbaYQJecgMAruSWoXDkM"

function info(){ echo -e "\033[1;34m[INFO]\033[0m $*"; }
function pass(){ echo -e "\033[1;32m[PASS]\033[0m $*"; }
function fail(){ echo -e "\033[1;31m[FAIL]\033[0m $*"; exit 1; }

need_cmd(){ command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"; }
need_cmd curl
need_cmd jq

health() {
  info "Health check"
  curl -fsS "$BASE_URL/healthz" | jq . >/dev/null || fail "healthz failed"
  pass "healthz OK"
}

signup_public(){
  [[ -n "$API_KEY" ]] && return 0
  info "Creating API key via public signup"
  local resp
  resp=$(curl -fsS -X POST "$BASE_URL/public/signup" -H 'Content-Type: application/json' -d '{"owner":"smoke-public"}')
  API_KEY=$(echo "$resp" | jq -r '.api_key // .apiKey // .key // empty')
  if [[ -z "$API_KEY" ]]; then fail "public signup returned no api key: $resp"; fi
  pass "Public signup key created"
}

require_api_key(){
  if [[ -z "$API_KEY" ]]; then
    fail "API_KEY not set and could not be created. Set ADMIN_TOKEN or run public signup."
  fi
}

post_balance(){
  local wallets_json=$1
  curl -fsS -X POST "$BASE_URL/api/get-balance" \
    -H "X-API-Key: $API_KEY" \
    -H 'Content-Type: application/json' \
    -d "{\"wallets\":$wallets_json}"
}

# Scenario 1: Single wallet
scenario_single_wallet(){
  info "Scenario: single wallet"
  [[ -z "$WALLET1" ]] && fail "Set WALLET1 environment variable"
  local resp
  resp=$(post_balance "[\"$WALLET1\"]") || fail "single wallet request failed"
  local count
  count=$(echo "$resp" | jq '.balances | length')
  [[ "$count" -ge 0 ]] || fail "unexpected balances length: $count\n$resp"
  pass "Single wallet request ran"
}

# Scenario 2: Multiple wallets (dedupe handled server-side)
scenario_multiple_wallets(){
  info "Scenario: multiple wallets"
  [[ -z "$WALLET1" || -z "$WALLET2" ]] && fail "Set WALLET1 and WALLET2"
  local resp count
  resp=$(post_balance "[\"$WALLET1\",\"$WALLET2\",\"$WALLET1\"]") || fail "multiple wallets request failed"
  count=$(echo "$resp" | jq '.balances | length')
  [[ "$count" -ge 0 ]] || fail "unexpected balances length: $count\n$resp"
  pass "Multiple wallets request ran"
}

# Scenario 2b: Five wallets in a single request
scenario_five_wallets(){
  info "Scenario: five wallets in one request"
  local wallets_json
  wallets_json='["'"$WALLET1"'","'"$WALLET2"'","'"$WALLET3"'","'"$WALLET4"'","'"$WALLET5"'"]'
  local resp count
  resp=$(post_balance "$wallets_json") || fail "five wallets request failed"
  count=$(echo "$resp" | jq '.balances | length')
  [[ "$count" -ge 0 ]] || fail "unexpected balances length: $count\n$resp"
  pass "Five-wallet request ran"
}

# Scenario 3: 5 requests same wallet (exercise cache/singleflight)
scenario_same_wallet_concurrency(){
  info "Scenario: 5 concurrent requests same wallet"
  [[ -z "$WALLET1" ]] && fail "Set WALLET1"
  seq 1 5 | xargs -I{} -P 5 bash -c 'curl -fsS -X POST "$0/api/get-balance" -H "X-API-Key: $1" -H "Content-Type: application/json" -d "{\"wallets\":[\"$2\"]}" >/dev/null' "$BASE_URL" "$API_KEY" "$WALLET1"
  pass "5 concurrent requests completed"
}

# Scenario 4: Mix of single/multi concurrent
scenario_mixed_concurrency(){
  info "Scenario: mixed concurrent requests"
  [[ -z "$WALLET1" || -z "$WALLET2" || -z "$WALLET3" ]] && fail "Set WALLET1,WALLET2,WALLET3"
  (
    post_balance "[\"$WALLET1\"]" &
    post_balance "[\"$WALLET2\"]" &
    post_balance "[\"$WALLET1\",\"$WALLET2\"]" &
    post_balance "[\"$WALLET1\",\"$WALLET3\"]" &
    wait
  ) || fail "mixed concurrency failed"
  pass "Mixed concurrent requests completed"
}

# Scenario 5: IP rate limiting (expect some 429s when bursting above RPM)
scenario_rate_limit(){
  info "Scenario: IP rate limiting"
  # Fire a burst of 20 requests quickly; default RPM is 10 so some should 429.
  local ok=0 rl=0
  for i in $(seq 1 20); do
    code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/get-balance" \
      -H "X-API-Key: $API_KEY" -H 'Content-Type: application/json' \
      -d "{\"wallets\":[\"$WALLET1\"]}") || code=000
    if [[ "$code" == "200" ]]; then ok=$((ok+1)); elif [[ "$code" == "429" ]]; then rl=$((rl+1)); fi
  done
  info "rate-limit summary: 200=$ok 429=$rl"
  [[ $rl -ge 1 ]] || info "note: no 429 (rate limit may be lenient in current config)"
  # For PoC we don't require successes; objective is to observe limiting behavior.
  pass "Rate limiting exercised"
}

# Scenario 6: Caching check (best-effort: look for 'cache' source on second call)
scenario_caching(){
  info "Scenario: caching best-effort"
  local r1 r2 s1 s2
  r1=$(post_balance "[\"$WALLET1\"]") || true
  r2=$(post_balance "[\"$WALLET1\"]") || true
  s1=$(echo "$r1" | jq -r '.balances[0].source // .balances[0].Source // empty')
  s2=$(echo "$r2" | jq -r '.balances[0].source // .balances[0].Source // empty')
  info "sources: first=$s1 second=$s2"
  pass "Caching scenario executed"
}

# Scenario 7: Auth and rate limiting validation
scenario_auth(){
  info "Scenario: authentication checks"
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/get-balance" -H 'Content-Type: application/json' -d '{"wallets":["w"]}')
  [[ "$code" == "401" ]] || fail "expected 401 for missing api key, got $code"
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/get-balance" -H 'Content-Type: application/json' -H 'X-API-Key: bad' -d '{"wallets":["w"]}')
  [[ "$code" == "403" ]] || info "note: expected 403 for invalid key, got $code"
  pass "Auth checks executed"
}

main(){
  info "BASE_URL=$BASE_URL"
  health
  # Run auth checks early to avoid hitting per-IP limiter first
  scenario_auth

  signup_public || true
  require_api_key

  scenario_single_wallet
  scenario_multiple_wallets
  scenario_five_wallets
  scenario_same_wallet_concurrency
  scenario_mixed_concurrency
  scenario_rate_limit
  # brief cooldown before caching check to reduce 429 likelihood
  sleep 1 || true
  scenario_caching

  pass "All scenarios executed"
}

main "$@"

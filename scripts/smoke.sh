#!/usr/bin/env bash
# End-to-end smoke test against a running server.
# Usage: PORT=8080 ./scripts/smoke.sh   (needs curl and jq)
set -euo pipefail

BASE="http://localhost:${PORT:-8080}/v1"
BODY="$(mktemp)"
trap 'rm -f "$BODY"' EXIT

fail() {
  echo "FAIL: $*" >&2
  echo "  response body: $(cat "$BODY")" >&2
  exit 1
}

# request <label> <method> <path> [curl args...]
# Stores the response body in $BODY and the status code in $STATUS.
request() {
  local label="$1" method="$2" path="$3"
  shift 3
  STATUS="$(curl -sS -o "$BODY" -w '%{http_code}' -X "$method" "$BASE$path" "$@")" ||
    fail "$label: could not reach $BASE$path"
}

# expect_2xx <label>
expect_2xx() {
  [[ "$STATUS" =~ ^2[0-9][0-9]$ ]] || fail "$1: expected 2xx, got $STATUS"
}

# expect_json <label> <jq filter that must be true>
expect_json() {
  jq -e "$2" "$BODY" >/dev/null || fail "$1: JSON check failed: $2"
}

pass() { echo "ok   $1 ($STATUS)"; }

UNIQUE="$(date +%s)-$RANDOM"

request "healthz" GET /healthz
expect_2xx "healthz"
pass "healthz"

request "create user" POST /users -H 'Content-Type: application/json' \
  -d "{\"name\":\"smoke-$UNIQUE\"}"
expect_2xx "create user"
expect_json "create user" '.id and .api_key and .name'
API_KEY="$(jq -r '.api_key' "$BODY")"
USER_ID="$(jq -r '.id' "$BODY")"
AUTH="Authorization: ApiKey $API_KEY"
pass "create user"

request "get user" GET /users -H "$AUTH"
expect_2xx "get user"
expect_json "get user" ".id == \"$USER_ID\""
pass "get user"

# .invalid never resolves, so the scraper cannot reach a real host with it.
FEED_URL="http://smoke-$UNIQUE.invalid/feed.xml"
request "create feed" POST /feeds -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"name\":\"smoke feed $UNIQUE\",\"url\":\"$FEED_URL\"}"
expect_2xx "create feed"
expect_json "create feed" ".id and .url == \"$FEED_URL\" and .user_id == \"$USER_ID\""
FEED_ID="$(jq -r '.id' "$BODY")"
pass "create feed"

request "list feeds" GET /feeds
expect_2xx "list feeds"
expect_json "list feeds" "type == \"array\" and any(.[]; .id == \"$FEED_ID\")"
pass "list feeds"

request "follow feed" POST /feed_follows -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"feed_id\":\"$FEED_ID\"}"
expect_2xx "follow feed"
expect_json "follow feed" ".id and .feed_id == \"$FEED_ID\" and .user_id == \"$USER_ID\""
FOLLOW_ID="$(jq -r '.id' "$BODY")"
pass "follow feed"

request "list follows" GET /feed_follows -H "$AUTH"
expect_2xx "list follows"
expect_json "list follows" "type == \"array\" and any(.[]; .id == \"$FOLLOW_ID\")"
pass "list follows"

request "get posts" GET /posts -H "$AUTH"
expect_2xx "get posts"
expect_json "get posts" 'type == "array"'
pass "get posts"

request "unfollow feed" DELETE "/feed_follows/$FOLLOW_ID" -H "$AUTH"
expect_2xx "unfollow feed"
pass "unfollow feed"

echo "smoke test passed"

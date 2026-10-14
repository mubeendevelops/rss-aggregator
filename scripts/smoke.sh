#!/usr/bin/env bash
# End-to-end smoke test against a running server.
# Usage: PORT=8080 ./scripts/smoke.sh   (needs curl and jq)
# Asserts exact status codes.
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

# expect_status <label> <code>
expect_status() {
  [[ "$STATUS" == "$2" ]] || fail "$1: expected $2, got $STATUS"
}

# expect_json <label> <jq filter that must be true>
expect_json() {
  jq -e "$2" "$BODY" >/dev/null || fail "$1: JSON check failed: $2"
}

pass() { echo "ok   $1 ($STATUS)"; }

UNIQUE="$(date +%s)-$RANDOM"

request "healthz" GET /healthz
expect_status "healthz" 200
pass "healthz"

request "create user" POST /users -H 'Content-Type: application/json' \
  -d "{\"name\":\"smoke-$UNIQUE\"}"
expect_status "create user" 201
expect_json "create user" '.id and .api_key and .name'
expect_json "create user" '.api_key | test("^[0-9a-f]{64}$")'
API_KEY="$(jq -r '.api_key' "$BODY")"
USER_ID="$(jq -r '.id' "$BODY")"
AUTH="Authorization: ApiKey $API_KEY"
pass "create user"

request "second user" POST /users -H 'Content-Type: application/json' \
  -d "{\"name\":\"smoke-$UNIQUE-b\"}"
expect_status "second user" 201
expect_json "second user" ".api_key != \"$API_KEY\""
pass "second user"

request "empty user name" POST /users -H 'Content-Type: application/json' -d '{"name":"  "}'
expect_status "empty user name" 400
pass "empty user name"

request "no auth header" GET /users
expect_status "no auth header" 401
pass "no auth header"

request "malformed auth header" GET /users -H 'Authorization: Bearer nope'
expect_status "malformed auth header" 401
pass "malformed auth header"

request "wrong key" GET /users -H 'Authorization: ApiKey not-a-real-key'
expect_status "wrong key" 401
pass "wrong key"

request "get user" GET /users -H "$AUTH"
expect_status "get user" 200
expect_json "get user" ".id == \"$USER_ID\""
pass "get user"

# .invalid never resolves, so the scraper cannot reach a real host with it.
FEED_URL="http://smoke-$UNIQUE.invalid/feed.xml"
request "create feed" POST /feeds -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"name\":\"smoke feed $UNIQUE\",\"url\":\"$FEED_URL\"}"
expect_status "create feed" 201
expect_json "create feed" ".id and .url == \"$FEED_URL\" and .user_id == \"$USER_ID\""
FEED_ID="$(jq -r '.id' "$BODY")"
pass "create feed"

request "empty feed name" POST /feeds -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"name\":\"\",\"url\":\"http://smoke-$UNIQUE-x.invalid/feed.xml\"}"
expect_status "empty feed name" 400
pass "empty feed name"

for BAD_URL in "" "file:///etc/passwd" "ftp://example.invalid/feed.xml" "example.invalid/feed.xml" "http://"; do
  request "bad feed url '$BAD_URL'" POST /feeds -H "$AUTH" -H 'Content-Type: application/json' \
    -d "{\"name\":\"smoke bad url\",\"url\":\"$BAD_URL\"}"
  expect_status "bad feed url '$BAD_URL'" 400
  pass "bad feed url '$BAD_URL'"
done

request "duplicate feed url" POST /feeds -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"name\":\"smoke feed again\",\"url\":\"$FEED_URL\"}"
expect_status "duplicate feed url" 409
expect_json "duplicate feed url" '.error'
pass "duplicate feed url"

request "list feeds" GET /feeds
expect_status "list feeds" 200
expect_json "list feeds" "type == \"array\" and any(.[]; .id == \"$FEED_ID\")"
pass "list feeds"

request "follow feed" POST /feed_follows -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"feed_id\":\"$FEED_ID\"}"
expect_status "follow feed" 201
expect_json "follow feed" ".id and .feed_id == \"$FEED_ID\" and .user_id == \"$USER_ID\""
FOLLOW_ID="$(jq -r '.id' "$BODY")"
pass "follow feed"

request "duplicate follow" POST /feed_follows -H "$AUTH" -H 'Content-Type: application/json' \
  -d "{\"feed_id\":\"$FEED_ID\"}"
expect_status "duplicate follow" 409
pass "duplicate follow"

request "follow unknown feed" POST /feed_follows -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"feed_id":"00000000-0000-4000-8000-000000000000"}'
expect_status "follow unknown feed" 404
pass "follow unknown feed"

request "list follows" GET /feed_follows -H "$AUTH"
expect_status "list follows" 200
expect_json "list follows" "type == \"array\" and any(.[]; .id == \"$FOLLOW_ID\")"
pass "list follows"

request "get posts" GET /posts -H "$AUTH"
expect_status "get posts" 200
expect_json "get posts" 'type == "array"'
pass "get posts"

request "get posts page" GET "/posts?limit=5&offset=0" -H "$AUTH"
expect_status "get posts page" 200
expect_json "get posts page" 'type == "array"'
pass "get posts page"

for BAD_QUERY in "limit=0" "limit=101" "limit=abc" "offset=-1"; do
  request "get posts $BAD_QUERY" GET "/posts?$BAD_QUERY" -H "$AUTH"
  expect_status "get posts $BAD_QUERY" 400
  pass "get posts $BAD_QUERY"
done

request "unfollow feed" DELETE "/feed_follows/$FOLLOW_ID" -H "$AUTH"
expect_status "unfollow feed" 200
pass "unfollow feed"

request "unfollow again" DELETE "/feed_follows/$FOLLOW_ID" -H "$AUTH"
expect_status "unfollow again" 404
pass "unfollow again"

request "unfollow unknown id" DELETE /feed_follows/00000000-0000-4000-8000-000000000000 -H "$AUTH"
expect_status "unfollow unknown id" 404
pass "unfollow unknown id"

request "unfollow bad id" DELETE /feed_follows/not-a-uuid -H "$AUTH"
expect_status "unfollow bad id" 400
pass "unfollow bad id"

echo "smoke test passed"

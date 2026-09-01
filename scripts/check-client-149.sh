#!/usr/bin/env bash

set -euo pipefail

api_url="${DESKLINK_API_URL:-http://127.0.0.1:21114}"
username="${DESKLINK_TEST_USERNAME:-admin}"
: "${DESKLINK_TEST_PASSWORD:?Set DESKLINK_TEST_PASSWORD before running this check}"

curl_args=(--fail --silent --show-error --retry 8 --retry-delay 1 --retry-connrefused)

login_payload=$(jq -n \
    --arg username "$username" \
    --arg password "$DESKLINK_TEST_PASSWORD" \
    '{username: $username, password: $password, id: "149-contract-test", uuid: "149-contract-test", type: "account", deviceInfo: {name: "DeskLink 1.4.9 contract test", os: "linux", type: "app"}}')

login=$(curl "${curl_args[@]}" -X POST "$api_url/api/login" \
    -H 'Content-Type: application/json' --data "$login_payload")
token=$(jq -er '.access_token' <<<"$login")
jq -e '.type == "access_token" and (.user | has("name") and has("display_name") and has("avatar") and has("status"))' <<<"$login" >/dev/null

auth_header="Authorization: Bearer $token"

curl "${curl_args[@]}" -X POST "$api_url/api/currentUser" -H "$auth_header" \
    -H 'Content-Type: application/json' --data '{"id":"149-contract-test","uuid":"149-contract-test"}' \
    | jq -e 'has("name") and has("display_name") and has("avatar") and (.status == 0 or .status == 1 or .status == -1)' >/dev/null

curl "${curl_args[@]}" "$api_url/api/login-options" | jq -e 'type == "array"' >/dev/null

for endpoint in \
    'users?current=1&pageSize=100&status=1&accessible=' \
    'peers?current=1&pageSize=100&status=1&accessible=' \
    'device-group/accessible?current=1&pageSize=100'; do
    curl "${curl_args[@]}" "$api_url/api/$endpoint" -H "$auth_header" \
        | jq -e 'has("total") and (.data | type == "array")' >/dev/null
done

curl "${curl_args[@]}" -X POST "$api_url/api/ab/settings" -H "$auth_header" \
    -H 'Content-Type: application/json' --data '{}' | jq -e 'has("max_peer_one_ab")' >/dev/null

personal=$(curl "${curl_args[@]}" -X POST "$api_url/api/ab/personal" -H "$auth_header" \
    -H 'Content-Type: application/json' --data '{}')
jq -e '. == null or (has("guid") and .rule == 3)' <<<"$personal" >/dev/null

curl "${curl_args[@]}" -X POST "$api_url/api/ab/shared/profiles?current=1&pageSize=100" \
    -H "$auth_header" -H 'Content-Type: application/json' --data '{}' \
    | jq -e 'has("total") and (.data | type == "array")' >/dev/null

curl "${curl_args[@]}" -X POST "$api_url/api/logout" -H "$auth_header" \
    -H 'Content-Type: application/json' --data '{}' >/dev/null

echo "DeskLink 1.4.9 API contract: PASS"

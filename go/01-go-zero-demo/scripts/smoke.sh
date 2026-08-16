#!/usr/bin/env bash

set -euo pipefail

gateway_addr="${GOZERO_GATEWAY_ADDR:-http://127.0.0.1:8180}"
email="demo-$(date +%s)@example.com"
password="secret"

register_response=$(curl --fail --silent --show-error \
  --request POST "${gateway_addr}/api/v1/auth/register" \
  --header 'Content-Type: application/json' \
  --data "{\"email\":\"${email}\",\"password\":\"${password}\"}")

token=$(printf '%s' "${register_response}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
if [[ -z "${token}" ]]; then
  printf 'register response does not contain a token: %s\n' "${register_response}" >&2
  exit 1
fi

post_response=$(curl --fail --silent --show-error \
  --request POST "${gateway_addr}/api/v1/posts" \
  --header 'Content-Type: application/json' \
  --header "Authorization: Bearer ${token}" \
  --data '{"title":"go-zero smoke test","content":"created through the REST gateway"}')

curl --fail --silent --show-error "${gateway_addr}/api/v1/posts" >/dev/null

printf 'smoke test passed; created post: %s\n' "${post_response}"

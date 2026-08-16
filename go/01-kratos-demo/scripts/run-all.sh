#!/usr/bin/env sh
set -eu

GATEWAY_CONFIG="${GATEWAY_CONFIG:-configs/local.yaml}"
USER_CONFIG="${USER_CONFIG:-configs/local.yaml}"
POST_CONFIG="${POST_CONFIG:-configs/local.yaml}"

pids=""
cleanup() {
	if [ -n "$pids" ]; then
		kill $pids 2>/dev/null || true
	fi
}
trap cleanup EXIT INT TERM

(cd user-service && CONFIG_PATH="$USER_CONFIG" go run ./cmd) &
pids="$pids $!"
(cd post-service && CONFIG_PATH="$POST_CONFIG" go run ./cmd) &
pids="$pids $!"
(cd gateway && CONFIG_PATH="$GATEWAY_CONFIG" go run ./cmd) &
pids="$pids $!"

wait

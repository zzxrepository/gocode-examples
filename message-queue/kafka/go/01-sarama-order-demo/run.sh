#!/bin/sh
set -eu
cd "$(dirname "$0")"
# 本机默认 Go 1.22 且关闭了工具链自动切换；仅调整这次命令的环境。
export GOTOOLCHAIN=auto
export GOPROXY="${KAFKA_GO_PROXY:-https://proxy.golang.org,direct}"
export GOSUMDB=sum.golang.org
exec go run "$@"

#!/bin/sh
set -eu
cd "$(dirname "$0")"
mkdir -p .cache
# 每次使用本项目源码构建，避免运行旧二进制。
GOPROXY="${KAFKA_GO_PROXY:-https://proxy.golang.org,direct}" GOSUMDB=sum.golang.org GOTOOLCHAIN=auto "${GO:-go}" build -o .cache/demo .
exec .cache/demo "$@"

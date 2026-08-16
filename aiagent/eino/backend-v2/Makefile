GO ?= /Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go
# 本机 shell 可能遗留旧版 GOROOT；v2 依赖要求 Go 1.24+，因此强制使用 1.26。
GOROOT := /Users/mmzhang/DevTools/SDK/go/1.26.5
GOCACHE ?= $(CURDIR)/.cache/go-build

.PHONY: run test fmt tidy

run:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./cmd/server

test:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) test ./...

fmt:
	$(GO) fmt ./...

tidy:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) mod tidy

GO ?= /Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go
GOCACHE ?= $(CURDIR)/.cache/go-build

.PHONY: run test tidy

run:
	GOROOT=/Users/mmzhang/DevTools/SDK/go/1.26.5 GOCACHE=$(GOCACHE) $(GO) run ./cmd/server

test:
	GOROOT=/Users/mmzhang/DevTools/SDK/go/1.26.5 GOCACHE=$(GOCACHE) $(GO) test ./...

tidy:
	GOROOT=/Users/mmzhang/DevTools/SDK/go/1.26.5 GOCACHE=$(GOCACHE) $(GO) mod tidy

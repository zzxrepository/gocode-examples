GO := /Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go
GOROOT := /Users/mmzhang/DevTools/SDK/go/1.26.5
GOCACHE ?= $(CURDIR)/.cache/go-build

.PHONY: run init tidy test

run:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run .

init:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run . -init

tidy:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) mod tidy

test:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) test ./...

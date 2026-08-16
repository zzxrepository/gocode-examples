GO := /Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go
GOROOT := /Users/mmzhang/DevTools/SDK/go/1.26.5
GOCACHE := $(CURDIR)/.cache/go-build

.PHONY: run tidy test docker-up docker-down

run:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./cmd/api

tidy:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) mod tidy

test:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) test ./...

docker-up:
	docker compose -f scripts/docker-compose.yml up -d

docker-down:
	docker compose -f scripts/docker-compose.yml down

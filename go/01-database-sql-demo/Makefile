GO := /Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go
GOROOT := /Users/mmzhang/DevTools/SDK/go/1.26.5
GOCACHE ?= $(CURDIR)/.cache/go-build

.PHONY: run run-sqlx tidy test docker-up docker-down docker-reset

run:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./cmd/database-sql-demo

run-sqlx:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./cmd/sqlx-demo

tidy:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) mod tidy

test:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) test ./...

docker-up:
	docker compose -f scripts/docker-compose.yml up -d

docker-down:
	docker compose -f scripts/docker-compose.yml down

docker-reset:
	docker compose -f scripts/docker-compose.yml down -v

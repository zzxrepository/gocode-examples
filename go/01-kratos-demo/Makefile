GO := /Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go
GOROOT := /Users/mmzhang/DevTools/SDK/go/1.26.5
GOCACHE := $(CURDIR)/.cache/go-build

.PHONY: run-gateway run-user run-post run-all tidy test generate docker-up docker-down

GATEWAY_CONFIG ?= configs/local.yaml
USER_CONFIG ?= configs/local.yaml
POST_CONFIG ?= configs/local.yaml

run-gateway:
	cd gateway && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) CONFIG_PATH=$(GATEWAY_CONFIG) $(GO) run ./cmd

run-user:
	cd user-service && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) CONFIG_PATH=$(USER_CONFIG) $(GO) run ./cmd

run-post:
	cd post-service && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) CONFIG_PATH=$(POST_CONFIG) $(GO) run ./cmd

run-all:
	PATH=$(dir $(GO)):$$PATH GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) GATEWAY_CONFIG=$(GATEWAY_CONFIG) USER_CONFIG=$(USER_CONFIG) POST_CONFIG=$(POST_CONFIG) sh scripts/run-all.sh

tidy:
	cd gateway && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) mod tidy
	cd user-service && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) mod tidy
	cd post-service && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) mod tidy

test:
	cd gateway && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) test ./...
	cd user-service && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) test ./...
	cd post-service && GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) test ./...

docker-up:
	docker compose -f scripts/docker-compose.yml up -d

docker-down:
	docker compose -f scripts/docker-compose.yml down

generate:
	PATH=$(dir $(GO)):$$PATH GOROOT=$(GOROOT) protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative user-service/api/user.proto
	PATH=$(dir $(GO)):$$PATH GOROOT=$(GOROOT) protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative post-service/api/post.proto

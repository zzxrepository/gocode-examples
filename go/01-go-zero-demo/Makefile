GO := /Users/mmzhang/DevTools/SDK/go/1.26.5/bin/go
GOROOT := /Users/mmzhang/DevTools/SDK/go/1.26.5
GOCACHE ?= $(CURDIR)/.cache/go-build

.PHONY: run-gateway run-user run-post run-all test tidy generate docker-up docker-down

run-gateway:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./gateway -f gateway/etc/gateway.yaml

run-user:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./user -f user/etc/user.yaml

run-post:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./post -f post/etc/post.yaml

run-all:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./user -f user/etc/user.yaml & \
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./post -f post/etc/post.yaml & \
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) run ./gateway -f gateway/etc/gateway.yaml

test:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) test ./...

tidy:
	GOROOT=$(GOROOT) GOCACHE=$(GOCACHE) $(GO) mod tidy

generate:
	PATH=/Users/mmzhang/DevWorkSpace/.go/bin:$(dir $(GO)):$$PATH GOROOT=$(GOROOT) goctl rpc protoc user/user.proto --go_out=./user/pb --go-grpc_out=./user/pb --zrpc_out=./user --module github.com/zzxrepository/gocode-examples/go/01-go-zero-demo
	PATH=/Users/mmzhang/DevWorkSpace/.go/bin:$(dir $(GO)):$$PATH GOROOT=$(GOROOT) goctl rpc protoc post/post.proto --go_out=./post/pb --go-grpc_out=./post/pb --zrpc_out=./post --module github.com/zzxrepository/gocode-examples/go/01-go-zero-demo
	PATH=/Users/mmzhang/DevWorkSpace/.go/bin:$(dir $(GO)):$$PATH GOROOT=$(GOROOT) goctl api go --api gateway/gateway.api --dir gateway

docker-up:
	docker compose -f scripts/docker-compose.yml up -d

docker-down:
	docker compose -f scripts/docker-compose.yml down

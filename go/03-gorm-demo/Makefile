GO ?= go

.PHONY: run init sql tidy test test-integration

run:
	$(GO) run .

init:
	$(GO) run . -init

sql:
	$(GO) run . -sql

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

test-integration:
	MYSQL_INTEGRATION=1 $(GO) test -count=1 -race -cover ./...

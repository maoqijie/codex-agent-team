CODEX_BIN ?= codex2
GO := go

.PHONY: build run test clean test-rpc

build:
	$(GO) build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	$(GO) test ./...

test-rpc:
	$(GO) run ./cmd/test-rpc

clean:
	rm -rf bin/

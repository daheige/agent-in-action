.PHONY: run build test vet run-web build-web

QUESTION ?= 你好

run:
	go run ./cmd/llm-gateway "$(QUESTION)"

build:
	mkdir -p bin
	go build -o bin/llm-gateway ./cmd/llm-gateway

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

run-web:
	go run ./cmd/web

build-web:
	mkdir -p bin
	go build -o bin/llm-gateway-web ./cmd/web

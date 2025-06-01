.DEFAULT_GOAL := build

fmt:
	go fmt ./...
.PHONY: fmt

lint: fmt
	staticcheck
.PHONY: lint

vet: fmt
	go vet ./...
.PHONY: vet

build: vet
	go mod tidy
	go build -ldflags="-s -w" -o bot
.PHONY: build

test:
	go test -v ./...

GOOSE_DIR := migrations

goose-up:
	goose --dir=$(GOOSE_DIR) up
.PHONY: goose-up

goose-down:
	goose --dir=$(GOOSE_DIR) down
.PHONY: goose-down

goose-status:
	goose --dir=$(GOOSE_DIR) status
.PHONY: goose-status

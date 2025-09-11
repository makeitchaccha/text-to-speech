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

test-coverage:
	go test -cover ./...

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

DOCKER_IMAGE_NAME := text-to-speech
DOCKER_IMAGE_TAG := latest

.PHONY: docker-build
docker-build:
	docker build -f docker/Dockerfile -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

.PHONY: docker-run
docker-run:
	docker run --rm -it \
		--env-file config.toml \
		$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)

.PHONY: docker-compose-up
docker-compose-up:
	docker compose up -d

.PHONY: docker-compose-down
docker-compose-down:
	docker compose down

.PHONY: docker-compose-logs
docker-compose-logs:
	docker compose logs -f


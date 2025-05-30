fmt:
	go fmt ./...
.PHONY: fmt

lint: fmt
	staticcheck
.PHONY: lint

vet: fmt
	go vet ./...
.PHONY: vet
.PHONY: all build test lint clean install coverage help

BINARY_NAME=intentra
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

all: lint test build

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/intentra

install:
	go install $(LDFLAGS) ./cmd/intentra

test:
	go test -race -v ./...

coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, running go vet only"; \
		go vet ./...; \
	fi

fmt:
	go fmt ./...

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

help:
	@echo "Available targets:"
	@echo "  all       - Run lint, test, and build"
	@echo "  build     - Build the binary"
	@echo "  install   - Install to GOPATH/bin"
	@echo "  test      - Run tests"
	@echo "  coverage  - Run tests with coverage"
	@echo "  lint      - Run linter"
	@echo "  fmt       - Format code"
	@echo "  clean     - Remove build artifacts"
	@echo "  snapshot  - Create snapshot release"
	@echo "  release   - Create release (requires GITHUB_TOKEN)"
	@echo "  help      - Show this help"

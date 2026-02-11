.PHONY: build clean test install lint fmt deps help

BINARY_NAME=buildbuddy-cli
GO_FILES=$(shell find . -type f -name '*.go' | grep -v vendor/)
VERSION=$(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X github.com/roboalchemist/buildbuddy-cli/pkg/api.version=$(VERSION)"

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	go clean

test:
	@echo "Running smoke tests..."
	@./smoke_test.sh

test-verbose:
	@echo "Running smoke tests (verbose)..."
	@bash -x ./smoke_test.sh

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Linting code..."
	golangci-lint run

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

dev-install: build
	@echo "Creating development symlink..."
	sudo ln -sf $(PWD)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .

release: clean
	@echo "Preparing release..."
	mkdir -p dist
	$(MAKE) build-all

run: build
	./$(BINARY_NAME)

help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run smoke tests"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  install      - Install binary to system"
	@echo "  dev-install  - Create development symlink"
	@echo "  build-all    - Cross-compile for all platforms"
	@echo "  release      - Prepare release builds"
	@echo "  help         - Show this help"

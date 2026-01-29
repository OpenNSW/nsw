# ==============================================================================
# Variables
# ==============================================================================
# Binary & Project
BINARY_NAME  := server
MAIN_PACKAGE := ./cmd/server/main.go
BIN_DIR      := bin
DOCKER_IMAGE := backend-service

# Linker Flags
# -s -w: Strips debug info for smaller binary size
LDFLAGS := -ldflags "-s -w"

# Tooling
LINTER_VERSION := v1.63.4

# ==============================================================================
# Environment Configuration
# ==============================================================================
# Load .env file if it exists, but do not error if it doesn't (useful for CI)
ifneq (,$(wildcard ./.env))
	include .env
	export
endif

# ==============================================================================
# Targets
# ==============================================================================
.PHONY: all run build build-linux deps test test-cov lint docker clean help

.DEFAULT_GOAL := help

all: deps build ## Run dependencies check and build binary

run: ## Run the server locally without compiling a binary
	go run $(LDFLAGS) $(MAIN_PACKAGE)

build: ## Build binary for the current OS (Mac/Linux/Windows)
	@echo "Building $(BINARY_NAME) for local OS..."
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

build-linux: ## Build static binary for Linux/AMD64 (Best for Docker/Alpine)
	@echo "Building static $(BINARY_NAME) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux $(MAIN_PACKAGE)

deps: ## Download and tidy Go modules
	go mod tidy
	go mod verify

test: ## Run unit tests (with race detection)
	go test -race -v ./...

test-cov: ## Run tests with coverage reporting
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

lint: ## Run golangci-lint (downloads binary to tmp)
	@echo "Running Linter..."
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@$(LINTER_VERSION) run ./...

docker: ## Build Docker image (Passes build args)
	docker build \
		-t $(DOCKER_IMAGE):latest .

clean: ## Cleanup binaries and coverage files
	rm -rf $(BIN_DIR) coverage.out

help: ## Show this help message
	@echo "Usage: make [target]"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
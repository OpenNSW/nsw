.PHONY: all run build linux test lint clean docker help

# ==============================================================================
# Main Configuration
# ==============================================================================
BINARY_NAME := server
MAIN_PACKAGE := ./cmd/server/main.go
DOCKER_IMAGE := backend-service

# Versioning (Injects Git info)
VERSION := $(shell git describe --tags --always --dirty)
COMMIT  := $(shell git rev-parse --short HEAD)
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

# Tooling Versions
LINTER_VERSION := v1.63.4

# Load .env if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# ==============================================================================
# Targets
# ==============================================================================

.DEFAULT_GOAL := help

all: build

run: ## Run the server locally
	go run $(LDFLAGS) $(MAIN_PACKAGE)

build: ## Build binary for current OS
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

linux: ## Build binary for Linux (production/docker)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux $(MAIN_PACKAGE)

test: ## Run tests (includes race detection & vet)
	go test -v -race ./...

lint: ## Run linter (installs automatically via go run)
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@$(LINTER_VERSION) run

docker: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

clean: ## Cleanup build artifacts
	rm -rf bin/

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
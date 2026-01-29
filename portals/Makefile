.PHONY: help setup install clean dev-oga dev-trader dev-all build build-oga build-trader build-ui lint lint-fix test preview-oga preview-trader

# Default target
help:
	@echo "📦 NSW Portals - Development Commands"
	@echo ""
	@echo "Setup & Installation:"
	@echo "  make setup          - Install pnpm globally and setup workspace"
	@echo "  make install        - Install all dependencies"
	@echo "  make clean          - Remove node_modules and lock files"
	@echo ""
	@echo "Development:"
	@echo "  make dev-oga        - Run OGA app in dev mode"
	@echo "  make dev-trader     - Run Trader app in dev mode"
	@echo "  make dev-all        - Run all apps concurrently"
	@echo ""
	@echo "Build:"
	@echo "  make build          - Build all workspaces"
	@echo "  make build-oga      - Build OGA app only"
	@echo "  make build-trader   - Build Trader app only"
	@echo "  make build-ui       - Build UI library only"
	@echo ""
	@echo "Quality:"
	@echo "  make lint           - Run ESLint on all workspaces"
	@echo "  make lint-fix       - Run ESLint with --fix"
	@echo ""

# Setup pnpm if not installed
setup:
	@echo "🔧 Setting up development environment..."
	@command -v pnpm >/dev/null 2>&1 || { \
		echo "📥 Installing pnpm globally..."; \
		npm install -g pnpm@10.28.1; \
	}
	@echo "✅ pnpm is installed"
	@$(MAKE) install
	@echo "✅ Setup complete!"

# Install dependencies
install:
	@echo "📦 Installing dependencies with pnpm..."
	@pnpm install

# Clean all node_modules and lock files
clean:
	@echo "🧹 Cleaning workspace..."
	@rm -rf node_modules
	@rm -rf apps/*/node_modules
	@rm -rf ui/node_modules
	@rm -rf ui/dist apps/*/dist
	@rm -rf apps/*/.turbo
	@echo "✅ Cleaned!"

# Development servers
dev-oga:
	@echo "🚀 Starting OGA app..."
	@pnpm --filter oga-app dev

dev-trader:
	@echo "🚀 Starting Trader app..."
	@pnpm --filter trader-app dev

dev-all:
	@echo "🚀 Starting all apps concurrently..."
	@pnpm --parallel --filter "./apps/*" dev

# Build commands
build:
	@echo "🏗️  Building all workspaces..."
	@pnpm -r build

build-oga:
	@echo "🏗️  Building OGA app..."
	@pnpm --filter oga-app build

build-trader:
	@echo "🏗️  Building Trader app..."
	@pnpm --filter trader-app build

build-ui:
	@echo "🏗️  Building UI library..."
	@pnpm --filter @lsf/ui build

# Linting
lint:
	@echo "🔍 Running ESLint..."
	@pnpm -r lint

lint-fix:
	@echo "🔧 Running ESLint with --fix..."
	@pnpm -r exec eslint . --fix

# Preview production builds
preview-oga:
	@echo "👀 Previewing OGA app..."
	@pnpm --filter oga-app preview

preview-trader:
	@echo "👀 Previewing Trader app..."
	@pnpm --filter trader-app preview

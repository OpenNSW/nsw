# Portals Workspace

A monorepo containing shared UI components and multiple applications built with React and Radix UI.

> **📦 Using pnpm** - Faster installs, better disk usage, single lock file for the entire monorepo

## Quick Start

```bash
# First time setup
make setup      # Installs pnpm (if needed) + all dependencies

# Start developing
make dev-oga    # Start OGA app
make dev-trader # Start Trader app
make help       # See all available commands
```

### Migrating from npm?
Run `make clean && make setup` to remove old npm artifacts and install with pnpm.

## Project Structure

```
portals/
├── Makefile               # Team development commands
├── pnpm-workspace.yaml    # pnpm workspace configuration
├── pnpm-lock.yaml         # Single lock file for entire monorepo
├── package.json           # Root workspace configuration
├── tsconfig.json          # Shared TypeScript configuration
├── ui/                    # Shared UI component library (@lsf/ui)
│   └── src/
│       └── components/    # Reusable components built on Radix UI
└── apps/                  # Consumer applications
    ├── oga-app/           # OGA portal application
    └── trader-app/        # Trading application
```

## Overview

### UI Library (`@lsf/ui`)

The `ui` package is a shared component library built from the ground up using [Radix UI](https://www.radix-ui.com/) primitives. It provides accessible, unstyled, and customizable components that can be consumed by any application in the monorepo.

**Key features:**
- Built on Radix UI primitives for accessibility and flexibility
- React 19 support
- TypeScript-first approach
- Bundled with Vite for ESM and CJS outputs

**Available components:**
- `Button`
- `Card`

### Apps

The `apps/` directory contains applications that consume the shared UI library. Each app is a standalone project that imports components from `@lsf/ui`.

**Current apps:**
- `trader-app` - Trading application

## Getting Started

### Prerequisites

- Node.js >= 18
- pnpm >= 9 (installed automatically via `make setup`)

### Setup

```bash
# First time setup (installs pnpm + dependencies)
make setup
```

If you prefer to install pnpm manually:
```bash
npm install -g pnpm
pnpm install
```

## Development Commands

Run `make help` to see all available commands.

### Common Tasks

```bash
# Development
make dev-oga        # Start OGA app
make dev-trader     # Start Trader app
make dev-all        # Start all apps in parallel

# Building
make build          # Build all workspaces
make build-ui       # Build UI library only

# Code quality
make lint           # Run linter
make lint-fix       # Auto-fix linting issues
```

### Adding Dependencies

```bash
# To a specific app
pnpm --filter oga-app add axios

# To UI library
pnpm --filter @lsf/ui add lodash

# To workspace root (dev dependencies)
pnpm add -w prettier -D
```

### Troubleshooting

**"pnpm: command not found"**
```bash
make setup  # Installs pnpm automatically
```

**"Cannot find module '@lsf/ui'"**
```bash
make build-ui  # Build the UI library first
```

**TypeScript errors after pulling**
```bash
make clean && make install && make build
```

## Using the UI Library

Import components from `@lsf/ui` in any app:

```tsx
import { Button, Card } from '@lsf/ui'

function MyComponent() {
  return (
    <Card>
      <Button>Click me</Button>
    </Card>
  )
}
```

## Adding New Components

1. Create a new component in `ui/src/components/`
2. Export it from `ui/src/index.ts`
3. Rebuild the UI library

## Adding New Apps

1. Create a new directory in `apps/`
2. Initialize the app with your preferred framework
3. Add `@lsf/ui` as a dependency in the app's `package.json`:
   ```json
   {
     "dependencies": {
       "@lsf/ui": "workspace:*"
     }
   }
   ```
4. Run `pnpm install` from the root

## Tech Stack

- **React** 19
- **Radix UI** - Unstyled, accessible component primitives
- **TypeScript** - Type safety
- **Vite** - Build tooling
- **pnpm** - Fast, efficient package manager

## Why pnpm?

- ⚡ **2x faster** than npm
- 💾 **30-50% less disk space** via content-addressable storage
- 🔒 **Stricter** - prevents phantom dependencies
- 🎯 **Single lock file** - better for monorepos
- ✅ **Industry standard** - used by Vue, Vite, Svelte, and more
- **npm Workspaces** - Monorepo management
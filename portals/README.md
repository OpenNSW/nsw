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

### 📚 New to the Project?
See **[SETUP_WORKSPACE.md](docs/SETUP_WORKSPACE.md)** for:
- Complete setup instructions (new developers & existing team migration)
- Version requirements and enforcement
- Troubleshooting common issues
- Verification checklist

---

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
- `oga-app` - OGA portal application
- `trader-app` - Trading application

---

## Development Workflow

### Common Commands

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

---

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

---

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

---

## Need Help?

- **Setup & Troubleshooting:** See [SETUP_WORKSPACE.md](docs/SETUP_WORKSPACE.md)
- **Available Commands:** Run `make help`
- **Issues:** Check the [troubleshooting section](docs/SETUP_WORKSPACE.md#troubleshooting) in SETUP_WORKSPACE.md
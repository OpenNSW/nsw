# Portals Workspace

A monorepo containing shared UI components and multiple applications built with React and Radix UI.

## Project Structure

```
portals/
├── ui/                    # Shared UI component library (@lsf/ui)
│   └── src/
│       └── components/    # Reusable components built on Radix UI
├── apps/                  # Consumer applications
│   └── trader-app/        # Trading application
├── package.json           # Root workspace configuration
└── tsconfig.json          # Shared TypeScript configuration
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
- npm >= 9

### Installation

Install all dependencies from the root:

```bash
npm install
```

### Building the UI Library

```bash
cd ui
npm run build
```

### Running an App

```bash
cd apps/trader-app
npm run dev
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
       "@lsf/ui": "*"
     }
   }
   ```
4. Run `npm install` from the root

## Tech Stack

- **React** 19
- **Radix UI** - Unstyled, accessible component primitives
- **TypeScript** - Type safety
- **Vite** - Build tooling
- **npm Workspaces** - Monorepo management
# Workspace Setup Guide

This guide helps you set up a consistent development environment for the NSW Portals monorepo. Follow the appropriate section based on whether you're a new developer or migrating from an existing setup.

## 📋 Table of Contents

- [Version Requirements](#version-requirements)
- [New Developer Setup](#new-developer-setup)
- [Existing Developer Migration](#existing-developer-migration)
- [Troubleshooting](#troubleshooting)
- [Verification Checklist](#verification-checklist)

---

## 🔧 Version Requirements

This project uses **strict version enforcement** to ensure consistent dependency resolution and lockfile generation across all team members.

| Tool | Required Version | Why? |
|------|-----------------|------|
| **Node.js** | `v22.18.0` | Locked to prevent lockfile inconsistencies |
| **pnpm** | `v10.28.1` | Enforced via `packageManager` field |

> **⚠️ Important**: Using different versions will cause `pnpm-lock.yaml` to change unexpectedly, creating merge conflicts and CI failures.

### Why These Specific Versions?

- **Node v22.x**: Latest stable version with modern JavaScript features
- **pnpm v10.28.x**: Latest stable version with improved monorepo support
- **Locked versions**: Prevents platform-specific lockfile differences (e.g., `libc: [glibc]` appearing/disappearing)

---

## 🆕 New Developer Setup

Follow these steps if you're setting up the project for the first time.

### Step 1: Install Node Version Manager (nvm)

**macOS/Linux:**
```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash

# Restart your terminal, then verify installation
nvm --version
```

**Windows:**
- Download and install [nvm-windows](https://github.com/coreybutler/nvm-windows/releases)
- Or use [Volta](https://volta.sh/) as an alternative

### Step 2: Clone and Navigate to Project

```bash
git clone https://github.com/OpenNSW/nsw.git
cd nsw/portals
```

### Step 3: Install Required Node Version

```bash
# Install the exact Node version (reads from .nvmrc)
nvm install

# Use the installed version
nvm use

# Verify installation
node --version
# Expected output: v22.18.0
```

### Step 4: Install pnpm

pnpm will be automatically installed via **Corepack** (built into Node.js):

```bash
# Enable Corepack (already included in Node 22+)
corepack enable

# Verify pnpm installation
pnpm --version
# Expected output: 10.28.1
```

**Alternative (manual installation):**
```bash
npm install -g pnpm@10.28.1
```

### Step 5: Install Dependencies

```bash
# Install all workspace dependencies
pnpm install

# This will install dependencies for:
# - Root workspace
# - apps/oga-app
# - apps/trader-app
# - ui package
```

### Step 6: Build Shared Packages

```bash
# Build the shared UI library
make build-ui

# Or use pnpm directly
pnpm --filter @lsf/ui build
```

### Step 7: Start Development

```bash
# Start the OGA app
make dev-oga

# OR start the Trader app
make dev-trader

# OR start all apps in parallel
make dev-all
```

### Step 8: Verify Everything Works

Run the verification checklist at the end of this document.

---

## 🔄 Existing Developer Migration

Follow these steps if you're already working on the project and need to migrate to the standardized setup.

### Why Migrate?

You may experience:
- ❌ `pnpm-lock.yaml` changes on every `pnpm install`
- ❌ Lockfile conflicts with teammates
- ❌ CI/CD failures due to lockfile mismatches
- ❌ Mysterious dependency issues

### Migration Steps

#### 1. Commit or Stash Your Work

```bash
# Save your current work
git add .
git stash

# Or commit if ready
git commit -m "WIP: save current work"
```

#### 2. Pull Latest Changes

```bash
git checkout main
git pull origin main

# Switch back to your branch
git checkout your-branch-name
git merge main  # or rebase: git rebase main
```

#### 3. Clean Old Installations

```bash
cd portals

# Remove ALL node_modules (including nested)
rm -rf node_modules
rm -rf apps/*/node_modules
rm -rf ui/node_modules

# Remove old npm artifacts (if migrating from npm)
rm -f package-lock.json
rm -rf .npm
```

#### 4. Install/Update Node Version

```bash
# If you don't have nvm, install it first (see New Developer Setup)

# Install the correct Node version
nvm install

# Use it
nvm use

# Verify
node --version
# Must show: v22.18.0
```

#### 5. Install/Update pnpm

```bash
# Method 1: Via Corepack (recommended)
corepack enable
pnpm --version

# Method 2: Manual installation
npm install -g pnpm@10.28.1

# Verify
pnpm --version
# Must show: 10.28.1
```

#### 6. Fresh Install

```bash
# Install all dependencies with correct versions
pnpm install

# Rebuild shared packages
make build-ui

# Or manually
pnpm --filter @lsf/ui build
```

#### 7. Verify Lockfile Stability

```bash
# Run install again - lockfile should NOT change
pnpm install

# Check git status
git status

# If pnpm-lock.yaml shows as modified, you may have version mismatches
# Run the verification checklist below
```

#### 8. Resume Development

```bash
# If you stashed changes
git stash pop

# Start your development server
make dev-oga  # or make dev-trader
```

---

## 🔍 Troubleshooting

### Problem: `engine-strict` error when running `pnpm install`

```
ERR_PNPM_BAD_NODE_VERSION  Unsupported Node.js version
```

**Solution:**
```bash
# You're not using the correct Node version
nvm use

# Verify
node --version  # Must be v22.18.0
```

---

### Problem: `pnpm-lock.yaml` keeps changing

**Symptoms:**
- Running `pnpm install` modifies the lockfile
- Lines like `libc: [glibc]` appear/disappear
- Lockfile conflicts with teammates

**Solution:**
```bash
# 1. Verify Node version
node --version
# Expected: v22.18.0
# If wrong: nvm use

# 2. Verify pnpm version
pnpm --version
# Expected: 10.28.1
# If wrong: npm install -g pnpm@10.28.1 OR corepack enable

# 3. Clean install
rm -rf node_modules apps/*/node_modules ui/node_modules
pnpm install

# 4. Test stability
pnpm install
git diff pnpm-lock.yaml
# Should show no changes
```

---

### Problem: `Cannot find module '@lsf/ui'`

**Symptoms:**
- TypeScript or runtime errors about missing `@lsf/ui` module
- Import statements fail

**Solution:**
```bash
# The UI library needs to be built first
make build-ui

# Or manually
pnpm --filter @lsf/ui build
```

---

### Problem: `pnpm: command not found`

**Solution:**
```bash
# Enable Corepack
corepack enable

# Or install manually
npm install -g pnpm@10.28.1
```

---

### Problem: Different behavior between team members

**Symptoms:**
- Works on one machine, fails on another
- Different test results or build outputs

**Root Cause:** Version mismatches

**Solution:**
Everyone runs the verification checklist below.

---

## ✅ Verification Checklist

Run these commands to verify your setup is correct:

```bash
# 1. Check Node version
node --version
# ✅ Expected: v22.18.0

# 2. Check pnpm version
pnpm --version
# ✅ Expected: 10.28.1

# 3. Check if in correct directory
pwd
# ✅ Should end with: /portals

# 4. Verify .nvmrc exists
cat .nvmrc
# ✅ Should show: 22.18.0

# 5. Verify .npmrc exists
cat .npmrc
# ✅ Should contain: engine-strict=true

# 6. Test lockfile stability
pnpm install
git diff pnpm-lock.yaml
# ✅ Should show: no changes

# 7. Verify workspace packages
pnpm list --depth=0
# ✅ Should list: oga-app, trader-app, @lsf/ui

# 8. Test build
make build-ui
# ✅ Should complete without errors

# 9. Test development server
make dev-oga
# ✅ Should start without errors
# Press Ctrl+C to stop
```

### All Green? ✅

You're all set! Start coding:

```bash
# See all available commands
make help

# Common workflows:
make dev-oga        # Start OGA app
make dev-trader     # Start Trader app
make build          # Build all packages
make lint           # Run linter
make lint-fix       # Auto-fix linting issues
```

---

## 🆘 Still Having Issues?

### Check Team Consistency

Ask a teammate to run:

```bash
node --version && pnpm --version
```

Compare outputs. Everyone should have:
- Node: `v22.18.0`
- pnpm: `10.28.1`

### Clean Slate Reset

Nuclear option if nothing else works:

```bash
# 1. Remove everything
cd portals
rm -rf node_modules apps/*/node_modules ui/node_modules
rm -rf .pnpm-store pnpm-lock.yaml

# 2. Reinstall Node/pnpm (start from scratch)
nvm install 22.18.0
nvm use 22.18.0
npm install -g pnpm@10.28.1

# 3. Fresh install
pnpm install
make build-ui
```

### Contact the Team

If you're still stuck:
1. Share your verification checklist output
2. Share `git diff pnpm-lock.yaml` output
3. Ask in the team channel

---

## 📚 Additional Resources

- [pnpm Documentation](https://pnpm.io/)
- [nvm Documentation](https://github.com/nvm-sh/nvm)
- [Node.js Releases](https://nodejs.org/en/about/previous-releases)
- [Makefile Commands](./Makefile) - Run `make help`

---

## 🔐 Security Note

Always keep your Node.js and pnpm versions up to date with security patches. The team will coordinate version updates through pull requests to ensure everyone stays synchronized.

**Current versions locked as of:** January 2026
- Node.js: v22.18.0
- pnpm: v10.28.1

When updating these versions, the team lead will:
1. Update `.nvmrc`
2. Update `package.json` `packageManager` field
3. Update `.npmrc` if needed
4. Regenerate `pnpm-lock.yaml`
5. Notify all team members to migrate

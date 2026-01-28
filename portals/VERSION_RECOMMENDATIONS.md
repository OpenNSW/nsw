# Version Update Recommendation

## Current Status

**Current Versions (as of January 2026):**
- Node.js: `v22.18.0` 
- pnpm: `v10.28.1`

**Latest Versions Available:**
- Node.js LTS: `v24.13.0` (Krypton) - **Recommended upgrade**
- pnpm: `v10.28.2` - Minor patch available

## Should We Update?

### ⚠️ Node.js: v22.18.0 → v24.13.0

**Status:** Node v24 is now the **Active LTS** (Long Term Support)

**Benefits of upgrading:**
- ✅ Active LTS support until April 2027
- ✅ Latest security patches
- ✅ Better performance
- ✅ Extended support timeline

**Considerations:**
- ⚠️ Requires team coordination (everyone must update together)
- ⚠️ May need to test compatibility with all dependencies
- ⚠️ Breaking changes possible (though v22→v24 is usually smooth)

**Recommendation:** 
> **UPGRADE RECOMMENDED** - Node v24 is the current LTS and provides better long-term support. However, coordinate with the team and test thoroughly before migrating.

### 📦 pnpm: v10.28.1 → v10.28.2

**Status:** Minor patch update

**Benefits:**
- ✅ Bug fixes
- ✅ Small improvements
- ✅ No breaking changes

**Recommendation:**
> **SAFE TO UPDATE** - This is a patch version with bug fixes only. Low risk.

---

## How to Update (If Approved by Team)

### Step 1: Update Node.js to v24.13.0

1. **Update `.nvmrc`:**
   ```bash
   echo "24.13.0" > .nvmrc
   ```

2. **Update `package.json`:**
   ```json
   {
     "engines": {
       "node": ">=24.0.0",
       "pnpm": ">=10.28.2"
     }
   }
   ```

3. **Update `.npmrc`:**
   ```ini
   use-node-version=24.13.0
   ```

### Step 2: Update pnpm to v10.28.2

1. **Update `package.json`:**
   ```json
   {
     "packageManager": "pnpm@10.28.2"
   }
   ```

### Step 3: Test Locally

```bash
# Install new Node version
nvm install 24.13.0
nvm use 24.13.0

# Verify
node --version  # Should show v24.13.0

# Update pnpm
corepack enable
# OR
npm install -g pnpm@10.28.2

# Clean install
rm -rf node_modules apps/*/node_modules ui/node_modules
pnpm install

# Run tests
make build
make dev-oga  # Verify app starts

# Check lockfile changes
git diff pnpm-lock.yaml
```

### Step 4: Team Migration Plan

1. **Create a branch:** `chore/update-node-pnpm-versions`
2. **Update all version files** (as shown above)
3. **Regenerate lockfile:** Run `pnpm install` on macOS/Linux to get canonical lockfile
4. **Create PR** with clear migration instructions
5. **Notify team** via Slack/Teams before merging
6. **Coordinate merge:** Pick a time when everyone can migrate together
7. **Team updates:** Everyone follows migration steps in SETUP_WORKSPACE.md

### Step 5: Update Documentation

Update `SETUP_WORKSPACE.md` and `README.md` with new version numbers.

---

## Decision Matrix

| Scenario | Action | Priority |
|----------|--------|----------|
| **Critical security vulnerability** | Update immediately | 🔴 High |
| **Active LTS available (v24)** | Plan upgrade within 1-2 weeks | 🟡 Medium |
| **Patch version (10.28.2)** | Update when convenient | 🟢 Low |
| **New major version (v25)** | Wait for LTS status | ⚪ Low |

## Current Recommendation

```bash
# Immediate action: Update pnpm patch version
# Priority: 🟢 Low - Safe, but optional

# Planned action: Update to Node v24 LTS
# Priority: 🟡 Medium - Should plan migration
# Timeline: Within next 2 weeks
# Reason: v24 is now the active LTS with better support
```

---

## Testing Checklist Before Team Rollout

- [ ] Fresh install works (`rm -rf node_modules && pnpm install`)
- [ ] All apps build successfully (`make build`)
- [ ] All apps run in dev mode (`make dev-oga`, `make dev-trader`)
- [ ] Tests pass (if you have tests)
- [ ] Linting passes (`make lint`)
- [ ] No new warnings in console
- [ ] Dependencies resolve correctly
- [ ] Lockfile stable (run `pnpm install` twice, no changes)
- [ ] Works on macOS
- [ ] Works on Linux (test in CI or Docker)
- [ ] Works on Windows (if team uses Windows)

---

## Rollback Plan

If the upgrade causes issues:

```bash
# 1. Revert version files
git checkout main -- .nvmrc package.json .npmrc

# 2. Switch back to old Node version
nvm use 22.18.0

# 3. Reinstall old pnpm
npm install -g pnpm@10.28.1

# 4. Restore lockfile
git checkout main -- pnpm-lock.yaml

# 5. Clean reinstall
rm -rf node_modules apps/*/node_modules ui/node_modules
pnpm install
```

---

## Summary

**Recommendation for your team:**

1. **✅ Update pnpm to v10.28.2** - Safe patch update, minimal risk
2. **🟡 Plan Node.js v24 upgrade** - Current LTS, but coordinate with team first
3. **📝 Create migration plan** - Use this document as guide
4. **🧪 Test thoroughly** - Run full test suite before team rollout
5. **👥 Coordinate rollout** - Pick a time when team can update together

**Next steps:**
1. Discuss with team lead
2. Create upgrade branch
3. Test locally
4. Schedule team migration
5. Update all documentation

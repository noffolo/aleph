# Final macOS Cycle — E2E Dedup + Push

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Consolidate orphaned Playwright E2E test files, verify all macOS-testable code, push final commit.

**Architecture:** The Playwright config at `frontend/playwright.config.ts` (testDir: `./tests/e2e/`) is the active config via `npm run e2e`. 11 spec files exist in `frontend/e2e/` but are NEVER executed — they need moving into `tests/e2e/`. The orphaned config `frontend/tests/e2e/playwright.config.ts` has a richer webServer array (starts both Go backend + Vite) but is never loaded. Root config only starts Vite.

**Tech Stack:** Playwright 1.52, TypeScript, Vite

---

### Task A1: Consolidate orphaned E2E test files

**Files:**
- Modify: `frontend/playwright.config.ts:31-36` — replace simple webServer with full-stack config
- Move: `frontend/e2e/*.spec.ts` → `frontend/tests/e2e/`
- Move: `frontend/e2e/connect-mock-helper.ts` → `frontend/tests/e2e/`
- Delete: `frontend/e2e/`
- Delete: `frontend/tests/e2e/playwright.config.ts` (orphaned duplicate)

**Problem:** `frontend/e2e/` has 11 spec files and 1 helper: auth-flow, commands, error-states, journey, onboarding, ontology-flow, sanitization, settings-flow, slideover, tool-lifecycle. None are executed because `frontend/playwright.config.ts` has `testDir: './tests/e2e'`. The orphaned config at `frontend/tests/e2e/playwright.config.ts` starts both Go backend + Vite but is never loaded (Playwright auto-discovers `frontend/playwright.config.ts` from `package.json`).

- [ ] **Step 1: Move all files from `frontend/e2e/` to `frontend/tests/e2e/`**

```bash
mv frontend/e2e/*.spec.ts frontend/tests/e2e/
mv frontend/e2e/connect-mock-helper.ts frontend/tests/e2e/
```

- [ ] **Step 2: Update root config to start both Go backend and Vite**

Replace the root `playwright.config.ts` webServer entry with the full-stack version:

```typescript
  // Replace lines 31-36:
  webServer: [
    {
      command: 'go run ./cmd/aleph serve',
      url: 'http://localhost:8080/api/v1/health',
      reuseExistingServer: !process.env.CI,
      timeout: 30000,
      cwd: __dirname + '/..',
    },
    {
      command: 'npx vite --port 5173',
      url: 'http://localhost:5173',
      reuseExistingServer: !process.env.CI,
      timeout: 30000,
    },
  ],
```

- [ ] **Step 3: Delete orphaned directory and duplicate config**

```bash
rm frontend/tests/e2e/playwright.config.ts
rmdir frontend/e2e/
```

- [ ] **Step 4: Verify**

```bash
ls frontend/tests/e2e/
# Expected: all .spec.ts files + connect-mock-helper.ts + smoke.spec.ts
# NO playwright.config.ts in tests/e2e/

ls frontend/e2e/
# Expected: "No such file or directory"
```

- [ ] **Step 5: Commit**

```bash
git add frontend/
git rm --cached frontend/e2e/ 2>/dev/null; true
git commit -m "fix: consolidate orphaned E2E test files into tests/e2e/"
```

---

### Task C1: Full suite verification + push + gitnexus

- [ ] **Step 1: Run full Go suite**

```bash
go build ./...
go vet ./...
go test -count=1 -race ./internal/... 2>&1 | tail -5
# Expected: ok or FAIL (report failures)
```

- [ ] **Step 2: Run frontend suite**

```bash
npx tsc --noEmit 2>&1 | tail -10
# Expected: 0 errors

npx vitest run 2>&1 | tail -5
# Expected: 1358 tests, all pass
```

- [ ] **Step 3: GitNexus reindex**

```bash
npx gitnexus analyze
```

- [ ] **Step 4: Push to main**

```bash
git push origin main
```

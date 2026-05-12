# W0 — Foundation: Audit, Feature Flags, Bugfix, and Test Baseline

- **Wave**: W0 (UX Redesign Foundation)
- **Status**: Draft
- **Owner**: Sisyphus-Junior
- **Dependencies**: None (precedes all other waves)
- **Risk**: LOW — no UI changes, only audit artifacts and targeted bugfixes
- **Estimated effort**: 5 deliverables, ~2–3 days

---

## 1. Purpose & Scope

W0 establishes the factual baseline for the entire UX redesign. Before touching a single component, we must know exactly what we're working with: every store field, every subscriber, every rendering path, every test. This wave produces no UI changes — only audit documents, a feature-flag system, one critical bugfix, and a test migration note.

**What W0 is NOT:**
- Not a refactor
- Not a redesign
- Not a performance optimization
- Not a component rewrite

**What W0 IS:**
- A rigorous inventory of current frontend state
- The scaffolding (feature flags) that lets subsequent waves ship incrementally
- A single targeted bugfix (resetHealth) that prevents cascading failures in W1
- A migration roadmap for the 10 Playwright tests that will need updating

---

## 2. Deliverables with Acceptance Criteria

### D1 — Full Codebase Audit Document

**File**: `docs/specs/ux-redesign-w0-audit.md`

**Description**: A complete catalog of every Zustand store field, its type, current slice location, and all consumer components. Also catalogs the 3 full-store subscribers, 11 InlineRenderer views, 20 SlideOverContent types, and the dual command system.

**Acceptance criteria**:
- [ ] Lists all 61 state fields across all 6 slices with exact TypeScript types
- [ ] For each field, documents every consumer component with file path and line reference
- [ ] Identifies the 3 files calling `useStore()` without a selector (CopilotView, OracleView, TerminalEffects)
- [ ] Lists all 11 InlineRenderer view types with the props each view receives
- [ ] Lists all 20 implemented SlideOverContent types (distinct from the 24 defined in the union)
- [ ] Documents the dual command system: 5 hardcoded commands in CommandPalette + 16 dynamic slashCommands
- [ ] Maps the fragmented explorer state across workspaceSlice (projectID, selectedObject, data, searchQuery), navigationSlice (activeView), and uiSlice (filteredTools concept)
- [ ] Identifies dead fields: `showGuide` (zero consumers), `pendingCrud` (CRUD actions manage own state), `ontologyVersions`/`selectedVersionId`/`isVersionHistoryOpen` (view-local)
- [ ] Documents the 3 rendering modes: SetupWizard (`showWizard`), WorkspaceOnboarding (`showOnboarding`), Full App

**Momus review pass**: Must verify the audit against actual source code via grep — no AI hallucinated fields or consumers.

### D2 — Feature Flags System

**File**: `frontend/src/config/featureFlags.ts` (new)

**Description**: A compile-time feature flag module that gates every UX redesign wave. Each wave gets a boolean flag. When a flag is `false`, the code path is identical to current production behavior. When `true`, the new code path activates.

**Design**:
```typescript
// frontend/src/config/featureFlags.ts
export const featureFlags = {
  /** W1: Store refactor — field consolidation, slice merge, dead field removal */
  storeRefactor: false,
  /** W2: Navigation redesign — sidebar 13→5, scene routing, command palette unification */
  navRedesign: false,
  /** W3: SlideOver unification — SHOW_INLINE reroute to SlideOver, InlineRenderer removal */
  slideOverUnification: false,
  /** W4: Copilot slim — CopilotView decomposition, TerminalView cleanup */
  copilotSlim: false,
  /** W5: Progressive disclosure — onboarding revamp, settings reorganization */
  progressiveDisclosure: false,
} as const

export type FeatureFlag = keyof typeof featureFlags
export function isEnabled(flag: FeatureFlag): boolean {
  return featureFlags[flag]
}
```

**Consumer convention**: Any conditional code path based on a feature flag MUST be clearly marked with a comment:
```typescript
// [W2-navRedesign] New sidebar scene routing path
if (isEnabled('navRedesign')) { ... }
```

**Acceptance criteria**:
- [ ] File created at `frontend/src/config/featureFlags.ts`
- [ ] Exports 5 boolean flags typed as `const` (literal types for dead-code elimination)
- [ ] Exports `isEnabled()` function that TypeScript can narrow via type guard
- [ ] Each flag defaults to `false`
- [ ] TypeScript compiles cleanly: `npx tsc --noEmit`
- [ ] Vite build tree-shakes disabled branches in production bundles

### D3 — Fix resetHealth() Bug

**File**: `frontend/src/store/healthSlice.ts`

**Description**: Current `resetHealth()` only resets `dataHealthStats` and `nlpHealthy`, leaving `ollamaHealthy`, `lastError`, and `ollamaModels` at their current values. This means that when `setProjectContext` fires (which calls all slice resets), the health status from the old project leaks into the new one.

**Fix**: `resetHealth()` must set every health field to its initial value:

```typescript
resetHealth: () => set({
  ollamaHealthy: false,
  nlpHealthy: false,
  dataHealthStats: [],
  lastError: null,
  ollamaModels: [],
})
```

**Acceptance criteria**:
- [ ] All 5 health fields are reset to initial values in `resetHealth()`
- [ ] `healthSlice.test.ts` updated to assert all 5 fields are cleared
- [ ] `npx vitest run src/store/healthSlice.test.ts` passes
- [ ] `npx tsc --noEmit` clean

### D4 — Playwright Test Migration Notes

**File**: `docs/specs/ux-redesign-w0-playwright-migration.md`

**Description**: Documents all 10 existing Playwright tests and which UX redesign waves will break them. For each test, describes:
- What the test currently covers
- Which W1–W5 changes will break it (specific: field renames, component moves, view type changes)
- What the updated test should cover
- Priority: MUST, SHOULD, NICE

**The 10 tests** (`frontend/e2e/`):

| File | Coverage | Breaks In | Priority |
|------|----------|-----------|----------|
| `auth-flow.spec.ts` | Login/logout flow, API key management | W2 (nav restructure) | MUST |
| `onboarding.spec.ts` | SetupWizard flow, initial project creation | W5 (progressive disclosure) | MUST |
| `journey.spec.ts` | Full user journey: auth → query → explore | W2, W3, W4 | MUST |
| `commands.spec.ts` | Command palette, slash commands | W2 (command unification) | MUST |
| `slideover.spec.ts` | SlideOver open/close, content types | W3 (SlideOver unification) | MUST |
| `ontology-flow.spec.ts` | Ontology CRUD, version management | W3 (InlineRenderer → SlideOver) | SHOULD |
| `tool-lifecycle.spec.ts` | Tool create/edit/execute/delete | W4 (CopilotView decomposition) | SHOULD |
| `settings-flow.spec.ts` | Settings navigation, API key CRUD | W2 (sidebar changes) | SHOULD |
| `error-states.spec.ts` | Error boundaries, error display | W3, W4 | NICE |
| `sanitization.spec.ts` | Input sanitization, XSS prevention | None (backend-focused) | NICE |

**Acceptance criteria**:
- [ ] All 10 test files listed with file path, current test count, and coverage summary
- [ ] For each test, specific breaking changes identified (field name, component name, route)
- [ ] Updated test structure described for each test
- [ ] Priority assignment with rationale
- [ ] Implementation order: higher-priority tests updated BEFORE their corresponding wave

### D5 — TerminalView Props Audit

**Deliverable**: Section in the audit document (D1) or a standalone note.

**Description**: TerminalView is a pass-through component that reads 11 fields from the store and 2 callbacks from useAppActions, then forwards them all to CopilotView. Determine if all 13 props are actually consumed by CopilotView or if any are dead.

**Analysis**: TerminalViewInner reads:
- `currentView` → passed? No — read but NOT passed to CopilotView (line 7, only used in local TerminalView context)
- `showInlinePanel` → read but NOT passed to CopilotView (line 8)
- `agents` → passed to CopilotView (line 32)
- `selectedAgent` → passed to CopilotView (line 33)
- `setSelectedAgent` → passed to CopilotView (line 34)
- `chat` → passed to CopilotView (line 35)
- `input` → passed to CopilotView (line 36)
- `setInput` → passed to CopilotView (line 37)
- `isStreaming` → passed to CopilotView (line 39)
- `cancelStream` → wrapped as `onCancelStream` (line 40)
- `clearChat` → wrapped as `onClearChat` (line 42)
- `onSend` from useAppActions → passed to CopilotView (line 38)
- `onConfirmAction` from useAppActions → passed to CopilotView (line 41)

**Findings**: `currentView` and `showInlinePanel` are read but NOT passed to CopilotView. They appear unused within TerminalView itself (the component only renders the header bar and CopilotView). These are likely legacy reads that should be cleaned up. All other 11 props are actively forwarded.

**Acceptance criteria**:
- [ ] Each of the 13 TerminalView store reads is documented as USED or UNUSED
- [ ] `currentView` and `showInlinePanel` confirmed as dead reads in TerminalView
- [ ] Recommendation: remove these 2 reads from TerminalView in W4 (CopilotSlim)

---

## 3. Key Findings from Codebase Audit

### 3.1 Zustand Store Inventory

**Total: 61 state fields across 6 slices**

| Slice | Fields | Key Issues |
|-------|--------|------------|
| `authSlice` | 5 | `projectID` is workspace-scope, misplaced here |
| `navigationSlice` | 7 | `activeView` duplicates URL state in nuqs |
| `copilotSlice` | 10 | Artificial split from navigation; `splitView` collides with `showInlinePanel` |
| `workspaceSlice` | 19 | Largest slice; 5 fields are view-local (`ontologyVersions`, `selectedVersionId`, `isVersionHistoryOpen`, `scenarios`, `selectedScenarioIds`) |
| `healthSlice` | 5 | **BUG**: `resetHealth()` omits 3 fields |
| `uiSlice` | 15 | 1 dead field (`showGuide`); 3 effects toggles better as local state |

**Full-Store Subscribers** (call `useStore()` with no selector → re-render on ANY state change):

| File | Line | Fields Actually Needed |
|------|------|----------------------|
| `CopilotView.tsx` | 40 | `splitView`, `setSplitView`, `chatSearchQuery`, `setChatSearchQuery` |
| `OracleView.tsx` | 30 | `projectID`, `predictions`, `setPredictions` |
| `TerminalEffects.tsx` | 7 | `enableScanline`, `enableGlow`, `enableFlicker` |

**Impact**: Chat streaming (every 50ms) and health polling (every 5s) cascade re-renders through all 3 components.

### 3.2 View System

**11 views in InlineRenderer**: explore, agent, ontology, data, health, skill, tool, component, settings, library, predict

**20 implemented SlideOverContent types** (of 24 defined in the union — 4 have no render path):
- Implemented: explore, ontology, data, health, skill, tool, sandbox, agent, datasource, component, settings, library, predict, asset, detail, agent-form, skill-form, tool-form, datasource-form, component-form
- Defined but NOT implemented: component-detail, tool-intelligence, scenario-comparison, dashboard

### 3.3 Command System Duality

- **5 hardcoded commands** in CommandPalette component (inline definitions)
- **16 dynamic slashCommands** loaded from API/registry
- Results in two different rendering paths for what should be a unified command interface
- No deduplication: a command could exist in both lists with different behavior

### 3.4 Explorer State Fragmentation

Explorer-related state is scattered across 3 slices:

| Slice | Field | Purpose |
|-------|-------|---------|
| `workspaceSlice` | `projectID` | Current project scope |
| `workspaceSlice` | `selectedObject` | Active navigation target |
| `workspaceSlice` | `searchQuery` | Current search string |
| `workspaceSlice` | `data` | Loaded query data |
| `workspaceSlice` | `availableObjects` | Navigation tree |
| `navigationSlice` | `activeView` | Current view mode (table/graph/etc.) |

This fragmentation means a single explorer action (e.g., "select object X") dispatches to 3 different slices, making it hard to reason about or add middleware.

### 3.5 Rendered Modes

The app renders in 1 of 3 modes based on store state:

1. **SetupWizard** (`showWizard === true`): First-run setup with 4 steps
2. **WorkspaceOnboarding** (`showOnboarding === true`): Guided tour overlay
3. **Full App** (both false): Normal terminal + inline panels + slide overs

These modes are gated by 2 boolean fields in `uiSlice`. As features grow, a proper mode enum (`'wizard' | 'onboarding' | 'app'`) would prevent invalid states (both true).

---

## 4. Implementation Order Within Wave

```
Step 1: D2 — Create featureFlags.ts
  ├── Single file, zero dependencies
  ├── Must exist before any other W1-W5 work
  └── Production behavior unchanged (all flags false)

Step 2: D3 — Fix resetHealth() bug
  ├── One function change in healthSlice.ts
  ├── Update healthSlice.test.ts
  └── Verified via vitest

Step 3: D1 — Full codebase audit document
  ├── Iterative: explore each slice file, document fields
  ├── Requires grepping consumer references
  ├── Output: docs/specs/ux-redesign-w0-audit.md
  └── Momus review pass to verify accuracy

Step 4: D4 — Playwright migration notes
  ├── Requires reading all 10 test files
  ├── Maps each test to its breaking wave
  └── Output: docs/specs/ux-redesign-w0-playwright-migration.md

Step 5: D5 — TerminalView props audit
  ├── Read TerminalView.tsx and trace each prop
  ├── Read CopilotView props interface
  └── Output: section in audit doc or standalone note
```

**Parallelism**: Steps 1 and 2 are independent and can run simultaneously. Steps 3–5 can run in any order after 1+2 are complete.

---

## 5. Dependencies & Risks

### Dependencies

| Deliverable | Depends On | Needed By |
|-------------|-----------|-----------|
| D2 (feature flags) | Nothing | W1–W5 (all waves) |
| D3 (resetHealth fix) | Nothing | W1 (store refactor) |
| D1 (audit doc) | Nothing | W1–W5 planning |
| D4 (test migration notes) | 10 e2e test files (read-only) | Before each wave touches UI |
| D5 (TerminalView audit) | TerminalView.tsx, CopilotView.tsx (read-only) | W4 (CopilotSlim) |

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Audit misses a store consumer | Medium | MEDIUM — W1 migration misses a reference | Use `grep` with exact field names + cross-reference with lsp_find_references |
| Feature flag import causes circular dep | Low | HIGH — build break | Keep featureFlags.ts dependency-free (no imports from store or components) |
| resetHealth fix breaks existing test | Medium | LOW — easy fix | Update healthSlice.test.ts assertion in same commit |
| Playwright test migration notes get stale | High | LOW — notes are advisory | Flag tests as "verify before wave" in each wave spec |
| Audit produces inaccurate field count | Medium | MEDIUM — downstream planning wrong | Momus review pass required before linking audit from W1 spec |

---

## 6. Verification Criteria

### Build Verification
- [ ] `npx tsc --noEmit` passes with zero errors
- [ ] `npx vite build` completes successfully (check for tree-shaking of disabled branches)
- [ ] `npx vitest run` passes (healthSlice.test.ts updated)

### Deliverable Verification
- [ ] All 61 store fields documented with consumers in audit doc
- [ ] All 5 feature flags exported, default to `false`, typed as `const`
- [ ] `resetHealth()` clears all 5 fields (verified via unit test)
- [ ] All 10 Playwright tests listed with breakage wave and migration plan
- [ ] TerminalView props categorized as USED (11) or UNUSED (2)

### Downstream Gate
- [ ] Before any W1 task starts: `featureFlags.storeRefactor` flipped to `true` in PR, verified by checking a gated code path compiles and works
- [ ] No W0 deliverable modifies any UI component's behavior (feature flags are compile-time only, bugfix restores correct initial state)

---

## 7. Momus Review Requirements

The following items MUST survive adversarial review (Momus) before W0 is considered complete:

1. **Audit accuracy**: Every field count and consumer claim verified against actual source, not extrapolated. Run `grep` for each field name across the frontend source.
2. **Feature flag design**: Confirm tree-shaking works by inspecting a Vite production build with all flags false — disabled branches must not appear in bundle.
3. **resetHealth test**: Verify the bug was NOT in the test itself (i.e., the test was passing because it wasn't checking the omitted fields).
4. **Test migration accuracy**: Verify each of the 10 Playwright test files exists and that the described coverage matches the actual test content.
5. **TerminalView dead reads**: Confirm the 2 "unused" props (`currentView`, `showInlinePanel`) have no side effects in TerminalView — verify via execution trace, not just grep.

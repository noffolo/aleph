# W1 — UX Redesign: Store Refactor

> **Wave:** W1 (Store Refactor)
> **Status:** Draft
> **Target:** Reduce Zustand store from 61 → 42 state fields across 5 slices
> **Risk:** HIGH — touches every frontend component via selector/import changes

---

## 1. Current State Analysis

The Zustand store (`frontend/src/store/`) has **61 state fields** across **6 slices**. Every component subscribes individually, but 3 files use the bare `useStore()` with no selector, subscribing to the entire state tree.

### authSlice (5 fields)

| Field | Type | Consumers |
|-------|------|-----------|
| `projectID` | `string` | App.tsx, OracleView, DashboardView, SlideOverContent, InlineRenderer |
| `apiKeys` | `ApiKey[]` | SlideOverContent, InlineRenderer |
| `projects` | `Project[]` | App.tsx |
| `notificationChannels` | `NotificationChannel[]` | SlideOverContent, InlineRenderer |
| `registryComponents` | `RegistryComponent[]` | SlideOverContent, InlineRenderer, ComponentDetailSlideOver |

### navigationSlice (7 fields)

| Field | Type | Consumers |
|-------|------|-----------|
| `currentView` | `'copilot' \| 'inline'` | Sidebar, StatusBar |
| `inlineContent` | `InlineContent \| null` | InlineRenderer |
| `showInlinePanel` | `boolean` | InlineRenderer, (UseAppActions sets it) |
| `commandHistory` | `string[]` | (persisted to sessionStorage, zero component readers) |
| `slideOverContent` | `SlideOverContent \| null` | App.tsx, Sidebar, StatusBar, SlideOverContent |
| `isCommandPaletteOpen` | `boolean` | App.tsx |
| `activeView` | `string` | SlideOverContent, InlineRenderer |

### copilotSlice (10 fields)

| Field | Type | Consumers |
|-------|------|-----------|
| `chat` | `ChatMessage[]` | TerminalView, useAppActions |
| `input` | `string` | useAppActions (onSend reads + writes) |
| `isStreaming` | `boolean` | useAppActions (onSend reads/writes) |
| `streamAbortController` | `AbortController \| null` | useAppActions (onSend creates, cancelStream reads) |
| `pendingConfirmation` | `PendingConfirmation \| null` | useAppActions (onConfirmAction reads/writes) |
| `selectedAgent` | `string` | App.tsx, useAppActions |
| `splitView` | `boolean` | CopilotView |
| `bookmarkedIds` | `Set<number>` | **ℹ️ Defined in slice, ZERO component consumers** |
| `chatSearchQuery` | `string` | CopilotView |
| `onlyBookmarks` | `boolean` | **ℹ️ Defined in slice, ZERO component consumers** |

**Issue:** `splitView` (copilot) and `showInlinePanel` (navigation) can both be true simultaneously, producing conflicting layout states. Also, `bookmarkedIds` + `onlyBookmarks` are dead — never read by any `.tsx` file.

### workspaceSlice (19 fields)

| Field | Type | Consumers |
|-------|------|-----------|
| `sandboxResult` | `SandboxResult \| null` | SandboxResultSlideOver |
| `sandboxInput` | `string` | SkillExecuteSlideOver, ToolExecuteSlideOver |
| `searchQuery` | `string` | SlideOverContent, InlineRenderer |
| `selectedObject` | `string` | SlideOverContent, InlineRenderer, App.tsx |
| `predictions` | `Prediction[]` | OracleView |
| `data` | `QueryData \| null` | SlideOverContent, InlineRenderer |
| `selectedRow` | `Row \| null` | SlideOverContent, InlineRenderer |
| `agents` | `Agent[]` | SlideOverContent, InlineRenderer, TerminalView |
| `ingestionTasks` | `IngestionTask[]` | SlideOverContent, InlineRenderer |
| `ontologyRaw` | `string` | SlideOverContent, InlineRenderer |
| `ontologyVersions` | `any[]` | **❌ Dead — zero consumers** |
| `selectedVersionId` | `string \| null` | **❌ Dead — zero consumers** |
| `isVersionHistoryOpen` | `boolean` | **❌ Dead — zero consumers** |
| `availableObjects` | `string[]` | App.tsx, SlideOverContent, InlineRenderer |
| `scenarios` | `Scenario[]` | ScenarioComparisonView |
| `selectedScenarioIds` | `string[]` | ScenarioComparisonView |
| `taskLogs` | `string` | SlideOverContent, InlineRenderer |
| `skills` | `Skill[]` | SlideOverContent, InlineRenderer, SkillsView, ToolManagementView |
| `tools` | `Tool[]` | SlideOverContent, InlineRenderer, ToolsView, ToolManagementView, SkillExecuteSlideOver |

### healthSlice (5 fields)

| Field | Type | Consumers |
|-------|------|-----------|
| `ollamaHealthy` | `boolean` | SlideOverContent, InlineRenderer, App.tsx, useAppActions (sets it) |
| `nlpHealthy` | `boolean` | App.tsx, useAppActions (sets it) |
| `dataHealthStats` | `ColumnStats[]` | SlideOverContent, InlineRenderer |
| `lastError` | `string \| null` | App.tsx, useAppActions (sets via handleError) |
| `ollamaModels` | `string[]` | SlideOverContent, InlineRenderer |

**Bug:** `resetHealth()` only clears `dataHealthStats` and `nlpHealthy`, leaving `ollamaHealthy`, `lastError`, and `ollamaModels` stale.

### uiSlice (15 fields)

| Field | Type | Consumers |
|-------|------|-----------|
| `showOnboarding` | `boolean` | App.tsx |
| `showWizard` | `boolean` | App.tsx |
| `showGuide` | `boolean` | **❌ Dead — only reference is a comment in contextualGuides.ts** |
| `isExplorerLoading` | `boolean` | SlideOverContent, InlineRenderer, App.tsx (sets) |
| `selectedAssetContent` | `string \| null` | SlideOverContent, InlineRenderer |
| `selectedAssetId` | `string \| null` | SlideOverContent, InlineRenderer |
| `globalSearchResults` | `Record \| null` | SlideOverContent, InlineRenderer |
| `assets` | `Asset[]` | SlideOverContent, InlineRenderer, LibraryView |
| `confirmDialog` | `ConfirmDialog` | (actions only, no direct UI reader) |
| `enableScanline` | `boolean` | TerminalEffects, SettingsView |
| `enableGlow` | `boolean` | TerminalEffects, SettingsView |
| `enableFlicker` | `boolean` | TerminalEffects, SettingsView |
| `toastMessages` | `ToastMessage[]` | Toast component |
| `inputMode` | `boolean` | TerminalPrompt, StatusBar |
| `pendingCrud` | `Record<string, boolean>` | useAgentActions, useToolActions (CRUD lock) |

### Full-Store Subscription Bug

3 files call `useStore()` without a selector. This triggers a re-render of the full component tree on every state change (chat keystrokes, health polling, streaming tokens):

| File | Line | Actually Needs | Fix |
|------|------|----------------|-----|
| `CopilotView.tsx` | 40 | `splitView`, `setSplitView`, `chatSearchQuery`, `setChatSearchQuery` | 4 individual selectors |
| `OracleView.tsx` | 30 | `projectID`, `predictions`, `setPredictions` | 3 individual selectors |
| `TerminalEffects.tsx` | 7 | `enableScanline`, `enableGlow`, `enableFlicker` | 3 individual selectors |

### Summary of Dead / Removable Fields

| Field | Slice | Reason | Consumers |
|-------|-------|--------|-----------|
| `bookmarkedIds` | copilot | Dead code | 0 (test only) |
| `onlyBookmarks` | copilot | Dead code | 0 (test only) |
| `showGuide` | ui | Dead code | 0 |
| `ontologyVersions` | workspace | Dead code | 0 |
| `selectedVersionId` | workspace | Dead code | 0 |
| `isVersionHistoryOpen` | workspace | Dead code | 0 |

---

## 2. Flat Store vs Sliced Store — Decision

**Recommendation: Keep 5 slices. Do NOT flatten.**

| Criterion | Flat Store | Sliced (current) |
|-----------|-----------|-------------------|
| Type inference | Same (`AppState = A & B & C & D & E`) | Same |
| Code organization | Single file or manual split | Per-concern files |
| Test isolation | Single test file | Per-slice test files (6 existing) |
| Merge conflicts | Worse (same file) | Better (separate files) |
| Performance | Same (Zustand selectors work identically) | Same |
| Refactor cost | Rewrite all 6 files + tests | Merge copilot into navigation |

**Decision:** Keep the slice pattern. Remove `copilotSlice` (absorb into navigation). Keep `auth`, `navigation`, `workspace`, `health`, `ui`.

---

## 3. Target State (42 fields)

### authSlice — 5 fields (unchanged)

| Field | Type | Status |
|-------|------|--------|
| `projectID` | `string` | ✅ Keep |
| `apiKeys` | `ApiKey[]` | ✅ Keep |
| `projects` | `Project[]` | ✅ Keep |
| `notificationChannels` | `NotificationChannel[]` | ✅ Keep |
| `registryComponents` | `RegistryComponent[]` | ✅ Keep |

`projectID` stays in auth. It is a cross-cutting context key (used by health polling, chat history loading, workspace data loading), not a workspace data field. Moving it would require updating 5+ consumer files for no semantic gain.

### navigationSlice — 12 fields (absorbs copilot chat state)

| Field | Origin | Type | Status |
|-------|--------|------|--------|
| `currentView` | navigation | `'copilot' \| 'inline'` | ✅ Keep |
| `inlineContent` | navigation | `InlineContent \| null` | ✅ Keep |
| `showInlinePanel` | navigation | `boolean` | ✅ Keep |
| `commandHistory` | navigation | `string[]` | ✅ Keep |
| `slideOverContent` | navigation | `SlideOverContent \| null` | ✅ Keep |
| `isCommandPaletteOpen` | navigation | `boolean` | ✅ Keep |
| `activeView` | navigation | `string` | ✅ Keep |
| `chat` | copilot | `ChatMessage[]` | 🔄 Moved in |
| `input` | copilot | `string` | 🔄 Moved in |
| `isStreaming` | copilot | `boolean` | 🔄 Moved in |
| `selectedAgent` | copilot | `string` | 🔄 Moved in |
| `chatSearchQuery` | copilot | `string` | 🔄 Moved in |

**Demoted from store:**
| Field | New Home | Reason |
|-------|----------|--------|
| `streamAbortController` | `useRef` in `useAppActions` | Implementation detail, not reactive state |
| `pendingConfirmation` | Local state in `useAppActions` | Only read/written by onConfirmAction |
| `splitView` | Merged into `showInlinePanel` concept | Sec W3 for full unification |
| `bookmarkedIds` | ❌ Removed | Zero consumers |
| `onlyBookmarks` | ❌ Removed | Zero consumers |

### workspaceSlice — 16 fields (removed 3 dead)

**Removed:** `ontologyVersions`, `selectedVersionId`, `isVersionHistoryOpen` — zero consumers. If Ontology versioning is re-added later, it should use local state or URL params, not the global store.

All other fields kept unchanged. `scenarios` + `selectedScenarioIds` stay — they are consumed by `ScenarioComparisonView` and `OracleView` (though `OracleView` only reads `predictions`).

### healthSlice — 5 fields (bugfix)

**Bugfix only.** `resetHealth()` must clear all fields:

```typescript
resetHealth: () => set({
  ollamaHealthy: false,
  nlpHealthy: false,
  dataHealthStats: [],
  lastError: null,
  ollamaModels: [],
})
```

### uiSlice — 4 fields (removed 11)

| Field | Type | Fate |
|-------|------|------|
| `showOnboarding` | `boolean` | ✅ Keep |
| `showWizard` | `boolean` | ✅ Keep |
| `toastMessages` | `ToastMessage[]` | ✅ Keep |
| `confirmDialog` | `ConfirmDialog` | ✅ Keep |

**Removed or moved out of global store:**

| Field | New Home | Rationale |
|-------|----------|-----------|
| `showGuide` | ❌ Removed | Dead code |
| `isExplorerLoading` | → Local state in `App.tsx` + `ExplorerView` | Only toggled in App's loadData effect |
| `selectedAssetContent` | → `useLibraryContext` | Library-scoped state |
| `selectedAssetId` | → `useLibraryContext` | Library-scoped state |
| `globalSearchResults` | → Local state in `SlideOverContent` + `InlineRenderer` | Only used in explore view mode |
| `assets` | → `useLibraryContext` | Fetched per-project, library-scoped |
| `enableScanline` | → `useTerminalEffects` hook + localStorage | Config, not reactive state |
| `enableGlow` | → `useTerminalEffects` hook + localStorage | Config, not reactive state |
| `enableFlicker` | → `useTerminalEffects` hook + localStorage | Config, not reactive state |
| `inputMode` | → `InputModeContext` | Two-consumer concern (TerminalPrompt + StatusBar) |
| `pendingCrud` | → Local `Set<string>` via `useRef` in `useAgentActions` + `useToolActions` | Implementation detail |

### Target Field Count

| Slice | Before | Removed | Moved Out | Moved In | After |
|-------|--------|---------|-----------|----------|-------|
| auth | 5 | 0 | 0 | 0 | **5** |
| navigation | 7 | 0 | 0 | +5 | **12** |
| copilot | 10 | 2 | 3 | — (deleted) | **0** |
| workspace | 19 | 3 | 0 | 0 | **16** |
| health | 5 | 0 | 0 | 0 | **5** |
| ui | 15 | 1 | 10 | 0 | **4** |
| **Total** | **61** | **6** | **13** | **0** | **42** |

---

## 4. Migration Strategy

### Phase 1: Add-Only (backward compatible, parallel-safe)

1. Create `navigationSlice` v2 with `chat`, `input`, `isStreaming`, `selectedAgent`, `chatSearchQuery` fields added
2. Add `createNavigationSlice` alongside existing slices (they compose)
3. No slice is removed yet — all old selectors still compile

### Phase 2: Fix Full-Store Subscribers

| File | Change |
|------|--------|
| `CopilotView.tsx` L40 | `useStore()` → `useStore(s => s.splitView)` etc. (4 selectors) |
| `OracleView.tsx` L30 | `useStore()` → `useStore(s => s.projectID)` + `useStore(s => s.predictions)` etc. (3 selectors) |
| `TerminalEffects.tsx` L7 | `useStore()` → `useTerminalEffects()` hook |

### Phase 3: Demote Implementation Details

| Change | Files Touched |
|--------|---------------|
| `streamAbortController` → `useRef` in `useAppActions` | `copilotSlice.ts`, `useAppActions.ts` |
| `pendingConfirmation` → `useState` in `useAppActions` | `copilotSlice.ts`, `useAppActions.ts`, `CopilotView.tsx` (remove prop? No — it's a local concern now) |
| `pendingCrud` → `useRef<Set<string>>` in `useAgentActions`, `useToolActions` | `uiSlice.ts`, `useAgentActions.ts`, `useToolActions.ts`, test files |
| `bookmarkedIds` + `onlyBookmarks` → remove | `copilotSlice.ts`, `copilotSlice.test.ts` |

### Phase 4: Move Concern-Scoped State

| Change | Files Touched |
|--------|---------------|
| `assets/selectedAssetContent/selectedAssetId` → `useLibraryContext` | Create `LibraryContext.tsx`, update `SlideOverContent.tsx`, `InlineRenderer.tsx`, `LibraryView.tsx` |
| `isExplorerLoading/globalSearchResults` → local state in `App.tsx` + `ExplorerView` | `App.tsx`, `SlideOverContent.tsx`, `InlineRenderer.tsx` |
| `enableScanline/Glow/Flicker` → `useTerminalEffects` hook + localStorage | Create hook, update `TerminalEffects.tsx`, `SettingsView.tsx` |
| `inputMode` → `InputModeContext` | Create context + provider, update `TerminalPrompt.tsx`, `StatusBar.tsx` |

### Phase 5: Remove Dead Fields + Delete copilotSlice

1. Remove `showGuide` from `uiSlice.ts`
2. Remove `ontologyVersions`, `selectedVersionId`, `isVersionHistoryOpen` from `workspaceSlice.ts`
3. Delete `copilotSlice.ts` and `copilotSlice.test.ts`
4. Remove `copilotSlice` import from `useStore.ts`
5. Update `setProjectContext` to call `resetNavigation()` instead of `resetCopilot()`

### Phase 6: Fix resetHealth

```typescript
// healthSlice.ts
resetHealth: () => set({
  ollamaHealthy: false,    // was missing
  nlpHealthy: false,
  dataHealthStats: [],
  lastError: null,          // was missing
  ollamaModels: [],         // was missing
})
```

---

## 5. Consumer Migration Table

For each removed or relocated field, the migration path:

| Old Field | Migration | Consumers to Update |
|-----------|-----------|-------------------|
| `bookmarkedIds` | Remove; no replacement | `copilotSlice.test.ts` |
| `onlyBookmarks` | Remove; no replacement | `copilotSlice.test.ts` |
| `showGuide` | Remove; no replacement | `uiSlice.test.ts` |
| `ontologyVersions` | Remove; use local state if re-added | — |
| `selectedVersionId` | Remove; use URL param if re-added | — |
| `isVersionHistoryOpen` | Remove; use local useState | — |
| `streamAbortController` | `useRef` in `useAppActions` | `copilotSlice.ts`, `useAppActions.ts` |
| `pendingConfirmation` | `useState` in `useAppActions` | `copilotSlice.ts`, `useAppActions.ts` |
| `pendingCrud` | `useRef<Set<string>>` in hooks | `uiSlice.ts`, `useAgentActions.ts`, `useToolActions.ts`, 3 test files |
| `assets` | `useLibraryContext()` | `uiSlice.ts`, `SlideOverContent.tsx`, `InlineRenderer.tsx`, `LibraryView.tsx` |
| `selectedAssetContent` | `useLibraryContext()` | same set |
| `selectedAssetId` | `useLibraryContext()` | same set |
| `isExplorerLoading` | Local state in App + ExplorerView | `App.tsx`, `SlideOverContent.tsx`, `InlineRenderer.tsx` |
| `globalSearchResults` | Local state in SlideOverContent + InlineRenderer | `SlideOverContent.tsx`, `InlineRenderer.tsx` |
| `enableScanline` | `useTerminalEffects()` | `uiSlice.ts`, `TerminalEffects.tsx`, `SettingsView.tsx` |
| `enableGlow` | `useTerminalEffects()` | same set |
| `enableFlicker` | `useTerminalEffects()` | same set |
| `inputMode` | `InputModeContext` | `uiSlice.ts`, `TerminalPrompt.tsx`, `StatusBar.tsx` |

### Copy-Paste Grep Patterns

Before Phase 5, verify no straggling references:

```bash
# Verify dead fields have zero consumers
grep -r "bookmarkedIds" frontend/src/components/ frontend/src/hooks/ frontend/src/views/ frontend/src/App.tsx
grep -r "onlyBookmarks" frontend/src/components/ frontend/src/hooks/ frontend/src/views/
grep -r "showGuide" frontend/src/components/ frontend/src/hooks/ frontend/src/views/ frontend/src/App.tsx
grep -r "ontologyVersions" frontend/src/components/ frontend/src/hooks/ frontend/src/views/
grep -r "selectedVersionId" frontend/src/components/ frontend/src/hooks/ frontend/src/views/
grep -r "isVersionHistoryOpen" frontend/src/components/ frontend/src/hooks/ frontend/src/views/

# Verify full-store subscribers are fixed
grep -rn "useStore()" frontend/src/components/ --include="*.tsx"
```

---

## 6. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `splitView` and `showInlinePanel` collision not fully resolved in W1 | Medium | Medium | W3 (SlideOver Unification) replaces both with unified panel; W1 just merges copilot into navigation |
| Library context refactor breaks asset loading | Medium | High | Keep `assets` in store as fallback; move to context only after verifying all 3 consumers work |
| `inputMode` context exposes `<body>` class setting issues | Low | Low | Use `InputModeContext.Provider` wrapping only TerminalView subtree |
| Reset copilot → resetNavigation after copilotSlice removal | Low | Low | `setProjectContext` already calls `resetCopilot()`; change to `resetNavigation()` after merge |
| Test files miss updated import paths | Medium | Low | `vitest run` fails fast on import errors; fix per failing file |
| Stateful `streamAbortController` lost on component remount | Low | High | Use `useRef` at the hook level (`useAppActions`), not in components — survives remounts because hook lives in App |

---

## 7. Success Criteria

- [ ] `npx tsc --noEmit` passes with zero errors
- [ ] `npx vitest run` passes all store tests (updated for removals)
- [ ] `npx vite build` completes cleanly
- [ ] All 3 full-store subscribers (`CopilotView`, `OracleView`, `TerminalEffects`) use individual selectors
- [ ] `resetHealth()` clears all 5 fields: `ollamaHealthy`, `nlpHealthy`, `dataHealthStats`, `lastError`, `ollamaModels`
- [ ] `copilotSlice.ts` file deleted; all fields migrated to navigation or demoted
- [ ] `bookmarkedIds`, `onlyBookmarks`, `showGuide`, `ontologyVersions`, `selectedVersionId`, `isVersionHistoryOpen` — grep returns zero hits outside store/ directories
- [ ] Chat streaming flow works: `useAppActions` uses `useRef` for abort controller, no store `streamAbortController`
- [ ] Visual effects toggles persist across page reload (localStorage) without store
- [ ] Library asset CRUD works with context-based state (not store)

# Store Inventory & Consumer Audit

> **Date:** 2026-05-11
> **Scope:** Read-only analysis of Zustand stores under `frontend/src/store/`
> **Goal:** Foundational data for UX W1 store refactor (target: 38–42 state fields)

---

## 1. Store Files Found

| # | File | Slice | Lines | State Fields |
|---|------|-------|-------|-------------|
| 1 | `frontend/src/store/authSlice.ts` | AuthSlice | 36 | 5 |
| 2 | `frontend/src/store/navigationSlice.ts` | NavigationSlice | 78 | 7 |
| 3 | `frontend/src/store/copilotSlice.ts` | CopilotSlice | 78 | 10 |
| 4 | `frontend/src/store/workspaceSlice.ts` | WorkspaceSlice | 114 | 19 |
| 5 | `frontend/src/store/healthSlice.ts` | HealthSlice | 33 | 5 |
| 6 | `frontend/src/store/uiSlice.ts` | UISlice | 118 | 15 |
| 7 | `frontend/src/store/useStore.ts` | Composition root + cross-slice `setProjectContext` | 76 | — |
| 8 | `frontend/src/store/types.ts` | Shared type definitions | 191 | — |

**Architecture:** 6 Zustand slices combined in `useStore.ts` via `create<AppState>()`. Single monolithic `useStore` hook exported.

**Current state-field count: 61**

---

## 2. Per-Store Field Inventory

### 2.1 AuthSlice (`authSlice.ts`)

**State fields: 5**

| # | Field | Type | Consumers |
|---|-------|------|-----------|
| 1 | `projectID` | `string` | `App.tsx`, `useLibraryActions.ts`, `useOntologyActions.ts`, `useSettingsActions.ts`, `useAgentActions.ts`, `useSkillActions.ts`, `useDataSourceActions.ts`, `useToolActions.ts`, `useAppActions.ts`, `DashboardView.tsx`, `OracleView.tsx` |
| 2 | `apiKeys` | `ApiKey[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx` |
| 3 | `projects` | `Project[]` | `App.tsx` (renders project list), `CommandPalette.tsx` |
| 4 | `notificationChannels` | `NotificationChannel[]` | **DEAD** — set via `useAppActions.ts`, never read by any component |
| 5 | `registryComponents` | `RegistryComponent[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `ComponentDetailSlideOver.tsx` |

**Interface entries (including actions): 11** — 5 state fields + 5 setters + `setProjectContext` (cross-slice) + `resetAuth`

---

### 2.2 NavigationSlice (`navigationSlice.ts`)

**State fields: 7**

| # | Field | Type | Consumers |
|---|-------|------|-----------|
| 1 | `currentView` | `'copilot' \| 'inline'` | `Sidebar.tsx`, `TerminalView.tsx` |
| 2 | `inlineContent` | `InlineContent \| null` | `Sidebar.tsx`, `InlineRenderer.tsx`, `StatusBar.tsx` |
| 3 | `showInlinePanel` | `boolean` | `InlineRenderer.tsx`, `TerminalView.tsx` |
| 4 | `commandHistory` | `string[]` | **DEAD** — `addToHistory` called in `useAppActions.ts`, persisted to `sessionStorage`, but **no component reads it** |
| 5 | `slideOverContent` | `SlideOverContent \| null` | `SlideOverContent.tsx`, `Sidebar.tsx`, `StatusBar.tsx` |
| 6 | `isCommandPaletteOpen` | `boolean` | `App.tsx` |
| 7 | `activeView` | `string` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `ExplorerView.tsx` |

**Interface entries: 15** — 7 state + 7 actions/setters + `resetNavigation`

---

### 2.3 CopilotSlice (`copilotSlice.ts`)

**State fields: 10**

| # | Field | Type | Consumers |
|---|-------|------|-----------|
| 1 | `chat` | `ChatMessage[]` | `CopilotView.tsx` |
| 2 | `input` | `string` | `TerminalPrompt.tsx` |
| 3 | `isStreaming` | `boolean` | `TerminalPrompt.tsx` |
| 4 | `streamAbortController` | `AbortController \| null` | `TerminalPrompt.tsx` (via `cancelStream`) |
| 5 | `pendingConfirmation` | `PendingConfirmation \| null` | `useAppActions.ts` (set/cleared via hooks) |
| 6 | `selectedAgent` | `string` | `App.tsx`, `TerminalPrompt.tsx` |
| 7 | `splitView` | `boolean` | `CopilotView.tsx` |
| 8 | `bookmarkedIds` | `Set<number>` | **DEAD** — never read outside the store. Only `toggleBookmark` is called in `useAppActions.ts`. |
| 9 | `chatSearchQuery` | `string` | `CopilotView.tsx`, `ChatSearchBar.tsx` |
| 10 | `onlyBookmarks` | `boolean` | **DEAD** — never read or written outside the store |

**Interface entries: 24** — 10 state + 10 actions/setters + `clearChat` + `cancelStream` + `toggleBookmark` + `resetCopilot`

---

### 2.4 WorkspaceSlice (`workspaceSlice.ts`)

**State fields: 19** — the largest slice

| # | Field | Type | Consumers |
|---|-------|------|-----------|
| 1 | `sandboxResult` | `SandboxResult \| null` | **DEAD** — only set via `useAppActions.ts`, never read by any component |
| 2 | `sandboxInput` | `string` | `SkillExecuteSlideOver.tsx`, `ToolExecuteSlideOver.tsx` |
| 3 | `searchQuery` | `string` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `useExplorerActions.ts` |
| 4 | `selectedObject` | `string` | `App.tsx`, `InlineRenderer.tsx`, `SlideOverContent.tsx`, `AgentsView.tsx`, `ToolsView.tsx`, `useExplorerActions.ts` |
| 5 | `predictions` | `Prediction[]` | `OracleView.tsx` |
| 6 | `data` | `QueryData \| null` | `InlineRenderer.tsx`, `SlideOverContent.tsx` |
| 7 | `selectedRow` | `Row \| null` | `InlineRenderer.tsx`, `useExplorerActions.ts` |
| 8 | `agents` | `Agent[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `TerminalView.tsx`, `AgentsView.tsx`, `useAgentActions.ts`, `useAppActions.ts` |
| 9 | `ingestionTasks` | `IngestionTask[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx` |
| 10 | `ontologyRaw` | `string` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `useOntologyActions.ts` |
| 11 | `ontologyVersions` | `any[]` | **DEAD** — only set in `useOntologyActions.ts`, never read by any component |
| 12 | `selectedVersionId` | `string \| null` | **DEAD** — never read outside store |
| 13 | `isVersionHistoryOpen` | `boolean` | **DEAD** — never read outside store |
| 14 | `availableObjects` | `string[]` | `App.tsx`, `InlineRenderer.tsx`, `SlideOverContent.tsx`, `CommandPalette.tsx`, `ExplorerView.tsx` |
| 15 | `scenarios` | `Scenario[]` | `ScenarioComparisonView.tsx` (single consumer, rarely-used view) |
| 16 | `selectedScenarioIds` | `string[]` | `ScenarioComparisonView.tsx` |
| 17 | `taskLogs` | `string` | `InlineRenderer.tsx`, `SlideOverContent.tsx` |
| 18 | `skills` | `Skill[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `SkillsView.tsx`, `useSkillActions.ts` |
| 19 | `tools` | `Tool[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `ToolsView.tsx`, `ToolManagementView.tsx`, `SkillExecuteSlideOver.tsx`, `useToolActions.ts` |

**Interface entries: 39** — 19 state + 19 setters + `resetWorkspace`

---

### 2.5 HealthSlice (`healthSlice.ts`)

**State fields: 5**

| # | Field | Type | Consumers |
|---|-------|------|-----------|
| 1 | `ollamaHealthy` | `boolean` | `App.tsx`, `InlineRenderer.tsx`, `SlideOverContent.tsx` |
| 2 | `nlpHealthy` | `boolean` | `App.tsx` |
| 3 | `dataHealthStats` | `ColumnStats[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx` |
| 4 | `lastError` | `string \| null` | `App.tsx` (displayed in UI), `useSSE.ts` |
| 5 | `ollamaModels` | `string[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx` |

**Interface entries: 11** — 5 state + 5 setters + `resetHealth`

---

### 2.6 UISlice (`uiSlice.ts`)

**State fields: 15**

| # | Field | Type | Consumers |
|---|-------|------|-----------|
| 1 | `showOnboarding` | `boolean` | `App.tsx` |
| 2 | `showWizard` | `boolean` | `App.tsx` |
| 3 | `showGuide` | `boolean` | **DEAD** — only exists in `uiSlice.ts` and a comment in `contextualGuides.ts`. No component reads or sets it. |
| 4 | `isExplorerLoading` | `boolean` | `InlineRenderer.tsx`, `SlideOverContent.tsx` |
| 5 | `selectedAssetContent` | `string \| null` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `LibraryView.tsx` (passed as prop), `useLibraryActions.ts` |
| 6 | `selectedAssetId` | `string \| null` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `LibraryView.tsx`, `useLibraryActions.ts` |
| 7 | `globalSearchResults` | `Record<string, unknown> \| null` | `InlineRenderer.tsx`, `SlideOverContent.tsx` |
| 8 | `assets` | `Asset[]` | `InlineRenderer.tsx`, `SlideOverContent.tsx`, `AssetDetailSlideOver.tsx`, `useLibraryActions.ts`, `useAppActions.ts` |
| 9 | `confirmDialog` | `{ isOpen, message, confirmLabel?, onConfirm? }` | **DEAD** — `ConfirmDialog.tsx` is a controlled component (props-based). Nobody reads `confirmDialog`, calls `showConfirmDialog`, or calls `hideConfirmDialog` from the store. |
| 10 | `enableScanline` | `boolean` | `TerminalEffects.tsx`, `SettingsView.tsx` |
| 11 | `enableGlow` | `boolean` | `TerminalEffects.tsx`, `SettingsView.tsx` |
| 12 | `enableFlicker` | `boolean` | `TerminalEffects.tsx`, `SettingsView.tsx` |
| 13 | `toastMessages` | `ToastMessage[]` | `Toast.tsx`, `useSSE.ts` (adds toasts) |
| 14 | `inputMode` | `boolean` | `TerminalPrompt.tsx`, `StatusBar.tsx` |
| 15 | `pendingCrud` | `Record<string, boolean>` | **WRITE-ONLY** — set via `setPendingCrud` / `clearPendingCrud` in `useToolActions.ts` and `useAgentActions.ts`, but `isCrudPending` is **never called** outside the store and tests. No component reads `pendingCrud` for loading state UI. |

**Interface entries: 35** — 15 state + 13 setters/actions + `showConfirmDialog` + `hideConfirmDialog` + `addToast` + `removeToast` + `setPendingCrud` + `clearPendingCrud` + `isCrudPending` + `resetUI`

---

### 2.7 Cross-slice override in `useStore.ts`

| Extra entry | Purpose |
|-------------|---------|
| `setProjectContext` | Calls all 6 `reset*` functions + sets `projectID`. Overrides the per-slice version. |

---

## 3. Dead Field Analysis

**Completely dead fields** (state exists, never consumed by any component outside the store):

| # | Slice | Field | Evidence |
|---|-------|-------|----------|
| 1 | AuthSlice | `notificationChannels` | Only accessed via `setNotificationChannels` in `useAppActions.ts`. No component reads the array. |
| 2 | NavigationSlice | `commandHistory` | `addToHistory` called in `useAppActions.ts`. Persisted to `sessionStorage`. No component ever reads `commandHistory` from the store. |
| 3 | CopilotSlice | `bookmarkedIds` | `toggleBookmark` called in `useAppActions.ts`. No component reads `bookmarkedIds`. |
| 4 | CopilotSlice | `onlyBookmarks` | Entirely unused. Never read, never written outside the store. |
| 5 | WorkspaceSlice | `sandboxResult` | Only set via `useAppActions.ts`. No component reads it. |
| 6 | WorkspaceSlice | `ontologyVersions` | Only set in `useOntologyActions.ts`. No component reads it. |
| 7 | WorkspaceSlice | `selectedVersionId` | No reads outside the store at all. |
| 8 | WorkspaceSlice | `isVersionHistoryOpen` | No reads outside the store at all. |
| 9 | UISlice | `showGuide` | Only exists as a comment in `contextualGuides.ts`. Never set or read by any component. |
| 10 | UISlice | `confirmDialog` | The nested `{ isOpen, message, confirmLabel?, onConfirm? }` object is never consumed. `ConfirmDialog.tsx` is a controlled component using props. `showConfirmDialog` and `hideConfirmDialog` are never called. |
| 11 | UISlice | `pendingCrud` | `setPendingCrud`/`clearPendingCrud` are called in useAction hooks, but `isCrudPending` is **never called** anywhere. No loading-state UI reads this. |

**Total dead fields: 11**

---

## 4. Mergeable Field Groups

Fields that always change together and could be merged into compound state objects:

| Group | Fields | Slice | Rationale |
|-------|--------|-------|-----------|
| **Health state** | `ollamaHealthy`, `nlpHealthy`, `dataHealthStats`, `ollamaModels` | HealthSlice | All health/status data, always refetched together from the server |
| **Terminal effects** | `enableScanline`, `enableGlow`, `enableFlicker` | UISlice | All visual toggles in Settings, always read together by `TerminalEffects.tsx` |
| **Asset viewer** | `selectedAssetContent`, `selectedAssetId` | UISlice | Always read/written as a pair |
| **Explorer query** | `searchQuery`, `selectedObject`, `activeView`, `isExplorerLoading`, `globalSearchResults` | WorkspaceSlice + UISlice | All explorer interaction state |
| **Sandbox** | `sandboxInput` | WorkspaceSlice | `sandboxResult` is dead; `sandboxInput` is the only live field |
| **Ontology versions** | `ontologyVersions`, `selectedVersionId`, `isVersionHistoryOpen` | WorkspaceSlice | All 3 are dead, can be bulk-removed |
| **Onboarding** | `showOnboarding`, `showWizard` | UISlice | Both wizard flow state |
| **Prediction/Oracle** | `predictions`, `scenarios`, `selectedScenarioIds` | WorkspaceSlice | All oracle/forecast domain |
| **Ingestion monitoring** | `ingestionTasks`, `taskLogs` | WorkspaceSlice | Both ingested data lifecycle |
| **Registry data** | `apiKeys`, `notificationChannels` (dead), `registryComponents` | AuthSlice | All fetched from same API on project load |

---

## 5. Full-Store Subscription Report (`useStore()` with no selector)

These are the **highest-risk re-render bugs**. When `useStore()` is called without a selector, the component subscribes to **any state change** in any slice, causing re-render on every keystroke, every streamed token, every health check, etc.

### Found Vulnerabilities: 3 files

| # | File | Line | Code | Risk Level |
|---|------|------|------|------------|
| 1 | `src/components/terminal/TerminalEffects.tsx` | 7 | `const { enableScanline, enableGlow, enableFlicker } = useStore()` | **HIGH** — re-renders on EVERY state change, including: chat streaming (every token), input changes, SSE updates, health checks, etc. |
| 2 | `src/components/OracleView.tsx` | 30 | `const { projectID, predictions, setPredictions } = useStore()` | **HIGH** — re-renders on EVERY state change, including chat streaming, health checks, etc. |
| 3 | `src/components/CopilotView.tsx` | 40 | `const { splitView, setSplitView, chatSearchQuery, setChatSearchQuery } = useStore()` | **HIGH** — this is the MAIN CHAT VIEW. Re-renders the entire chat on every keystroke in the input field, every streamed token, every toast. This is likely the primary source of chat lag. |

**Fix pattern for all 3:** Replace `useStore()` destructuring with individual selectors:
```ts
// Instead of:
const { splitView, setSplitView, chatSearchQuery, setChatSearchQuery } = useStore()

// Use:
const splitView = useStore(s => s.splitView)
const setSplitView = useStore(s => s.setSplitView)
const chatSearchQuery = useStore(s => s.chatSearchQuery)
const setChatSearchQuery = useStore(s => s.setChatSearchQuery)
```

---

## 6. All Consumer Files (Complete Map)

33 files consume the Zustand store via `useStore()`:

| File | `useStore` calls | Accesses |
|------|-----------------|----------|
| `src/App.tsx` | 12 | `projects`, `projectID`, `selectedObject`, `selectedAgent`, `showWizard`, `showOnboarding`, `isCommandPaletteOpen`, `availableObjects`, `lastError`, `slideOverContent`, `ollamaHealthy`, `nlpHealthy` |
| `src/components/terminal/SlideOverContent.tsx` | 33 | 27+ selectors including `data`, `selectedObject`, `agents`, `tools`, `skills`, etc. |
| `src/components/terminal/InlineRenderer.tsx` | 32 | Same 27+ fields as SlideOverContent, plus inline-specific |
| `src/components/CopilotView.tsx` | 1**\*** | `splitView`, `setSplitView`, `chatSearchQuery`, `setChatSearchQuery` (full-store) |
| `src/components/terminal/TerminalView.tsx` | 11 | `currentView`, `showInlinePanel`, `agents` |
| `src/components/terminal/TerminalEffects.tsx` | 1**\*** | `enableScanline`, `enableGlow`, `enableFlicker` (full-store) |
| `src/components/terminal/TerminalPrompt.tsx` | 1 | `inputMode`, `setInputMode` (with selector) |
| `src/components/terminal/StatusBar.tsx` | 3 | `slideOverContent`, `inlineContent`, `inputMode` |
| `src/components/OracleView.tsx` | 1**\*** | `projectID`, `predictions`, `setPredictions` (full-store) |
| `src/components/Sidebar.tsx` | 3 | `inlineContent`, `slideOverContent`, `currentView` |
| `src/components/Toast.tsx` | 2 | `toastMessages`, `removeToast` |
| `src/components/AgentsView.tsx` | 2 | `setAgents`, `selectedObject` |
| `src/components/ToolsView.tsx` | 2 | `setTools`, `selectedObject` |
| `src/components/SkillsView.tsx` | 2 | `setSkills`, `selectedObject` |
| `src/components/SettingsView.tsx` | 3 | `enableScanline`, `enableGlow`, `enableFlicker` |
| `src/components/DashboardView.tsx` | 1 | `projectID` |
| `src/components/ToolManagementView.tsx` | 1 | `tools` |
| `src/components/forms/SkillExecuteSlideOver.tsx` | 3 | `tools`, `sandboxInput`, `setSandboxInput` |
| `src/components/forms/ToolExecuteSlideOver.tsx` | 2 | `sandboxInput`, `setSandboxInput` |
| `src/components/forms/ComponentDetailSlideOver.tsx` | 1 | `registryComponents` |
| `src/components/forms/AssetDetailSlideOver.tsx` | 1 | `assets` |
| `src/views/ScenarioComparisonView.tsx` | 3 | `scenarios`, `selectedScenarioIds`, `setSelectedScenarioIds` |
| `src/hooks/useAppActions.ts` | 1 | `projectID` (via `useStore.getState()` for many others) |
| `src/hooks/useSSE.ts` | 2 | `addToast` |
| `src/hooks/domain/useLibraryActions.ts` | 5 | `projectID`, `selectedAssetContent`, `setSelectedAssetContent`, `selectedAssetId`, `assets` |
| `src/hooks/domain/useExplorerActions.ts` | 4 | `setSelectedObject`, `setSearchQuery`, `setActiveView`, `setSelectedRow` |
| `src/hooks/domain/useOntologyActions.ts` | 3 | `projectID`, `setOntologyRaw`, `ontologyRaw` |
| `src/hooks/domain/useSettingsActions.ts` | 1 | `projectID` |
| `src/hooks/domain/useAgentActions.ts` | 1 | `projectID` (plus `useStore.getState()` for agents/pendingCrud) |
| `src/hooks/domain/useSkillActions.ts` | 1 | `projectID` |
| `src/hooks/domain/useToolActions.ts` | 1 | `projectID` (plus `useStore.getState()` for tools/pendingCrud) |
| `src/hooks/domain/useDataSourceActions.ts` | 1 | `projectID` |

---

## 7. Target Reduction Plan: 61 → 40

### Phase 1: Remove Dead Fields (61 → 50)

| Field | Slice | Reason |
|-------|-------|--------|
| `notificationChannels` | AuthSlice | Never read |
| `commandHistory` | NavigationSlice | Only persisted, never displayed |
| `bookmarkedIds` | CopilotSlice | Never read |
| `onlyBookmarks` | CopilotSlice | Never read or written |
| `sandboxResult` | WorkspaceSlice | Only written, never read |
| `ontologyVersions` | WorkspaceSlice | Only written, never read |
| `selectedVersionId` | WorkspaceSlice | Never read |
| `isVersionHistoryOpen` | WorkspaceSlice | Never read |
| `showGuide` | UISlice | Never read or set |
| `confirmDialog` (nested object) | UISlice | `ConfirmDialog.tsx` uses props, not store |
| `pendingCrud` | UISlice | Only written, `isCrudPending` never called |

**Result: 11 fields removed**

### Phase 2: Merge Co-occurring Fields (50 → 40)

| Merge Into | Fields Absorbed | Saving |
|------------|----------------|--------|
| `explorerQuery: { searchQuery, selectedObject, activeView }` | 3 → 1 | -2 |
| `oracleData: { predictions, scenarios, selectedScenarioIds }` | 3 → 1 | -2 |
| `terminalEffects: { scanline, glow, flicker }` | 3 → 1 | -2 |
| `assetViewer: { selectedAssetContent, selectedAssetId }` | 2 → 1 | -1 |
| `ingestionState: { ingestionTasks, taskLogs }` | 2 → 1 | -1 |
| `healthStatus: { ollamaHealthy, nlpHealthy, dataHealthStats, ollamaModels }` | 4 → 1 | -3 |

**Result: -11 fields**

These merge proposals combine fields that always change together in the same API response or user interaction. Each merged object reduces selector granularity slightly but eliminates cross-talk re-renders. For example, `explorerQuery` as a single selector means any explorer state change still triggers re-render, but it won't trigger unrelated slices.

### Target State: 40 fields

| Slice | Current | Dead Removed | Merged | Target |
|-------|---------|-------------|--------|--------|
| AuthSlice | 5 | -1 (notificationChannels) | — | 4 |
| NavigationSlice | 7 | -1 (commandHistory) | — | 6 |
| CopilotSlice | 10 | -2 (bookmarkedIds, onlyBookmarks) | — | 8 |
| WorkspaceSlice | 19 | -4 (sandboxResult, ontologyVersions, selectedVersionId, isVersionHistoryOpen) | -6 (oracle: -2, explorer: -2, ingestion: -1) | 9 |
| HealthSlice | 5 | — | -3 (→ healthStatus object) | 2 |
| UISlice | 15 | -3 (showGuide, confirmDialog, pendingCrud) | -3 (terminalEffects: -2, assetViewer: -1) | 9 |
| **Cross-slice** | — | — | — | 2 (wizard flags) |
| **Total** | **61** | **-11** | **-10** | **40** |

---

## 8. Recommendations for UX W1 Store Refactor

### Critical: Fix Full-Store Subscriptions First

The 3 full-store `useStore()` calls (TerminalEffects, OracleView, CopilotView) are the highest-impact quick fix. These are **guaranteed to cause unnecessary re-renders** on every Zustand state change — including each token of a streaming chat response, each keystroke in the input field, and each health-check poll.

**Priority:** #1 — fix CopilotView.tsx line 40. This is the main chat component.

### Dead Field Removal (Safe to Delete in W1)

All 11 dead fields can be removed without any functional impact:

- **AuthSlice:** remove `notificationChannels` + `setNotificationChannels`
- **NavigationSlice:** remove `commandHistory` + `addToHistory`
- **CopilotSlice:** remove `bookmarkedIds` + `toggleBookmark` + `onlyBookmarks` + `setOnlyBookmarks`
- **WorkspaceSlice:** remove `sandboxResult` + `setSandboxResult` + `ontologyVersions` + `setOntologyVersions` + `selectedVersionId` + `setSelectedVersionId` + `isVersionHistoryOpen` + `setVersionHistoryOpen`
- **UISlice:** remove `showGuide` + `setShowGuide` + `confirmDialog` + `showConfirmDialog` + `hideConfirmDialog` + `pendingCrud` + `setPendingCrud` + `clearPendingCrud` + `isCrudPending`

### Merge Strategy for W1

**Approach:** Compound objects for co-occurring fields, retaining Zustand 4.5.2 compatibility:

```typescript
// Before:
interface WorkspaceSlice {
  searchQuery: string
  selectedObject: string
  activeView: string
}

// After:
interface WorkspaceSlice {
  explorer: { query: string; object: string; view: string }
}
```

Selectors update minimally:
```typescript
// Before: useStore(s => s.searchQuery)
// After:  useStore(s => s.explorer.query)
```

### Slice Boundary Changes

**Current:** 6 slices + cross-slice override. WorkspaceSlice has 19 fields (31% of all state) and is a dumping ground for any data that doesn't fit elsewhere.

**Recommendation for W1:**
1. Rename `WorkspaceSlice` → `DataSlice` (clearer domain: entity data + explorer state)
2. Create `ExplorerSlice` from `{ searchQuery, selectedObject, activeView, isExplorerLoading, globalSearchResults }` currently split between WorkspaceSlice and UISlice
3. Keep `CopilotSlice` focused on chat UI state only (remove dead fields)
4. Keep `HealthSlice` focused on health status
5. Keep `AuthSlice` for project/session data
6. Keep `UISlice` for visual preferences + wizard state

### Target Intersection Map: Read Fields with Risk Scoring

| Field | Read By | Read Count | Risk if Changed |
|-------|---------|-----------|-----------------|
| `projectID` | 10 files | HIGH | CRITICAL (cross-cutting) |
| `agents` | 5 files | HIGH | MEDIUM |
| `tools` | 6 files | HIGH | MEDIUM |
| `skills` | 3 files | MEDIUM | MEDIUM |
| `lastError` | 2 files | LOW | LOW |
| `ollamaHealthy` | 3 files | MEDIUM | LOW |
| `assets` | 4 files | MEDIUM | MEDIUM |
| `selectedObject` | 5 files | HIGH | MEDIUM |
| `slideOverContent` | 3 files | MEDIUM | HIGH (navigation) |
| `inlineContent` | 3 files | MEDIUM | HIGH (navigation) |
| `chat` | 1 file | LOW | CRITICAL (core UX) |

---

## Appendix: Interface Entry Counts (including actions)

Full count of every method/action/setter in each interface, for reference:

| Slice | State | Setters/Actions | Reset | Total |
|-------|-------|----------------|-------|-------|
| AuthSlice | 5 | 6 | — | 11 |
| NavigationSlice | 7 | 7 | 1 | 15 |
| CopilotSlice | 10 | 13 | 1 | 24 |
| WorkspaceSlice | 19 | 19 | 1 | 39 |
| HealthSlice | 5 | 5 | 1 | 11 |
| UISlice | 15 | 19 | 1 | 35 |
| Cross-slice | — | 1 | — | 1 |
| **Grand Total** | **61** | **70** | **5** | **136** |

# SPEC-10: W5-W6 UX Redesign — Progressive Disclosure & Polish

**Spec version**: 1.0  
**Date**: 11 May 2026  
**Plan reference**: `docs/plans/piano-finale-aleph-26-apr.md` W5 (Progressive Disclosure) + W6 (Polish, Tests)  
**Findings addressed**: R11-R20 (information density), F15-F18 (missing loading/empty/error states)  
**Depends on**: W4 glass-panel refactor (`SlideOverPanel.tsx`, `glass-panel` CSS), Zustand `uiSlice`  
**Related specs**: `docs/specs/wave3-frontend-spec.md` (scene routing, component architecture)  
**Status**: 🔧 Draft — awaiting W4 completion

---

## 1. Progressive Disclosure Philosophy

Every Aleph view observes a **3-tier information model**. The goal is to reduce cognitive load on first encounter while preserving full access to power-user features:

| Tier | Visibility | Default | Mechanism |
|------|-----------|---------|-----------|
| **Summary** | Always visible | Open | Static render, no interaction required |
| **Details** | One click away | Closed | CSS `collapse` + click-to-expand header |
| **Advanced** | Opt-in only | Hidden | Zustand toggle or persistent settings flag |

This is **not** a tabbed interface. All three tiers coexist on the same scrollable page. The user expands sections inline, never navigates to a separate route. This matches Aleph's single-page scene architecture where each view is a self-contained `mountedView` in `App.tsx`.

### Which Views Get Progressive Disclosure

| View | Lines | Tier-1 Summary | Tier-2 Details | Tier-3 Advanced | Reason |
|------|-------|---------------|----------------|----------------|--------|
| `ToolsView` | 278 | Tool list (name, status, health) | Config, execution history | JSON editor, permissions | Highest complexity, most config surface |
| `OracleView` | 270 | Input + last response | Full conversation, context chain | Model config, temperature, system prompt | Heavy chat history |
| `SettingsView` | 225 | Critical settings (API keys, model) | All settings grouped groups | Developer/debug, feature flags | Many infrequently changed fields |
| `ComponentsView` | 197 | Component grid + health status | Per-component config | Dependency graph, metrics | Graphs can be expensive |
| `LibraryView` | 159 | Search bar + grid | Item metadata, preview | Batch ops, tags, categories | Search is primary action |
| `AgentsView` | 139 | Agent list + online status | Config steps (wizard) | System prompt editor, tool bindings | Wizard already progressive |

### Which Views Are Excluded

| View | Lines | Why |
|------|-------|-----|
| `ExplorerView` | 90 | Already simple — single file tree |
| `DataHealthView` | 93 | Already simple — single metrics panel |
| `Dashboard` | fullscreen | Different pattern — overview-first, no tiers needed |

---

## 2. Per-View Section Architecture

### 2.1 ToolsView — 3 Collapsible Sections

```
┌──────────────────────────────────┐
│ Summary: Tool Registry           │ ← always visible
│  ├─ Tool A  ● healthy   ⚡ 120ms │
│  ├─ Tool B  ○ idle      ⚡ 45ms  │
│  └─ Tool C  ✕ error     ⚡ —     │
├──────────────────────────────────┤
│ ▼ Details: Tool Configuration    │ ← expandable
│  ├─ Name, version, category      │
│  ├─ Execution history (last 10)  │
│  └─ Environment variables        │
├──────────────────────────────────┤
│ ⚙ Advanced: JSON / Permissions   │ ← opt-in toggle
│  ├─ Raw JSON config editor       │
│  └─ Role bindings (RBAC)         │
└──────────────────────────────────┘
```

- Summary header is always the full tool registry list (no collapse).
- Clicking a tool row expands its Details section inline (accordion pattern).
- The Advanced section is hidden behind a settings gear icon in the header.

### 2.2 OracleView — 2 Collapsible Sections + 1 Opt-in

```
┌──────────────────────────────────┐
│ Summary: Query Bar               │ ← always visible
│  └─ Input field + Send           │
├──────────────────────────────────┤
│ ▼ Details: Conversation          │ ← expandable
│  ├─ Q: "What are market risks?"  │
│  └─ A: "Based on Q3 data..."     │
├──────────────────────────────────┤
│ ⚙ Advanced: Model Config         │ ← opt-in toggle (gear icon)
│  ├─ Temperature (slider)         │
│  ├─ System prompt (textarea)     │
│  └─ Token limit                  │
└──────────────────────────────────┘
```

- Summary shows the input bar and the **last** exchange only.
- Details reveals the full conversation history as a scrollable chat log.
- The context chain viewer (how the decision engine arrived at the response) lives in Details.
- Advanced model tweaks are behind a settings gear in the Oracle header.

### 2.3 SettingsView — 3 Categories

```
┌──────────────────────────────────┐
│ Summary: Quick Settings          │ ← always visible
│  ├─ API Key (masked [●──])       │
│  ├─ Default Model                │
│  └─ Theme toggle                 │
├──────────────────────────────────┤
│ ▼ Details: All Settings          │ ← expandable
│  ├─ Project settings             │
│  ├─ Notification prefs           │
│  └─ Data source defaults         │
├──────────────────────────────────┤
│ ⚙ Advanced: Developer            │ ← opt-in (gear icon)
│  ├─ Debug logging level          │
│  ├─ Feature flags (genesis, etc) │
│  └─ DuckDB query inspector       │
└──────────────────────────────────┘
```

- Summary contains only the 3 most-changed settings (API keys, model, theme).
- Details groups remaining settings by category with collapsible sub-groups.
- Advanced is hidden unless `devMode` is enabled in the user profile.

### 2.4 ComponentsView — Status Overview

```
┌──────────────────────────────────┐
│ Summary: Component Grid          │ ← always visible
│  ├─ [🟢 Router] [🟢 Store]       │
│  └─ [🟡 NLP]    [🔴 Watcher]    │
├──────────────────────────────────┤
│ ▼ Details: Component Config      │ ← expandable
│  ├─ Per-component version, uptime│
│  └─ Health history (sparkline)   │
├──────────────────────────────────┤
│ ⚙ Advanced: Dependency Graph     │ ← opt-in tab (default hidden)
│  ├─ SVG dependency graph         │
│  └─ Component metrics table      │
└──────────────────────────────────┘
```

- Component grid is always rendered (lightweight — just name + status badge).
- Details expands to show config and health sparkline per component.
- Dependency graph is rendered only when the user explicitly opens the Advanced section.

### 2.5 LibraryView — Search-First

```
┌──────────────────────────────────┐
│ Summary: Search + Grid           │ ← always visible
│  └─ 🔍 Search... [Grid View]     │
├──────────────────────────────────┤
│ ▼ Details: Item Preview          │ ← expandable
│  ├─ Metadata (author, date, tags)│
│  ├─ Content preview              │
│  └─ Related items                │
├──────────────────────────────────┤
│ ⚙ Advanced: Batch Operations     │ ← opt-in toggle
│  ├─ Multi-select mode            │
│  ├─ Bulk tag / categorize        │
│  └─ Export selected              │
└──────────────────────────────────┘
```

- Summary is search bar + grid (primary interaction).
- Clicking a grid item opens Details as an inline expansion (not a modal).
- Advanced batch mode is toggled by a button in the view header.

### 2.6 AgentsView — Wizard-Native

```
┌──────────────────────────────────┐
│ Summary: Agent List              │ ← always visible
│  ├─ Agent A  ● online            │
│  └─ Agent B  ○ offline           │
├──────────────────────────────────┤
│ ▼ Details: Config Steps          │ ← expandable
│  ├─ Step 1: Name & description   │
│  ├─ Step 2: Model selection      │
│  └─ Step 3: Data sources         │
├──────────────────────────────────┤
│ ⚙ Advanced: System Prompt        │ ← opt-in toggle
│  ├─ System prompt editor         │
│  └─ Tool binding matrix          │
└──────────────────────────────────┘
```

- Agent list is always visible (the "explore" view).
- Clicking an agent expands its configuration wizard inline.
- System prompt and tool bindings are behind an "Advanced" toggle in the wizard header.

---

## 3. Implementation Pattern

### 3.1 Zustand State — `uiSlice` Extension

Add to `uiSlice.ts`:

```typescript
// New state field
expandedSections: Record<string, boolean>;

// New actions
toggleSection(viewId: string, sectionId: string): void;
collapseAll(viewId: string): void;
expandAll(viewId: string): void;
```

The key format is `"${viewId}:${sectionKey}"` — e.g. `"tools:details-configuration"`, `"oracle:advanced-model"`.

### 3.2 Collapsible Section Component

No new components — use the existing `glass-panel` with collapsible headers. Pattern:

```tsx
// In each view, wrap sections with:
<GlassPanel collapsible defaultCollapsed={true} title="Details">
  {...section content}
</GlassPanel>
```

The `collapsible` prop on `GlassPanel`:
- Renders a clickable header with a `▼` / `▶` indicator
- Toggles `expandedSections[viewSectionKey]` in Zustand
- Uses CSS `max-height` + `transition` for collapse animation (no JS animation)
- Remembers state across navigation if `persistKey` is provided

### 3.3 CSS-Only Collapse Animation

```css
.glass-panel-collapsible .panel-content {
  max-height: 0;
  overflow: hidden;
  transition: max-height 0.3s ease-out, opacity 0.3s ease-out;
  opacity: 0;
}

.glass-panel-collapsible.expanded .panel-content {
  max-height: 2000px; /* larger than any content */
  opacity: 1;
}
```

No JS animation libraries. No `element.animate()`. Pure CSS transitions with `will-change: max-height, opacity` for GPU acceleration.

### 3.4 URL-Persistable State

The `expandedSections` state for a given view is persisted as URL query params:

```
/tools?expand=details-configuration,advanced-permissions
```

This enables sharing links with specific sections expanded. The `mountedView` scene router reads these params on mount and sets `expandedSections` accordingly.

---

## 4. Polish Pass — Scope

### 4.1 Animation Consistency

| Element | Duration | Easing | Notes |
|---------|----------|--------|-------|
| SlideOver slide-in | 0.3s | `ease-out` | Right edge → center |
| Panel collapse/expand | 0.3s | `ease-out` | Height transition |
| Button hover | 0.15s | `ease-out` | Background/border tint |
| Tooltip fade | 0.2s | `ease-out` | Opacity only |
| Route transition | 0.2s | `ease-out` | Content fade-in |

All panel animations use `0.3s ease-out` as the default. No custom cubic-bezier curves unless a design review requires it.

### 4.2 CSS Volatility Layer Audit

Ensure every CSS rule in the frontend is assigned to one of four volatility layers (defined in `volatility.css`):

| Layer | Scope | Rules |
|-------|-------|-------|
| `vol-static` | Base styles | Reset, typography, color tokens |
| `vol-structural` | Layout | Grid, flex, spacing, positioning |
| `vol-interactive` | Behavior | Hover, focus, active, transitions |
| `vol-signal` | State-driven | Error, loading, empty, online/offline |

Audit target: zero rules in `:root` or `*` that should be in a named layer.

### 4.3 Empty States

Every view must render a meaningful empty state when its data source returns zero results:

| View | Empty State |
|------|-------------|
| `ToolsView` | "No tools installed. Add your first tool from the registry." |
| `OracleView` | "No conversation yet. Ask a question to begin." |
| `SettingsView` | N/A — always has defaults |
| `ComponentsView` | "No components loaded. Aleph requires at least the core engine." |
| `LibraryView` | "Your library is empty. Import data via the sidebar or drag files here." |
| `AgentsView` | "No agents configured. Create your first agent to get started." |
| `ExplorerView` | "No files to display. Connect a data source or upload a file." |
| `DataHealthView` | "No data quality metrics. Ingest data to see health scores." |

Each empty state includes:
- An icon (use the view's existing icon from `LucideIcon` map)
- A short message (≤20 words)
- An action button linking to the relevant create/import flow

### 4.4 Loading States

Every lazy-loaded view (via `React.lazy` in `App.tsx`) must have a `<Suspense>` wrapper with a skeleton loader:

```tsx
<React.Suspense fallback={<SkeletonLoader rows={4} />}>
  <MountedView />
</React.Suspense>
```

The `SkeletonLoader` component (already existing) renders 4 pulsing rows matching the view's layout structure.

### 4.5 Responsive Floor

Aleph is a desktop-first application. The minimum supported viewport width is **1024px**. No responsive breakpoints target widths below 1024px.

```
@media (max-width: 1024px) {
  /* Only: reduce sidebar icon size, collapse spacing */
}
@media (max-width: 1280px) {
  /* Optional: reduce grid columns from 4→3 */
}
```

### 4.6 Error Boundaries

One `<ErrorBoundary>` per scene component in `App.tsx`:

```tsx
// BEFORE: single error boundary wrapping all scenes
<ErrorBoundary><SceneRenderer /></ErrorBoundary>

// AFTER: per-scene error boundary
<ErrorBoundary key="tools"><ToolsView /></ErrorBoundary>
<ErrorBoundary key="oracle"><OracleView /></ErrorBoundary>
// ... etc
```

Each error boundary renders the view's icon + "Something went wrong" + a "Reload" button that calls `window.location.reload()`.

### 4.7 Keyboard Navigation

| Action | Key | Scope |
|--------|-----|-------|
| Close SlideOver | `Escape` | Global (already implemented in `SlideOverPanel.tsx`) |
| Tab through form fields | `Tab` / `Shift+Tab` | All forms (default browser behavior, verify no `tabindex` conflicts) |
| Expand/collapse section | `Space` or `Enter` | When focus is on a collapsible header |
| Cmd+K palette | `Cmd+K` / `Ctrl+K` | Global (already implemented in `CopilotView.tsx`) |

Audit all views for `tabindex={-1}` or `aria-hidden="true"` that traps keyboard focus in SlideOver or modal elements.

---

## 5. Test Migration Plan

### 5.1 Playwright Tests — 10 → 12

The existing 10 Playwright e2e tests (from the pre-redesign W0-W3 era) are rewritten for the new architecture:

| # | Name | What It Tests | Coverage |
|---|------|--------------|----------|
| 1 | `sidebar-navigation` | 5 sidebar items switch scenes correctly | Core routing |
| 2 | `slideover-opens` | Clicking an item opens SlideOver panel | Interaction |
| 3 | `command-palette` | Cmd+K opens search, enter navigates | Keyboard UX |
| 4 | `tool-results` | Tool execution result renders in SlideOver | Data flow |
| 5 | `scene-routing-url` | URL param `?scene=tools` renders ToolsView | URL routing |
| 6 | `scene-routing-default` | No URL param loads default scene | Edge case |
| 7 | `scene-routing-bad` | Invalid scene param falls back to default | Error handling |
| 8 | `progressive-disclosure` | Expanding sections shows/hides content | Disclosure tiers |
| 9 | `store-persist` | Zustand state persists view selection | Store contract |
| 10 | `store-reset` | Reset action clears expanded sections | Store contract |
| 11 | `regression-tools` | ToolsView loads tools list, empty state | Regression |
| 12 | `regression-oracle` | OracleView renders query bar | Regression |

### 5.2 Vitest Unit Tests

Existing vitest tests continue to pass. New tests for the `uiSlice` expanded sections:

```typescript
describe('uiSlice expandedSections', () => {
  it('toggles section', () => {
    store.getState().toggleSection('tools', 'details-configuration');
    expect(store.getState().expandedSections['tools:details-configuration']).toBe(true);
  });
  it('collapseAll resets view', () => {
    store.getState().collapseAll('tools');
    expect(Object.keys(store.getState().expandedSections)
      .filter(k => k.startsWith('tools:'))).toEqual([]);
  });
});
```

### 5.3 Go Backend Tests

No backend changes in W5-W6 — `go test -count=1 ./...` must continue passing unchanged.

---

## 6. Build Verification Criteria

All of the following must pass before W5-W6 can be marked complete:

| Check | Command | Expected |
|-------|---------|----------|
| Go build | `go build ./...` | Zero errors |
| Go tests | `go test -count=1 ./...` | All pass |
| Go vet | `go vet ./...` | Zero errors (exclude pre-existing `dsl/ast.go` PEG struct tag warnings) |
| TypeScript | `npx tsc --noEmit -p tsconfig.app.json` | Zero errors |
| Frontend unit | `npx vitest run` | All pass |
| Frontend build | `npx vite build` | Exit code 0 |
| Playwright | `npx playwright test` | 12/12 pass |
| LSP diagnostics | `lsp_diagnostics` on all changed files | Zero errors |

---

## Appendix A: Collapsible Section Data Flow

```
User clicks "▼ Details" header
  → GlassPanel onClick handler
  → store.toggleSection(viewId, sectionKey)
  → Zustand batch update
  → React re-render with .expanded class
  → CSS transition animates max-height: 0 → 2000px
  → URL param updated (if persist enabled)
```

## Appendix B: File Change Summary

| File | Change |
|------|--------|
| `frontend/src/store/uiSlice.ts` | Add `expandedSections`, `toggleSection`, `collapseAll`, `expandAll` |
| `frontend/src/components/terminal/GlassPanel.tsx` | Add `collapsible`, `defaultCollapsed`, `persistKey` props |
| `frontend/src/components/terminal/GlassPanel.css` | Add `.glass-panel-collapsible` styles with CSS transition |
| `frontend/src/views/ToolsView.tsx` | Wrap detail sections in collapsible GlassPanel |
| `frontend/src/views/OracleView.tsx` | Same |
| `frontend/src/views/SettingsView.tsx` | Same |
| `frontend/src/views/ComponentsView.tsx` | Same |
| `frontend/src/views/LibraryView.tsx` | Same |
| `frontend/src/views/AgentsView.tsx` | Same |
| `frontend/src/App.tsx` | Per-scene error boundaries, empty state wiring |
| `frontend/src/styles/volatility.css` | Layer audit fixes |
| `frontend/src/components/terminal/SkeletonLoader.tsx` | Verify existing — add if missing |
| `frontend/e2e/*.spec.ts` | Rewrite 10 tests → 12 |

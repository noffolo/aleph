# SPEC-W3: SlideOver Unification — Eliminate Dual-View Architecture

**Spec version**: 1.0  
**Date**: 11 May 2026  
**Plan reference**: `plans/aleph-ux-redesign-piano.md` Wave 3 (W3-01 through W3-04)  
**Depends on**: W2 (Navigation scene router with `?view` param) — must land first  
**Related specs**: `docs/specs/wave3-frontend-spec.md` (store contracts)  
**Status**: 🔴 Draft — pending W2 completion

---

## 1. Overview

Aleph-v2 currently maintains **two parallel rendering paths** for its views: `InlineRenderer` (inside CopilotView's split pane) and `SlideOverContent` (inside the slide-over panel, triggered by sidebar clicks). Both lazy-load the same 11 view components with near-identical prop-passing. This duplication adds ~574 lines of dead-weight code, 42 `as unknown as` type casts, and a confusing mental model where the same view can appear either inline or in a slideover depending on how the user triggered it.

**W3 eliminates InlineRenderer entirely** and folds its functionality into the SlideOver system. The split-view rendering inside CopilotView becomes a simple message-detail panel (no view loading). All slash-command routing (`/explore`, `/agent`, `/skills`, etc.) goes exclusively through `setSlideOverContent`.

**Critical ordering constraint**: W3-05 ("Delete InlineRenderer") must execute AFTER W3, not before. W2 (Navigation — scene routing via `?view` param) must land first so that the new Scene components exist before the old path is torn down.

---

## 2. Current Dual-View Problem

### 2.1 InlineRenderer (`frontend/src/components/terminal/InlineRenderer.tsx` — 266 lines)

- Imports 11 lazy views: ExplorerView, DataSourcesView, OntologyView, DataHealthView, SettingsView, ComponentsView, SkillsView, ToolsView, LibraryView, AgentsView, OracleView
- Renders them inside CopilotView (line 216: `<InlineRenderer />`) when `splitView` is enabled
- Guards rendering with `showInlinePanel` + `inlineContent` from navigationSlice
- Adds a header bar with close button using terminal styling (`bg-surface`, `animate-fade-in`)
- Handles 11 type cases: `explore`, `agent`, `ontology`, `data`, `health`, `skill`, `tool`, `component`, `settings`, `library`, `predict`
- Pulls 30+ Zustand selectors and 9 domain hooks for prop construction

**Dependents** (4 files):
- `CopilotView.tsx` — imports and renders `<InlineRenderer />` at line 215-217 inside an `InlineErrorBoundary`
- `App.tsx` — transitively via CopilotView/TerminalView
- `TerminalView.tsx` — reads `showInlinePanel` from store (currently dead — not used in render)
- `useAppActions.ts` — `SHOW_INLINE` action routes to `setInlineContent` + `setShowInlinePanel(true)`

### 2.2 SlideOverContent (`frontend/src/components/terminal/SlideOverContent.tsx` — 308 lines)

- Imports the same 11 lazy views (identical lazy imports) plus ToolIntelligenceView
- Renders them inside `SlideOverPanel` in App.tsx (lines 184-196)
- Handles 22 type cases: all 11 from InlineRenderer plus `sandbox`, `agent-form`, `skill-form`, `tool-form`, `datasource-form`, `component-form`, `component-detail`, `asset`, `detail`, `scenario-comparison`, `datasource`, `tool-intelligence`
- Also renders 9 form slideover components directly
- Pulls the same 30+ Zustand selectors and 9 domain hooks

### 2.3 Duplication Summary

| Dimension | InlineRenderer | SlideOverContent |
|-----------|---------------|-----------------|
| Lines | 266 | 308 |
| Lazy imports | 11 (shared) | 11 (shared) + 1 unique |
| Type cases | 11 | 22 |
| Form components | 0 | 9 |
| Domain hooks | 9 | 9 |
| `as unknown as` casts | 5 | 6 |
| Zustand selectors | ~30 | ~35 |

### 2.4 Bloat in SlideOverContent Type System

Current `SlideOverContent['type']` union has **22 variants**:

```
explore | ontology | data | health | skill | tool | sandbox | agent | datasource |
component | settings | library | predict | asset | detail | agent-form | skill-form |
tool-form | datasource-form | component-form | component-detail | tool-intelligence |
scenario-comparison | dashboard
```

Notable duplication:
- `datasource` (line 280, dead-ish duplicate of `data`)
- `scenario-comparison` (never used — `ScenarioComparisonView` may be removed)
- `asset` (duplicates `library`'s detail mode)
- `detail` (internal-only, can inline into consumers)

---

## 3. Unified Architecture

### 3.1 After W3

```
User Action
    │
    ├── Sidebar click → useStore.setState({ scene: 'terminal'|'explore'|'agents'|'system' })
    │
    ├── Slash command → setSlideOverContent({ type, title, data })
    │
    ├── Item click in view → setSlideOverContent({ type: '...-form', data: item })
    │
    └── Cmd+K → CommandPalette → setSlideOverContent(...) or switch scene
```

**No more `inlineContent`, `showInlinePanel`, `currentView: 'inline'`.**

All views render through **exactly one path**: `SlideOverPanel → SlideOverContent` (for detail/edit/form) or full-scene replacement (for TerminalScene/ExploreScene/AgentsScene/SystemScene).

### 3.2 Scene Components Replace Full-Screen Views

Instead of InlineRenderer occupying the CopilotView split pane, 5 **Scene components** occupy the entire main content area. They are full-screen list/dashboard views. The SlideOver opens on top for detail/edit.

```
┌──────────────────────────────────────────┐
│  Sidebar   │   Scene (full area)          │
│            │                              │
│  [#]       │  ┌────────────────────────┐  │
│  Terminal  │  │  TerminalScene          │  │
│            │  │  ┌──────────────────┐   │  │
│  [@]       │  │  │ CopilotView      │   │  │
│  Explore   │  │  │ (chat only,      │   │  │
│            │  │  │  no split)       │   │  │
│  [%]       │  │  └──────────────────┘   │  │
│  Agents    │  │                          │  │
│            │  └────────────────────────┘  │
│  [&]       │                              │
│  System    │  SlideOverPanel (overlay)    │
│            │  ┌──────────────────────┐    │
│            │  │ Detail / Edit / Form │    │
│            │  └──────────────────────┘    │
└──────────────────────────────────────────┘
```

---

## 4. Component Consolidation Plan

### 4.1 Files to Delete

| File | Lines | Reason |
|------|-------|--------|
| `InlineRenderer.tsx` | 266 | Entirely replaced by Scene components + SlideOver |
| `SlideOverContent.tsx` | 308 | Split into Scene components (view parts) + simplified SlideOver (form/edit parts) |

### 4.2 Files to Modify

| File | Changes |
|------|---------|
| `CopilotView.tsx` | Remove `import { InlineRenderer }`, remove `<InlineRenderer />` at lines 215-217, simplify `splitView` to message-detail only |
| `App.tsx` | Replace `/SlideOverContent` import with simplified version; add Scene component mounting logic keyed by `scene` state |
| `TerminalView.tsx` | Remove dead `showInlinePanel` reference (line 8); no functional change |
| `Sidebar.tsx` | Remove `inlineContent`/`inlineType` references (lines 60-63); switch to `scene` state |
| `StatusBar.tsx` | Remove `inlineContent`/`inlineType` references (lines 12-15); switch to `scene` state |
| `useAppActions.ts` | Rewrite `SHOW_INLINE` case (lines 121-139) to always call `setSlideOverContent` and never touch `inlineContent`/`showInlinePanel` |
| `navigationSlice.ts` | Remove `inlineContent`, `showInlinePanel`, `setCurrentView('inline')` |
| `useStore.ts` | Remove `InlineContent` interface; trim `SlideOverContent['type']` from 22 to ~12 variants |

### 4.3 SlideOverContent Replacement

After W3, what remains of SlideOverContent is a **simplified switch** (~100 lines) handling only:

| Type | Component |
|------|-----------|
| `skill` (with data) | `SkillExecuteSlideOver` |
| `tool` (with data) | `ToolExecuteSlideOver` |
| `sandbox` | `SandboxResultSlideOver` |
| `agent-form` | `AgentFormSlideOver` |
| `skill-form` | `SkillFormSlideOver` |
| `tool-form` | `ToolFormSlideOver` |
| `datasource-form` | `DataSourceFormSlideOver` |
| `component-form` | `ComponentFormSlideOver` |
| `component-detail` | `ComponentDetailSlideOver` |
| `tool-intelligence` | `ToolIntelligenceView` |

All `explore`, `agent`, `ontology`, `data`, `health`, `skill` (list), `tool` (list), `component` (list), `settings`, `library`, `predict` view types are **removed from SlideOverContent** — they render as full Scene content, not in the slideover.

### 4.4 SlideOverPanel (`SlideOverPanel.tsx` — 176 lines)

**Unchanged.** Already handles focus trap, inert on `#main-content`, Escape-to-close, fullscreen toggle, aria-modal. No modifications needed.

---

## 5. Scene Component Design

Each Scene component is a small (~40–60 lines) routing shell that receives a `?view` query parameter and renders the appropriate sub-view. Scenes occupy the full main content area and never appear in the SlideOver.

### 5.1 TerminalScene (`scenes/TerminalScene.tsx`)

```
┌──────────────────────────────────┐
│  TerminalScene                    │
│  ┌────────────────────────────┐  │
│  │ CopilotView (full height,  │  │
│  │  no split, no inline)      │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

- Wraps `CopilotView` directly (no TerminalView wrapper redirect needed)
- **Only** the chat — no splitView, no InlineRenderer, no message-detail pane
- The splitView toggle in CopilotView header stays but becomes a simple **message detail** pane (right half showing message contents/tool calls) — no view loading
- `?view` param: ignored (only one view)

### 5.2 ExploreScene (`scenes/ExploreScene.tsx`)

```
┌──────────────────────────────────┐
│  ExploreScene                     │
│  ┌────────────────────────────┐  │
│  │ ?view=explorer → Explorer  │  │
│  │ ?view=library  → Library   │  │
│  │ ?view=ontology → Ontology   │  │
│  │ ?view=data     → DataSrc   │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

- Default: `?view=explorer`
- Each sub-view renders without the `inline` prop (no header bar/close button)
- The `inline` prop in all view components should be deprecated — it was only meaningful for InlineRenderer's embedded header
- Sidebar "Explore" click sets `scene: 'explore'` with `view: 'explorer'`

### 5.3 AgentsScene (`scenes/AgentsScene.tsx`)

```
┌──────────────────────────────────┐
│  AgentsScene                      │
│  ┌────────────────────────────┐  │
│  │ ?view=agents → AgentsView  │  │
│  │ ?view=skills → SkillsView  │  │
│  │ ?view=tools  → ToolsView   │  │
│  │ ?view=components → Comp    │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

- Default: `?view=agents`
- Create/Edit actions open SlideOver (agent-form, skill-form, tool-form, component-form)

### 5.4 SystemScene (`scenes/SystemScene.tsx`)

```
┌──────────────────────────────────┐
│  SystemScene                      │
│  ┌────────────────────────────┐  │
│  │ ?view=health   → Health    │  │
│  │ ?view=settings → Settings  │  │
│  │ ?view=oracle   → Oracle    │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

- Default: `?view=health`
- OracleView moves from predict to system scene

### 5.5 DashboardScene (`scenes/DashboardScene.tsx`)

```
┌──────────────────────────────────┐
│  DashboardScene                   │
│  ┌────────────────────────────┐  │
│  │ DashboardView (fullscreen) │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

- Already handled in App.tsx as a special case (lines 175-179)
- No sidebar entry — invoked programmatically (e.g., project load)

### 5.6 Scene Component Template

Each scene follows this pattern:

```tsx
// scenes/ExploreScene.tsx
import { Suspense } from 'react'
import { useSearchParams } from 'react-router-dom'  // or a simple store-based param
import { ExplorerView } from '../components/ExplorerView'
import { LibraryView } from '../components/LibraryView'
// ...

export const ExploreScene = () => {
  const params = new URLSearchParams(window.location.search)
  const view = params.get('view') || 'explorer'

  const renderView = () => {
    switch (view) {
      case 'library':  return <LibraryView />
      case 'ontology': return <OntologyView />
      case 'data':     return <DataSourcesView />
      default:         return <ExplorerView />
    }
  }

  return (
    <div className="h-full bg-background overflow-auto">
      <Suspense fallback={<SkeletonLoader rows={12} cols={1} />}>
        {renderView()}
      </Suspense>
    </div>
  )
}
```

---

## 6. Store Contract Changes

### 6.1 Remove from `NavigationSlice`

| Field | Reason |
|-------|--------|
| `currentView` (`'copilot' | 'inline'`) | No more inline view — replace with `scene` |
| `inlineContent` | Replaced by SlideOverContent or Scene routing |
| `showInlinePanel` | Always false — InlineRenderer deleted |
| `setShowInlinePanel` | Dead code |
| `setInlineContent` | Dead code |
| `setCurrentView('inline')` | Dead branch |

### 6.2 Add to `NavigationSlice`

| Field | Type | Purpose |
|-------|------|---------|
| `scene` | `'terminal' \| 'explore' \| 'agents' \| 'system'` | Active scene selector |
| `setScene` | `(s: SceneType) => void` | Scene switcher |

The `?view` param within each scene can be derived from URL search params or a lightweight store field.

### 6.3 Trim `SlideOverContent['type']`

**Before:** 22 variants  
**After:** ~12 variants (removing all view-list types)

**Removed:** `explore`, `agent`, `ontology`, `data`, `health`, `skill` (list), `tool` (list), `component` (list), `settings`, `library`, `predict`, `datasource`, `asset`, `detail`, `scenario-comparison`

**Retained:** `sandbox`, `agent-form`, `skill-form`, `tool-form`, `datasource-form`, `component-form`, `component-detail`, `tool-intelligence`, `skill` (detail/execute only), `tool` (detail/execute only)

**Moved to App.tsx special case:** `dashboard` (stays as-is, lines 175-179)

---

## 7. Execution Order & Dependencies

### 7.1 Wave Dependency Graph

```
W1 (Sidebar 5 items)            W3 BEGINS HERE
    │
    ▼
W2 (Navigation scene router) ──── W3 CAN START ────► AFTER W2 LANDS:
    │                                                   │
    │                                                   ▼
    │                              ┌─────────────────────────────┐
    │                              │ W3-01: Create 5 Scene       │
    │                              │        components           │
    │                              └─────────────────────────────┘
    │                                                   │
    │                              ┌─────────────────────────────┐
    │                              │ W3-02: Simplify             │
    │                              │        SlideOverContent     │
    │                              │        (remove view cases)  │
    │                              └─────────────────────────────┘
    │                                                   │
    │                              ┌─────────────────────────────┐
    │                              │ W3-03: Rewrite SHOW_INLINE  │
    │                              │        → setSlideOverContent│
    │                              └─────────────────────────────┘
    │                                                   │
    │                              ┌─────────────────────────────┐
    │                              │ W3-04: Update App.tsx       │
    │                              │        scene mounting +     │
    │                              │        remove old imports   │
    │                              └─────────────────────────────┘
    │                                                   │
    │                              ┌─────────────────────────────┐
    │                              │ W3-05: DELETE InlineRenderer│
    │                              │ DELETE SlideOverContent     │
    │                              │ (old file, replaced by      │
    │                              │  simplified version)        │
    │                              └─────────────────────────────┘
    │                                                   │
    │                              ┌─────────────────────────────┐
    │                              │ W3-06: Clean store +        │
    │                              │        update E2E tests     │
    │                              └─────────────────────────────┘
    │                                                   │
    ▼                                                   ▼
W4 (Zustand state refactor) ──── W3 MUST BE DONE FIRST
```

### 7.2 E2E Tests Referencing InlineRenderer

| File | What to update |
|------|----------------|
| `frontend/e2e/ontology-flow.spec.ts` (line 48) | Replace `inlineContent` mock with `slideOverContent` |
| `frontend/e2e/slideover.spec.ts` | Panel assertions remain valid; no change needed |
| `frontend/e2e/commands.spec.ts` | `/explore` command — verify it routes to `slideOverContent` |
| `frontend/e2e/tool-lifecycle.spec.ts` | Tool detail routing — already uses `slideOverContent` |
| 6 other spec files | No direct InlineRenderer references; verify still pass |

Total test files to audit: 10. Estimated changes: 3 files (ontology-flow, commands, slideover assertions).

### 7.3 Risk Items

| Risk | Impact | Mitigation |
|------|--------|------------|
| `inline` prop removal breaks view layouts | Visual regression in Scene views | Audit all 11 views for `inline`-dependent CSS; make `inline` no-op or remove after Scene migration |
| CopilotView `splitView` loses functionality | Users lose message-detail view | Keep splitView toggle but make it purely a message detail pane — no view loading |
| Scene component mounting flash | Perceived slowness | Pre-warm lazy imports via prefetch; re-use Suspense boundaries |
| Old `inlineContent` calls in domain hooks | Runtime errors | Search all `setInlineContent` call sites; grep for `inlineContent` across entire frontend after migration |

---

## 8. Files Changed Summary

| Action | File | Est. Δ |
|--------|------|--------|
| 🗑️ DELETE | `frontend/src/components/terminal/InlineRenderer.tsx` | −266 lines |
| 🗑️ DELETE | `frontend/src/components/terminal/SlideOverContent.tsx` | −308 lines (replaced) |
| ✏️ REWRITE | New simplified `terminal/SlideOverContent.tsx` (~100 lines) | −208 net |
| ✏️ CREATE | `frontend/src/scenes/TerminalScene.tsx` | +40 lines |
| ✏️ CREATE | `frontend/src/scenes/ExploreScene.tsx` | +55 lines |
| ✏️ CREATE | `frontend/src/scenes/AgentsScene.tsx` | +55 lines |
| ✏️ CREATE | `frontend/src/scenes/SystemScene.tsx` | +50 lines |
| ✏️ CREATE | `frontend/src/scenes/DashboardScene.tsx` | +15 lines |
| 🔧 MODIFY | `App.tsx` — scene mounting + slideover wiring | ~30 lines changed |
| 🔧 MODIFY | `CopilotView.tsx` — remove InlineRenderer, simplify split | ~5 lines changed |
| 🔧 MODIFY | `useAppActions.ts` — SHOW_INLINE → always slideover | ~15 lines changed |
| 🔧 MODIFY | `navigationSlice.ts` — swap inline fields for scene | ~15 lines changed |
| 🔧 MODIFY | `useStore.ts` — trim SlideOverContent type union | ~5 lines changed |
| 🔧 MODIFY | `Sidebar.tsx` — switch inline→scene references | ~5 lines changed |
| 🔧 MODIFY | `StatusBar.tsx` — remove inlineContent ref | ~3 lines changed |
| 🔧 MODIFY | `TerminalView.tsx` — remove dead showInlinePanel | ~3 lines changed |
| 🔧 MODIFY | `frontend/e2e/ontology-flow.spec.ts` — inlineContent→slideOverContent | ~5 lines changed |
| 🔍 AUDIT | `frontend/e2e/commands.spec.ts` — verify SHOW_INLINE routing | ~2 lines changed |

**Net delta**: approximately −550 lines of code, +220 lines of new code = **−330 lines net reduction**.

---

## 9. Verification Criteria

1. **No more InlineRenderer imports**: `grep -r "InlineRenderer" frontend/src/` returns only deleted file
2. **No more `inlineContent` in store**: `useStore.getState().inlineContent` is `undefined`
3. **No more `showInlinePanel`**: `grep -r "showInlinePanel" frontend/src/` returns 0
4. **All 11 views render correctly in their Scene**: manual check of each `?view` param
5. **Slash commands route to SlideOver**: `/explore`, `/agent`, `/skills` etc. all call `setSlideOverContent`
6. **CopilotView splitView still toggles**: but shows message detail only (no InlineRenderer)
7. **E2E tests pass**: `npx playwright test` (10 spec files)
8. **Frontend build clean**: `npx tsc --noEmit && npx vite build`
9. **SlideOverPanel unchanged**: focus trap, inert, escape key, fullscreen all work
10. **No "inline" prop warnings**: all views rendered without `inline` prop in Scene context

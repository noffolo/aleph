# W2 — Navigation Redesign

**Status:** Draft  
**Wave:** W2 — Navigation Redesign  
**Depends on:** W1 (design tokens, theme foundation)  
**Leads into:** W3 (SlideOver unification — InlineRenderer elimination)  
**Key constraint:** No React Router. Scene routing extends the existing nuqs-based `NavigationStateSync` approach.

## Sidebar Architecture: 13 → 5 Items

The current sidebar carries 13 items (Dashboard, Explorer, Data Health, Copilot, Oracle, Library, Ontologies, Data Sources, Agents, Skills, Tools, Components, Settings) with dividers after positions 1, 4, 7. Twelve items map 1:1 to `SlideOverContent['type']` via the `ID_TO_INLINE_TYPE` lookup table in Sidebar.tsx. Copilot is special — clicking it resets to chat state.

The new sidebar collapses related views into 5 scene-mapped items. Each item maps to a **Scene component** (not directly to a view type). Scene components internally dispatch to the correct view based on the `?view` URL parameter.

### New 5-Item Structure

| # | Item | Icon | Scene | Merges From |
|---|------|------|-------|-------------|
| 1 | Terminal | `Terminal` | `terminal` | Dashboard |
| 2 | Explore | `Compass` | `explore` | Explorer, Library, Ontologies, Data Sources |
| 3 | Agents | `UserCog` | `agents` | Agents, Skills, Tools, Components |
| 4 | System | `MonitorCog` | `system` | Data Health, Settings, Oracle |
| 5 | Copilot | `Bot` | `null` (chat only) | Copilot (unchanged) |

No dividers between items. The bottom Settings icon (project selector) remains unchanged.

### Sidebar.tsx Changes

Replace the 13-item `sidebarItems` array with the 5-item array. Remove `ID_TO_INLINE_TYPE` and `PANEL_TITLES` lookup tables — they are replaced by a shared `VIEW_TO_SCENE` mapping (see §Scene Routing). The `handleClick` function rewrites to set `currentScene` via the store:

```typescript
const handleClick = (item: SidebarItem) => {
  if (!projectID) { onShowOnboarding(); return }
  if (item.id === 'copilot') {
    useStore.getState().setCurrentScene(null)
    useStore.getState().setSlideOverContent(null)
    return
  }
  useStore.getState().setCurrentScene(item.scene!)
}
```

The old `Dashboard → 'dashboard'` special case is removed — TerminalScene handles it. The `Settings` bottom button is removed (Settings moves into the System scene as a SlideOver view).

## Scene Routing System

Scene routing extends the existing `NavigationStateSync` hook (51 lines, nuqs-based). No new routing library is introduced. The `?scene` URL parameter is added alongside existing `?view`, `?tab`, `?slide`.

### URL Parameter Hierarchy

```
?scene=terminal|explore|agents|system   (NEW — top-level routing key)
?view=explore|agent|ontology|...        (existing — scoped within active scene)
?tab=table|graph|map|timeline           (existing — unchanged)
?slide=<SlideOverContent['type']>       (existing — unchanged)
```

### NavigationStateSync Extension

The existing hook is extended — **not rewritten** — with three additions:

1. **New `useQueryState` call** for `?scene`:
   ```typescript
   const [scene, setScene] = useQueryState('scene', {
     defaultValue: null,
     parse: (v: string | null) => v || null,
     serialize: (v: string | null) => v || '',
   })
   ```

2. **Store sync on init**: The first `useEffect` reads `?scene` and calls `setCurrentScene(scene)`. If `?scene` is absent but `?view` is present, the scene is derived via `VIEW_TO_SCENE` mapping (see below).

3. **Store → URL sync**: The second `useEffect` subscribe watches `currentScene` changes and writes to `?scene`. A comparison guard prevents feedback loops.

### Shared Scene Mapping

A new file `src/store/sceneMapping.ts` defines the mapping tables used by both NavigationStateSync and useAppActions:

```typescript
export const VIEW_TO_SCENE: Record<string, string | null> = {
  explore:  'explore',
  library:  'explore',
  ontology: 'explore',
  data:     'explore',
  agent:    'agents',
  skill:    'agents',
  tool:     'agents',
  component:'agents',
  health:   'system',
  settings: 'system',
  predict:  'system',
  dashboard:'terminal',
}

export const EXPLORE_VIEWS = ['explore', 'library', 'ontology', 'data']
export const AGENT_VIEWS   = ['agent', 'skill', 'tool', 'component']
export const SYSTEM_VIEWS  = ['health', 'settings', 'predict']
```

### Store Changes (navigationSlice)

Add to `NavigationSlice` interface:

```typescript
currentScene: string | null  // 'terminal' | 'explore' | 'agents' | 'system' | null
setCurrentScene: (s: string | null) => void
```

The `resetNavigation` method clears `currentScene` back to `null`. No `sceneViews` map or per-scene memory is needed in W2 — scenes simply open to the view indicated by `?view`.

## Scene Components

### SceneSelector.tsx (NEW)

A central dispatch component rendered in App.tsx that replaces the conditional slide-over and dashboard rendering:

```typescript
function SceneSelector() {
  const scene = useStore(s => s.currentScene)
  switch (scene) {
    case 'terminal': return <TerminalScene />
    case 'explore':  return <ExploreScene />
    case 'agents':   return <AgentsScene />
    case 'system':   return <SystemScene />
    default:         return null  // Copilot mode — no scene dispatch
  }
}
```

SceneSelector is rendered inside `<main>` in App.tsx. The slide-over panel renders independently (controlled by `slideOverContent` state, same as today).

### TerminalScene.tsx (NEW)

Renders the fullscreen terminal. When `?view=dashboard` is set (or inferred from old state), renders `DashboardView` as a fullscreen overlay matching current App.tsx behavior. Otherwise renders `TerminalView`. No SlideOver.

### ExploreScene.tsx (NEW)

Reads `?view` from URL and dispatches to the correct view by setting `slideOverContent`. Default view: `'explore'`. Valid views: `explore`, `library`, `ontology`, `data`. Each view is lazy-loaded by the existing `SlideOverContent` component — the scene does not import views directly.

```typescript
function ExploreScene() {
  const view = useQueryState('view')[0] ?? 'explore'
  useEffect(() => {
    if (EXPLORE_VIEWS.includes(view)) {
      useStore.getState().setSlideOverContent({ type: view, title: VIEW_LABELS[view] ?? view })
    }
  }, [view])
  return null
}
```

### AgentsScene.tsx (NEW)

Same pattern as ExploreScene. Default view: `'agent'`. Valid views: `agent`, `skill`, `tool`, `component`.

### SystemScene.tsx (NEW)

Same pattern as ExploreScene. Default view: `'health'`. Valid views: `health`, `settings`, `predict`.

## SHOW_INLINE Rerouting (useAppActions.ts)

Per Oracle review finding: the `SHOW_INLINE` action in `handleCommandResult` currently routes 9 of 13 targets through `setInlineContent` (creating an inline panel). The current code has a partial fix via the `shouldUseSlideOver` allowlist, but it only covers `['explore', 'map', 'timeline', 'graph', 'explorer']`.

**W2 change:** All `SHOW_INLINE` targets route through `setSlideOverContent`. No target creates an inline panel:

```typescript
case 'SHOW_INLINE':
  const targetScene = VIEW_TO_SCENE[result.target as string] ?? null
  if (targetScene) store.setCurrentScene(targetScene)
  store.setSlideOverContent({
    type: result.target as SlideOverContent['type'],
    title: result.target ?? 'View',
    data: result.args ? { text: result.args } : undefined,
  })
  return true
```

The `setCurrentView('inline')`, `setShowInlinePanel(true)`, and `setInlineContent(...)` calls become dead code and are removed. The `InlineRenderer` component is deprecated (marked with a header comment) but not deleted in W2 — full elimination occurs in W3.

## Migration Path

### No Feature Flags

The old 13-item sidebar and new 5-item sidebar coexist during the transition. All old `?view` values continue to work because:

- `?scene` is inferred from `?view` via `VIEW_TO_SCENE` when absent
- Scene components dispatch to the same SlideOver views as the old system
- No view types are removed — only the sidebar UI changes
- Slash commands map through `VIEW_TO_SCENE` and set scenes automatically

### Phase Plan

| Phase | When | Action |
|-------|------|--------|
| 1 | W2 start | Add scene infrastructure (`?scene`, `currentScene`, sceneMapping.ts) with no visible change |
| 2 | W2 mid | Reroute SHOW_INLINE → SlideOver; verify all slash commands still work |
| 3 | W2 merge | Replace sidebar items; SceneSelector goes live |
| 4 | W3 | Delete InlineRenderer; remove inlineContent/showInlinePanel/currentView from store |

### Backward Compatibility Test Cases

- `?view=health` (no scene) → scene inferred as `system`, DataHealthView renders correctly
- `?slide=dashboard` (old bookmark) → mapped to `scene='terminal'`, slideover cleared, fullscreen renders
- `/skills` slash command → sets scene `agents`, opens SkillsView in SlideOver
- Sidebar click on old 13 items → derives scene, sets `?view`, works as expected

## App.tsx Changes

The main rendering block changes from:

```
Before: <TerminalView /> always → {slideOverContent?.type === 'dashboard' && <DashboardView />} → SlideOverPanel

After:  <SceneSelector /> → SlideOverPanel (guarded by currentScene !== 'terminal')
```

`TerminalView` no longer renders unconditionally — `SceneSelector` dispatches to `TerminalScene` only when `currentScene === 'terminal'`. The `DashboardView` lazy import remains alive during transition (used by TerminalScene).

## Edge Cases

- **Invalid `?scene` value**: Treated as `null` (copilot mode). NavigationStateSync clamps via the union fallback in `parse`.
- **Double scene set**: Sidebar click sets `currentScene` → store subscription writes to `?scene` → no loop because the comparison guard (`if state.currentScene !== scene`) breaks the cycle.
- **Direct URL entry with `?scene=explore&view=ontology`**: NavigationStateSync sets both. SceneSelector renders ExploreScene, which reads `?view` and dispatches OntologyView.
- **`?scene` absent, `?view` present**: Derived via `VIEW_TO_SCENE` — old bookmarks work.
- **Race condition**: Store → URL sync uses `useStore.subscribe` comparison guard. URL → store sync happens once on mount in the init effect. They cannot conflict.

## Acceptance Criteria

1. Sidebar shows exactly 5 items: Terminal, Explore, Agents, System, Copilot
2. Clicking each item activates the correct scene and sets `?scene` in URL
3. Old URLs with only `?view` correctly infer the scene
4. All SHOW_INLINE slash commands open SlideOver (not inline)
5. Copilot click clears all panels and resets to chat
6. `npx tsc --noEmit` passes with no new errors
7. `npx vite build` succeeds
8. All existing vitest tests pass (no regressions from sidebar or action changes)


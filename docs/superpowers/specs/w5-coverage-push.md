# W5 — Coverage Push 62.76% → 75%

## Goal
Push frontend statement coverage from 1549/2468 (62.76%) to 1851/2468 (75%) by writing ~15-20 new test files targeting worst-offender files.

## Current baseline
- 1549/2468 statements covered
- +302 statements needed to reach 75%
- tsc 0 err, vitest 714 pass, coverage excludes `_pb.ts` files

## Target files (worst offenders by statements %)

### Priority group — <20% coverage
| File | Stmts % | Stmts uncovered | Est. gain |
|------|---------|-----------------|-----------|
| ToolResultDisplay.tsx | 5.88% | ~51 | +45 |
| CommandRegistry.ts | 16.12% | ~26 | +22 |
| useSSE.ts | 17.02% | ~39 | +32 |
| workspaceSlice.ts | 18.75% | ~26 | +22 |

### Priority group — <40% coverage
| File | Stmts % | Stmts uncovered | Est. gain |
|------|---------|-----------------|-----------|
| SkillForm.tsx | 20% | ~20 | +16 |
| AgentForm.tsx | 25.92% | ~20 | +16 |
| useStore.ts | 30.76% | ~36 | +28 |
| SourceForm.tsx | 33.33% | ~26 | +20 |
| LibraryView.tsx | 38.88% | ~30 | +24 |
| uiSlice.ts | 40.9% | ~26 | +20 |

### Priority group — <50% coverage
| File | Stmts % | Stmts uncovered | Est. gain |
|------|---------|-----------------|-----------|
| OracleView.tsx | 41.3% | ~90 | +55 |
| AgentSlideOver.tsx | 35.48% | ~20 | +15 |
| DataSourceSlideOver.tsx | 47.16% | ~28 | +18 |
| App.tsx | 44.44% | ~20 | +15 |

### Total estimated gain
~368 statements — exceeds the +302 target. Safety margin: ~20%.

## Groups (independent, parallel)

### Group 1: ToolResultDisplay + CommandRegistry
- `ToolResultDisplay.tsx` (components/tools/): renders tool execution results (data/error/loading states)
- `CommandRegistry.ts` (commands/): command registration/lookup with keyboard shortcuts

### Group 2: useSSE + workspaceSlice
- `useSSE.ts` (hooks/): SSE connection management, reconnection, event parsing
- `workspaceSlice.ts` (store/): workspace state (files, active document, tabs)

### Group 3: SkillForm + AgentForm + SourceForm
- `SkillForm.tsx` (components/): skill creation/editing form
- `AgentForm.tsx` (components/): agent creation/editing form  
- `SourceForm.tsx` (components/): data source creation form

### Group 4: useStore + uiSlice + App.tsx
- `useStore.ts` (store/): combined store entry point
- `uiSlice.ts` (store/): UI state (sidebar, panels, expanded sections)
- `App.tsx` (components/): root application component

### Group 5: LibraryView + OracleView + SlideOver forms
- `LibraryView.tsx` (components/): asset library grid
- `OracleView.tsx` (components/): oracle predictions/sentiment/advanced
- `AgentSlideOver.tsx` (components/): agent detail slideover
- `DataSourceSlideOver.tsx` (components/): data source detail slideover

## Mock templates
Each test file follows established patterns:
- Store: `vi.mock('../../store/useStore', ...)` with `expandedSections`, `toggleSection`
- i18n: `vi.mock('../../i18n', ...)` returning `{ t: (k) => k }`
- Icons: `vi.mock('lucide-react', ...)` for chevron, bot, settings icons
- Proto types: direct import from generated types
- React.lazy: `vi.importActual` or waitFor for async components
- SSE/events: `vi.fn()` + `fireEvent` patterns from existing tests

## Verification
- `npx tsc --noEmit: 0 errors`
- `npx vitest run --reporter=verbose: all pass`
- `npx vite build: pass`
- `npx vitest run --coverage: ≥75% statements`

# SPEC-07: Frontend Hardening — State Security, Accessibility, Quality

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 3, tasks W3-4 through W3-11  
**Findings addressed**: F1-F10 (frontend cluster), L2-L4 (data leakage)  
**Depends on**: `docs/specs/wave0-secrets-spec.md` (apiKey removal, Zustand cleanup)  
**Related specs**: `docs/specs/wave3-api-spec.md` (CSP policy for frontend assets)  
**Status**: ✅ Approved — ready for execution

---

## 1. Zustand Store Contract

### Store Exposure — Dev-Only Gating

```typescript
// frontend/src/store/useStore.ts
// BEFORE: unconditional exposure
if (typeof window !== 'undefined') {
    window.__ALEPH_STORE__ = useStore;
}

// AFTER: dev-only + opt-in feature flag
if (import.meta.env.DEV && import.meta.env.VITE_ALEPH_DEV_TOOLS === 'true') {
    (window as any).__ALEPH_STORE__ = useStore;
}
```

```typescript
// frontend/src/App.tsx
// BEFORE: duplicate exposure
(window as any).__ALEPH_STORE__ = useStore;

// AFTER: same dev-only gate
```

### apiKey Removal

See SPEC-02 (wave0-secrets-spec.md) Section 3 for full specification.

### Store Slice Contracts (unchanged — already decomposed)

```
useStore
├── authSlice       — projectID, projects, apiKeys (NO apiKey)
├── navigationSlice — currentView, inlineContent, slideOverContent
├── copilotSlice    — chat[], input, isStreaming, pendingConfirmation
├── workspaceSlice  — data, agents, skills, tools, scenarios
├── healthSlice     — ollamaHealthy, nlpHealthy, lastError
└── uiSlice         — showOnboarding, toastMessages, confirmDialog, pendingCrud
```

---

## 2. AbortController Gaps

### Missing in App.tsx

```typescript
// frontend/src/App.tsx
useEffect(() => {
    const abortController = new AbortController();
    
    const loadData = async () => {
        try {
            await Promise.all([
                listProjects({ signal: abortController.signal }),
                executeQuery({ signal: abortController.signal }),
                getDataStats({ signal: abortController.signal }),
                getChatHistory({ signal: abortController.signal }),
            ]);
        } catch (err) {
            if (err.name === 'AbortError') {
                return; // Expected on unmount
            }
            reportError('App.dataLoad', err);
        }
    };
    
    loadData();
    
    return () => {
        abortController.abort();
    };
}, []); // Or with appropriate dependencies
```

### Existing AbortControllers (verify working)

- `useAppActions.ts` `loadAbortRef` → `loadProjectData` ✅
- `useAppActions.ts` `streamAbortController` → `sendMessage` ✅
- `useSSE.ts` → SSE connection ✅
- `useDataSourceActions.ts` → DataSource operations ✅
- `OracleView.tsx` → Oracle query ✅

---

## 3. Chat Streaming Memory Fix

### Current (O(n²) memory)

```typescript
// frontend/src/hooks/useAppActions.ts:205-207
// Each token: create new array copy of entire chat history
store.setChat([...chat]);  // ⚠️ O(n) per token → O(n²) total
```

### Fix: immer-based immutable update

```typescript
import { produce } from "immer";

// In the streaming handler:
onToken: (token: string) => {
    store.setState(produce((state: AppState) => {
        const lastMessage = state.chat[state.chat.length - 1];
        if (lastMessage && lastMessage.role === 'assistant') {
            lastMessage.content += token;
        } else {
            state.chat.push({
                id: generateId(),
                role: 'assistant',
                content: token,
                timestamp: Date.now(),
            });
        }
    }));
}
```

### Or: Zustand middleware for immer

```typescript
import { immer } from "zustand/middleware/immer";

export const useStore = create<AppState>()(
    immer((...a) => ({
        ...createAuthSlice(...a),
        ...createNavigationSlice(...a),
        ...createCopilotSlice(...a),
        ...createWorkspaceSlice(...a),
        ...createHealthSlice(...a),
        ...createUiSlice(...a),
    }))
);
```

---

## 4. useStore.getState() Audit Plan

### 136 call sites across 29 files

**Phase 1: Render paths (most critical)**
- Replace `useStore.getState().field` in component render bodies with `useStore(s => s.field)`
- Priority: Component files (not hooks)

**Phase 2: Hook callbacks (acceptable)**
- Keep `useStore.getState()` in event handlers and callbacks (not in render)
- Pattern: `const field = useStore(s => s.field)` at top level, then use `field` in callbacks

**Phase 3: add React.memo + useCallback**
- Wrap pure presentational components in `React.memo`
- Wrap event handlers in `useCallback`
- Target: top 10 most-rendered components

### File Priority

| Priority | Files | Reason |
|----------|-------|--------|
| **P0** | `App.tsx`, `SlideOverContent.tsx`, `SlideOverPanel.tsx` | Render every frame |
| **P1** | `Sidebar.tsx`, `CopilotView.tsx`, `DashboardView.tsx` | High frequency renders |
| **P2** | All view components (AgentsView, SkillsView, ToolsView, etc.) | Medium frequency |
| **P3** | Hook files (useAppActions, useAgentActions, useToolActions) | Acceptable — in callbacks, not render |

---

## 5. Form Accessibility

### Contract for ALL Forms

```typescript
// EVERY form SlideOver must follow this pattern:

export const MyFormSlideOver: React.FC = () => {
    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        // ... validation and submission ...
    };

    return (
        <SlideOverPanel title="My Form">
            <form onSubmit={handleSubmit} noValidate>
                {/* Inputs */}
                <div>
                    <label htmlFor="field-name">Field Name</label>
                    <input
                        id="field-name"
                        name="name"
                        type="text"
                        required
                        minLength={3}
                        maxLength={100}
                        aria-describedby="field-name-error"
                    />
                    <span id="field-name-error" role="alert" className="text-red-500">
                        {/* Validation error */}
                    </span>
                </div>
                
                {/* Submit */}
                <button type="submit">Save</button>
                <button type="button" onClick={onClose}>Cancel</button>
            </form>
        </SlideOverPanel>
    );
};
```

### Forms to Fix

| File | Current | Fix |
|------|---------|-----|
| `AgentFormSlideOver.tsx` | `onClick={handleSubmit}` | `<form onSubmit={handleSubmit}>` + `<button type="submit">` |
| `ToolFormSlideOver.tsx` | `onClick={handleSubmit}` | Same |
| `SkillFormSlideOver.tsx` | `onClick={handleSubmit}` | Same |
| `DataSourceFormSlideOver.tsx` | `onClick` per step | Same + step navigation as `<button type="button">` |
| `ComponentFormSlideOver.tsx` | `onClick={handleSubmit}` | Same |

---

## 6. SlideOverPanel Accessibility

### Required Changes

```typescript
// SlideOverPanel.tsx — add:

// 1. Backdrop click to close
<div 
    className="fixed inset-0 bg-black/50 z-40"
    onClick={onClose}
    aria-hidden="true"
/>

// 2. Inert background (modern alternative to aria-hidden on siblings)
useEffect(() => {
    if (isOpen) {
        const main = document.getElementById('main-content');
        if (main) main.setAttribute('inert', '');
        
        return () => {
            if (main) main.removeAttribute('inert');
        };
    }
}, [isOpen]);

// 3. aria-describedby
<div 
    role="dialog"
    aria-modal="true"
    aria-label={title || 'Slide over panel'}
    aria-describedby="slideover-description"
>
    <div id="slideover-description" className="sr-only">
        {description || 'Dialog panel for ' + (title || 'content')}
    </div>
</div>

// 4. aria-live announcement
<div aria-live="polite" className="sr-only">
    {isOpen ? `${title} dialog opened` : `${title} dialog closed`}
</div>

// 5. Replace emoji icon with SVG
<button
    onClick={toggleFullscreen}
    aria-label={isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen'}
>
    <MaximizeIcon className="w-5 h-5" />
</button>
```

---

## 7. Error Reporter Service

```typescript
// frontend/src/lib/errorReporter.ts
import { useStore } from '@/store/useStore';

type ErrorSeverity = 'error' | 'warning' | 'info';

interface ErrorContext {
    component: string;
    action: string;
    severity?: ErrorSeverity;
    metadata?: Record<string, unknown>;
}

export function reportError(ctx: ErrorContext, error: unknown) {
    // Development: console + full stack trace
    if (import.meta.env.DEV) {
        console.error(`[${ctx.component}/${ctx.action}]`, error);
    }
    
    // Production: toast only (no console leak)
    const message = error instanceof Error ? error.message : String(error);
    
    useStore.getState().addToast({
        type: ctx.severity || 'error',
        title: ctx.component,
        message: message,
        duration: ctx.severity === 'info' ? 3000 : 8000,
    });
    
    // Future: send to error tracking service (Sentry, etc.)
    // if (import.meta.env.PROD) {
    //     Sentry.captureException(error, { tags: ctx });
    // }
}
```

### Migration: All console.error Sites

| File | Line | Replace With |
|------|------|-------------|
| `ToolIntelligenceView.tsx` | 34 | `reportError({component:'ToolIntelligence', action:'fetch'}, e)` |
| `AlephErrorBoundary.tsx` | 22 | Keep `console.error` but gate on `import.meta.env.DEV` |
| `ToolConfigPanel.tsx` | 18 | `reportError({component:'ToolConfigPanel', action:'copy'}, err)` |
| `OracleView.tsx` | 72 | `reportError({component:'OracleView', action:'query'}, err)` |
| `DashboardView.tsx` | 60 | `reportError({component:'DashboardView', action:'fetch'}, e)` |
| `InlineErrorBoundary.tsx` | 21 | Keep `console.error` but gate on `import.meta.env.DEV` |
| `useCursorPagination.ts` | 38 | `reportError({component:'CursorPagination', action:'fetch'}, error)` |

---

## 8. Vitest Fixes

### Root Cause

All 9 failures: mock `useStore.getState()` returns `{}` (empty object), but hooks access `.agents`, `.pendingCrud`, `.setPendingCrud`, etc.

### Fix Pattern

```typescript
// BEFORE (broken)
vi.mock('@/store/useStore', () => ({
    useStore: {
        getState: vi.fn(() => ({})),  // ⚠️ Empty object
    },
}));

// AFTER (fixed)
import { createAppState } from '@/store/useStore';

const baseState = {
    agents: [],
    tools: [],
    skills: [],
    pendingCrud: false,
    setPendingCrud: vi.fn(),
    // ... all fields accessed by hooks under test
};

vi.mock('@/store/useStore', () => ({
    useStore: {
        getState: vi.fn(() => baseState),
    },
}));
```

### Files to Fix

1. `useAgentActions.test.ts` — Add `agents`, `pendingCrud`, `setPendingCrud`
2. `useToolActions.test.ts` — Add `tools`, `pendingCrud`, `setPendingCrud`
3. Any other test files using `useStore.getState()`

---

## 9. Verification

### Test Coverage

- [ ] `authSlice.test.ts`: apiKey removed; no __ALEPH_STORE__ in prod build
- [ ] `useAgentActions.test.ts`: All 3 previously-failing tests pass
- [ ] `useToolActions.test.ts`: All 2 previously-failing tests pass
- [ ] `App.test.tsx`: AbortController cleanup on unmount

### Manual Verification

```bash
# No console.error in production build code (except dev-gated)
grep -rn "console.error" frontend/src/components/ frontend/src/hooks/ \
  | grep -v "_test." | grep -v "AlephErrorBoundary" | grep -v "InlineErrorBoundary"
# → 0 matches (all migrated to reportError)

# No window.__ALEPH_STORE__ in production
# Build and check:
grep -rn "__ALEPH_STORE__" frontend/dist/
# → 0 matches in production assets

# All forms use <form onSubmit>
grep -rn "<form" frontend/src/components/forms/
# → All 5 form files have <form> with onSubmit

# All tests pass
npx vitest run
# → 216+ tests, 0 failures
```

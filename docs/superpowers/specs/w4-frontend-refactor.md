# W4: Frontend Refactor — Specifica

## Stato Attuale

Branch: `remediation/w1-w2-type-safety-panic`
Baseline: tsc 0 err, vitest 714/714 pass, vite build 778ms, go build clean
Coverage: 62.76%

---

## W4-01: Consolidate handleError

### Problema

Due funzioni `handleError` in `frontend/src/hooks/useAppActions.ts` con logica duplicata:

1. **Linea 44 — Module-level export**: `export const handleError = (err, context) => { ... }` — usata da 8 domain hooks (useComponentActions, useLibraryActions, useOntologyActions, useSettingsActions, useAgentActions, useSkillActions, useDataSourceActions, useToolActions)
2. **Linea 59 — useCallback interna**: `const handleError = useCallback((err, context) => { ... }, [])` — usata dentro useAppActions()

**Differenza critica:** La versione 2 gestisce `errorTimerRef` (clearTimeout prima di setTimeout), prevenendo accumulo di timer. La versione 1 no.

### Fix

1. Spostare timer management a livello di modulo (variabile `errorTimer` fuori dal componente)
2. Unificare in una sola funzione esportata:

```ts
// Module-level timer (sostituisce useRef)
let errorTimer: ReturnType<typeof setTimeout> | null = null

export const handleError = (err: unknown, context: string) => {
  const store = useStore.getState()
  const msg = err instanceof Error ? err.message : `Errore in ${context}`
  store.setLastError(msg)
  store.addToast({ message: msg, type: 'error', context })
  if (errorTimer) clearTimeout(errorTimer)
  errorTimer = setTimeout(() => useStore.getState().setLastError(null), 5000)
}
```

3. Rimuovere la `const handleError = useCallback(...)` interna a `useAppActions()`
4. Sostituire chiamate interne a `handleError(err, '...')` — ora usano quella esportata

**Attenzione:** La `handleError` esportata viene chiamata anche fuori da React component context (dentro catene `.catch()` nei domain hooks). Il module-level timer è perfetto per questo scenario — non serve useRef.

### Files modificati
- `frontend/src/hooks/useAppActions.ts`

### Verifica
- `npx tsc --noEmit` 0 errori
- `npx vitest run` all pass
- Nessun cambiamento funzionale

---

## W4-02: Rimuovere `as unknown as ToolsViewProps['onExecuteTool']`

### Problema

`frontend/src/components/terminal/SlideOverContent.tsx` linea 183:
```tsx
onExecuteTool={onExecuteTool as unknown as ToolsViewProps['onExecuteTool']}
```

`useToolActions` definisce:
```ts
onExecuteTool: useCallback((id: string) => { ... })
```

`ToolsViewProps` definisce:
```ts
onExecuteTool: (id: string) => void;
```

I tipi sono compatibili — `(id: string) => void` accetta sia callback sincroni che asincroni (TypeScript converte `void` return). Il cast `as unknown` è ridondante.

### Fix
Rimuovere il cast: `onExecuteTool={onExecuteTool}`
**DOPO** aver verificato che tsc 0 err senza cast.

Test: se tsc lamenta incompatibilità, aggiungere tipo esplicito a `useToolActions` return:
```ts
onExecuteTool: (id: string) => void = useCallback((id: string) => { ... })
```

### Files modificati
- `frontend/src/components/terminal/SlideOverContent.tsx`

### Verifica
- `npx tsc --noEmit` 0 errori
- `npx vitest run` all pass

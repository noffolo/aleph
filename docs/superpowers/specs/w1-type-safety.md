# W1: Type Safety — Eliminate `any` in Production Code

> **Branch:** `remediation/w1-w2-type-safety-panic`
> **Target:** 55 `any` occurrences in 20 production files → 0
> **Stima:** 1-2gg

## Regole Generali

1. **NON toccare test files** — i 197 `as any` nei test sono intenzionali (mock/partial)
2. **NON toccare `_pb.ts`** — auto-generati, esclusi da coverage
3. **NON toccare `node_modules`, vendor, `.d.ts`**
4. **Usare tipi esistenti** dal progetto — non creare nuove interfacce generiche dove già esistono quelle giuste
5. **Ogni commit deve mantenere:** `npx tsc --noEmit` 0 errors, vitest all pass, vite build ok

## Task W1-01: `any` in `hooks/useAppActions.ts` (17 occorrenze)

**File:** `frontend/src/hooks/useAppActions.ts`

**Occorrenze:**
| Line | Pattern | Fix |
|------|---------|-----|
| 29 | `handleError = (err: any, context: string)` | `err: unknown` |
| 44 | `handleError = useCallback((err: any, context: string)` | `err: unknown` |
| 61,70,74,78,82,86,90,106,110,114 | `then((res: any) => { ... })` | Usare response type da ConnectRPC (es. `listProjectsResponse`, `listAgentsResponse`, ecc.) |
| 200 | `produce((state: any) => { ... })` | Usare `AppState` type dallo store |
| 212,233,250,265 | `catch (err: any) { ... }` | `err: unknown` + controllare `instanceof Error` prima di usare `.message` |

**Pattern di fix per risposte ConnectRPC:**
```typescript
// PRIMA
agentClient.listAgents({ projectId: projectID }, opts).then((res: any) => {
// DOPO (usa il type generato da protobuf)
import { ListAgentsResponse } from '../api/proto/aleph/v1/agent_pb';
agentClient.listAgents({ projectId: projectID }, opts).then((res: ListAgentsResponse) => {
```

**Pattern per catch:**
```typescript
// PRIMA
catch (err: any) { setError(err.message); }
// DOPO
catch (err: unknown) {
  const msg = err instanceof Error ? err.message : String(err);
  setError(msg);
}
```

## Task W1-02: `any` in `lib/AlephGraph.tsx` (13 occorrenze)

**File:** `frontend/src/lib/AlephGraph.tsx`

**Occorrenze d3 force graph:**
| Line | Pattern | Fix |
|------|---------|-----|
| 24 | `const nodes: any[] = ...` | `const nodes: d3.SimulationNodeDatum[]` |
| 25 | `const links: any[] = []` | `const links: d3.SimulationLinkDatum<d3.SimulationNodeDatum>[]` |
| 54 | `d3.forceLink(links).id((d: any) => d.id)` | `(d: d3.SimulationNodeDatum & {id: number}) => d.id` |
| 71 | `(event: any, d: any) =>` | `(event: d3.D3DragEvent<any, any, any>, d: d3.SimulationNodeDatum) =>` |
| 75-76 | `(event: any, d: any) => {...}` | Same pattern |
| 80 | `(event: any, d: any) => onRowClick?.(d.data)` | D3 click type |
| 92 | `(d: any) => d.label.length > 15` | d3 text type |
| 102-107 | `(d: any) => d.source.x` etc | d3 link type |

**Import d3 types già presenti** — aggiungere solo i tipi mancanti nelle callback.

## Task W1-03: `any` in `App.tsx` (3 occorrenze)

**File:** `frontend/src/App.tsx`

| Line | Pattern | Fix |
|------|---------|-----|
| 47 | `(res: { projects: any[] })` | `ListProjectsResponse['projects']` |
| 82 | `(res: { messages?: any[] })` | `GetChatHistoryResponse['messages']` |
| 130 | `(res: { projects: any[] })` | Same as line 47 |
| 148 | `projects.find((x: any) => x.id === id)` | `(x: Project) => x.id === id` |

## Task W1-04: `any` in components (~22 occorrenze, 12 file)

**Files da sistemare:**

1. **`ToolForm.tsx:68`** — `catch (e: any)` → `catch (e: unknown)`
2. **`ToolIntelligenceView.tsx:8`** — `icon: any` → `icon: React.ComponentType<{size?: number}>`
3. **`SkillExecuteSlideOver.tsx:30`** — `(t: any) => t.id === tid` → `(t: Tool) => t.id === tid`
4. **`DataSourceFormSlideOver.tsx:26`** — `value: any` → tipizzare col type del config
5. **`CommandPalette.tsx:10`** — `projects: any[]` → `projects: Project[]`
6. **`OracleView.tsx:77,96,114,172`** — `err: any`, `err: any`, `err: any`, `pred: any` → `err: unknown`, `pred: Prediction`
7. **`DetailPanel.tsx:6`** — `selectedRow: any` → tipizzare dal contesto d'uso
8. **`ToolResultDisplay.tsx:11`** — `parsed: any` → tipizzare da `ToolResult`
9. **`ExplorerView.tsx:18-19`** — `data: any`, `row: any` → tipi dal contesto
10. **`SkillForm.tsx:63`** — `catch (e: any)` → `catch (e: unknown)`
11. **`DataPanel.tsx:4`** — `data: any` → tipizzare
12. **`SlideOverContent.tsx:222`** — `(a: any) => a.id === selectedAssetId` → `(a: Asset) => a.id`
13. **`api/utils.ts:3`** — `assertType<T extends Message>(data: any)` → lasciare (type guard, usa generic intenzionalmente)

**Pattern per catch blocks:**
```typescript
catch (err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  handleError(err, 'context');
}
```

## Verifica W1

```bash
# Nessun any residuo in produzione
grep -rn ': any\|as any' frontend/src/ --include='*.ts' --include='*.tsx' | grep -v '.test.' | grep -v '__tests__' | grep -v '_pb.ts' | grep -v 'node_modules'
# Output: 0 linee

npx tsc --noEmit
# Output: 0 errors

npx vitest run --reporter=verbose 2>&1 | tail -5
# Output: Tests all pass

npx vite build 2>&1 | tail -3
# Output: Build ok
```

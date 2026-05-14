# Aleph-v2 Remediation Plan — Verified (3 Reviewers + GitNexus + Graphify)

> **Stato:** 12 Maggio 2026
> **Baseline:** tsc 0 err ✅, vitest 76 file / 714 test pass ✅, vite build 778ms ✅, go build clean ✅
> **Coverage:** 62.76% statements, 54.12% branches, 63.91% functions (dopo esclusione `_pb.ts`)
> **Verifica:** 3 reviewer indipendenti (Momus, Oracle, Metis) + 2 code intelligence tools (GitNexus, Graphify)

---

## Riepilogo Findings

### Frontend (57 `any` in produzione + 197 `any` in test)

| Categoria | Gravità | Quantità |
|-----------|---------|----------|
| `any` type leak (production, ~20 file) | 🔴 CRITICA | 57 occorrenze |
| `any` type in test files (22 file) | ⚪ INTENZIONALE | 197 occorrenze (mock/partial — esclusi) |
| handleError duplicato | 🟡 MEDIA | 2 (App.tsx + useAppActions) |
| onExecuteTool signature conflitto | 🟡 MEDIA | 1 |

### Backend (8 context.TODO, 3 panic, 1 CB mancante)

| Categoria | Gravità | Quantità |
|-----------|---------|----------|
| context.TODO() in codice produzione | 🔴 CRITICA | 8 |
| panic() in production path | 🔴 CRITICA | 3 (1 bug reale, 2 init-time) |
| Circuit breaker non cablato in middleware chain | 🟠 ALTA | 1 |
| Tool packages senza integration test | 🟡 MEDIA | 4 package (finance, osint, humanecosystems, adaptation) |

### Coverage Gap (target 75% → attuale 62.76%)

| Categoria | Attuale | Target | Gap |
|-----------|---------|--------|-----|
| Statements | 62.76% | 75% | −12.24% |
| Branches | 54.12% | 60% | −5.88% |
| Functions | 63.91% | 75% | −11.09% |
| Lines | 62.25% | 75% | −12.75% |

---

## Wave 1: Type Safety 🔴 (1-2gg)

Eliminare i 57 `any` nel codice di produzione (~20 file). I 197 `any` nei test (mock con `as any`) sono intenzionali e vanno esclusi dallo scope.

### W1-01: `any` nei file store (5 file)
- **File**: `frontend/src/store/slices/*.ts` (authSlice, copilotSlice, workspaceSlice, inlineSlice, uiSlice)
- **Tipo**: `(action: any)` in zustand slice actions
- **Fix**: Sostituire con tipi esatti: `Agent | Partial<Agent>`, `Skill | Partial<Skill>`
- **Stima**: ~15 occorrenze, ~2h

### W1-02: `any` nei componenti (~12 file)
- **Top offenders**: `useAppActions.ts` (17), `AlephGraph.tsx` (13)
- **Tipo**: `props: any`, `data: any`, `result: any`
- **Fix**: Tipizzare con interfacce esistenti (Tool, Agent, Skill, ecc.)
- **Stima**: ~30 occorrenze, ~4h

### W1-03: `any` in API layer (3 file)
- **File**: `frontend/src/api/*.ts`
- **Tipo**: risposte ConnectRPC non tipizzate
- **Fix**: Usare `fromProto` mapper + response type inference
- **Stima**: ~12 occorrenze, ~2h

**Verifica W1:** `npx tsc --noEmit` 0 errors. I `any` nei test non sono bloccanti. Usare `grep -rn ': any\|as any' frontend/src/ --include='*.ts' --include='*.tsx' | grep -v '.test.' | grep -v '__tests__'` per verificare 0 residui in produzione.

---

## Wave 2: Panic & Context.TODO 🔴 (1gg)

### W2-01: panic() → error return (1 bug reale)
- `internal/mcp/ssrf/validator.go:179` — CIDR parse failure (🔴 REALE, da fixare)
- `internal/config/secrets.go:49` — required key missing (🟡 INIT-TIME, lasciare)
- `internal/storage/context.go:32` — DB schema assertion (🟡 INIT-TIME, lasciare)
- **Fix W2-01a**: validatore CIDR → `return fmt.Errorf(...)` invece di `panic()`
- **Stima**: ~30min

### W2-02: context.TODO() → context propagato (8 siti)
1. `internal/tools/registry.go:154` — tool registry timeout
2. `internal/health/checker.go:61` — health check
3. `internal/storage/duckdb_backup.go:39` — backup
4. `internal/storage/context.go:82` — schema DDL
5. `internal/storage/context.go:96` — schema DDL
6. `internal/storage/duckdb.go:128` — query exec
7. `internal/sandbox/namespace_isolated.go:38` — isolated exec
8. `internal/sandbox/container_sandbox.go:78` — container sandbox
- **Fix**: Aggiungere ctx param alla funzione chiamante, propagare dal chiamato
- **Stima**: ~3h

### ~~W2-03: ValidateAPIKey deprecato~~ — RIMOSSO
- **Motivo**: ValidateAPIKey è già stato migrato ad argon2id. Non richiede intervento.

**Verifica W2:** `go vet ./...` clean + `go build ./...` clean

---

## Wave 3: Circuit Breaker Wiring + Integration Tests 🟠 (1-2gg)

Verifica indipendente (GitNexus + Oracle): **ZERO stub handler** in `internal/api/handler/`. Tutti gli handler hanno implementazioni reali. Le ~46 funzioni in `finance/`, `osint/`, `humanecosystems/`, `adaptation/` hanno codice reale.

Wave re-scopata su due gap confermati:

### W3-01: Circuit breaker nella middleware chain
- **File**: `internal/app/app.go:186-198`
- **Situazione attuale**: middleware chain ha subsystem, errorHandler, audit, auth, authRateLimit, timeout, retry, bulkhead, tracking — MA circuit breaker mancante
- **Fix**: Cablare CircuitBreakerInterceptor nella middleware chain. Verificare i 2 CB esistenti (NLP, OSINT) siano corretti.
- **Stima**: ~3h

### W3-02: Tool package smoke tests
- **Package senza test**: `finance/`, `osint/`, `humanecosystems/`, `adaptation/` — ~46 file con codice reale, zero test
- **Fix**: Aggiungere smoke test (1-2 test per package) che verificano: (a) compilazione, (b) chiamata base non panica, (c) error handling path
- **Stima**: ~4h

**Verifica W3:** `go test -race -count=1 ./internal/tools/...` passa + `go build ./...` clean

---

## Wave 4: Frontend Refactor 🟠 (1-2gg)

### W4-01: handleError duplicato
- **File**: `App.tsx` e `hooks/useAppActions.ts` — stessa logica di error handling
- **Fix**: Estrarre in `hooks/useErrorHandler.ts` o simile. Consolidare in una funzione condivisa.
- **Stima**: ~1h

### W4-02: onExecuteTool signature conflitto
- **File**: `hooks/useAppActions.ts`
- **Problema**: firma diversa tra chiamante e dichiarazione — confermato da Momus e Oracle
- **Fix**: Allineare firma o creare adapter
- **Stima**: ~1h

### ~~W4-03: AssetDetailSlideOver stub~~ — RIMOSSO
- **Motivo**: AssetDetailSlideOver.tsx esiste ed è funzionale (58 linee, pulsanti, store interaction). Non è uno stub.

### ~~W4-04: loadProjectData stale closure~~ — RIMOSSO
- **Motivo**: Usa `useStore.getState()` che è sempre corrente. Falso allarme.

**Verifica W4:** `npx tsc --noEmit` 0 errors + vitest all pass

---

## Wave 5: Coverage Push → 75% 🟡 (4-6gg)

Strategia: 3 leve per coprire il gap di ~12 punti.

### Leva A — Esclusioni già fatte (+15.47 punti)
- **Fatty**: Esclusi `_pb.ts` (auto-generati, non testabili)
- **Risultato**: 47.29% → **62.76%** ✅

### Leva B — Test per i file worst-offender (~8 punti)
Individuare i file con coverage più basso:
```bash
npx vitest run --coverage 2>&1 | grep -E 'src/' | sort -t'%' -k5 -n | head -20
```
Target: portare ogni file sotto il 40% di coverage ad almeno il 60%.

### Leva C — Integration/E2E (~4 punti)
I test Playwright (56 già esistenti) contribuiscono indirettamente al coverage se eseguiti con strumentazione Istanbul. Configurazione già presente:
```bash
npx vitest run --coverage.include='src/**'
```

### Calcolo copertura:
```
Attuale: 62.76% (1551/2471 statements)
Target:  75%   (1853/2471)
Servono: ~302 statements aggiuntivi coperti
```

**Stima**: 4-6gg per raggiungere 75% (stima conservativa dato il volume di test da scrivere)

---

## Wave 6: CI/CD Hardening 🟢 (0.5gg)

### W6-01: Coverage threshold update
- **File**: `frontend/vitest.config.ts`
- **Threshold corrente**: statements=60, branches=50, functions=60, lines=60
- **Nuova soglia**: statements=65, branches=55, functions=65 (dopo W5)

### W6-02: Post-commit hook verification
- **Hook già creato**: `.git/hooks/post-commit`
- **Azione**: Verificare che l'hook funzioni correttamente con `git commit --allow-empty -m "test: verify post-commit hook"`

### W6-03: Docker immagine ottimizzata
- Verificare che `docker compose build` passi con nuovo coverage config
- **Stima**: ~30min

---

## Riepilogo Temporale

| Wave | Descrizione | Giorni | Dipende da |
|------|-------------|--------|------------|
| W1 | Type Safety (57 `any` produzione) | 1-2 | — |
| W2 | Panic + Context.TODO (8+1 reale) | 1 | — |
| W3 | Circuit Breaker + Tool Tests | 1-2 | W2 (ctx propagation) |
| W4 | Frontend Refactor (2 item) | 1-2 | — |
| W5 | Coverage Push 75% | 4-6 | W1 (type safety) |
| W6 | CI/CD Hardening | 0.5 | W5 (thresholds) |
| **Totale** | | **~7-14gg** | |

**Ordine consigliato:** W1+W2 in parallelo → W3 → W4+W5 in parallelo → W6

---

## Ship Gate Checklist

- [ ] W1: `tsc --noEmit` 0 err + 0 `any` residui in produzione (test esclusi)
- [ ] W2: `go vet ./...` clean + `go build ./...` clean
- [ ] W3: `go test -race -count=1 ./internal/tools/...` passa + circuit breaker cablato
- [ ] W4: `tsc --noEmit` 0 err + vitest 714+ pass + handleError consolidato
- [ ] W5: `vitest run --coverage` statements ≥ 75%
- [ ] W6: CI + Docker build passano + post-commit hook verificato

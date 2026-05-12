# Handoff — Aleph-v2 Full Audit Plan (11 Maggio 2026)

> ⛔ UX W1 (Store Refactor) cancellato dopo 45 min di blocco.
> **Riprendere da UX W1**, poi Phase 8 → Phase 5 → Phase 6 → Phase 9.

---

## Riepilogo Esecuzione

**Piano**: `docs/superpowers/plans/2026-05-11-aleph-full-audit-plan.md` (9 fasi, ~140 task)
**Stato**: Phases 0-4 + UX W0 + Phase 7 ✅ — UX W1 ⛔ — Remaining: Phase 8, Phase 5, Phase 6, Phase 9

## ✅ COMPLETATE

### Phase 0 (14 Probes) — 4 agenti paralleli
- **Go**: go.mod 1.26 vs CI 1.24 (HIGH mismatch). Tutti i 42 select blocco hanno ctx.Done(). Catena interceptor corretta. memory.go già protetto da ValidateIdentifier. query.go ha 8 fmt.Sprintf %s (MEDIUM — trust compiler compromise).
- **Frontend**: Solo 7/82 componenti testati (HIGH gap). 7 catch {} vuoti, tutti intenzionali. 1 `as any` in useStore.ts:75 (dead code in prod).
- **CI/Docker**: VITE_API_BASE_URL mai consumato. NLP pytest mai eseguito in CI (HIGH). gitleaks presente in security.yml. No Docker --mount=type=cache.
- **Cross**: 8 pacchetti Go non testati (7 protobuf = OK). 8 env var in codice mancanti da .env.example.

### Phase 1 (9 Architecture) — 3 agenti paralleli
- **Frontend**: SlideOverContent↔InlineRenderer 85% duplicato (28 as-unknown-as cast). 2 full-store subscriptions (OracleView, CopilotView). 6 raw fetch endpoint senza equivalente ConnectRPC.
- **Backend**: Tutti i 30 lock senza defer sono intenzionali. Nessuna race in MCP Discovery. memory.go usa raw `"%s"` invece di QuoteIdentifier() — MEDIUM defense-in-depth gap.
- **NLP**: 16 test Python senza servizi esterni. conftest.py ha fixture gRPC morta. 8 test duplicati.

### Phase 2 (Bug Verification — 31 test) — 3 agenti
- Go: 4 test PASS (SQL injection, DuckDB ctx cancel, memory.go guard, Discovery leak con -race).
- Frontend: 26 test PASS (catch-block-inventory, AlephErrorBoundary, useAppActions) + tsc clean.
- NLP: 1 test PASS (cmdstanpy presente).

### Phase 3 (Bug Fix) — 2 agenti
- **FE** (tsc+vite+vitest clean ✅):
  - 7 catch → console.error con contesto (navigationSlice 3, AlephErrorBoundary 1, DataSourceForm 2, SetupWizard 1)
  - `(window as any).__ALEPH_STORE__` → `window.__ALEPH_STORE__` (type-safe via Window augmentation)
- **Go** (go build+vet+test 46/46 ✅):
  - `internal/memory/memory.go`: tableName() ora usa `safeident.QuoteIdentifier(m.schema)` invece di raw `"%s"`
  - `internal/api/handler/query.go`: validateIdentifier aggiunto a 8 siti fmt.Sprintf %s (difesa-in-profondità)

### Phase 4 (Backend Tests) — 3 agenti
- `internal/concurrency/safego_test.go`: 10 test (panic recovery, ctx cancellation, 50 goroutine concorrenti) ✅
- `internal/repository/metadata_test.go`: 18 test paginazione cursore aggiunti (55 repo test totali) ✅
- `internal/sandbox/verification_test.go`: 26 test + 2 fuzz (VerifyToolCode, isOutputSafe, VerifyMultipleTools) ✅

### UX W0 (Foundation) — 5 agenti
- W0-01: `frontend/src/config/features.ts` (6 flag, default false, VITE_FEATURE_* env override)
- W0-02+03: `docs/store-inventory-audit.md` — 61→38-42 campi, 11 morti, 4 derivati, 3 full-store subscriptions
- W0-04: `docs/api-coverage-audit.md` — 79 endpoint (48 ConnectRPC + 31 HTTP), 93.75% coverage
- W0-05: Regression gate GREEN (go build ⏐ tsc ⏐ vite build ⏐ go test 44/44 ⏐ go vet)

### Phase 7 (Integration/Security) — 6 agenti, tutti ✅ VERIFICATI
- **7-3 SSRF**: `internal/mcp/ssrf_test.go` — test MCP passano ✅
- **7-5 Memory SQL**: `internal/memory/memory_test.go` — test SQL injection passano ✅
- **7-7 Race script**: `scripts/race-check.sh` (92 righe, chmod +x, bash -n clean) ✅
- **7-8 CORS**: `internal/middleware/cors_test.go` — test middleware passano ✅
- **7-9 DuckDB integ**: `internal/storage/duckdb_integration_test.go` — storage test (incl fuzz 18 seed) passano ✅
- **7-10 Protobuf valid**: `tests/integration/protobuf_validation_test.go` (build tag: integration) — integration tests passano ✅
- **Skip**: 7-1 (no Docker), 7-6 (API key E2E, serve backend running)

---

## ⛔ CANCELLATO: UX W1 (Store Refactor)

**Task**: 5 task sequenziali su modello deep. Bloccato dopo 45 min.
**Da rilanciare da capo**.

I task W1:
1. **W1-01**: Aggiungere selettori a chiamate `useStore()` senza selettore (≥3 full-store subscriptions in CopilotView, OracleView, TerminalEffects)
2. **W1-02**: Bugfix + cleanup: resetHealth() aggiungere ollamaHealthy:true, rimuovere 11 campi morti, rinominare splitView→showMessageDetail
3. **W1-03**: Unificare campi explorer da workspaceSlice/navigationSlice/uiSlice in explorerSlice
4. **W1-04**: Riduzione 61→38-42 campi, aggiornare TUTTI i consumer, verificare tsc --noEmit
5. **W1-05**: Rimuovere subscription duplicate (useAppStore vs useStore) — canonicalizzare

**Store audit**: `docs/store-inventory-audit.md`
**API audit**: `docs/api-coverage-audit.md`

---

## ⏳ RIMANENTI

### Phase 8: CI Integration & Coverage Gates
- 8-1: Fix Go version mismatch go.mod 1.26 vs CI 1.24 (install 1.26.2 in CI)
- 8-2: Fix VITE_API_BASE_URL build config (vite define{} invece di ARG/ENV)
- 8-3: Add NLP pytest to CI (3-step: pip install → pytest nlp/tests/)
- 8-4: Verify gitleaks + npm audit in CI
- 8-5: Add Docker --mount=type=cache optimization
- 8-6: Add Go benchmark step to CI
- 8-7: Add frontend coverage gate (min 20% → 50%) + 8 env vars to .env.example

### Phase 5: Frontend Unit Tests (dopo UX W1)
Da fare SOLO dopo che UX W1 ha completato il refactoring dello store.
- Test per componenti non testati (20+): AgentFormSlideOver, SkillFormSlideOver, ToolFormSlideOver, DataSourceFormSlideOver, ComponentFormSlideOver
- Hooks non testati (7): useComponentActions, useDataSourceActions, useExplorerActions, useLibraryActions, useOntologyActions, useSettingsActions, useSkillActions
- View non testata: ScenarioComparisonView

### Phase 6: Playwright E2E Tests (dopo UX W6)
Da fare SOLO dopo UX W6 (polish + a11y). 24+ nuovi spec.
- Task 6-0a: Aggiungere data-testid a 40+ componenti
- Coprire: DashboardView, ExplorerView, DataHealthView, ToolIntelligenceView, LibraryView, ComponentsView, ScenarioComparisonView, form views, SetupWizard, GuideTour

### Phase 9: Final Report
- Riepilogo metriche (nuovi test, coverage gaps colmati, bug fixati)

---

## Build State Attuale
- `go build ./...` ✅
- `go vet ./...` ✅
- `go test -count=1 ./...` ✅ (46/46 packages)
- `npx tsc --noEmit` ✅
- `npx vite build` ✅
- `scripts/race-check.sh` ✅ (nuovo)
- `npx vitest run` ✅ (nuovi test catch/AlephErrorBoundary/useAppActions)

## File Chiave Creati
| File | Scopo |
|------|-------|
| `frontend/src/config/features.ts` | Feature flags |
| `internal/concurrency/safego_test.go` | SafeGo panic/cancel test |
| `internal/repository/metadata_test.go` | Cursor pagination |
| `internal/sandbox/verification_test.go` | Tool verification |
| `internal/mcp/ssrf_test.go` | SSRF protection |
| `internal/memory/memory_test.go` | SQL injection guard |
| `internal/middleware/cors_test.go` | CORS headers |
| `internal/storage/duckdb_integration_test.go` | DuckDB round-trip |
| `tests/integration/protobuf_validation_test.go` | Protobuf validation |
| `scripts/race-check.sh` | Race detector script |
| `docs/store-inventory-audit.md` | Store inventory (UX W0) |
| `docs/api-coverage-audit.md` | API coverage (UX W0) |

# Post-Audit Resolution Plan — aleph-v2 (v3, post-Assemblea)

**Data**: 2026-04-25
**Fonte**: Assemblea 12 personas audit
**Risultato iniziale**: 3 ASSENSO, 9 VETO — piano risoluzione obbligatorio
**Momus Review**: REJECT v1 → Correzione 3 blocchi critici, 8 issue
**Assemblea Finale**: 7 APPROVE, 4 REJECT — integrati in v3
**Stato**: ✅ COMPLETATO — 32/32 item eseguiti e verificati (2026-04-25)

---

## Summary

| Priorità | Item | Veto da | Effort | File | Miglioria |
|----------|------|---------|--------|------|-----------|
| Priorità | Item | Veto da | Effort | File | Status |
|----------|------|---------|--------|------|--------|
| 🔴 P0 | Migration gap: tabelle `system_*` non migrate in duckdb/ | Go, API, Debug, DevOps, Aleph | M | `migrations/duckdb/` | ✅ |
| 🔴 P0 | 3 TypeScript errors (App.tsx:381/764/830) | React/TS, Aleph, Debug | S | `App.tsx` | ✅ |
| 🔴 P0 | `deploy.yml:134` YAML invalido | DevOps | S | `.github/workflows/deploy.yml` | ✅ |
| 🔴 P0 | Integration test 6/10 fail | Aleph, Go, Debug | M | `internal/api/handler/*_test.go` | ✅ |
| 🟠 P1 | `isMutatingOperation` gap (esclude "Send") | API, Go, Debug | S | `internal/middleware/audit.go:79` | ✅ |
| 🟠 P1 | `ProjectIDFromContext` senza garanzia | API, Go, Debug | S | `internal/middleware/audit.go:52` | ✅ |
| 🟠 P1 | `alert()` nativi 7x invece di Toast | UX/UI | S | Vari componenti | ✅ |
| 🟠 P1 | 3 form inline in App.tsx | UX/UI | M | `App.tsx: skill/tool/component cases` | ✅ |
| 🟠 P1 | `font-mono` su body | UX/UI | S | `index.css` | ✅ |
| 🟠 P1 | Label contrast WCAG AA fail (3.0:1) | UX/UI | S | `index.css` / `design-tokens.json` ternary token | ✅ |
| 🟠 P1 | `read_only:true` + `INSTALL vss` conflict | DevOps | S | `duckdb.go` / migration | ✅ |
| 🟠 P1 | `continue-on-error:true` in CI | DevOps | S | `.github/workflows/ci.yml:42` | ✅ |
| 🟡 P2 | Standardizzare errori REST | API | M | `internal/api/handler/*.go` | ✅ |
| 🟡 P2 | D3 zoom mancante AlephGraph | Data Viz | S | `frontend/src/lib/AlephGraph.tsx` | ✅ |
| 🟡 P2 | ToolIntelligenceView dati mock | Data Viz | M | `ToolIntelligenceView.tsx` | ✅ |
| 🟡 P2 | Leaflet clustering mancante | Data Viz | S | `ExplorerView.tsx` | ✅ |
| 🟡 P2 | Claim "autocoscienza" non supportato | Filosofo | S | `AGENTS.md`, doc | ✅ |
| 🟡 P2 | Bias recalibration mancante | Filosofo | M | `docs/development-bias-checklist.md` | ✅ |
| 🟡 P2 | HEALTHCHECK + USER mancanti Dockerfile | DevOps | S | `Dockerfile` | ✅ |
| 🟡 P2 | `predict_probs` duplicato in `nlp/ensemble.py` | Python | S | `nlp/ensemble.py` | ✅ |
| 🟡 P2 | `torch_geometric` non in `nlp/requirements.txt` | Python | S | `nlp/requirements.txt` | ✅ |
| 🟡 P2 | DataHealthView usa div CSS invece di charting | Data Viz | S | `DataHealthView.tsx` | ✅ |
| 🟡 P2 | Timeline CSS-only (no proper visualization) | Data Viz | S | Timeline componente | ✅ |
| 🟡 P2 | Responsive/mobile design mancante | UX/UI | M | Multipli view | ✅ |
| 🟡 P2 | Focus trap modali (↑ da P3) | UX/UI | S | `SlideOverPanel.tsx` | ✅ |
| 🟡 P2 | Bookmark mobile/touch support (↑ da P3) | UX/UI | S | `CopilotView.tsx` | ✅ |
| 🟢 P3 | Brandmark, onboarding più ispirato | Art Director | M | SetupWizard + brand | ✅ |
| 🟢 P3 | Swagger spec incompleta | API | L | `docs/swagger/` | ✅ |
| 🟢 P3 | Paginazione inconsistente | API | M | Multipli handler | ✅ |
| 🟢 P3 | Backup/restore DuckDB | DevOps | M | `internal/storage/` | ✅ |
| 🟢 P3 | nlp gitignore \_\_pycache\_\_/venv | Python | S | `.gitignore` | ✅ |

---

## P0 — Critical (bloccanti, risolvere immediatamente)

### 🔴 P0-01: Migration gap — tabelle `system_*` non migrate in duckdb/

**Veto da**: Fullstack Go Dev, Debug Engineer, API Engineer, DevOps, Aleph
**File**: `migrations/duckdb/` (dir), `internal/storage/duckdb.go` (RunDuckDBMigrations path)
**Problema**: Le migration DuckDB (000001-000003 in `migrations/duckdb/`) NON creano le tabelle `system_*` (system_tasks, system_simulations, system_proposals, system_chat_history, system_agents, system_skills, system_tools, system_api_keys, system_notification_channels, system_features — 10 tabelle). Il SQL esiste già in `migrations/000001_init_schema.up.sql` (root, linee 34-126) MA `RunDuckDBMigrations` legge da `migrations/duckdb/` che ha un 000001 diverso (solo `components` + `system_features`). Senza queste tabelle, integration test falliscono (6/10) e database fresh è inutilizzabile.
**Nota**: Root `000001_init_schema.up.sql` contiene `INSTALL vss;` — va gestito per non confliggere con P1-07.
**Risoluzione**:
1. Creare `migrations/duckdb/000004_system_tables.up.sql` con le 10 tabelle system_* (copiare da root migration, rimuovere INSTALL vss)
2. Aggiungere down.sql corrispondente
3. `RunDuckDBMigrations` lo eseguirà automaticamente (ordine numerico)
4. Opzionale: decidere se root o duckdb/ è il path authoritative
**Effort**: M (30-45 min)
**Dipendenze**: Nessuna
**QA**: `go test ./internal/api/handler/...` passa verde

### 🔴 P0-02: 3 TypeScript errors (App.tsx)

**Veto da**: Fullstack React/TS Dev, Debug Engineer, Aleph
**File**: `frontend/src/App.tsx` — righe 381, 764, 830
**Problemi**:
  - **Riga 381**: `onRegisterComponent` chiamato con oggetto senza `id` e `version` (richiesti da `RegistryComponent`)
  - **Riga 764**: La callback locale accetta un tipo `Skill` SENZA index signature `[key: string]: unknown`, ma la firma attesa usa il tipo store `Skill` CON index signature. Mismatch strutturale.
  - **Riga 830**: Callback accetta `RegistryComponent` ma la firma attesa è `Partial<ComponentMetadata>` (direzione opposta a quanto descritto in v1). `id` in RegistryComponent è `string`, in Partial<ComponentMetadata> è `string | undefined`.
**Risoluzione**:
  - Riga 381: Aggiungere `id` (uuid v4 o timestamp) e `version` (es. "1.0.0") all'oggetto passato a `onRegisterComponent`
  - Riga 764: Allineare i due tipi Skill. Opzioni: (a) rimuovere index signature dal tipo store, (b) aggiungere index signature al tipo locale, (c) convertire esplicitamente
  - Riga 830: Convertire `RegistryComponent` in `Partial<ComponentMetadata>` prima di passare, o allineare i tipi d'attesa
**Effort**: S (10-15 min)
**Dipendenze**: Nessuna
**QA**: `npx tsc --noEmit` passa con 0 errori

### 🔴 P0-03: `deploy.yml:134` YAML invalido

**Veto da**: DevOps
**File**: `.github/workflows/deploy.yml`
**Problema**: `retention-days:` (riga 134) seguito da caratteri non ASCII, YAML parsing fallisce.
**Risoluzione**: Sostituire con `retention-days: 30` (valore numerico valido).
**Effort**: S (2 min)
**Dipendenze**: Nessuna
**QA**: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy.yml'))"` passa

### 🔴 P0-04: Integration test 6/10 fail

**Veto da**: Aleph, Debug Engineer, Fullstack Go Dev
**File**: `internal/api/handler/*_test.go`
**Problema**: 6 su 10 integration test falliscono perché migration duckdb/ non include tabelle system_*.
**Risoluzione**: Agganciato a P0-01. Dopo migration 000004, i test dovrebbero passare. Se non basta, aggiungere `RunMigrations()` esplicito nel setup degli integration test.
**Effort**: M (30 min, accoppiato a P0-01)
**Dipendenze**: **P0-01**
**QA**: `go test ./internal/api/handler/... -v` 10/10 passano

---

## P1 — Major (funzionalità rotte o insicure)

### 🟠 P1-01: `isMutatingOperation` gap (esclude "Send")

**Veto da**: API Engineer, Fullstack Go Dev, Debug Engineer
**File**: `internal/middleware/audit.go:79-104`
**Problema**: `isMutatingOperation` non riconosce operazioni con prefisso "Send" (es. SendNotification). Ogni chiamata SendNotification (operazione mutante per definizione) bypassa audit logging.
**Risoluzione**: Aggiungere `case strings.Contains(procedure, "Send"): return true` alla switch.
**Effort**: S (5 min)
**Dipendenze**: Nessuna
**QA**: `go test ./internal/middleware/...` passa

### 🟠 P1-02: `ProjectIDFromContext` senza garanzia

**Veto da**: API Engineer, Fullstack Go Dev, Debug Engineer
**File**: `internal/middleware/audit.go:52`
**Problema**: `logAuditEvent` chiama `ProjectIDFromContext(ctx)` senza verificare che il valore nel contesto esista e sia del tipo corretto. Se il contesto non ha un projectID, la type assertion dentro `ProjectIDFromContext` può panicare. Inoltre, `logAuditEvent` usa `context.Background()` per l'insert (linea 72) perdendo il contesto originale (cancellazione, deadline).
**Risoluzione**:
1. Verificare `ProjectIDFromContext` abbia nil-safe type assertion (pattern `val, ok := ctx.Value(key).(string)`)
2. Opzionale: propagare il ctx originale invece di Background()
**Effort**: S (10 min)
**Dipendenze**: Nessuna
**QA**: `go test ./internal/middleware/...` passa, nessun panic su route anonime

### 🟠 P1-03: `alert()` nativi 7x invece di Toast

**Veto da**: UX/UI Designer
**File**: AgentForm.tsx, DataSourceForm.tsx, ToolManagementView.tsx, altri
**Problema**: Toast system già implementato (W5-05+12) ma 7 chiamate `alert()` native persistono in vari componenti.
**Risoluzione**: Sostituire ogni `alert(...)` con `useAppActions().addToast({ type: 'error', message: ... })` (o hook diretto). Seguire pattern stabilito.
**Effort**: S (10 min)
**Dipendenze**: Nessuna (Toast già deployato)
**QA**: `grep -r "alert(" frontend/src/ | grep -v "eslint\|comment\|// "` restituisce 0

### 🟠 P1-04: 3 form inline in App.tsx

**Veto da**: UX/UI Designer
**File**: `frontend/src/App.tsx` — casi SlideOver: skill-form, tool-form, component-form
**Problema**: Stesso pattern di agent-form (W5-01) e datasource-form (W5-02) MA non ancora estratti. 3 form inline in App.tsx (circa 80 righe ciascuno) rendono il file ingombrante (1162 righe) e violano la separazione delle responsabilità.
**Risoluzione**: Estrarre ciascuno in componente standalone: `SkillForm.tsx`, `ToolForm.tsx`, `ComponentForm.tsx`. Seguire pattern:
- CVA variant dal design system
- Validazione client-side (Zod schema da W5-09)
- Nessun alert()
- Stili design system tokens
**Effort**: M (45-60 min)
**Dipendenze**: Nessuna
**QA**: `npx tsc --noEmit` passa. Form funzionanti in SlideOver.

### 🟠 P1-05: `font-mono` su body

**Veto da**: UX/UI Designer
**File**: `frontend/src/index.css`
**Problema**: `font-family: var(--font-mono)` o equivalente applicato a body/html. Testo lungo illeggibile. `font-mono` deve essere limitato a contesti terminal/code.
**Risoluzione**: Spostare font-mono dal body a classi specifiche (`.terminal-output`, `.code-block`, etc.). Body/sans-serif generale con `var(--font-sans)` già definito in design token.
**Effort**: S (10 min)
**Dipendenze**: Nessuna
**QA**: `grep "font.*mono\|font-mono" frontend/src/index.css` non mostra regole globali

### 🟠 P1-06: Label contrast WCAG AA fail

**Veto da**: UX/UI Designer
**File**: `frontend/src/index.css` (CSS custom properties per --color-tertiary o equivalente)
**Problema**: Colore tertiary `#6B7280` su sfondo `#0F0F1A` ha contrasto ~3.0:1. WCAG AA richiede 4.5:1 per testo <18px. Le label a 10px usano questo colore.
**Risoluzione**: Aumentare luminanza del tertiary token a `#9CA3AF` (stima ~4.8:1), o limitare uso a testo decorativo/deactivated.
**Effort**: S (10 min)
**Dipendenze**: Nessuna
**QA**: Verificare contrasto con strumento (es. WebAIM contrast checker)

### 🟠 P1-07: `read_only:true` + `INSTALL vss` conflict

**Veto da**: DevOps
**File**: `internal/storage/duckdb.go` + `migrations/000001_init_schema.up.sql`
**Problema**: DuckDB aperto in read_only ma `INSTALL vss` richiede scrittura (nella root migration). Crash garantito su database fresh.
**Risoluzione**: Separare fase di setup estensioni (connessione read_write) da fase operativa (read_only). O usare `ATTACH '' AS algae_db (READ_ONLY)` dopo setup.
**Effort**: S (15 min)
**Dipendenze**: P0-01 (migration path unification)
**QA**: DuckDB si apre e migration passano senza "cannot write to read-only database"

### 🟠 P1-08: `continue-on-error:true` in CI

**Veto da**: DevOps
**File**: `.github/workflows/ci.yml:42`
**Problema**: Step "Run Go tests with race detector" ha `continue-on-error: true`, nascondendo fallimenti dei test in CI.
**Risoluzione**: Rimuovere `continue-on-error: true` dallo step (riga 42). Assicurarsi P0-01 e P0-04 siano risolti PRIMA di questo fix, altrimenti CI fallirà.
**Effort**: S (5 min)
**Dipendenze**: **P0-04** (test devono passare prima di rimuovere continue-on-error)
**QA**: CI run con test che passano senza continue-on-error

---

## P2 — Medium (qualità/struttura)

### 🟡 P2-01: Standardizzare error handling REST

**Veto da**: API Engineer
**File**: `internal/api/handler/*.go`
**Problema**: Pattern misti: alcuni handler usano `{"success":false,"error":"..."}` con HTTP 200, altri usano codici HTTP veri. Mix di `connect.NewError` e errori raw.
**Risoluzione**: Unificare su formato standard con codici HTTP appropriati. Convertire casi 200+false in 4xx/5xx.
**Effort**: M (45 min)
**Dipendenze**: Nessuna
**QA**: Tutti gli handler REST usano codici HTTP coerenti

### 🟡 P2-02: D3 zoom mancante AlephGraph

**Veto da**: Data Viz Developer
**File**: `frontend/src/lib/AlephGraph.tsx`
**Problema**: `d3.forceSimulation` usata senza `d3.zoom()`. Nodi >300 degradano senza pan/zoom.
**Risoluzione**: Aggiungere `d3.zoom()` con pan+zoom. Virtualizzazione oltre 300 nodi.
**Effort**: M (30 min)
**Dipendenze**: Nessuna

### 🟡 P2-03: ToolIntelligenceView dati mock

**Veto da**: Data Viz Developer
**File**: `frontend/src/components/ToolIntelligenceView.tsx`
**Problema**: Vista usa `mockToolData` invece di dati reali dal backend.
**Risoluzione**: Collegare a endpoint API `/api/v1/tools/intelligence` (codeflow/synthesis — W5-16 già completato).
**Effort**: M (45 min)
**Dipendenze**: W5-16 (codeflow+synthesis — completato)

### 🟡 P2-04: Leaflet clustering mancante

**Veto da**: Data Viz Developer
**File**: `frontend/src/components/ExplorerView.tsx` (mappa Leaflet)
**Problema**: Marker Leaflet senza clustering. >100 marker degradano.
**Risoluzione**: Aggiungere `leaflet.markercluster` o `supercluster`.
**Effort**: S (15 min)
**Dipendenze**: Nessuna

### 🟡 P2-05: Claim "autocoscienza" non supportato

**Veto da**: Filosofo
**File**: `AGENTS.md`, doc di progetto
**Problema**: Progetto descritto come "autocosciente" ma non ha meccanismi autoriflessivi (nessun modello di sé, nessuna introspezione computazionale). I sistemi riflessivi esistenti (RepairEngine, ToolUsageTracker, CodeFlow) sono di orchestrazione, non di coscienza.
**Risoluzione**: Ridefinire come "piattaforma di orchestrazione con sistemi riflessivi". Documentare esplicitamente il gap tra orchestrazione e autocoscienza nei file doc.
**Effort**: S (15 min)
**Dipendenze**: Nessuna

### 🟡 P2-06: Bias recalibration mancante

**Veto da**: Filosofo
**File**: `docs/development-bias-checklist.md`
**Problema**: Bias checklist creata (W6-11) ma nessun meccanismo di recalibration applicato. I bias identificati rimangono senza remediation.
**Risoluzione**: Per ogni bias identificato, aggiungere una remediation action implementabile (es. test specifici, linter rules, validatori runtime in repair.go o security.go).
**Effort**: M (30 min)
**Dipendenze**: Nessuna

### 🟡 P2-07: HEALTHCHECK + USER mancanti Dockerfile

**Veto da**: DevOps
**File**: `Dockerfile`
**Problema**: Nessun HEALTHCHECK per readiness probe. Container eseguito come root (nessuna USER directive).
**Risoluzione**: Aggiungere `HEALTHCHECK --interval=30s CMD curl -f http://localhost:8080/healthz || exit 1` e `USER nobody` (o utente non-root).
**Effort**: S (10 min)
**Dipendenze**: Nessuna
**QA**: `docker build` passa, container parte come non-root

---

## P3 — Low (migliorie non bloccanti)

| Item | Descrizione | File | Effort | Status |
|------|-------------|------|--------|--------|
| P3-01 | Brandmark + onboarding più ispirato | `frontend/src/components/` | M | ✅ |
| P3-02 | Swagger spec: endpoint mancanti | `docs/swagger/` | L | ✅ |
| P3-04 | Unificare pattern paginazione | Multipli handler | M | ✅ |
| P3-05 | Backup/restore DuckDB | `internal/storage/` | M | ✅ |
| P3-08 | nlp gitignore | `.gitignore` | S | ✅ |

---

## Item Python (ripristinati da v2 con path corretto)

⚠️ **Attenzione**: La v2 del piano aveva RIMOSSO questi item perchè Momus ha cercato `python/` (inesistente). Il codice Python reale è in `nlp/`. Item ripristinati con path corretto.

### 🟡 P2-08: `predict_probs` duplicato in nlp/ensemble.py

**Veto da**: Fullstack Python Dev
**File**: `nlp/ensemble.py:251` e `:275`
**Problema**: `predict_proba` definito due volte (stessa firma, stesse features). La seconda sovrascrive la prima.
**Risoluzione**: Rimuovere la definizione duplicata (conservare la più completa).
**Effort**: S (5 min)
**QA**: `python3 -c "import sys; sys.path.insert(0, 'nlp'); from ensemble import EnsemblePredictor; p = EnsemblePredictor(...);"` non lancia errore

### 🟡 P2-09: `torch_geometric` non in nlp/requirements.txt

**Veto da**: Fullstack Python Dev
**File**: `nlp/requirements.txt`
**Problema**: `from torch_geometric...` importato in predict.py ma non in requirements.txt. Crash su esecuzione.
**Risoluzione**: Aggiungere `torch-geometric` a `nlp/requirements.txt`.
**Effort**: S (2 min)
**QA**: `grep "torch" nlp/requirements.txt` matcha torch-geometric

### 🟡 P2-10: DataHealthView usa div CSS invece di charting

**Veto da**: Data Viz Developer
**File**: `frontend/src/components/DataHealthView.tsx`
**Problema**: Health metrics renderizzate come raw CSS `<div>` bars invece di usare una libreria di charting (recharts, d3, vega-lite).
**Risoluzione**: Migrare bars CSS a recharts `BarChart` o equivalente d3.
**Effort**: S (15 min)
**Dipendenze**: Nessuna
**QA**: DataHealthView mostra barre con assi e scale proporzionali

### 🟡 P2-11: Timeline CSS-only (no proper visualization)

**Veto da**: Data Viz Developer
**File**: Componente timeline (ExplorerView o TimelineView)
**Problema**: Timeline renderizzata con puro CSS invece di time-series chart (d3-scale, recharts LineChart).
**Risoluzione**: Sostituire timeline CSS con componente graph-based (recharts LineChart o d3-time).
**Effort**: S (15 min)
**Dipendenze**: Nessuna
**QA**: Timeline mostra data axis, zoom temporale, tooltip

### 🟡 P2-12: Responsive/mobile design mancante

**Veto da**: UX/UI Designer
**File**: Multipli view in `frontend/src/`
**Problema**: Applicazione non responsive. Layout fisso per 1920×1080. Nessun breakpoint mobile. Timeline fissa. SlideOver full-width. Sidebar non collassabile su mobile. Tabella overflow orizzontale.
**Risoluzione**: Aggiungere breakpoint CSS per mobile. SlideOver full-screen su mobile. Sidebar collassabile. Terminal responsive. Tabella scroll orizzontale.
**Effort**: M (2-3 ore)
**Dipendenze**: Nessuna
**QA**: Layout funziona a 390px (iPhone 14) e 768px (iPad)

### 🟡 P2-13: Focus trap modali (↑ da P3 a P2)

**Veto da**: UX/UI Designer
**File**: `frontend/src/components/SlideOverPanel.tsx`
**Problema**: Modali e SlideOver non hanno focus trap. Utenti tastiera (Tab) escono dal modal. Violazione WCAG 2.1.1 Keyboard.
**Risoluzione**: Aggiungere `@react-aria/focus` o implementare focus trap manuale in SlideOverPanel.
**Effort**: S (15 min)
**Dipendenze**: Nessuna
**QA**: Tab cicla all'interno del modal, non esce

### 🟡 P2-14: Bookmark mobile/touch support (↑ da P3 a P2)

**Veto da**: UX/UI Designer
**File**: `frontend/src/components/CopilotView.tsx`
**Problema**: Bookmark visibile solo su hover. Nessun bookmark accessibile su touch/mobile o da screen reader.
**Risoluzione**: Aggiungere bookmark button sempre visibile (icona + label), non solo su hover.
**Effort**: S (10 min)
**Dipendenze**: Nessuna
**QA**: Bookmark visibile senza hover, cliccabile su touch

### 🟢 P3-08: `__pycache__/` e `venv/` in nlp/ committati

**Veto da**: Fullstack Python Dev
**File**: `.gitignore`
**Problema**: `nlp/__pycache__/` e `nlp/venv/` committati nel repository.
**Risoluzione**: Aggiungere `nlp/__pycache__/` e `nlp/venv/` a `.gitignore`, rimuovere dal tracking.
**Effort**: S (5 min)
**QA**: `git status nlp/__pycache__` non mostra file tracked

---

## Wave Execution Plan

### Wave 1 (Immediata, 0 dipendenze — 16 task paralleli)
```
P0-02: Fix 3 TS errors (S, 10min) 
P0-03: Fix deploy.yml YAML (S, 2min)
P1-01: isMutatingOperation + "Send" (S, 5min)
P1-02: Fix ProjectIDFromContext nil-safety (S, 10min)
P1-03: Sostituire alert() con Toast (S, 10min)
P1-05: font-mono limitato a terminal (S, 10min)
P1-06: Fix label contrast WCAG AA (S, 10min)
P0-01: Migration 000004 system_* tables (M, 30min)
P2-02: D3 zoom AlephGraph (M, 30min)
P2-04: Leaflet clustering (S, 15min)
P2-05: Fix claim "autocoscienza" doc (S, 15min)
P2-07: HEALTHCHECK + USER Dockerfile (S, 10min)
P2-08: predict_probs duplicato nlp (S, 5min)
P2-09: torch_geometric in nlp/requirements.txt (S, 2min)
P2-14: Bookmark mobile/touch (S, 10min)
P3-03: Dead code AlephPrediction (S, 5min)
```

### Wave 2 (Dopo P0-01)
```
P0-04: Integration test pass (M, 30min)
P1-07: read_only + INSTALL fix (S, 15min)
P1-08: continue-on-error fix (S, 5min)
P1-04: 3 form inline -> componenti (M, 45min)
P2-01: Standardizzare error handling REST (M, 45min)
P2-03: ToolIntelligenceView dati reali (M, 45min)
P2-06: Bias recalibration actions (M, 30min)
P2-10: DataHealthView charting (S, 15min)
P2-11: Timeline visualization (S, 15min)
P2-12: Responsive design (M, 2-3h)
P2-13: Focus trap modali (S, 15min)
P3-05: DuckDB backup/restore (M)
```

### Wave 3 (Secondarie)
```
P3-01: Brandmark + onboarding (M)
P3-02: Swagger spec complete (L)
P3-04: Paginazione unificata (M)
P3-08: nlp gitignore (S, 5min)
```

---

## Criteri di Accettazione

| # | Criterio | Verifica |
|---|----------|----------|
| 1 | `go build ./...` zero errori | `go build ./...` exit 0 |
| 2 | `npx tsc --noEmit` zero errori | `npx tsc --noEmit` exit 0 (0 errori, non 3) |
| 3 | `go test ./...` pancia verde | `go test ./...` exit 0 |
| 4 | `npx vite build` zero errori | `npx vite build` exit 0 |
| 5 | WCAG AA label contrast ≥4.5:1 | Lighthouse/axe-core audit |
| 6 | `deploy.yml` YAML valido | `python3 -c "import yaml; yaml.safe_load(...)"` |
| 7 | Zero `alert()` nativi UI | `grep -r "alert(" frontend/src/ \| grep -v eslint\|comment` 0 match |
| 8 | Zero `font-mono` su body | CSS: font-mono solo su classi terminal/code |
| 9 | Zero form inline in App.tsx | App.tsx: nessun JSX >20 righe inline nei case |
| 10 | Zero nil pointer audit middleware | `go test -race ./internal/middleware/...` passa |
| 11 | Zero "autocoscienza" claim non supportato | grep doc per claim e fix |
| 12 | Docker HEALTHCHECK + USER presenti | `grep -E "HEALTHCHECK|USER" Dockerfile` |
| 13 | CI senza continue-on-error | `ci.yml` riga 42 senza continue-on-error |
| 14 | Responsive design base funziona 390px-1920px | Browser resize / Playwright responsive test |
| 15 | DataHealthView usa charting library | DataHealthView.tsx importa recharts/d3 |
| 16 | Timeline con data axis e zoom | Timeline componente ha asse temporale |
| 17 | Modali con focus trap | Tab all'interno modal non esce |
| 18 | `nlp/ensemble.py` senza predict_probs duplicato | `grep -c "def predict_proba" nlp/ensemble.py` = 1 |
| 19 | `torch-geometric` in `nlp/requirements.txt` | `grep "torch-geometric" nlp/requirements.txt` matcha |

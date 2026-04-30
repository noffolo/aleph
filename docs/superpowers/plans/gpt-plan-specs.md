# Execution Specifications — gpt-plan-revised.md

**Collegato a**: [`gpt-plan-revised.md`](gpt-plan-revised.md) (piano strategico + gap analysis)
**Ruolo**: Task breakdown concreto, acceptance criteria eseguibili, dipendenze tra task

**Relazione tra i due documenti**:
```
gpt-plan-revised.md (strategico)          gpt-plan-specs.md (esecutivo)
─────────────────────────────────────     ─────────────────────────────────────
Scope e gap analysis (A-G)          ──→   Task breakdown per ogni area
3 tracce parallele                   ──→   WBS con durata e dipendenze
Principi e governance                ──→   Quality gates matrix per-traccia
Risk register                        ──→   Dependency graph + blocchi
KPI di successo                      ──→   Acceptance criteria come comandi shell
```

---

## Track A — Discovery + Backend/NLP Redesign

### A-W1: Discovery tecnico onesto

**Durata**: 2-3gg | **Dipendenza**: Nessuna | **Bloccante per**: Tutto Track A

#### Task A-W1.1 — Code walk internal/ packages
- [ ] Elencare OGNI package in `internal/` con classificazione: Stub/Parziale/Funzionale/Produzione
- [ ] Per ogni capability README, comando shell che ne verifica lo stato
- [ ] Output: W1_GAP_MATRIX.md con tabella capability → stato → comando verifica → owner

**Accettazione**:
```bash
# Ogni capability README deve avere una riga nella matrice
grep -c "|" docs/superpowers/plans/gpt-plan-revised.md
# Deve esistere una capability matrix esportabile
test -f W1_GAP_MATRIX.md && head -5 W1_GAP_MATRIX.md
```

#### Task A-W1.2 — SQL injection audit
- [ ] Grep per tutti i `fmt.Sprintf` in `internal/memory/store.go` e `internal/ingestion/`
- [ ] Classificare ogni occorrenza: parametrizzata vs. non sicura
- [ ] Output: elenco vector con severity (Critical/High/Medium/Low)

**Accettazione**:
```bash
# Trovare tutti i fmt.Sprintf con input non parametrizzati
grep -rn 'fmt\.Sprintf.*%s' internal/memory/ internal/ingestion/ | grep -i 'select\|insert\|update\|delete'
```

#### Task A-W1.3 — gRPC contract audit
- [ ] Elencare tutti i metodi gRPC definiti in `api/proto/`
- [ ] Confrontare con implementazione Python in `nlp/`
- [ ] Confrontare con consumer Go in `internal/nlp_adapter/`
- [ ] Output: matrice metodo → implementato Python? → consumato Go?

**Accettazione**:
```bash
# Metodi proto
grep -E 'rpc\s+\w+' api/proto/*.proto
# Metodi Python implementati
grep -E 'def\s+\w+' nlp/server.py
# Metodi Go consumati
grep -rn 'nlp\.\|AnalyzeSentiment\|StreamPredictions' internal/nlp_adapter/
```

---

### A-W2: Backend + NLP gap closure

**Durata**: 10-15gg | **Dipendenza**: A-W1 completato | **Bloccante per**: A-W2f (gRPC contract)

#### Task A-W2.1 — PAORA Reflect redesign (2-3gg)

**Scope**:
- Sostituire `return plan, nil` in `internal/decision/observer.go` con reflection strutturata
- Reflect deve: analizzare scostamenti piano↔risultato, classificare gap (atteso/inatteso/critico), produrre reflection strutturata
- Aggiungere Observer fallback per keyword-only deadlock (se Observer non produce output, usare fallback pattern matching)

**Cosa NON fare**:
- Non integrare LLM nel Reflect (post-MVP)
- Non toccare Plan/Act/Admit già funzionanti

**Test**:
```go
// Test che Reflect produce output non vuoto per input realistico
func TestReflectProducesOutput(t *testing.T) {
    engine := NewDecisionEngine()
    plan := Plan{Steps: []Step{{Action: "analyze", Params: map[string]string{"query": "test"}}}}
    result := PlanResult{Outcome: "success", Data: map[string]interface{}{"confidence": 0.85}}
    reflection, err := engine.Reflect(plan, result)
    assert.NoError(t, err)
    assert.NotEmpty(t, reflection.Analysis)
    assert.NotEmpty(t, reflection.GapClassification)
}
```

**Accettazione**:
```bash
go test -race -count=1 -run TestReflect ./internal/decision/... -v
# Deve mostrare "PASS" per TestReflectProducesOutput
grep -c "return plan, nil" internal/decision/observer.go
# → DEVE essere 0
```

---

#### Task A-W2.2 — GENESIS Suggester reale (2gg)

**Scope**:
- `Suggester.Analyze()`: implementare pattern-matching basato su embedding similarity (DuckDB VSS) + rule-based fallback
- NON usare LLM per suggerimenti. Usare similarità con tool esistenti + pattern history.
- Output: `Suggestion{Score: float64, Rationale: string, ToolConfig: ToolDefinition}`

**Cosa NON fare**:
- Non implementare sandbox containerizzata (troppo scope per ora)
- Documentare sandbox come "heuristic filter" (regex + resource timeout)

**Test**:
```go
func TestSuggesterProducesValidSuggestion(t *testing.T) {
    suggester := NewSuggester(engine)
    input := ToolInput{Name: "fetch-data", Parameters: map[string]string{"url": "https://example.com/data.csv"}}
    suggestions, err := suggester.Analyze(context.Background(), input)
    assert.NoError(t, err)
    assert.Greater(t, len(suggestions), 0)
    assert.Greater(t, suggestions[0].Score, 0.0)
    assert.NotEmpty(t, suggestions[0].Rationale)
}
```

**Accettazione**:
```bash
go test -race -count=1 -run TestSuggester ./internal/genesis/... -v
# Deve mostrare "PASS" per TestSuggesterProducesValidSuggestion
```

---

#### Task A-W2.3 — NLP Sentiment: deprecare o sostituire (3-5gg)

**OPZIONE A (consigliata)** — Deprecazione onesta:
- [ ] Aggiungere flag `is_calibrated: false` all'output sentiment
- [ ] Documentare in README: "Sentiment analysis: baseline euristica, non ML. Confidence non calibrata."
- [ ] Rimuovere claim "modelli ensemble NLP" dal README

**OPZIONE B** — Sostituzione:
- [ ] Training script per classificatore testo leggero (ONNX, bert-small)
- [ ] Pipeline di calibrazione (Platt scaling)
- [ ] Sostituire word-counting con inferenza ONNX
- [ ] gRPC contract aggiornato

**Test**:
```python
# test_sentiment.py (nuovo)
def test_sentiment_returns_confidence_range():
    result = analyze_sentiment("Questo è un test positivo")
    assert 0.0 <= result.confidence <= 1.0
    assert result.label in ["positive", "negative", "neutral"]
```

**Accettazione**:
```bash
# Se OPZIONE A
grep -c "is_calibrated: false" nlp/sentiment.py
# → DEVE essere 1

# Se OPZIONE B
python -m pytest nlp/tests/test_sentiment.py -v
# → Deve passare
```

---

#### Task A-W2.4 — Repair engine AST-aware (2-3gg)

**Scope**:
- `fixPerformance`: Implementare logica reale o rimuovere (attualmente no-op)
- `fixTimeout`/`fixCaching`: Sostituire string-replace con AST transformation (usando `go/ast` o regex strutturate)
- `executeRegenerate`: Implementare tentativo di rigenerazione (template + compilazione)
- Documentare ogni strategia come "production" o "heuristic" in base all'implementazione

**Cosa NON fare**:
- Non riscrivere tutto da zero. Mantenere strategie funzionanti.
- Non aggiungere nuove strategie.

**Test**:
```go
func TestFixTimeoutProducesValidGo(t *testing.T) {
    input := `func slow() { time.Sleep(10 * time.Second) }`
    result, err := repairEngine.Fix("timeout", input)
    assert.NoError(t, err)
    // Il risultato deve essere Go compilabile
    _, err = syntax.ParseGo(result)
    assert.NoError(t, err)
}
```

**Accettazione**:
```bash
go test -race -count=1 -run TestFixTimeout ./internal/repair/... -v
# Tutti i test repair devono passare
```

---

#### Task A-W2.5 — SQL injection fix (BLOCKING, 1-2gg)

**Scope**:
- [ ] Parametrizzare TUTTI i `fmt.Sprintf` con input utente in `internal/memory/store.go` e `internal/ingestion/`
- [ ] Usare `database/sql` parametrized queries (`$1`, `$2`, ecc.) o `DuckDB` prepared statements
- [ ] Rafforzare `validateSQLName`: regex `^[a-zA-Z_][a-zA-Z0-9_]*$`, lunghezza max 64 caratteri, whitelist di nomi consentiti

**Vincolo**: QUESTO è BLOCKING per qualsiasi deploy. Non si passa a W5 senza questo fix.

**Test**:
```go
func TestSQLInjectionBlocked(t *testing.T) {
    malicious := "users; DROP TABLE users; --"
    err := store.Store(context.Background(), malicious, data)
    assert.Error(t, err)  // Deve rifiutare
    assert.Contains(t, err.Error(), "invalid table name")  
}
```

**Accettazione**:
```bash
go test -race -count=1 -run TestSQL ./internal/memory/... -v
go test -race -count=1 -run TestSQL ./internal/ingestion/... -v
# Nessun fmt.Sprintf con input non parametrizzato rimasto
grep -rn 'fmt\.Sprintf.*SELECT\|fmt\.Sprintf.*INSERT\|fmt\.Sprintf.*UPDATE\|fmt\.Sprintf.*DELETE' internal/ --include='*.go'
# → DEVE essere 0 (o solo query senza input utente)
```

---

#### Task A-W2.6 — gRPC contract testing + NLP output standardization (1-2gg)

**Scope**:
- [ ] Pinnare proto: aggiungere `nlp.proto` version comment (`// version: 1.0.0`)
- [ ] Contract test Go: chiamare Python sidecar via gRPC, validare output schema
- [ ] Standardizzare output probabilistico:

**Proto update minimo**:
```protobuf
message SentimentResult {
  string label = 1;
  double score = 2;           // 0.0 - 1.0
  double confidence = 3;       // 0.0 - 1.0 (1.0 = calibrated)
  bool is_calibrated = 4;
  string method = 5;           // "heuristic" | "onnx" | "ensemble"
}
```

**Test**:
```go
//go:build contract
func TestNLPContract(t *testing.T) {
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    client := pb.NewNLPServiceClient(conn)
    resp, err := client.AnalyzeSentiment(ctx, &pb.SentimentRequest{Text: "test"})
    assert.NoError(t, err)
    assert.InDelta(t, 0.0, resp.Confidence, 1.0)
    assert.NotEmpty(t, resp.Label)
}
```

**Accettazione**:
```bash
docker compose up -d nlp-sidecar
go test --tags=contract -run TestNLPContract ./internal/nlp/... -v
docker compose down nlp-sidecar
# Deve mostrare "PASS"
```

---

## Track B — Monitoring + Frontend

### B-W3-Mon: Monitoring infrastructure

**Durata**: 3-5gg | **Dipendenza**: Nessuna (parallelo a W2) | **Bloccante per**: Track C W5-Rel

#### Task B-W3-Mon.1 — Prometheus metrics endpoint (1-2gg)

**Scope**:
- [ ] Aggiungere `promhttp` Handler al server HTTP
- [ ] Metriche minime:
  - `aleph_http_requests_total{method, path, status}`
  - `aleph_http_request_duration_seconds{method, path, quantile}`
  - `aleph_nlp_requests_total{method, status}`
  - `aleph_db_query_duration_seconds{engine, operation}`
  - `aleph_paora_cycle_total{phase, outcome}`

**Accettazione**:
```bash
go build ./... && go run . &
sleep 2
curl -s http://localhost:9090/metrics | grep aleph_http_requests_total
# Deve mostrare counter >= 0
kill %1
```

#### Task B-W3-Mon.2 — Grafana + Alertmanager integration (1-2gg)

**Scope**:
- [ ] Aggiungere a `docker-compose.yml`: prometheus, grafana, alertmanager
- [ ] 3 dashboard minime:
  1. **API/Servizi**: request rate, error rate, p50/p95/p99 latency
  2. **NLP sidecar**: request rate, error rate, method breakdown
  3. **DB/Storage**: query duration, connection pool, VSS query count
- [ ] SLO alerting rules: error budget 5% rolling 30gg, latency p95 > 500ms warning

**Accettazione**:
```bash
docker compose config
# Deve mostrare prometheus, grafana, alertmanager services
docker compose up -d prometheus grafana alertmanager
curl -s http://localhost:3000/api/health
# → {"status": "ok"}
curl -s http://localhost:9090/-/healthy
# → OK
docker compose down
```

---

### B-W3-UX: Scenario comparison + Frontend polish

**Durata**: 2-3gg | **Dipendenza**: Nessuna (parallelo a W2) | **Bloccante per**: W5 exit criteria

#### Task B-W3-UX.1 — Scenario comparison view (1-2gg)

**Scope**:
- [ ] Nuova view o modal: seleziona 2-3 scenari → confronto side-by-side
- [ ] Elementi da confrontare: confidence score, segnali chiave, assumptions, trend direction
- [ ] Visual diff: probability distribution chart, signal strength comparison
- [ ] Integrazione con Zustand `scenarioSlice`

**Cosa NON fare**:
- Non modificare la logica di scenario generation
- Non toccare la structure di App.tsx

**Test**:
```tsx
// test scenario comparison rendering
it('renders side-by-side comparison of 2 scenarios', () => {
  render(<ScenarioComparison scenarios={mockScenarios.slice(0, 2)} />);
  expect(screen.getByTestId('scenario-a')).toBeInTheDocument();
  expect(screen.getByTestId('scenario-b')).toBeInTheDocument();
});
```

**Accettazione**:
```bash
npx vitest run -- --testPathPattern="ScenarioComparison" -v
# Deve passare
npx tsc --noEmit
# 0 errors
```

#### Task B-W3-UX.2 — WCAG 2.1 AA + Lighthouse (1gg)

**Scope**:
- [ ] Keyboard navigation audit: tutti i flussi navigabili con Tab/Enter/Escape
- [ ] Focus management: modali, slide-over, dropdown
- [ ] Contrast ratio: testo su sfondo ≥ 4.5:1
- [ ] Aria labels: tutti gli elementi interattivi
- [ ] React.memo sui componenti che usano `useStore()` con slice selettori

**Accettazione**:
```bash
npx lighthouse http://localhost:5173 --output=json | jq '.categories.performance.score, .categories.accessibility.score, .categories."best-practices".score'
# Tutti ≥ 0.9
```

---

## Track C — Security, Resilience, Release

### C-W4-Sec: Security hardening

**Durata**: 3-5gg | **Dipendenza**: A-W2.5 (SQL injection fix) | **Parallelo a**: W2+W3

#### Task C-W4-Sec.1 — SAST/DAST in CI (1gg)

**Scope**:
- [ ] Aggiungere `govulncheck ./...` al workflow CI
- [ ] Aggiungere `npm audit --audit-level=high` al workflow CI
- [ ] Aggiungere `trivy image aleph-v2:latest` al workflow CI
- [ ] Tutti devono fallire la pipeline se trovano vulnerability

**Accettazione**:
```bash
# Verificare che i tool funzionino localmente
govulncheck ./...
# exit code 0 → no high/critical

npm audit --audit-level=high
# exit code 0 → no high vulnerabilities

trivy image aleph-v2:latest | grep CRITICAL | wc -l
# → 0
```

#### Task C-W4-Sec.2 — Threat model + audit trail (2gg)

**Scope**:
- [ ] Produrre threat model: data flow diagram, trust boundaries, attori, attack surface
- [ ] Audit trail per eventi PAORA: ogni ciclo decisionale loggato con correlation ID + timestamp + outcome
- [ ] Auth hardening: API key supporta scope (admin/user/readonly). Rimuovere chiave da localStorage

**Accettazione**:
```bash
# Audit trail esistente e popolato
grep -rn 'audit\.Log\|AuditLog' internal/audit/
# Deve mostrare implementazione reale, non stub
```

#### Task C-W4-Sec.3 — Chaos drills (1-2gg)

**Scope**:
- [ ] NLP sidecar down: sistema deve degradare graceful senza crash
- [ ] DB latency injection: query devono timeout secondo configurazione
- [ ] MCP discovery failure: sistema deve usare cache

**Accettazione**:
```bash
# Test manuale documentato in runbook
# NLP down: curl API endpoint → deve restituire errore strutturato 503, non panic
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/predictions
# → Deve essere 503 (non 500)
```

---

### C-W5-Rel: Release candidate + go-live

**Durata**: 2-3gg | **Dipendenza**: A-W2 completato, B-W3-Mon completato, C-W4-Sec completato

#### Task C-W5-Rel.1 — CI gate integration (1gg)

**Scope**:
- [ ] Contract test Go↔Python in CI: `docker compose up nlp-sidecar && go test --tags=contract`
- [ ] E2E Playwright smoke in CI: scenario utente critico (login→scenario→confronto)
- [ ] Load test baseline: `go test -bench=. ./...` registrato per confronto futuro

**Accettazione**:
```bash
# Verificare che i workflow CI esistano e includano i nuovi gate
grep -c 'contract\|playwright\|e2e\|bench' .github/workflows/ci.yml
# Deve essere > 0
```

#### Task C-W5-Rel.2 — Documentation + release (1-2gg)

**Scope**:
- [ ] README aggiornato: capacità classificate onestamente, non rivendicare ML dove non c'è
- [ ] Documentazione architettura: diagramma aggiornato con dual-engine, NLP sidecar, monitoring stack
- [ ] Runbook incident response: NLP down, DB failure, deploy rollback
- [ ] Release tag semantico: `v2.x.x`
- [ ] Release notes automatiche da commit log

**Accettazione**:
```bash
# README non deve contenere claim non verificati
grep -ci 'produzione\|production-grade\|completato' README.md
# Deve essere coerente con la matrice W1
```

---

## Quality Gates Matrix (per traccia)

| Gate | Comando | Track A | Track B | Track C |
|------|---------|---------|---------|---------|
| Go build | `go build ./...` | ✅ | ✅ | ✅ |
| Go test (race) | `go test -race -count=1 ./...` | ✅ | ✅ | ✅ |
| Go vet | `go vet ./...` | ✅ | ✅ | ✅ |
| PAORA Reflect | `grep "return plan, nil" internal/decision/observer.go` → 0 | ✅ | — | — |
| GENESIS output | `go test -run TestSuggester ./internal/genesis/...` ✅ | ✅ | — | — |
| SQL injection | `grep -c 'fmt\.Sprintf.*SELECT' internal/` → 0 | ✅ | — | — |
| gRPC contract | `go test --tags=contract ./internal/nlp/...` ✅ | ✅ | — | — |
| Frontend tsc | `npx tsc --noEmit` ✅ | — | ✅ | ✅ |
| Frontend vite | `npx vite build` ✅ | — | ✅ | ✅ |
| Vitest | `npx vitest run` ✅ | — | ✅ | ✅ |
| Lighthouse | ≥90 per performance/accessibility/best-practices | — | ✅ | — |
| Metrics endpoint | `curl localhost:9090/metrics \| grep aleph_` ✅ | — | ✅ | — |
| Grafana health | `curl localhost:3000/api/health` ✅ | — | ✅ | — |
| Govulncheck | `govulncheck ./...` → no high/critical | — | — | ✅ |
| npm audit | `npm audit --audit-level=high` → exit 0 | — | — | ✅ |
| Trivy scan | `trivy image aleph-v2:latest \| grep CRITICAL` → 0 | — | — | ✅ |
| Docker config | `docker compose config` ✅ | — | — | ✅ |
| E2E Playwright | playwright smoke test ✅ | — | — | ✅ |
| Release tag | `git tag v2.x.x` | — | — | ✅ |

---

## Dependency Graph

```
A-W1 (Discovery)
  ├── A-W2.1 (PAORA Reflect)
  ├── A-W2.2 (GENESIS Suggester)
  ├── A-W2.3 (NLP Sentiment)
  ├── A-W2.4 (Repair AST)
  ├── A-W2.5 (SQL injection) ─── BLOCKING ─── C-W4-Sec
  ├── A-W2.6 (gRPC contract)
  │
  ├── B-W3-Mon.1 (Prometheus) ─── B-W3-Mon.2 (Grafana)
  │                              └── C-W5-Rel
  ├── B-W3-UX.1 (Scenario comp) ─── C-W5-Rel (E2E)
  ├── B-W3-UX.2 (WCAG+Lighthouse)
  │
  ├── C-W4-Sec.1 (SAST/DAST) ─── C-W5-Rel (CI gate)
  ├── C-W4-Sec.2 (Threat model)
  └── C-W4-Sec.3 (Chaos drills) ─── C-W5-Rel

C-W5-Rel (Release) dipende da: A-W2 completo + B-W3-Mon completo + C-W4-Sec completo
```

---

## Legend — Stato task

| Stato | Significato |
|-------|-------------|
| ⬜ Pending | Non ancora iniziato |
| 🔄 In progress | In esecuzione |
| ✅ Completato | Verificato con comando shell |
| ❌ Bloccato | Dipendenza non soddisfatta |
| 🚫 Cancellato | Non più in scope (con reason) |

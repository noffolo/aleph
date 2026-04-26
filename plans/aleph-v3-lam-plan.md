# aleph-v3: LAM Integration — Indice

> **Review**: Momus ✅ | Aleph (self) ✅ | Metis ✅
> **Decisione**: Split in due piani separati
> **Data**: 2026-04-25

---

## Piano A: aleph-v2.1 — Secondo Hardening

**File**: `a-v2.1-hardening-2.md` (17 task, 2 wave)

Cosa si fa:
- **W1**: NotificationService stop, tool_suggest ticker, goroutine leak fix, app.go God Object decomposition, ToolCodeWriter ctx, DuckDB backup lock, vitest config, store test, SlideOver triage
- **W2**: DELETE NLP dead code (CalibrationWrapper, predict_probs fittizio, sentiment fake, train_link_prediction, unused deps), fix gRPC proto, honest responses (mai score fabbricati), Decision Loop Plan→Act→Observe→Reflect→Admit, decision trace spans, hook unit+integration test, component test

**Prerequisito per v3**: v2.1 completato + build check passato

---

## Piano B: aleph-v3 — LAM Pivot

**File**: `b-v3-lam-pivot.md` (34 task, 7 fasi)

Cosa si fa (solo DOPO v2.1):
- **F1**: MCP JSON-RPC 2.0 + STDIO + tool registry unico
- **F2**: Vector store + embeddings + retrieval in Chat()
- **F3**: Decision Loop reale (Plan→Act→Observe→Reflect→AdmitFailure) con OTEL trace
- **F4**: A2A protocol + AgentOrchestrator + capability registry
- **F5**: Policy engine + Judge (advisory) + AST scanner + sandbox Verifier + defense in depth
- **F6**: *Spike* multimodal (OCR) + computer-use (Playwright sandbox) — opzionali, non gate
- **F7**: Benchmark E2E + audit tool + load test + coverage 60% + rollback plan

---

## Dipendenza

```
v2.1 (W1+W2) ──complete──→ v3 (F1→F7)
```

Non iniziare v3 prima che v2.1 abbia build check passante.

---

## Note Review

### Momus (CONDITIONAL PASS)
- File paths corretti dopo review

### Aleph (ACCEPT WITH CONDITIONS)
1. Multi-agente SOLO DOPO Decision Loop + Memoria
2. Judge Model solo advisory
3. AdmitFailure esplicito
4. Multimodal/Computer-use come spike, non gate

### Metis (SPLIT RECOMMENDED)
- v2.1 + v3 separati
- Blind spot corretti: rollback plan, migration doc

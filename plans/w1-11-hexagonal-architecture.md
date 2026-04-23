# W1-11 — Hexagonal Architecture Plan

> *"Hexagonal architecture è overengineering per un monolite con tre entrypoint (di cui uno morto). Mi serve Zustand che non esplode, DuckDB che non perde dati, e streaming che si ferma quando gli dico basta."*
> — Aleph

## Stato: PLAN-ONLY (nessuna implementazione)

## Problema

Il codebase ha 3 entrypoint Go (main.go, cmd/aleph-server/main.go, internal/app.go) di cui uno potenzialmente morto. La logica di business è accoppiata direttamente a:
- DuckDB (storage)
- Ollama/Anthropic/OpenAI (LLM via HTTP)
- gRPC NLP sidecar
- Y.js WebRTC (collaborazione frontend)

L'accoppiamento rende difficile testare, sostituire componenti, o evolvere indipendentemente.

## Perché NON ora

1. **Il monolite funziona** — W0 e W0.5 hanno risolto i bug critici (SQLi, SSRF, context leak)
2. **W1-03 (Provider interface)** introduce già un'abstrazione chiave per LLM
3. **W1-02 (Migrations)** introduce già separazione schema-vs-code
4. **L'effort è sproporzionato** — richiederebbe cambi in ogni handler, ogni test, ogni constructor
5. **I test non esistono ancora** — rifattorizzare senza test è rischioso

## Piano (per wave futura, W3+)

### Fase 1: Ports (interfacce)

```
internal/port/
  query_port.go      — QueryService interface
  ingestion_port.go  — IngestionService interface  
  nlp_port.go        — NLPAnalyzer interface (GIÀ ESISTE in ingestion/)
  llm_port.go        — Provider interface (GIÀ ESISTE in llm/)
  registry_port.go   — RegistryService interface
```

### Fase 2: Adapters (implementazioni)

```
internal/adapter/
  duckdb_query.go      — DuckDBQueryAdapter implements QueryService
  duckdb_registry.go   — DuckDBRegistryAdapter implements RegistryService
  grpc_nlp.go          — gRPC NLP adapter (GIÀ ESISTE in nlp_adapter/)
  ollama_llm.go        — OllamaProvider (GIÀ ESISTE in llm/)
  anthropic_llm.go     — AnthropicProvider (GIÀ ESISTE in llm/)
  openai_llm.go        — OpenAIProvider (GIÀ ESISTE in llm/)
```

### Fase 3: Dependency Injection

- App.go diventa composition root: crea adapters, inietta nei handlers
- Handler non importano più `storage.DuckDB` o `http.Client` direttamente
- Accettano interfacce port

### Fase 4: Eliminare entrypoint morto

- Determinare quale di main.go / cmd/aleph-server/main.go è vivo
- Eliminare l'altro
- Unificare in singolo entrypoint

## Pre-requisiti

- [ ] Test suite minima (almeno E2E) — senza test, rifattorizzare è a rischio
- [ ] W1-02 (migrations) completata — schema stabile
- [ ] W1-03 (Provider interface) completata ✅
- [ ] Tutti gli handler hanno interfacce port definite

## Metriche di successo

- Handler importano solo `port/` e `adapter/`, mai `storage/` o `http` direttamente
- DuckDB può essere sostituito con PostgreSQL cambiando solo adapter
- LLM provider può essere aggiunto senza toccare handler
- Test possono usare mock delle interfacce port

## Rischio

- **MEDIO**: Richiede cambi pervasivi ma incrementali. Ogni fase può essere rilasciata indipendentemente.
- Il rischio principale è la mancanza di test. Prima i test (W2), poi l'architettura (W3+).
# Gap Analysis: README Claims vs Codebase Realtà

> Analisi degli scostamenti tra le affermazioni del README e l'implementazione effettiva nel codebase, con indicazioni su cosa serve per colmare ogni gap.

**Data**: 28 Aprile 2026
**Progetto**: aleph-v2
**Commit**: `6335dac`

---

## Riepilogo

| Livello | Gap trovati |
|---------|-------------|
| ❌ Claim assente dal codice (rimosso dal README) | 3 |
| ⚠️ Parzialmente implementato | 3 |
| ✅ Corretto nel README attuale | 3 |

---

## Gap risolti (rimossi dal README)

### 1. LangChain ❌

**Claim originale**: Elencato in architettura e stack table come componente dell'intelligence layer.

**Realtà**: Zero riferimenti in qualsiasi file `.py`, `.txt`, `requirements.txt` o `.md` nella directory `nlp/`. Non presente in `go.mod`. Non installato né importato.

**Cosa servirebbe per implementarlo**:
- Decidere il ruolo: agent orchestration (LangChain AgentExecutor), RAG su DuckDB, o tool-use chain
- Aggiungere `langchain`, `langchain-community` a `nlp/requirements.txt`
- Creare un `nlp/langchain_agent.py` o integrare in `nlp/main.py` con un nuovo RPC
- Documentare il nuovo flusso prima di ri-aggiungere al README

### 2. XGBoost (come modello attivo) ❌

**Claim originale**: Elencato nell'intelligence layer come componente attivo.

**Realtà**: `xgboost` è in `requirements.txt` e in `site-packages`, MA:
- Mai importato in nessun file `.py`
- `nlp/main.py` riga 117 stampa "Ensemble (Prophet/XGBoost) loaded" — falso positivo
- Non esiste codice che istanzi o chiami `xgb.XGBRegressor` o simili

**Cosa servirebbe per implementarlo**:
- Creare `nlp/xgboost_model.py` con training, inference, e interfaccia unificata
- Integrare in `nlp/ensemble.py` come alternativa/componente del PredictiveEnsemble
- Correggere la stampa in `nlp/main.py` (già non referenziata dal README corrente)

### 3. Protocollo Genesis ❌

**Claim originale**: Meccanismo sperimentale per auto-proposta di tools/skill/agenti, con revisione umana.

**Realtà**: Zero occorrenze di "genesis" o "Genesis" in qualsiasi file `.go`. Nessun meccanismo di auto-proposta implementato.

**Cosa servirebbe per implementarlo**:
- Progettare formato delle proposte (tool/skill/agent schema)
- Definire interfaccia `ProposalEngine` in Go
- Implementare `internal/genesis/` con: `proposal.go`, `review.go`, `approval.go`
- Integrare nel decision loop: dopo Reflect, se Observe rileva gap → Genesis propone
- Aggiungere UI per revisione e approvazione umana
- Documentare e ri-aggiungere al README solo quando esiste un MVP funzionante

---

## Gap parziali (monitorare)

### 4. Ensemble "multi-modello" ⚠️

**Claim**: "modelli statistici, machine learning, modelli linguistici, analisi del sentiment, agenti AI specializzati"

**Realtà**: Solo Prophet + GBM geometrico in `ensemble.py`. Il sentiment analysis è embedding-based con fallback euristico. Non ci sono modelli linguistici indipendenti nell'ensemble. Gli "agenti specializzati" sono tool-dispatch wrappers (ricerca, sentiment, trust score), non modelli ensemble.

**Per colmare**:
- Aggiungere un modello basato su LLM (ad es. chiamata a Ollama per forecast testuale) nell'ensemble
- Aggiungere XGBoost reale
- Documentare onestamente lo stato attuale: "Prophet + GBM + sentiment score"

### 5. Decision Loop (Observe/Reflect) ⚠️

**Claim implicito**: Ciclo Plan→Act→Observe→Reflect completo e funzionante.

**Realtà**:
- `observer.go` — valida solo errori, output vuoti e troncamento. Nessuna valutazione LLM.
- `reflector.go` — minimale: marca `CanProceed=false` all'ultimo fallimento. Nessun re-planning.
- `planner.go` — usa keyword matching quando non c'è LLM provider, non costruisce piani strutturati.

**Per colmare**:
- Implementare Observer LLM-based (descrivi risultato atteso, chiedi a LLM se matcha)
- Implementare Reflect con Re-plan phase (se fallisce, rivedi piano e riprova)
- Aggiungere plannedTool con rationale/expected/fallback
- Vedi piano `piano-finale-aleph-26-apr.md` W4-W5 per dettagli

### 6. GNN Link Prediction ⚠️

**Claim**: Non esplicitamente nel README, ma `gnn_adapter.go` esiste.

**Realtà**: Il GNN adapter è presente ma non è chiaro se il modello sia addestrato o usato in produzione.

**Per colmare**:
- Aggiungere test di integrazione
- Documentare stato (sperimentale/non attivo)
- Opzionale: rimuovere o completare

---

## Cosa è stato corretto nel README

| Modifica | Dettaglio |
|----------|-----------|
| Rimosso "Protocollo Genesis" | Sostituito con "in fase di sviluppo sperimentale" |
| Rimosso `XGBoost` e `LangChain` da architettura | Sostituito con `GBM`, `PyTorch (fallback)` |
| Rimosso `LangChain` da stack table | Allineato alla realtà |
| Rettificato "modelli linguistici" nell'ensemble | Specificato: Prophet, GBM, sentiment NLP, agenti dispatch |

---

## Raccomandazioni

1. **Mantenere il README onesto**: ogni feature menzionata deve avere un codice corrispondente o un ticket aperto. Il README è il primo contatto — se mente, mina la fiducia.
2. **Sync periodico**: ogni volta che si aggiunge/rimuove una dipendenza maggiore, aggiornare l'architecture diagram e lo stack table.
3. **Labels "sperimentale" e "beta"**: usare per feature non complete. Il README attuale ha un'onesta sezione "Stato del progetto" — rafforzarla per ogni feature gap.
4. **Roadmap realistica**: la roadmap indicativa è vaga ma onesta. Considerare di aggiungere milestone con ticket collegati.
5. **La stampa "Ensemble (Prophet/XGBoost) loaded"** in `nlp/main.py:117` va corretta — produce un falso positivo.

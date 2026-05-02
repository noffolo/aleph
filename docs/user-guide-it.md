# Guida Utente — Aleph-v2

> **Versione:** 2.0.0 · **Ultimo aggiornamento:** Aprile 2026 · **Lingua:** Italiano

Benvenuto in Aleph-v2, la piattaforma di intelligenza predittiva multi-agente. Questa guida ti aiuta a usare l'applicazione partendo dai concetti base fino ai workflow avanzati.

---

## Indice

1. [Primi passi](#1-primi-passi)
2. [Concetti chiave](#2-concetti-chiave)
3. [Interfaccia terminale](#3-interfaccia-terminale)
4. [Workflow principali](#4-workflow-principali)
5. [Gestione progetti](#5-gestione-progetti)
6. [Agenti e competenze](#6-agenti-e-competenze)
7. [Tool e sandbox](#7-tool-e-sandbox)
8. [Chat e analisi dati](#8-chat-e-analisi-dati)
9. [Ingestione dati](#9-ingestione-dati)
10. [Notifiche e webhook](#10-notifiche-e-webhook)
11. [Sicurezza e privacy](#11-sicurezza-e-privacy)
12. [Troubleshooting](#12-troubleshooting)

---

## 1. Primi passi

### Accesso

Apri il browser all'indirizzo del tuo server Aleph (per esempio `http://localhost:5173` in locale). Ti trovi davanti a un terminale interattivo, l'interfaccia principale dell'applicazione.

### Primo login

All'avvio devi inserire una API key. Se non ne hai una, chiedi all'amministratore di sistema di crearla tramite `AuthService > CreateApiKey`. La chiave ti viene mostrata una sola volta: copiala e conservala in un password manager.

### Navigazione rapida

| Scorciatoia | Azione |
|-------------|--------|
| `Cmd+K` (Mac) / `Ctrl+K` (Win) | Apri la palette comandi |
| `↑` / `↓` | Scorri la cronologia dei comandi |
| `Tab` | Autocompletamento comandi |
| `Esc` | Chiudi pannelli e modali |

---

## 2. Concetti chiave

### Progetto

Un progetto è un contenitore di dati, agenti, tool e competenze. Ogni progetto ha la sua directory separata (`raw/`, `ontologies/`, `agents/`, `skills/`). Puoi passare da un progetto all'altro senza uscire dall'interfaccia.

### Agente

Un agente è un'istanza di IA configurata con un provider LLM (per esempio Ollama), un modello (per esempio llama3) e un system prompt che definisce il suo comportamento. Ogni agente può avere una o piú competenze (skill).

### Competenza (Skill)

Una competenza raggruppa uno o piú tool per scopi specifici. Per esempio, la competenza "Analisi Finanziaria" puó includere tool per il sentiment analysis, il fetching di dati di mercato e la generazione di report.

### Tool

Un tool è un pezzo di codice eseguibile che svolge un compito preciso: importare un CSV, analizzare il sentiment di un testo, eseguire una query su DuckDB. I tool girano dentro una sandbox isolata.

### Ontologia

L'ontologia descrive la struttura dei dati del progetto (tabelle, colonne, relazioni). Puoi leggerla, modificarla o generarla automaticamente dal database DuckDB.

### Ciclo PAORA

Ogni interazione con l'agente segue il ciclo decisionale Plan > Act > Observe > Reflect > Admit. Questo garantisce che l'agente pianifichi, esegua, verifichi e ammetta i risultati in modo autonomo.

---

## 3. Interfaccia terminale

### Layout

L'interfaccia é divisa in tre zone:

- **Header in alto**: nome del progetto attivo, nome dell'agente selezionato, stato connessione
- **Area chat centrale**: messaggi dell'utente e risposte dell'agente, con rendering inline di tabelle, grafici e tool
- **Pannello laterale (SlideOver)**: si apre a destra per form complessi (creazione agente, upload file, impostazioni)

### Comandi slash

Digita `/` per vedere l'elenco dei 16 comandi built-in:

| Comando | Cosa fa |
|---------|---------|
| `/help` | Mostra tutti i comandi disponibili |
| `/clear` | Pulisce la sessione di chat |
| `/model` | Cambia modello LLM |
| `/agent` | Elenca o cambia agente attivo |
| `/tool` | Gestione tool (installa, elenca, controlla salute) |
| `/skills` | Mostra le competenze dell'agente corrente |
| `/status` | Stato della connessione e dei servizi |
| `/export` | Esporta la conversazione in Markdown |
| `/diagnose` | Esegue una diagnostica rapida |
| `/theme` | Cambia tema chiaro o scuro |
| `/debug` | Attiva/disattiva modalitá debug |

### Effetti visivi

Puoi attivare effetti scanline, flicker e glow dal menu impostazioni (`/theme`). Questi sono puramente estetici e non influenzano le funzionalitá.

---

## 4. Workflow principali

### 4.1 Analisi dati con chat

1. Seleziona un progetto dall'header
2. Digita una domanda nel terminale: `SHOW ME sales BY month`
3. L'agente pianifica l'azione (Plan), esegue la query DuckDB (Act), mostra i risultati (Observe)
4. Se il risultato é buono, l'agente lo conferma (Admit). Altrimenti, riprova (Reflect)
5. Puoi continuare la conversazione con domande di approfondimento

### 4.2 Importazione dati

1. Apri il pannello laterale con `Cmd+K` > "New Data Source"
2. Scegli la sorgente: CSV, JSON, API URL, Google Sheets, RSS, GitHub
3. Configura i parametri (per esempio, URL del file o query SQL)
4. Avvia il task di ingestione
5. Controlla il progresso con `/status` o nel pannello Ingestion

### 4.3 Creazione di un agente personalizzato

1. Digita `/agent` > "Create new agent"
2. Compila il form nel pannello laterale:
   - Nome (per esempio, "Market Analyst")
   - Provider LLM (Ollama, OpenAI)
   - Modello (llama3, gpt-4)
   - System prompt (istruzioni di comportamento)
   - Competenze da assegnare
3. Salva e attiva l'agente

### 4.4 Registrazione di un nuovo tool

1. Digita `/tool` > "Register tool"
2. Inserisci nome, descrizione e codice Go del tool
3. Il sistema esegue uno scan di sicurezza (SecurityScanner) prima della registrazione
4. Il tool diventa disponibile per tutti i progetti

---

## 5. Gestione progetti

### Creare un progetto

```
Cmd+K > New Project
```

Inserisci nome e descrizione. Il sistema crea automaticamente la struttura di directory:
```
data/projects/<nome-progetto>/
├── raw/           # File sorgenti
├── ontologies/    # Definizioni ontologiche
├── agents/        # Configurazioni agenti
└── skills/        # Configurazioni competenze
```

### Cambiare progetto

Clicca sul nome del progetto nell'header e seleziona un altro dall'elenco, oppure usa:
```
/project <nome-progetto>
```

### Ontologia

Per vedere la struttura dati corrente:
```
/ontology show
```

Per generarla automaticamente dal database:
```
/ontology emerge
```

Per modificarla manualmente:
```
/ontology edit
```

---

## 6. Agenti e competenze

### Switchare agente

```
/agent <nome-agente>
```

Oppure usa la palette `Cmd+K` > "Switch Agent".

### Competenze predefinite

| Competenza | Tool inclusi | Uso tipico |
|------------|--------------|------------|
| Query Dati | `execute_query`, `get_data_stats` | Esplorazione database |
| Analisi Sentiment | `analyze_sentiment` | Opinion mining su testi |
| Previsione | `stream_predictions` | Forecasting time series |
| Ingestione | `csv_ingester`, `json_ingester` | Importazione dati |
| CodeFlow | `code_metrics`, `dependency_graph` | Analisi codice |

### Assegnare competenze

Nel form di modifica agente (SlideOver > AgentForm), spunta le competenze che vuoi attivare. L'agente le userá automaticamente quando rileva un task compatibile.

---

## 7. Tool e sandbox

### Eseguire un tool

Puoi chiamare un tool direttamente dalla chat:
```
Run csv_ingester on data/raw/sales.csv
```

Oppure via endpoint REST:
```bash
curl -X POST http://localhost:8080/api/v1/tools/call \
  -H "X-Aleph-Api-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"tool_id": "tool_csv_001", "parameters": {"file_path": "sales.csv"}}'
```

### Sicurezza sandbox

I tool girano in un ambiente isolato con queste restrizioni:

- Timeout di esecuzione configurabile
- Lista di comandi permessi (14 comandi)
- Flag bloccati (`-rf`, `--force`, `--no-dry-run`, `-exec`, `--allow-root`)
- Regex per bloccare metacaratteri shell
- Nessun accesso di rete (`network_mode: none`)
- Filesystem in sola lettura (`read_only: true`)

### Health check dei tool

Controlla lo stato di tutti i tool:
```
/tool health
```

Storico di un tool specifico:
```
/tool health <tool-id>
```

---

## 8. Chat e analisi dati

### Chat streaming

La chat usa SSE (Server-Sent Events) per lo streaming in tempo reale. Vedi la risposta dell'agente apparire parola per parola, senza attesa.

### Tool call inline

Quando l'agente decide di usare un tool, vedi un riquadro inline nella chat con:
- Nome del tool
- Parametri passati
- Output restituito
- Tempo di esecuzione

### Conferma azioni

Per azioni distruttive (delete, update massivo), l'agente chiede conferma:
```
⚠️ Azione richiesta: delete table 'sales_2025'
Conferma? (yes/no)
```

### Esportare conversazioni

```
/export
```

La conversazione viene scaricata come file Markdown con timestamp.

---

## 9. Ingestione dati

### Fonti supportate

| Fonte | Configurazione | Esempio |
|-------|----------------|---------|
| CSV | Path file locale | `./data/raw/sales.csv` |
| JSON | URL o path | `https://api.example.com/data.json` |
| API | Endpoint + headers | `GET /api/v1/users` |
| Google Sheets | ID spreadsheet + range | `1BxiMV.../Sheet1!A1:D10` |
| RSS | URL feed | `https://news.ycombinator.com/rss` |
| GitHub | Repo + path | `owner/repo/data/` |
| Email | IMAP config | `imap.gmail.com:993` |

### Monitorare l'ingestione

Durante un task di ingestione attivo:
```
/status
```

Mostra:
- Percentuale di completamento
- Righe elaborate / totali
- Errori eventuali
- Log in tempo reale

---

## 10. Notifiche e webhook

### Configurare un webhook

1. Apri il pannello Notifiche (SlideOver)
2. Aggiungi un canale di tipo webhook
3. Inserisci URL e secret (opzionale)
4. Scegli gli eventi da notificare (ingestione complete, tool failure, health alert)

### Invio manuale

Puoi inviare un webhook di test:
```
/notify send https://hooks.example.com/aleph {"event": "test"}
```

---

## 11. Sicurezza e privacy

### Chiavi API

- Le API key sono hashate con SHA-256 prima del salvataggio
- Cifrate a riposo con AES-256-GCM
- Mostrate in plaintext solo al momento della creazione
- Revocabili in qualsiasi momento

### Dati sensibili

- Nessun log contiene API key o password
- I parametri dei tool sono validati con regex
- SQL injection é impossibile grazie a query parametrizzate
- Il codice dei tool é scansionato prima dell'esecuzione

### Audit

Ogni operazione di scrittura (create, update, delete) é registrata con:
- Timestamp
- ID progetto
- Azione eseguita
- Dettagli JSON dell'operazione

Puoi consultare l'audit log tramite l'API REST o il pannello amministrazione.

---

## 12. Troubleshooting

### L'agente non risponde

1. Controlla `/status` — il servizio LLM (Ollama) é attivo?
2. Verifica che l'agente abbia un provider e modello validi
3. Se Ollama é down, l'agente passa in modalitá degradata (heuristic planning)

### Errore "tool not found"

1. Verifica che il tool sia registrato: `/tool list`
2. Controlla che la competenza dell'agente includa quel tool
3. Se é un tool MCP, verifica la connettivitá: `/mcp status`

### Query troppo lenta

1. Controlla le statistiche della tabella: `GET /api/v1/query/data-stats`
2. Verifica se ci sono indici mancanti su colonne filtrate
3. Per tabelle molto grandi, usa `LIMIT` nelle query

### Problemi di autenticazione

1. Verifica che l'header `X-Aleph-Api-Key` sia presente
2. Controlla che la chiave non sia scaduta o revocata
3. Per problemi di CORS, verifica che l'origine sia in `CORS_ALLOWED_ORIGINS`

### Docker Healthcheck fallito

| Servizio | Comando diagnostico |
|----------|---------------------|
| Backend | `docker compose exec aleph-backend wget -qO- http://localhost:8080/readyz` |
| NLP | `docker compose exec aleph-nlp-sidecar python -c "import grpc; print('ok')"` |
| DB | `docker compose exec aleph-db pg_isready -U postgres` |

---

## Glossario rapido

| Termine | Significato |
|---------|-------------|
| **Agente** | Istanza IA con modello LLM, prompt e competenze |
| **Competenza (Skill)** | Raggruppamento di tool per uno scopo |
| **Tool** | Codice eseguibile in sandbox |
| **Progetto** | Contenitore isolato di dati e configurazioni |
| **Ontologia** | Schema dei dati del progetto |
| **PAORA** | Ciclo Plan > Act > Observe > Reflect > Admit |
| **SSE** | Server-Sent Events, streaming HTTP |
| **MCP** | Model Context Protocol, per tool esterni |

---

## Altre guide

- [`docs/user-guide-en.md`](./user-guide-en.md) — English user guide
- [`docs/api-reference.md`](./api-reference.md) — API reference completa
- [`docs/deployment-guide.md`](./deployment-guide.md) — Guida al deployment

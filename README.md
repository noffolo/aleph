# Aleph-v2

**Sistema di Decision Intelligence che trasforma dati eterogenei, segnali ambigui e variabili incerte in scenari espliciti, probabilistici e azionabili.**

---

## Cos'è Aleph

Il problema non è la mancanza di dati. È capire cosa significano, insieme, in un momento specifico.

Aleph è un sistema di Decision Intelligence: un ambiente in cui dati, modelli predittivi e agenti AI collaborano per rendere leggibile la complessità. Non produce risposte automatiche. Produce una mappa migliore dell'incertezza — così che le persone possano decidere con più consapevolezza.

In pratica, Aleph aiuta a rispondere a domande che i dashboard tradizionali non riescono a fare:

- Cosa significano questi dati, nel loro insieme?
- Quali scenari sono plausibili da qui in avanti?
- Quali segnali stanno cambiando, e quanto velocemente?
- Dove il rischio è più alto? Dove c'è margine di manovra?
- Dove serve ancora giudizio umano?

> Aleph è uno dei punti dello spazio che contengono tutti i punti.

---

## A cosa serve

Aleph è pensato per contesti con molte informazioni, segnali deboli e bisogno di sintesi. Può supportare:

- analisi di trend e scenari futuri
- confronto tra ipotesi diverse
- monitoraggio di segnali da fonti eterogenee
- letture strategiche di un contesto
- ricerca, pianificazione e analisi strutturata
- prototipazione di sistemi predittivi e decisionali
- sperimentazione con workflow basati su agenti AI

Aleph non decide al posto di nessuno. Aiuta a vedere meglio il contesto in cui si deve decidere.

---

## Per chi è pensato

### Utenti non tecnici

Aleph si può usare come strumento esplorativo: si osservano scenari, si leggono sintesi, si valutano possibili evoluzioni. L'interfaccia è pensata per rendere accessibili anche situazioni complesse, senza richiedere conoscenze di programmazione.

### Analisti, ricercatori, strategist

Aleph offre un ambiente di esplorazione per dati, scenari, ipotesi e simulazioni. Il sistema combina modelli statistici, machine learning, modelli linguistici e workflow decisionali in un unico spazio di lavoro.

### Sviluppatori

Aleph è una piattaforma tecnica modulare: backend in Go, frontend React/TypeScript, componenti Python per l'intelligence layer, orchestrazione Docker. È possibile estendere il sistema con nuovi strumenti, agenti e skill — sia importandoli da fonti esterne, sia costruendoli direttamente.

---

## Le capacità agentiche

Una delle caratteristiche centrali di Aleph è la sua architettura estendibile tramite agenti e strumenti.

Gli agenti possono analizzare, sintetizzare, proporre interpretazioni o attivare procedure controllate. Possono operare in modo autonomo o collaborativo, all'interno di workflow configurabili.

Gli strumenti (*tools*) e le skill possono essere:

- **importati** da fonti esterne, librerie o sistemi già esistenti;
- **creati da sviluppatori umani** attraverso l'interfaccia o direttamente nel codice;
- **proposti dal sistema** (in fase di sviluppo sperimentale), con meccanismi di revisione e approvazione umana prima dell'attivazione.

---

## Funzionalità principali

### Scenari predittivi

Il sistema trasforma dati grezzi in scenari leggibili. Uno scenario non è una certezza: è una possibile evoluzione costruita sulla base dei dati disponibili, dei modelli usati e delle ipotesi configurate. Ogni scenario porta con sé un livello di confidenza e le assunzioni su cui si basa.

### Modelli ensemble

Aleph combina più approcci analitici: modelli statistici (Prophet), machine learning (GBM), analisi del sentiment tramite NLP, e agenti AI per il dispatch di strumenti. L'idea è che nessun modello sia sufficiente da solo — più prospettive riducono il rischio di interpretazioni parziali o distorte.

### Workspace adattivo

L'interfaccia è pensata come spazio di lavoro, non come dashboard. L'obiettivo non è mostrare dati: è aiutare a orientarsi tra informazioni, scenari e azioni possibili.

### Architettura resiliente

Il sistema è progettato per essere modulare e resistente agli errori. I componenti possono essere isolati, aggiornati o sostituiti senza compromettere il sistema nel suo insieme.

---

## Architettura

```
Utente
  │
  ▼
Interfaccia Web
React · TypeScript · Vite · Tailwind CSS
  │
  ▼
Backend
Go · Connect RPC
  │
  ├── Persistenza dati
  │   DuckDB
  │
  ├── Intelligence layer
  │   Python · ONNX · Prophet · GBM · PyTorch (fallback)
  │
  └── Orchestrazione
      Docker · Docker Compose
```

### Stack tecnologico

| Area | Tecnologia |
|---|---|
| Backend | Go, Connect RPC |
| Database / analisi locale | DuckDB |
| Intelligence | Python, ONNX, Prophet, GBM, PyTorch (fallback) |
| Frontend | React, TypeScript, Vite, Tailwind CSS |
| Orchestrazione | Docker, Docker Compose |

---

## Installazione

### Requisiti

- Git
- Docker e Docker Compose

Per lavorare direttamente sul codice: Go (backend), Node.js (frontend), Python (intelligence layer).

### Avvio con Docker Compose

```bash
git clone https://github.com/noffolo/aleph.git
cd aleph
cp .env.example .env
```

Configura le variabili nel file `.env`, poi:

```bash
docker compose up --build -d
```

L'interfaccia sarà disponibile su `http://localhost:5173`.

---

## Configurazione

La configurazione avviene tramite file `.env`. Il file `.env.example` contiene un esempio completo delle variabili disponibili, che includono chiavi API, modelli AI, porte dei servizi, impostazioni CORS, rate limiting, database, logging e ambiente di esecuzione.

Non committare mai `.env` nel repository.

---

## Sicurezza

Aleph può gestire chiavi API, richieste verso modelli AI e dati potenzialmente sensibili. Alcune regole essenziali:

- non salvare segreti nel codice;
- non pubblicare chiavi API;
- usare configurazioni distinte tra sviluppo e produzione;
- limitare l'accesso ai servizi esposti;
- controllare i log prima di condividerli.

Per segnalare una vulnerabilità, non aprire una issue pubblica. Consulta `SECURITY.md` e segui le istruzioni per la segnalazione responsabile.

---

## Uso per sviluppatori

### Backend

```bash
go mod download
go build ./...
go test ./...
```

### Frontend

```bash
cd frontend
npm install
npm run dev    # sviluppo
npm run build  # produzione
```

### Docker

```bash
docker compose up --build   # ricostruisce tutto
docker compose down         # ferma i servizi
```

### Struttura del repository

```
.
├── frontend/              # Interfaccia web
├── internal/              # Codice backend Go
├── nlp/                   # Componenti NLP / intelligence
├── aleph_tools/           # Strumenti di supporto
├── docs/                  # Documentazione
├── migrations/            # Migrazioni dati
├── .github/               # Workflow GitHub
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── main.go
├── README.md
├── SECURITY.md
└── LICENSE
```

---

## Interpretare i risultati

Ogni output di Aleph va letto nel suo contesto. Prima di usare un risultato per prendere una decisione, vale la pena chiedersi:

- Da quali dati viene? Sono reali o sintetici?
- Qual è il livello di confidenza dichiarato?
- Quali ipotesi ha fatto il modello?
- In quale finestra temporale si collocano le analisi?

Una previsione con alta confidenza non è una garanzia. Una previsione con bassa confidenza non è inutile: spesso segnala incertezza reale, mancanza di dati o instabilità del fenomeno osservato — informazioni utili in sé.

---

## Stato del progetto

Aleph-v2 è in fase **beta**.

Alcune funzionalità possono cambiare. Le API possono essere modificate. L'interfaccia può evolvere. Le analisi prodotte dal sistema devono essere considerate come supporto esplorativo, non come raccomandazioni definitive. Qualsiasi decisione importante richiede giudizio umano.

### Roadmap indicativa

- stabilizzare le API e i workflow degli agenti
- migliorare documentazione ed esempi pratici
- aggiungere dataset e casi d'uso concreti
- rafforzare test e CI
- migliorare osservabilità, logging e metriche

---

## Contribuire

I contributi sono benvenuti in molte forme: segnalazioni di bug, miglioramenti alla documentazione, nuovi test, correzioni di sicurezza, esempi, workflow, interfaccia.

Prima di aprire una pull request:

1. controlla le issue esistenti;
2. descrivi chiaramente il problema o la proposta;
3. tieni le modifiche piccole e leggibili;
4. aggiungi test quando necessario;
5. aggiorna la documentazione se il comportamento del sistema cambia.

---

## Licenza

Distribuito con licenza **GPL-3.0**. Vedi `LICENSE` per i dettagli.

---

*Aleph nasce per esplorare una domanda semplice ma ambiziosa: come trasformare il rumore informativo in comprensione utile. Non promette decisioni automatiche né previsioni infallibili. Il suo obiettivo è costruire uno spazio in cui dati, modelli e persone possano collaborare per leggere meglio la complessità.*

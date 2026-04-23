# Piano Redesign: Aleph Terminal UI

## Visione

Trasformare Aleph da una web app dashboard-style a una **terminal-first intelligence platform**: il copilot diventa la home, l'interfaccia è CLI-like (stile opencode/warp/gemini-cli) con "super poteri" — slash commands, tab completion, markdown rendering, multiline editing — e l'Explorer si fonde col copilot in un sistema integrato dove puoi interrogare dati, richiedere visualizzazioni, e navigare ontologie tutto dalla stessa interfaccia.

Il layout è **ibrido tipo Warp**: sidebar sottile con icone + status indicator, terminale/area chat che prende 80%+ dello schermo.

### Principio: Meraviglia Controllata

L'interfaccia deve **sorprendere e meravigliare** pur restando semplice e funzionale. La regola:

- **Idle state**: quieto, monospace, quasi monastico. Zero rumore visivo inutile.
- **Active state**: l'informazione appare con intenzione — non confetti, ma precisione. Una barra di probabilità che si riempie, una predizione che si materializza, uno sparkline che racconta un trend.
- **Dati spiegati**: ogni visualizzazione porta con sé una **narrativa** — una riga che spiega cosa significano i numeri. Le previsioni non mostrano solo indici, li **interpretano**.

**Linguaggio narrativo**: italiano (coerente con l'interfaccia attuale). Il sistema spiega i dati in linguaggio naturale: "Mercato volatileo per il settore tech. Aziende con revenue >10M mostrano stabilità superiore del 37%."

---

## Fase 1: Design System — Terminal CLI Aesthetic (Dual Theme)

### 1.1 Token e Colori

Sostituire l'attuale design-tokens.json con un sistema dual-theme (dark/default + light):

```json
{
  "dark": {
    "bg":           { "primary": "#0a0a0f", "surface": "#12121a", "elevated": "#1a1a26" },
    "border":       { "default": "#2a2a3a", "subtle": "#1e1e2e" },
    "text":         { "primary": "#e4e4ef", "muted": "#6b6b80", "dim": "#3d3d50" },
    "accent":       { "blue": "#4f8fff", "green": "#34d399", "amber": "#fbbf24", "red": "#f87171", "violet": "#a78bfa" },
    "terminal":     { "bg": "#0d0d14", "green": "#4ade80", "prompt": "#4f8fff" }
  },
  "light": {
    "bg":           { "primary": "#fafafa", "surface": "#ffffff", "elevated": "#f5f5f5" },
    "border":       { "default": "#e5e5e5", "subtle": "#f0f0f0" },
    "text":         { "primary": "#0f172a", "muted": "#64748b", "dim": "#94a3b8" },
    "accent":       { "blue": "#2563eb", "green": "#16a34a", "amber": "#d97706", "red": "#dc2626", "violet": "#7c3aed" },
    "terminal":     { "bg": "#ffffff", "green": "#15803d", "prompt": "#2563eb" }
  }
}
```

### 1.2 Tipografia

- **Monospace-first**: il font primario per messaggi e output è JetBrains Mono (già nel token). Per label UI brevi usare Inter.
- Gerarchia: `font-mono` per tutto il contenuto del terminale, `font-sans` solo per badge/tiny header.
- Prompt symbol: `ᐅ` (unicode U+1440) per dark, `>` per light.

### 1.3 Componenti Base — Redesign

| Componente | Ora | Nuovo |
|---|---|---|
| Card | `rounded-[32px] shadow-2xl` bianche | Pannello `border` sottile, sfondo surface, angoli `rounded-lg`, monospace |
| Button primary | `bg-blue-600 rounded-2xl shadow-lg` | `bg-accent-blue text-bg-primary font-mono px-4 py-2 rounded-lg` |
| Badge | `bg-gray-100 rounded-full text-[10px]` | `border border-border px-2 py-0.5 font-mono text-xs` |
| Table/List | AlephTable custom | Output terminale: righe con `│` separatori, header bold mono |
| Input | `rounded-3xl shadow-xl text-lg` | Terminal block: `bg-terminal font-mono p-3 rounded-lg border` |
| Modal | `rounded-[32px] shadow-2xl` overlay blur | Panel slide-in dal basso o drawer laterale, stile terminale |

### 1.4 Nuovi Componenti da Creare

- **`TerminalPrompt`** — input multiline con prompt symbol, syntax highlighting per slash commands, history (↑/↓)
- **`TerminalOutput`** — renderizzatore di output strutturato (Markdown, tabelle ASCII, JSON, sparklines, narrative blocks)
- **`StatusBar`** — barra fondo-schermo con indicatori (modello attivo, progetto, Ollama/NLP status)
- **`SlashCommandBar`** — autocomplete dropdown per `/commands`
- **`DataPanel`** — pannello laterale/inline per visualizzazioni tabella/mappa/timeline/graph
- **`SparklineRenderer`** — componente che converte serie numeriche in sparkline Unicode inline (▂▃▅▇█▇▅▃▂)
- **`PredictionCard`** — card per predizioni con barre di probabilità a gradiente, animazione fill, e narrativa AI
- **`NarrativeBlock`** — blocco di spiegazione AI in italiano sotto ogni visualizzazione dati
- **`HealthStrip`** — dashboard strip inline per indici di salute dati con barre e indicatori

### 1.5 Meraviglia Controllata — Micro-interazioni e Animazioni

Filosofia: zero rumore a riposo, impatto quando serve. Ogni animazione ha uno scopo comunicativo.

| Elemento | Animazione | Durata | Easing |
|---|---|---|---|
| Messaggio AI appare | Typing animation con cursore Blink, poi fade-in blocco | 400ms typing + 200ms fade | ease-out |
| Slash command digitata | Highlight immediato del comando, autocomplete fade-in | 150ms | ease-in |
| DataPanel apre | Slide-in da destra con spring | 300ms | spring(1, 80, 10) |
| Barra probabilità | Animazione fill da 0% a valore | 800ms | ease-out |
| Cambio tema | Crossfade smooth | 200ms | linear |
| Errore | Shake orizzontale 4px | 200ms | ease-in-out |
| Predizione stream (Oracle) | Card appaiono una alla volta con stagger | 150ms per card | ease-out |
| Sparkline appare | Disegno sequenziale da sinistra | 500ms | ease-in-out |
| Tool call confermata | Flash border accent-blue + checkmark fade | 300ms | ease-out |

CSS Implementation (Tailwind + custom):
- Tutte le animazioni definite come `@keyframes` in `index.css`
- Classi utility: `animate-in fade-in`, `animate-in slide-in-from-right`, `animate-in slide-in-from-bottom`
- Il cursore Blink del terminale: `animate-pulse` customizzato (700ms, opacity 0→1)
- Spring animation per DataPanel: CSS `cubic-bezier(0.34, 1.56, 0.64, 1)` come approximation

---

## Fase 2: Layout Rewrite — Warp-style Ibrido

### 2.1 Architettura Layout

```
┌──────┬────────────────────────────────────────────────────┐
│      │  ᐅ Aleph copilot                    [⌘K]          │
│  E   ├────────────────────────────────────────────────────┤
│  X   │                                                    │
│  P   │  terminale conversazione (chat + output)           │
│  L   │  - messaggi utente con stile terminale             │
│  R   │  - risposte AI con Markdown renderizzato           │
│  E   │  - /commands inline                                │
│      │  - viste dati inline (tabelle, grafi)              │
│  ──  ├────────────────────────────────────────────────────┤
│  S   │  ᐅ digita un comando o fai una domanda...          │
│  I   │  [/explore · /predict · /agent · /help]            │
│  D   └────────────────────────────────────────────────────┘
│  E   │  ● ollama  ● nlp  │  project: xxx  │  model: yy  │
│  B   │  status bar                                        │
└──────┴────────────────────────────────────────────────────┘
```

- Sidebar: ~48px (icone) expandabile a ~200px con label
- L'header sparisce (sostituito da StatusBar + info nel prompt)
- Main area = 100% terminale conversazionale
- Il Copilot è la **home page** (default tab)

### 2.2 Sidebar Redesign

La sidebar passa da `w-72` con label a una **thin icon bar** (~48px) che si espande on hover/shortcut:

- Icone: Explorer, Copilot (home/default), Oracle, Library, Data Sources, Ontologies, Agents, Skills, Tools, Settings
- Ogni icona ha tooltip e badge di stato
- Bottom: project selector + status dots
- Il tab "Explorer" diventa una funzione accessibile anche via `/explore <entity>` dal copilot

### 2.3 App.tsx Refactor

- Eliminare il pattern `switch(activeTab)` con componenti isolati
- Sostituire con un layout system:
  ```tsx
  <div class="flex h-screen bg-terminal-bg">
    <Sidebar />           {/* thin icon bar */}
    <TerminalArea>        {/* main = copilot + output */}
      <TerminalMessages />
      <TerminalPrompt />
    </TerminalArea>
    <DataPanel />         {/* slide-in panel per viste dati */}
    <StatusBar />         {/* fondo */}
  </div>
  ```

---

## Fase 3: Copilot come Home — Fusione con Explorer

### 3.1 Copilot = Home

Il copilot diventa la vista di default. Tutte le operazioni passano per il terminale:

- `/explore <entity>` → popola una vista Explorer nel DataPanel a destra
- `/explore <entity> as map` → mostra mappa
- `/explore <entity> as graph` → mostra grafo
- `/predict` → Oracle predictions inline nel terminale
- `/agent list` → lista agenti nel terminale (tabella ASCII)
- `/agent create` → form inline o step-by-step nel terminale
- `/ontology` → editor ontologia in DataPanel
- `/datasource add` → wizard inline
- `/library` → lista asset nel terminale + preview in DataPanel

### 3.2 Explorer Integration nel Copilot

Quando l'utente chiede al copilot "mostrami i dati di X":
1. Il copilot chiama l'API `executeQuery`
2. L'output appare come **tabella ASCII** nel terminale (stile `psql`, `mysql` CLI)
3. Un pulsante "Apri vista" apre il DataPanel con la visualizzazione completa (tabella interattiva, mappa, timeline, grafo)
4. Livelli di sintesi:
   - **Quick** — 3-5 righe di summary nel terminale
   - **Table** — output ASCII tabellare
   - **Full** — apre DataPanel con vista completa

### 3.3 Slash Commands System

Creare un engine di comandi:

```typescript
const COMMANDS = {
  '/explore':  { description: 'Esplora entità ontologica', args: '<entity>', action: 'EXPLORE' },
  '/predict':  { description: 'Esegui predizione Oracle', args: '[entity]', action: 'PREDICT' },
  '/agent':    { description: 'Gestione agenti', subcommands: ['list', 'create', 'delete'] },
  '/ontology': { description: 'Modellazione ontologia', subcommands: ['edit', 'emerge', 'save'] },
  '/data':     { description: 'Gestione data source', subcommands: ['list', 'add', 'run'] },
  '/library':  { description: 'Biblioteca asset', subcommands: ['list', 'upload', 'open'] },
  '/skill':    { description: 'Esegui skill', args: '<skill-id>' },
  '/tool':     { description: 'Esegui tool', args: '<tool-id>' },
  '/settings': { description: 'Impostazioni' },
  '/help':     { description: 'Mostra comandi disponibili' },
  '/clear':    { description: 'Pulisci conversazione' },
  '/theme':    { description: 'Cambia tema (dark/light)', args: '[dark|light]' },
}
```

- Tab completion per comandi e argomenti
- `⌘K` rimane come quick-switch globale

### 3.4 TerminalPrompt Component

```
┌──────────────────────────────────────────────────────┐
│ ᐅ /explore companies                                │
│   ↓ autocomplete popup                               │
│   /explore  /explore <entity> as map                  │
│   /explore companies · /explore people                │
└──────────────────────────────────────────────────────┘
```

Features:
- Multiline (Shift+Enter per newline)
- History (↑/↓)
- Tab completion
- Slash command detection + autocomplete
- Syntax highlighting minimo per `/commands`

### 3.5 TerminalOutput — Formattazione Risposte

Il copilot renderizza output in diversi formati:

| Tipo | Rendering |
|---|---|
| Testo AI | Markdown con syntax highlight, typing animation |
| Dati tabellari | Tabella ASCII con bordi `┌─┬─┐`, allineamento |
| Predizioni | PredictionCard con barre gradiente, animazione fill, narrativa |
| Serie temporali | Sparkline Unicode inline (▂▃▅▇█▇▅▃▂) |
| Codice SQL | Blocco monospace con syntax highlight |
| Status/Info | `✓` verde, `✗` rosso, `◉` blue per indicatori |
| Errori | `✗ Errore: messaggio` in rosso monospace + shake animation |
| Tool calls | `⚡ tool_name(args)` collapsible, flash on confirm |
| Salute dati | HealthStrip con barre e indicatori colorati |

### 3.6 Predictions Come Meraviglia

Le predizioni Oracle sono il momento "wow" dell'interfaccia. Ogni predizione si materializza con:

```
◆ STABILITY_INDEX    ▓▓▓▓▓▓▓▓▓░ 92%  ← barra gradiente verde solido
◆ MARKET_RISK        ▓▓▓▓▓░░░░░ 48%  ← barra gradiente ambra, pulse animation
⚠ ACTION_REQUIRED    ▓▓▓▓▓▓▓▓░░ 81%  ← barra gradiente rossa, badge ⚠

  "Mercato volatileo per il settore tech. Le aziende con 
   revenue >10M mostrano stabilità superiore del 37%."
```

Implementazione:
- Barre di probabilità con **gradiente** (non colore piatto): il riempimento transiziona da accent-color chiaro a accent-color scuro
- Animazione: fill da sinistra a destra, 800ms ease-out, stagger 150ms tra predizioni
- `▶`/`◀` per espandere/comprimere la narrativa
- Il testo narrativo è generato dalla spiegazione AI dell'Oracle, in italiano

### 3.7 Sparklines Inline nei Dati

Quando il terminale mostra dati tabellari, le colonne numeriche con trend integrato:

```
│ revenue │ 4.2M  ▂▃▅▇█▇▅▃▂  │ ↑12% │
│ churn   │ 2.1%  ▇▅▃▂▂▃▅▇█  │ ↓8%  │
│ users   │ 15K   ▃▃▄▅▆▇██▇  │ ↑23% │
```

- Sparkline renderizzate da array numerici → Unicode block characters (▁▂▃▄▅▆▇█)
- Freccia trend ↑/↓ con colore verde/rosso
- Attivazione automatica quando il dataset ha ≥5 punti storici
- Clic sulla riga → apre DataPanel con vista completa del trend

### 3.8 Narrative Layer — Dati Spiegati

Ogni visualizzazione significativa è seguita da un **NarrativeBlock** — una o due righe che interpretano i dati:

- Dopo tabella ASCII: `▸ 42 aziende, revenue medio 3.1M, range 0.2M-89M`
- Dopo sparkline: `▸ Trend positivo: +12% rispetto al periodo precedente`
- Dopo HealthStrip: `▸ 3 su 5 indici eccellenti, attenzione alla freschezza dei dati`
- Dopo mappa (in DataPanel): `▸ Cluster concentrato in Nord Italia, 3 outlier al sud`
- Dopo grafo: `▸ 12 nodi centrali con forte connessione, 4 periferici isolati`

Il NarrativeBlock è stilizzato diversamente dal testo AI:
- Font: `font-mono`, colore `text-muted`, prefisso `▸`
- Non animato (appare subito, è contesto, non rivelazione)
- In italiano

### 3.9 HealthStrip — Indici di Salute Dati

Comando `/health` o automatico quando si accede a un'entità:

```
 ╔══════════════════════════════════════════╗
 ║  HEALTH  companies                       ║
 ╠══════════════════════════════════════════╣
 ║  ● Completeness  94%  ████████░░        ║
 ║  ● Freshness      67%  ██████░░░░  ⚠    ║
 ║  ● Consistency    88%  ████████░░        ║
 ╠══════════════════════════════════════════╣
 ║  ▸ 3 su 5 indici buoni. Freschezza       ║
 ║    dei dati sotto soglia.                 ║
 ╚══════════════════════════════════════════╝
```

---

## Fase 4: DataPanel — Viste Dati nel Copilot

### 4.1 Layout Dual-Pane

Quando una vista dati è attiva (esplicitamente o via comando), appare un **DataPanel** laterale che prende ~50% dello schermo:

```
┌────┬──────────────────────┬─────────────────────────┐
│SB  │ terminale             │ DATA PANEL               │
│    │                       │ ┌─ tabs ────────────────┐│
│    │ ᐅ /explore companies │ │Table│Map│Timeline│Graph││
│    │                       │ │                       ││
│    │ Mostrando 42 aziende… │ │  [vista interattiva]  ││
│    │                       │ │                       ││
│    │ ᐀                    │ │                       ││
│    ├──────────────────────┤ │                       ││
│    │ ᐅ _                  │ │                       ││
│    │ [/explore·/help]      │ └───────────────────────┘│
└────┴──────────────────────┴─────────────────────────┘
```

- DataPanel è **resizable** (drag del divider)
- Si chiude con `Escape` o click su "close"
- Supporta i 4 tipi di vista: Tabella, Mappa, Timeline, Grafo (riutilizzando AlephTable, AlephMap, etc.)
- Livelli di dettaglio: **Overview** (card sintesi) → **Detail** (tabella completa) → **Raw** (JSON)

### 4.2 Vista Tabella Redesign

La tabella ASCII nel terminale:
```
┌───────┬──────────────┬──────────┐
│ id    │ name         │ revenue  │
├───────┼──────────────┼──────────┤
│ 1     │ Acme Corp    │ 4.2M     │
│ 2     │ TechStart    │ 1.1M     │
└───────┴──────────────┴──────────┘
▸ 42 aziende, revenue medio 3.1M, range 0.2M-89M
Mostrando 2 di 42 risultati. /explore companies --all per tutto.
```

La tabella interattiva nel DataPanel: mantiene AlephTable ma con styling terminal-compatible (font mono, bordi sottili, hover highlight).

### 4.3 Livelli di Sintesi nel DataPanel

Ogni vista nel DataPanel ha 3 livelli di dettaglio, accessibili via tab o shortcut:

1. **Overview** — Card sintetiche con:
   - Titolo entità + conteggio
   - 3 metriche chiave con sparkline
   - NarrativeBlock: una frase di interpretazione
   
2. **Detail** — Tabella interattiva completa con:
   - Ordinamento colonne
   - Filtri inline
   - Clic su riga → dettaglio nel terminale
   
3. **Raw** — JSON dell'oggetto/query, per debugging

Toggle tra livelli: `1` `2` `3` sulla tastiera quando il DataPanel è attivo, o pulsanti nel DataPanel header.

---

## Fase 5: Dual Theme System

### 5.1 Implementazione

- Aggiungere `data-theme="dark"` al `<html>`
- Usa CSS custom properties per i token, mappate da Tailwind:
  ```css
  :root { /* light tokens */ }
  [data-theme="dark"] { /* dark tokens */ }
  ```
- Default: `dark` (l'esperienza CLI è naturale in dark)
- Toggle via `/theme dark|light` o shortcut `⌘⇧D`
- Tutti i componenti esistenti ricevono varianti theme-aware

### 5.2 Store Update

Aggiungere a `useStore`:
```typescript
theme: 'dark' | 'light'
setTheme: (t: string) => void
```

Persistere in `localStorage`.

---

## Fase 6: Ristrutturazione Store e Routing

### 6.1 Store Refactor

Estendere `useStore` per supportare il nuovo modello:

```typescript
// Nuovi campi
theme: 'dark' | 'light'
setTheme: (t: 'dark' | 'light') => void
commandHistory: string[]
addCommandHistory: (cmd: string) => void
dataPanelOpen: boolean
setDataPanelOpen: (v: boolean) => void
dataPanelView: 'table' | 'map' | 'timeline' | 'graph' | 'overview'
setDataPanelView: (v: string) => void
activeCommand: string | null  // slash command in corso
setActiveCommand: (c: string | null) => void
```

### 6.2 Rimosso vs Rinominato

- `activeTab` rimane ma default → `'Copilot'`
- Le viste Explorer/Oracle/etc. rimangono accessibili via sidebar E via slash commands
- Il DataPanel è un overlay laterale, non un tab separato

---

## Fase 7: Implementazione — Ordine di Esecuzione

### Sprint 1 — Fondazioni (3-4 giorni)
1. Creare nuovo design-tokens.json (dual theme)
2. Aggiornare tailwind.config.js con il sistema di colori theme-aware
3. Creare `index.css` con CSS custom properties per dual theme
4. Creare componente `TerminalPrompt` (input con prompt symbol, slash detection, history)
5. Creare componente `TerminalOutput` (markdown + ASCII table renderer)
6. Creare `StatusBar` component

### Sprint 2 — Layout & Copilot Home (3-4 giorni)
7. Refactor `App.tsx` nel nuovo layout (Sidebar thin + TerminalArea + DataPanel)
8. Riscrivere `Sidebar.tsx` come thin icon bar (48px) con expand
9. Riscrivere `CopilotView.tsx` come terminale conversazionale (usando TerminalPrompt + TerminalOutput)
10. Implementare slash command engine (`/explore`, `/predict`, `/agent`, etc.)
11. Tab completion per slash commands + entità ontologiche

### Sprint 3 — Explorer + DataPanel (3-4 giorni)
12. Creare `DataPanel` component (slide-in laterale con tabs Tabella/Mappa/Timeline/Grafo)
13. Integrare il copilot con Explorer: quando domanda dati → ASCII table nel terminale + pulsante "Apri vista"
14. Riscrivere `ExplorerView.tsx` come vista dentro DataPanel (non più tab separato)
15. Implementare livelli di sintesi (quick/table/full)
16. Refactor di AlephTable con styling terminal

### Sprint 4 — Dual Theme & Rimanenti Views (2-3 giorni)
17. Implementare toggle dark/light + persistenza localStorage
18. Aggiornare tutti i componenti per essere theme-aware
19. Riscrivere `OracleView.tsx` — predizioni come output terminale + card inline
20. Riscrivere `AgentsView.tsx`, `SkillsView.tsx`, `ToolsView.tsx`, `SettingsView.tsx` — stile terminale
21. CommandPalette → stile terminale (già parzialmente CLI-like, serve solo styling)

### Sprint 5 — Polish & Comandi Avanzati (2-3 giorni)
22. `/agent create` — wizard step-by-step nel terminale
23. `/ontology edit` — editor inline con syntax highlighting
24. `/data add` — wizard data source nel terminale
25. `/library list` + preview
26. Animazioni e transizioni fluide
27. Keyboard shortcuts globali (`⌘K`, `⌘⇧D`, `Escape`, `/`)
28. Testing e bug fix

---

## Dipendenze

- **react-markdown** + **react-syntax-highlighter** per il rendering Markdown nel terminale
- **framer-motion** per le micro-interazioni (slide-in, stagger, spring)
- **xterm.js** (opzionale, per input avanzato) → inizialmente usiamo `<textarea>` con styling terminal
- Le librerie esistenti (D3, Leaflet, Zustand) rimangono

## Rischi e Mitigazioni

| Rischio | Mitigazione |
|---|---|
| Complessità terminal prompt | Iniziare con textarea styled, evolvere verso xterm.js se necessario |
| Performance ASCII tables con dati grandi | Paginazione + "mostra altri" on demand |
| Transizione UX brusca | Mantenere sidebar con navigation tra views tradizionali come fallback |
| Mobile responsiveness | Terminal UI funziona bene su mobile con viewport stretto |
| Over-animation se framer-motion è troppo | Tutte le animazioni sono opzionali via prefers-reduced-motion; fallback instant |
| Narrative layer richiede dati AI | Se l'AI non è disponibile, usare template statici basati su statistiche calcolate (avg, min, max, count) |

## File Coinvolti

**Nuovi:**
- `src/components/TerminalPrompt.tsx`
- `src/components/TerminalOutput.tsx`
- `src/components/StatusBar.tsx`
- `src/components/DataPanel.tsx`
- `src/components/SlashCommandBar.tsx`
- `src/components/SparklineRenderer.tsx`
- `src/components/PredictionCard.tsx`
- `src/components/NarrativeBlock.tsx`
- `src/components/HealthStrip.tsx`
- `src/lib/slashCommands.ts`
- `src/lib/terminalRenderer.ts`
- `src/lib/asciiTable.ts`
- `src/lib/sparkline.ts` (array → Unicode block characters)
- `src/lib/narrative.ts` (generazione narrative statiche da statistiche)

## File Coinvolti

**Nuovi:**
- `src/components/TerminalPrompt.tsx`
- `src/components/TerminalOutput.tsx`
- `src/components/StatusBar.tsx`
- `src/components/DataPanel.tsx`
- `src/components/SlashCommandBar.tsx`
- `src/lib/slashCommands.ts`
- `src/lib/terminalRenderer.ts`
- `src/lib/asciiTable.ts`

**Riscritti:**
- `src/App.tsx` → nuovo layout system
- `src/components/Sidebar.tsx` → thin icon bar
- `src/components/CopilotView.tsx` → terminal-style chat con sparklines, narrative, prediction cards
- `src/components/ExplorerView.tsx` → DataPanel view
- `src/components/OracleView.tsx` → terminal output con PredictionCard e narrative
- `src/components/DataHealthView.tsx` → HealthStrip inline nel terminale
- `src/styles/design-tokens.json` → dual theme tokens
- `src/index.css` → theme custom properties + animation keyframes
- `tailwind.config.js` → theme-aware config + animation utilities
- `src/store/useStore.ts` → nuovi campi (theme, commandHistory, dataPanel, narrative)

**Aggiornati (styling):**
- `AgentsView.tsx`, `SkillsView.tsx`, `ToolsView.tsx`, `SettingsView.tsx`, `LibraryView.tsx`, `DataSourcesView.tsx`, `OntologyView.tsx`, `ComponentsView.tsx`

**Rimossi:**
- Header bar (integrato in StatusBar)
- Card-based layout globale (sostituito da terminal layout)
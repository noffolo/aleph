# Aleph-v2 Redesign: Terminal-First, Copilot-Centric UI/UX

## Executive Summary

Trasformare l'interfaccia di Aleph-v2 da un tradizionale layout tab-based a un'esperienza **terminal-first, copilot-centric**. Il Copilot diventa la home principale e unico punto di ingresso per navigare, esplorare dati, gestire ontologie e interagire con agenti. Le viste esistenti (Explorer, Data Health, Ontologie, ecc.) vengono integrate come output inline all'interno del flusso conversazionale, raggiungibili tramite slash commands (`/`) o tramite richiesta naturale all'LLM.

## Stato Attuale & Problemi

1. **Copilot isolato**: E' una tab laterale, non la home. L'utente deve sapere a priori dove cliccare.
2. **Slash commands non integrate**: Esistono in `slashCommands.ts` ma non vengono eseguite prima di inviare il messaggio all'agente. `/explore foo` viene trattato come testo libero.
3. **Navigazione duplicata**: Sidebar a icone + Tab bar nel main area. L'utente ha due modelli mentali per la stessa cosa.
4. **Esperienza esplorativa frammentata**: Le viste (mappa, timeline, grafo) sono in tab separate, non integrate nel flusso di lavoro del Copilot.
5. **Stile terminale sotto-utilizzato**: Design tokens, font mono, colori dark esistono, ma mancano effetti controllati (CRT, scanline, glow dinamico).

## Architettura Target

```
+-------------------------------------------------------------+
|  Aleph v2 — Terminal-First Interface                        |
+-------------------------------------------------------------+
|  λ > [Input utente]                                          |
+-------------------------------------------------------------+
|  +-------------------+  +-------------------------------+    |
|  | Storia Messaggi   |  | Vista Attiva (Inline/Modal)  |    |
|  | (Chat Stream)     |  | - Explorer Output             |    |
|  |                   |  | - Data Tables                  |    |
|  |                   |  | - Graph / Map / Timeline       |    |
|  |                   |  | - Ontology Tree               |    |
|  |                   |  | - Settings Panel               |    |
|  +-------------------+  +-------------------------------+    |
+-------------------------------------------------------------+
|  Status Bar | Context: Project X | Agent: Oracle | Mode: CMD |
+-------------------------------------------------------------+
```

### Flusso di Interazione

1. **Landing**: L'utente entra e vede il Copilot con una breve intro (`λ > Welcome to Aleph. Type /help to see available commands.`).
2. **Navigazione via Comandi**: L'utente scrive `/explore` o `/data`. Il sistema mostra una GUI inline (es. una tabella o un grafico) direttamente nel flusso della chat, senza cambiare tab.
3. **Navigazione via LLM**: L'utente scrive "Show me the sales data for Q3". L'agente riconosce l'intento e restituisce una risposta strutturata. Il frontend renderizza un componente `InlineExplorer` o `InlineDataTable` all'interno della chat.
4. **Modali Controllati**: Per azioni complesse (es. Wizard di importazione dati), il Copilot può aprire un overlay modale centrato, ma il focus rimane sul terminale sottostante.
5. **Sidebar Ridotta**: La sidebar rimane come riferimento visivo rapido, ma cliccando un'icona si traduce in un comando `/switch explore` o `/switch data` inviato al Copilot, che poi aggiorna la vista inline.

## Piano di Implementazione

### Fase 1: Fondamenta & Build Check
**Goal**: Assicurarsi che il codebase sia stabile prima del refactoring massivo.
- [ ] **Task 1.1**: Eseguire build check frontend (`npm run build`).
- [ ] **Task 1.2**: Eseguire build check backend (`go build ./...`).
- [ ] **Task 1.3**: Eseguire check sintassi sidecar Python (`python -m py_compile`).
- [ ] **Task 1.4**: Verificare dipendenze mancanti e aggiornare `package.json`/`go.mod` se necessario.

### Fase 2: Struttura & Stato Globale
**Goal**: Preparare lo store per la nuova architettura single-view.
- [ ] **Task 2.1**: Rifattorizzare `useStore` (Zustand):
  - Rimuovere `activeTab` come driver primario.
  - Aggiungere `currentView: 'copilot' | 'inline'`.
  - Aggiungere `inlineContent: { type: 'table' | 'graph' | 'map' | 'settings' | ...; data: any } | null`.
  - Aggiungere `commandQueue: string[]` per comandi da eseguire all'avvio.
- [ ] **Task 2.2**: Creare componente `MainLayout`:
  - Rimuovere tab bar dall'area principale.
  - Layout a due colonne: Storia Messaggi (sinistra/unica colonna) + Area Inline (destra, condizionale, resizable).
  - In fullscreen mobile, l'area inline diventa un overlay a schermo intero con un tasto "Back to Chat".

### Fase 3: Integrazione Slash Commands
**Goal**: Rendere le `/` commands il motore di navigazione primario.
- [ ] **Task 3.1**: Modificare `TerminalPrompt.tsx`:
  - Intercettare l'evento `onSubmit`.
  - Eseguire `parseSlashCommand(input)` PRIMA di inviare qualsiasi cosa al backend.
  - Se è una slash command, non inviare il messaggio all'agente LLM, ma eseguire `executeSlashCommand`.
  - Aggiungere autocomplete dropdown quando l'utente digita `/`.
- [ ] **Task 3.2**: Rifattorizzare `slashCommands.ts`:
  - Ogni comando deve restituire un oggetto strutturato `{ action: 'SWITCH_VIEW' | 'OPEN_MODAL' | 'TRIGGER_AGENT' | 'UI_ACTION', payload: {...} }`.
  - Aggiornare handler di `/explore` per impostare `inlineContent` invece di cambiare tab.
  - Aggiungere handler per `/data`, `/ontology`, `/settings`.
- [ ] **Task 3.3**: Aggiornare `CopilotView.tsx`:
  - Rimuovere logica tab switching.
  - Gestire la coda di messaggi. Se un messaggio è una risposta strutturata (es. `{ component: 'DataTable', props: {...} }`), renderizzare il componente inline.

### Fase 4: Componenti Inline
**Goal**: Permettere al Copilot di renderizzare viste complesse dentro la chat.
- [ ] **Task 4.1**: Creare `InlineRenderer.tsx`:
  - Riceve `type` e `props`.
  - Mappa `type` ai componenti esistenti: `ExplorerView`, `DataTable`, `OntologyGraph`, `SettingsPanel`.
  - Wrapper per garantire padding, bordi e stile terminale.
- [ ] **Task 4.2**: Adattare le viste esistenti per il rendering inline:
  - Rimuovere header ridondanti (il contesto è già nella chat).
  - Supportare `height` flessibile (`max-h-[60vh]` con scroll interno).
  - Assicurarsi che i tooltip e i modali secondari non escano dal container inline.
- [ ] **Task 4.3**: Aggiungere azioni contestuali su ogni componente inline:
  - Pulsante "Pop out" (apre in modale a schermo intero).
  - Pulsante "Close" (rimuove `inlineContent` dallo store).
  - Pulsante "Export" (dove applicabile).

### Fase 5: Copilot come Home Principale
**Goal**: Rendi il Copilot la prima cosa che l'utente vede.
- [ ] **Task 5.1**: Modificare `App.tsx`:
  - Default view: `CopilotView` a schermo intero o come componente radice.
  - Aggiungere state di benvenuto con suggerimenti di comandi (`Tip: try /explore`).
  - Rimuovere o nascondere la tab bar tradizionale. Se necessaria per retrocompatibilità, spostarla in un pannello laterale collassabile.
- [ ] **Task 5.2**: Aggiornare Sidebar:
  - Clic su icona = triggera un comando (es. `/explore`) invece di cambiare tab direttamente.
  - Aggiungere stato "active" basato sul `currentView` dello store.
  - Aggiungere tooltip con il nome del comando corrispondente.
- [ ] **Task 5.3**: Gestione URL/Routing:
  - Aggiungere `react-router-dom` (o gestione hash) per URL come `/#/explore/dataset-123`.
  - Al caricamento, parsare l'URL e inserire il comando corrispondente nella `commandQueue`.

### Fase 6: Microtipografia & Effetti Terminal
**Goal**: Raffinare l'estetica senza invadere l'usabilità.
- [ ] **Task 6.1**: Creare `TerminalEffects.tsx`:
  - **Scanline overlay**: CSS `radial-gradient` o `linear-gradient` con `background-size` ripetuto. Opacità 3-5%. Toggle on/off nelle impostazioni.
  - **CRT curvature**: Leggero `border-radius` sui bordi dello schermo o `box-shadow` interno.
  - **Bloom/Glow**: `text-shadow` sul prompt e sul caret (`terminal-glow` esiste, renderlo dinamico con CSS custom properties).
- [ ] **Task 6.2**: Aggiungere `prefers-reduced-motion`:
  - Se l'utente lo richiede, disabilitare tutti i glow e le animazioni di scanline.
- [ ] **Task 6.3**: Refinimento Input:
  - Aggiungere storia comandi locale (freccia su/giù per navigare i precedenti input).
  - Highlight sintassi basilare per le slash commands nel textarea (colore diverso per `/comando` e argomenti).

### Fase 7: Wizard & Onboarding nel Terminale
**Goal**: Intagrare i flussi guidati nell'esperienza chat.
- [ ] **Task 7.1**: Creare `WizardInline.tsx`:
  - Wrapper per i wizard esistenti.
  - Mostra i passaggi come messaggi di sistema o come form compatti inline.
- [ ] **Task 7.2**: Comando `/setup` o `/wizard`:
  - Avvia il wizard di onboarding come sequenza di messaggi bot + form inline, anziché modale separata.

### Fase 8: Testing & QA
**Goal**: Verificare stabilità e funzionalità end-to-end.
- [ ] **Task 8.1**: Unit test per `slashCommands.ts` (parsing e routing).
- [ ] **Task 8.2**: Integrazione test per `TerminalPrompt` (submit, autocomplete, history).
- [ ] **Task 8.3**: E2E test con Playwright/Cypress:
  - Flusso: Utente apre app -> Scrive `/explore` -> Vede tabella inline -> Scrive `/settings` -> Vede pannello inline -> Clicca "Pop out" -> Vede modale.
- [ ] **Task 8.4**: Performance check:
  - Lighthouse score > 90.
  - Verificare che i componenti inline pesanti (D3/Leaflet) non blocchino il render della chat (usare `React.lazy` + `Suspense`).

## Dipendenze tra Task

```
Fase 1 (Build) -> Fase 2 (Store) -> Fase 3 (Commands) -> Fase 4 (Inline) -> Fase 5 (Home)
Fase 6 (Effects) può procedere in parallelo con Fase 3/4/5.
Fase 7 (Wizard) dipende da Fase 4 e 5.
Fase 8 (QA) dipende da tutto.
```

## Note Tecniche

- **Stato**: Zustand è ideale per questo refactor data la sua semplicità e l'assenza di boilerplate.
- **Routing**: Considerare se aggiungere `react-router-dom`. Dato che l'esperienza è "single page" con viste inline, potrebbe non essere strettamente necessario; la gestione dello stato globale potrebbe bastare. Se si desidera URL sharing, un hash-based router leggero è preferibile.
- **Backend**: Le modifiche sono principalmente frontend. Il backend deve solo assicurarsi che gli endpoint RPC restituiscano dati strutturati che il frontend possa interpretare per renderizzare i componenti inline (es. schema JSON per tabelle/grafici).
- **Accessibilità**: Navigazione da tastiera è prioritaria (Tab per autocomplete, Enter per submit, Esc per chiudere inline views).

## Criteri di Accettazione

1. L'utente apre Aleph e vede un terminario con prompt `λ >`.
2. Scrivendo `/data`, una tabella dati appare inline sotto il messaggio, senza cambiare tab.
3. Scrivendo "Show me the ontology graph", l'agente risponde e un grafo appare inline.
4. La sidebar è presente ma cliccando un'icona il risultato appare nel flusso del copilot.
5. Tutti i build (Go, npm, Python) passano senza errori.
6. Lighthouse performance score >= 90 in modalità mobile.

# Aleph-v2 Redesign: Terminal-First, Copilot-Centric UI/UX

## Executive Summary

Trasforma l'interfaccia di Aleph-v2 da un tradizionale layout tab-based a un'esperienza **terminal-first, copilot-centric**. Il Copilot diventa la home principale e unico punto di ingresso per navigare, esplorare dati, gestire ontologie e interagire con agenti. Le viste esistenti (Explorer, Data Health, Ontologie, ecc.) vengono integrate come output inline all'interno del flusso conversazionale, raggiungibili tramite slash commands (`/`) o tramite richiesta naturale all'LLM.

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
- [x] **Task 1.1**: Eseguire build check frontend (`npm run build`).
- [x] **Task 1.2**: Eseguire build check backend (`go build ./...`).
- [x] **Task 1.3**: Eseguire check sintassi sidecar Python (`python -m py_compile`).
- [x] **Task 1.4**: Verificare dipendenze mancanti e aggiornare `package.json`/`go.mod` se necessario.

### Fase 2: Struttura & Stato Globale
**Goal**: Preparare lo store per la nuova architettura single-view.
- [x] **Task 2.1**: Rifattorizzare `useStore` (Zustand):
  - Rimuovere `activeTab` come driver primario (mantenerlo per retrocompatibilità).
  - Aggiungere `currentView: 'copilot' | 'inline'`.
  - Aggiungere `inlineContent: { type: 'table' | 'graph' | 'map' | 'settings' | ...; title: string; data: any } | null`.
  - Aggiungere `commandHistory: string[]` per comandi da eseguire all'avvio.
  - Aggiungere `showInlinePanel: boolean`.
- [x] **Task 2.2**: Creare `InlineRenderer.tsx`:
  - Mappa `type` ai componenti esistenti: `ExplorerView`, `DataTable`, `OntologyGraph`, `SettingsPanel`.
  - Wrapper per garantire padding, bordi e stile terminale.

### Fase 3: Integrazione Slash Commands nel Input
**Goal**: Rendere le `/` commands il motore di navigazione primario.
- [x] **Task 3.1**: Modificare `TerminalPrompt.tsx`:
  - Intercettare l'evento `onSubmit`.
  - Eseguire `parseCommand(input)` PRIMA di inviare qualsiasi cosa al backend.
  - Se è una slash command, non inviare il messaggio all'agente LLM, ma eseguire `executeSlashCommand`.
- [x] **Task 3.2**: Rifattorizzare `slashCommands.ts`:
  - Ogni comando restituisce `CommandResult` con `action: 'SHOW_INLINE' | 'SWITCH_COPILOT' | 'CLEAR_CHAT' | 'AGENT_COMMAND'`.
  - Aggiornare handler di `/explore` per impostare `inlineContent` invece di cambiare tab.
  - Aggiungere handler per `/data`, `/ontology`, `/settings`.
- [x] **Task 3.3**: Aggiornare `App.tsx`:
  - Introdurre `handleCommandResult()` in `onSend()`. Le slash commands vengono intercettate PRIMA di inviare al backend LLM.

### Fase 4: TerminalProgressBar (Dettaglio Maniacale)
**Goal**: Progress bar con micro-tipografia esemplare.
- [x] **Task 4.1**: Riscrivo `TerminalProgressBar` con:
  - **Smooth animation** via `requestAnimationFrame` + ease-out cubic
  - **Sub-character precision** usando i blocchi Unicode 258F–2597 (7 gradazioni per carattere)
  - **Braille spinner** (10 frame, 80ms)
  - **Sparkline ASCII inline** con `▁▂▃▄▅▆▇█`
  - **ETA intelligente** con formato `mm:ss`
  - **Throughput display** con unità arbitrarie (MB/s, rows/s, etc.)
  - **4 varianti**: `classic` (default), `compact`, `nested`, `full`
  - **Tab-size alignment** con padding monospaced per colonne perfette

### Fase 5: Inline Prop per tutte le viste
**Goal**: Permettere al Copilot di renderizzare viste complesse dentro la chat.
- [ ] **Task 5.1**: Aggiungere `inline?: boolean` a TUTTE le viste:
  - `ExplorerView` (già fatto)
  - `DataSourcesView`, `OntologyView`, `AgentsView`, `SkillsView`, `ToolsView`, `LibraryView`, `DataHealthView`, `SettingsView`, `ComponentsView`, `OracleView`
  - Quando `inline = true`, rimuovere header ridondanti (il contesto è già nella chat).
  - Supportare `height` flessibile (`max-h-[60vh]` con scroll interno).
- [ ] **Task 5.2**: Integrare `InlineRenderer` in `CopilotView`:
  - Posizionarlo nel flusso della chat, o come pannello fisso laterale.

### Fase 6: Copilot come Home Principale
**Goal**: Rendi il Copilot la prima cosa che l'utente vede.
- [ ] **Task 6.1**: Modificare `App.tsx`:
  - Default view: cambiare `activeTab` default da `Explorer` a `Copilot`.
  - Aggiugere messaggio di benvenuto con suggerimenti di comandi (`Tip: try /explore`).
  - Mantenere tab bar come fallback nascosto o secondario.
- [ ] **Task 6.2**: Aggiornare `Sidebar.tsx`:
  - Clic su icona = triggera comando (es. `/explore`) invece di cambiare tab direttamente.
  - Aggiungere tooltip con il nome del comando corrispondente.
- [ ] **Task 6.3**: Rimuovere o nascondere la tab bar tradizionale.

### Fase 7: Microtipografia & Effetti Terminal
**Goal**: Raffinare l'estetica senza invadere l'usabilità.
- [ ] **Task 7.1**: Creare `TerminalEffects.tsx`:
  - **Scanline overlay**: CSS `repeating-linear-gradient` con opacità 3-5%. Toggle on/off nelle impostazioni.
  - **CRT curvature**: Leggero `border-radius` sui bordi dello schermo o `box-shadow` interno.
  - **Bloom/Glow**: `text-shadow` sul prompt e sul caret.
- [ ] **Task 7.2**: Aggiungere `prefers-reduced-motion`:
  - Se l'utente lo richiede, disabilitare tutti i glow e le animazioni di scanline.
- [ ] **Task 7.3**: Refinimento Input:
  - Aggiungere storia comandi locale (freccia su/giù per navigare i precedenti input).
  - Highlight sintassi basilare per le slash commands nel textarea.

### Fase 8: Testing & QA
**Goal**: Verificare stabilità e funzionalità end-to-end.
- [ ] **Task 8.1**: Build check frontend (`npm run build`).
- [ ] **Task 8.2**: Build check backend (`go build ./...`).
- [ ] **Task 8.3**: E2E test con Playwright/Cypress:
  - Flusso: Utente apre app → Scrive `/explore` → Vede tabella inline → Scrive `/settings` → Vede pannello inline → Clicca "Pop out" → Vede modale.
- [ ] **Task 8.4**: Performance check:
  - Lighthouse score > 90.
  - Verificare che i componenti inline pesanti (D3/Leaflet) non blocchino il render della chat (usare `React.lazy` + `Suspense`).

## Dipendenze tra Task

```
Fase 1 (Build) → Fase 2 (Store) → Fase 3 (Commands) → Fase 4 (Inline) → Fase 5 (Home)
Fase 6 (Effects) può procedere in parallelo con Fase 3/4/5.
Fase 7 (Wizard) dipende da Fase 4 e 5.
Fase 8 (QA) dipende da tutto.
```

## Note Tecniche

- **Stato**: Zustand è ideale per questo refactor data la sua semplicità e l'assenza di boilerplate.
- **Routing**: Considerare se aggiungere `react-router-dom`. Dato che l'esperienza è "single page" con viste inline, potrebbe non essere strettamente necessario.
- **Backend**: Le modifiche sono principalmente frontend. Il backend deve solo assicurarsi che gli endpoint RPC restituiscano dati strutturati che il frontend possa interpretare per renderizzare i componenti inline.
- **Accessibilità**: Navigazione da tastiera è prioritaria (Tab per autocomplete, Enter per submit, Esc per chiudere inline views).

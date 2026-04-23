# Piano Definitivo: Terminal-First Frontend Aleph-v2
# Versione 2.0 — Post-Review Assemblea Completa

> **Principio fondante**: Copilot è l'unica vista principale. Tutto il resto è output inline nel flusso terminale o slide-over laterale. Zero tab, zero modali centrati, zero palette light. Ma fatto con intelligenza — non con violenza epistemologica.

---

## Premessa — Regole di Routing Inline vs SlideOver (Definitivo)

| View | Routing | Motivazione |
|------|---------|-------------|
| Explorer (tabella) | **Inline** | Lista verticale, scroll naturale nel terminale |
| Explorer (mappa/timeline/grafico) | **SlideOver** | Canvas interattivo, richiede spazio orizzontale e zoom/pan |
| Agents list | **Inline** | Lista verticale compatibile col flusso |
| Agents create/edit form | **SlideOver** | Form denso, evita scroll interminabile nel terminale |
| Ontology view | **Inline** per read / **SlideOver** per edit form | Editor DSL compatto in inline, form di modifica in slideOver |
| Data Sources list | **Inline** | Lista verticale con stato |
| Data Sources add form | **SlideOver** | Form multi-step denso |
| Skills list | **Inline** | Lista verticale |
| Skills detail/run | **SlideOver** | Form parametri + risultato sandbox |
| Tools list | **Inline** | Lista verticale |
| Tools detail/execute | **SlideOver** | Code preview + form parametri |
| Library list | **Inline** | Lista assets verticale |
| Library asset viewer | **SlideOver** | Lettura documento richiede spazio |
| Data Health stats | **Inline** | Metriche testuali |
| Settings | **Inline** per lista / **SlideOver** per form dettagliato | Impostazioni semplici inline, form webhook/apikey in slideOver |
| Components list | **Inline** | Lista verticale |
| Components register/detail | **SlideOver** | Form metadata denso |
| DetailPanel (record inspect) | **SlideOver** | Contenuto strutturato, lettura non lineare |
| Command Palette | **Overlay centrato** (esistente, convertito a dark) | Overlay temporaneo, ok centrato |
| Onboarding/Wizard | **Full-screen** (pre-project, mantenuto) | Senza progetto non ha senso un terminale |

---

## FASE 1 — Deprecazione Sicura State Store (PRIORITÀ ALTA)

1. **Non eliminare `activeTab` immediatamente**. Marcarlo `@deprecated` nell'interfaccia `AppState`.
2. **Rimuovere `activeTab` da `SYNCED_KEYS`** e aggiungere un effetto one-time che esegue `yMap.delete('activeTab')` per pulire i peer.
3. **Aggiungere initializer** per `slideOverContent: null`, `sandboxResult: null`, `sandboxInput: '{}'`, `setSlideOverContent`, `setSandboxResult`, `setSandboxInput` nel return di `create`.
4. **Pulire `setProjectContext`** dal reset di `activeTab`.
5. **Aggiungere campo `terminalMode: 'copilot'`** (per future estensioni).
6. **Attendere conferma build OK** prima di rimuovere totalmente `activeTab`.

---

## FASE 2 — Security Hardening (CRITICA — PRIMA di Refactoring UI)

1. **Slash Command Allow-list**: Solo comandi nel file `slashCommands.ts` sono eseguibili. Tutto il resto va al LLM.
2. **Categorizzazione comandi**: readonly (`/help`, `/explore`, `/health`) vs mutanti (`/agent create`, `/skills run`). I comandi mutanti richiedono **conferma esplicita** (Enter o checkbox) PRIMA di dispatch.
3. **Agent Output Sanitization**: L'output dell'agente LLM è sempre plain text escaped. Nessun HTML rendering. Se l'agente scrive `/explore` nel suo output, NON viene interpretato come comando — solo l'input dell'utente passa per `parseCommand()`.
4. **Yjs WebrtcProvider Room ID**: Cambiare naming. NON usare `simpleHash(apiKey)` — generare token JWT server-side per room auth. (Richiede modifica backend — segnare come collegamento backend).
5. **API Key Storage**: `getStoredApiKey()` legge da `sessionStorage` (non `localStorage`). La chiave non persiste oltre la sessione. Backend deve impostare cookie `httpOnly`, `Secure`, `SameSite=Strict`.
6. **Content Security Policy (meta)**: Aggiungere header CSP in produzione (da configurare con reverse proxy / CDN).
7. **AlephErrorBoundary globale**: Wrappa l'intera app. Wrappa anche `InlineRenderer` con boundary specifico per lazy loading.

---

## FASE 3 — App.tsx Riscrittura Radicale

1. **Eliminare tutti gli import statici delle 11 view tranne `CopilotView`**. CopilotView è l'unica import statico (core). Tutte le altre via `React.lazy`.
2. **Spostare `loadProjectData`, `onSend`, `onConfirmAction`, `handleCommandResult`** in `src/hooks/useAppActions.ts`.
3. **Il `renderMain()` sparisce**. Il contenuto principale è sempre `<CopilotView ... />` + `<InlineRenderer />` + `<SlideOverPanel />` (unico, unificato).
4. **Rimuovere i 3 modali hardcoded** (skill detail, tool detail, sandbox result) → SlideOver unificato che legge `store.slideOverContent`.
5. **Rimuovere `useState` locali `sandboxInput`/`sandboxResult`** → usare `store.sandboxInput` / `store.sandboxResult`.
6. **DetailPanel** (ispezione record) → SlideOverPanel (`selectedRow` trigger).
7. **Aggiungere `prefetchView(viewId)` utility**: preloads i chunk delle view comuni al hover della sidebar.
8. **Aggiungere `TerminalEffects`** al layout root (scanline + glow attivi, CRT disabilitato di default).
9. **Configurare Vite manual chunks**: raggruppare viste correlate in bundle logici (es. `map-views`, `form-views`, `graph-views`). Budget entry: 150KB gzipped.

---

## FASE 4 — Sidebar Refactor

1. **Rimuovere props `activeTab` e `setActiveTab`**.
2. **Click icona → dispatch slash command**: `store.setInput('/explore')` + auto-submit.
3. **Highlight attivo**: basato su `store.inlineContent?.type` o `store.slideOverContent?.type`.
4. **Rimuovere array `sections` con `id` legacy** → mappatura icona→comando pura.

---

## FASE 5 — StatusBar Refactor

1. **Rimuovere prop `activeTab`**.
2. **Layout**: `ALEPH │ {projectID || 'NO PROJECT'} │ {slideOverContext || 'READY'}`.
3. **Servizi**: Ollama, NLP indicatore con dot colorato.
4. **Se slideOver aperto**: mostrare tipo vista (es: `MAP`, `GRAPH`, `SKILL DETAIL`).

---

## FASE 6 — CopilotView + InlineRenderer Enhancement

1. **Autocompletamento slash commands**: `Tab` key nel `TerminalPrompt` → suggerimento da `getTabCompletion()`.
2. **Inline vs SlideOver decision logic**:
   - Quando `executeCommand()` ritorna `target='explore'` con sub-view `map|timeline|graph` → Inline renderizza riepilogo testuale, SlideOver aperta automatica con vista canvas.
   - Aggiungere `panelMode` a `InlineContent`: `'inline'` per liste/tabelle, `'slideover'` per canvas/form.
3. **Accessibility**: `InlineRenderer` wrapper con `role="region"` e `aria-label`.
4. **Streaming**: quando l'utente è in streaming, lo slideOver può rimanere aperto (non interrompe). AbortController gestito per-component (non globale).

---

## FASE 7 — Migrazione Palette Dark (11 view)

| Light Class → Dark Class |
|:---|:---|
| `bg-white` | `bg-surface` |
| `bg-gray-50` / `bg-gray-100` | `bg-surface-alt` o `bg-background` |
| `border-gray-100` / `border-gray-200` | `border-border` |
| `text-gray-900` / `text-gray-800` | `text-text` |
| `text-gray-500` / `text-gray-400` | `text-textMuted` |
| `text-gray-300` / `text-gray-200` | `text-textDim` |
| `bg-blue-600` / `bg-blue-500` | `bg-primary` |
| `text-blue-600` / `text-blue-700` | `text-primary` |
| `bg-blue-50` / `hover:bg-blue-50` | `bg-primary/10` / `hover:bg-primary/10` |
| `bg-green-100` / `text-green-600` | `bg-success/10` / `text-success` |
| `bg-red-100` / `text-red-600` | `bg-danger/10` / `text-danger` |
| `bg-amber-50` / `text-amber-600` | `bg-warning/10` / `text-warning` |
| `shadow-2xl` / `shadow-xl` | `shadow-lg shadow-primary/5` o rimosso |
| `rounded-[32px]` / `rounded-3xl` | `rounded-lg` |
| `focus:border-blue-300` | `focus:border-primary/50` |

**Esecuzione**: in parallelo con FASE 8, view per view.

---

## FASE 8 — Modali Interni → SlideOverPanel (6 view)

| View | Modali | Nuova Destinazione |
|------|--------|-------------------|
| AgentsView | Create Agent, Edit Agent | SlideOver (`type='agent-form'`) |
| SkillsView | Create Skill | SlideOver (`type='skill-form'`) |
| ToolsView | Create Tool | SlideOver (`type='tool-form'`) |
| DataSourcesView | Add Source | SlideOver (`type='datasource-form'`) |
| LibraryView | Asset viewer | SlideOver (`type='asset-detail'`) |
| ComponentsView | Register, Detail | SlideOver (`type='component-form'`, `type='component-detail'`) |

**SlideOverPanel enhancement**: aggiungere prop `fullscreen?: boolean` + pulsante ⛶ per espandere a schermo intero. Animazione `max-w-2xl` → `max-w-full` con `cubic-bezier(0.16, 1, 0.3, 1)`.

---

## FASE 9 — Microtipografia, Animazioni e "Magia" Terminal

### Microtipografia Ruleset
- **Font sizes**: `text-[10px]` per label/etichette sistema, `text-xs` per testo secondario, `text-sm` per testo corpo terminale, `text-base` per prompt input.
- **Line-height**: `leading-relaxed` (1.6) per flusso terminale, `leading-snug` per label.
- **Letter-spacing**: `tracking-widest` per uppercase labels, `tracking-tight` per titoli display.
- **Color hierarchy**: primary → azioni/il λ; success → stato positivo; danger → errore; warning → attenzione; textMuted → metadati; textDim → separatori/decorazioni.
- **Spaziatura**: 8px grid (gap-2 = 8px, gap-4 = 16px).

### Animazioni "Machine-Like" (Non bounce)
- **Easing primario**: `cubic-bezier(0.16, 1, 0.3, 1)` — preciso, rapido all'inizio, decelerazione controllata alla fine.
- **Durata standard**: 250ms per slideOver, 150ms per micro-interazioni (hover, focus), 300ms per fade-in inline.
- **No bounce, no elastic, no spring**. Il terminale è deterministico.
- **Scanline overlay**: opacity 0.02-0.04, scrolling lento (10s). Respects `prefers-reduced-motion`.
- **Glow**: solo su `λ`, cursore attivo, e input field focus. Niente glow neon ovunque.

### Magia vs Grottesco
✅ **Magia**:
1. Predictive command glow — il λ pulsa leggermente quando l'utente sta scrivendo un comando riconosciuto.
2. Smart scroll — quando l'utente scrive, l'auto-scroll si ferma se l'utente ha scrollato manualmente (flag `userHasScrolled`).
3. Riepilogo contestuale nella status bar — mostra non solo il progetto, ma cosa sto guardando ("viewing MAP: Projects cluster").
4. Fade-in sequenziale dei blocchi inline — i blocchi non appaiono tutti insieme, ma con stagger 50ms quando sono multipli nella stessa risposta.
5. "Show your work" — streaming del pensiero dell'agente, non solo risultato finale.

❌ **Grottesco**:
1. CRT curvature su tutto lo schermo (dà nausea).
2. Blinking ovunque — solo il cursore deve blinkare.
3. Neon su tutti i bordi (glow indiscriminato).
4. Font monospace anche sui titoli display (si rompe la leggibilità).
5. Background noise / griglie che rovinano la leggibilità del testo.

---

## FASE 10 — i18n Fix

1. OntologyView: `"Visual Glossary"` → `"Glossario Visivo"`
2. DataSourcesView: `"Execution Output (Real-time Logs)"` → `"Output Esecuzione (Log Real-time)"`
3. ToolsView: `"Code Preview"` → `"Anteprima Codice"`
4. ComponentsView: `"Exec"` → `"Esecuzione"`; select options in italiano (`Agente`, `Competenza`, `Strumento`, `Modello`, `Connettore`)
5. LibraryView: riga 193 `font-sans` → `font-mono`
6. Terminal voice: i messaggi di sistema nel terminale sono in italiano, ma le keyword tecniche (API, JWT, JSON) rimangono in inglese per consistenza codice.

---

## FASE 11 — useViewActions Refactor

1. **Collegare `onRunSkill` → `setSlideOverContent`**:
   ```
   { type: 'skill', title: skill.name, data: skill }
   ```
2. **Collegare `onExecuteTool` → `setSlideOverContent`**:
   ```
   { type: 'tool', title: tool.name, data: tool }
   ```
3. **After execution**: risultato sandbox → `store.setSandboxResult(result)`, e lo slideOver mostra automaticamente la vista risultato.
4. **Domain-specific hooks**: scomporre `useViewActions` in `useExplorerActions`, `useAgentActions`, `useOntologyActions` etc. Comporre sotto un `useViewActions` facade per backward compat.
5. **Gestione errori centralizzata**: `handleError` che logga a Sentry + mostra toast nel terminale.

---

## FASE 12 — Error Handling, Testing & Observability

1. **Error Boundaries**:
   - Globale: `<AlephErrorBoundary>` attorno all'app.
   - Per `InlineRenderer`: boundary dedicato che isola i crash delle view lazy.
   - Per `SlideOverPanel`: boundary dedicato.
2. **Test Unit (Vitest)**:
   - `parseCommand()`: fuzzing con payload di injection.
   - `TerminalOutput`: sanitization HTML/ANSI.
   - `SlideOverPanel`: apertura/chiusura/fullscreen.
3. **Test E2E (Playwright)**:
   - Typing commands: `page.keyboard.type('/explore cars')` + `Enter` → assert output.
   - SlideOver flows: trigger comando → assert panel mount + chunk load.
   - Real-time sync: due browser contexts, stesso progetto → assert sincronizzazione terminale.
   - Security regression: XSS injection → assert testo non eseguito.
4. **Observability**:
   - `AlephErrorBoundary` invia a Sentry con contesto (projectID, ultimo comando, vista attiva).
   - Log strutturati per tutti i 12 client RPC (latenza, gRPC status code).
5. **Bundle budget**: 150KB gzipped per entry chunk. CI fallisce se superato.
6. **Lighthouse CI**: performance, accessibility, best practices.

---

## FASE 13 — Yjs, Command History & Storage Cleanup

1. **Yjs cleanup**: Effetto one-time che chiama `yMap.delete('activeTab')` per tutti i peer.
2. **`commandHistory`**: persistenza in `sessionStorage` (non `localStorage`). Max 50 comandi. Non include testo dell'API key o segreti.
3. **Rimozione finale `activeTab`**: solo dopo build check OK e zero riferimenti via `grep`.
4. **Storage audit**: assicurarsi che `localStorage` non contenga API keys, token, o dati progetto.

---

## FASE 14 — Build Check + E2E Finale

1. `cd frontend && npx tsc --noEmit` — 0 errori.
2. `cd frontend && npx vite build` — verifica bundle < 150KB entry, chunks logici.
3. `cd frontend && npm audit` — 0 vulnerabilità critiche.
4. Playwright E2E suite — tutti i percorsi principali:
   - `/explore`, `/ontology`, `/data`, `/skills`, `/tools`, `/components`, `/settings`, `/help`, `/clear`
   - SlideOver open/close/fullscreen
   - Onboarding → Wizard → Terminale
   - Streaming agente
5. Verifica coerenza palette dark su tutte le view.

---

## Checklist Pre-Build

- [ ] `activeTab` marcato `@deprecated`, non più usato in nessun componente.
- [ ] `slideOverContent`, `sandboxResult`, `sandboxInput` inizializzati e funzionanti.
- [ ] `AlephErrorBoundary` attorno a app, InlineRenderer, SlideOverPanel.
- [ ] `parseCommand` allow-list implementata + conferma per comandi mutanti.
- [ ] Agent output in plain text escaped (no HTML rendering).
- [ ] `sessionStorage` per API key, niente `localStorage` per segreti.
- [ ] Vite manual chunks configurati, budget entry 150KB.
- [ ] `prefetchView` utility presente.
- [ ] Terminal effects (scanline, glow) rispetta `prefers-reduced-motion`.
- [ ] Tutte le 11 view migrated a dark palette.
- [ ] Tutti i modali interni convertiti a SlideOverPanel.
- [ ] i18n stringhe corrette (IT, tranne termini tecnici).
- [ ] Domain-specific hooks spezzati da `useViewActions` monolite.

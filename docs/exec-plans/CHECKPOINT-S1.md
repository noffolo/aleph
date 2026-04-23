# Aleph-v2 Redesign — Checkpoint Sessione

## Data: 2026-04-21
## Sessione: Redesign Terminal-First UI/UX (iniziata)

---

### ✅ Completato

#### 1. Build Check
- **Frontend (`npm run build`)**: OK. Chunk JS 833KB (ottimizzabile con lazy loading). 1 moderate vulnerability npm.
- **Backend (`go build ./...`)**: OK. Nessun errore.
- **Sidecar Python (`python3 -m py_compile main.py`)**: OK. Sintassi valida.

#### 2. Architettura & Piano
- Creato piano dettagliato: `docs/exec-plans/redesign-terminal-copilot.md`
- Definite 8 Fasi con dipendenze e criteri di accettazione.

#### 3. Refactor Store (`useStore.ts`)
Aggiunti campi per la navigazione terminale:
- `currentView: 'copilot' | 'inline'`
- `inlineContent: InlineContent | null`
- `showInlinePanel: boolean`
- `commandHistory: string[]` + `addToHistory(cmd)`

#### 4. Refactor Slash Commands (`slashCommands.ts`)
- Ristrutturato da `execute(context)` a `execute(): CommandResult`
- Ogni comando restituisce un oggetto strutturato: `{ handled, action, target, args, message }`
- Aggiunto `formatHelp()` per `/help`.
- Rimosso accoppiamento diretto con `setActiveTab`.

#### 5. Integrazione in `App.tsx`
- Aggiunta `handleCommandResult()` che smista le azioni:
  - `SHOW_INLINE` → imposta `inlineContent` e apre pannello laterale
  - `CLEAR_CHAT` → svuota chat
  - `SWITCH_COPILOT` → aggiunge messaggio di sistema
- Modificato `onSend()` per eseguire `executeCommand()` PRIMA di inviare al backend.
- Se il comando è riconosciuto, non viene inviato all'agente LLM.

#### 6. Creazione `InlineRenderer.tsx`
- Componente che renderizza le viste esistenti dentro un pannello inline.
- Usa `React.lazy()` + `Suspense` per code-splitting (ottimizza il chunk iniziale).
- Supporta tutte le viste: Explorer, Agents, Ontology, Data Sources, Data Health, Skills, Tools, Components, Settings, Library.
- Header con titolo + pulsante CLOSE.

#### 7. `TerminalPrompt.tsx`
- Aggiunta navigazione cronologia con ArrowUp/ArrowDown.
- Highlight del prefisso `λ` con glow.
- Focus automatico all'avvio.

---

### 🔄 In Progress / Prossimi Task

#### Fase 6: Terminal Effects
- [ ] Creare `TerminalEffects.tsx` (CRT curvature, scanline overlay, bloom glow)
- [ ] Aggiungere CSS custom properties per effetti dinamici
- [ ] Supportare `prefers-reduced-motion`

#### Fase 5: Copilot come Home
- [ ] Aggiornare `Sidebar.tsx`: click su icona → trigger comando `/switch` invece di `setActiveTab`
- [ ] Rimuovere/nascondere tab bar tradizionale in `App.tsx`
- [ ] Aggiungere stato di benvenuto nel Copilot

#### Fase 4: Adattamento Viste Inline
- [ ] Aggiungere prop `inline?: boolean` a tutte le viste (ExplorerView, AgentsView, ecc.)
- [ ] Quando `inline=true`, rimuovere header ridondante e adattare layout (padding, overflow)

#### Integrazione Finale
- [ ] Importare `InlineRenderer` in `CopilotView.tsx` e posizionarlo nell'area messaggi o laterale
- [ ] Verificare che i tooltip/modali interni non escano dai container inline

#### QA & Build
- [ ] Eseguire build frontend e correggere eventuali errori TypeScript
- [ ] Lighthouse performance check
- [ ] Test E2E: flusso `/explore` → vista inline → close → `/settings`

---

### 🗂️ File Modificati / Creati

| File | Azione |
|------|--------|
| `frontend/src/store/useStore.ts` | Modificato |
| `frontend/src/components/terminal/slashCommands.ts` | Riscritto |
| `frontend/src/App.tsx` | Modificato |
| `frontend/src/components/terminal/TerminalPrompt.tsx` | Modificato |
| `frontend/src/components/terminal/InlineRenderer.tsx` | Creato |
| `docs/exec-plans/redesign-terminal-copilot.md` | Creato |

### ⚠️ Note Tecniche

- **Chunk JS**: 833KB → da ottimizzare con `React.lazy` già applicato in `InlineRenderer`. Verificare se altri componenti pesanti (D3, Leaflet) possono essere lazy-loaded.
- **Yjs / WebrtcProvider**: Intatto, nessuna modifica alla logica di sincronizzazione.
- **Vite proxy config**: Invariata, funziona con backend Go su :8080.
- **TypeScript**: Verificare che `InlineContent` sia importato correttamente dove necessario.

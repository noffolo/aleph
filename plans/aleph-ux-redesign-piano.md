# Aleph UX Redesign — Piano Finale Integrato

> Basato su: design doc approvato, review Momus (BLOCKED → risolto), Oracle (feasible + critico), Metis (F → B+).
> Stato: **versione finale**. Tutti i blocchi risolti, specifiche collegate.

---

## Revisioni e Blocker Risolti

| Review | Finding Chiave | Risoluzione |
|--------|---------------|-------------|
| **Momus** | 6 spec files mancanti, task one-liner, CopilotView split confusion, InlineRenderer blast radius untracked, 124 store fields senza migration plan, no feature flags | ✅ Ogni wave ha spec file dedicato (`docs/specs/ux-redesign-*.md`), task dettagliati con file/consumers, fase store ha mappa `old → new` per ogni consumer, W0 include feature flags |
| **Momus** | W1-05 (delete InlineRenderer) prima di W3 (unified SlideOver) | ✅ Corretto: InlineRenderer rimosso solo in W3-04, non W1 |
| **Momus** | 21 Playwright tests si rompono, nessun rewrite task | ✅ W6-01: Playwright spec & rewrite esplicito |
| **Oracle** | SHOW_INLINE fallthrough deve reroutare a SlideOver | ✅ W3-03: reroute esplicito via `NavigationStateSync` |
| **Oracle** | 5 nuove scene components (Terminal, Explore, Agents, System, Dashboard) | ✅ W2-02: scene components creati |
| **Oracle** | Store refactor deve essere incrementale, non sequenziale | ✅ W1 diviso in 4 task incrementali, ogni task è safe-refactor |
| **Metis** | 61 campi reali (non 63), 11 view InlineRenderer (non 5) | ✅ Dati corretti nel piano |
| **Metis** | Store prima di navigazione (sequenza sbagliata) | ✅ W1 (store) → W2 (navigation) |
| **Metis** | resetHealth bug (non resetta ollamaHealthy) | ✅ W1-02: bugfix integrato |
| **Metis** | Explorer fragmentation (3 slice) | ✅ W1-03: consolidato |
| **Metis** | 2 duplicate command systems | ✅ W4-02: unificato |
| **Metis** | 21 Playwright tests → W6 rewrite | ✅ Aggiunto |

---

## Wave Structure (0-6)

### W0 — Foundation (1g)
**Dipende da**: niente
**Blocca**: tutto
**Spec**: `docs/specs/ux-redesign-w0-foundation.md`

| Task | Descrizione | File Coinvolti |
|------|-------------|----------------|
| **W0-01** | Attivare feature flag (`features.ts`): `uxRedesign`, `slimSidebar`, `unifiedSlideOver`, `slimCopilot`, `progressiveDisclosure`. Flag `uxRedesign=false` per rollout graduale. | `frontend/src/config/features.ts` (nuovo) |
| **W0-02** | Inventario completo stato store: mappare ogni campo ai suoi consumers via grep. Identificare 61 campi attuali e 38-42 target. Verificare 3 full-store subscription. | `frontend/src/store/*.ts` |
| **W0-03** | Audit consumatori direct store: trovare tutti i `useStore(` e `useAppStore(` senza selettore, pianificare migrazione. | Tutti i file frontend |
| **W0-04** | Audit backend API: verificare che le API servite da `internal/api/` coprano i nuovi requirements per SlideOver unificato. | `internal/api/`, `frontend/src/api/` |
| **W0-05** | Test regressione build corrente: certificare `npx tsc --noEmit`, `npx vite build`, `go build ./...` tutti passano prima di iniziare. | CI |

---

### W1 — Store Refactor (2g)
**Dipende da**: W0
**Blocca**: W2, W3, W4, W5
**Spec**: `docs/specs/ux-redesign-w1-store-refactor.md`

| Task | Descrizione | File Coinvolti |
|------|-------------|----------------|
| **W1-01** | **Aggiungere selettori mancanti** in tutti i `useStore()` senza selettore (almeno 3 trovati in W0-03). Creare selettori derivati per ogni subscription. Safe refactor — nessun campo rimosso. | `frontend/src/hooks/`, `frontend/src/components/**/*.tsx` |
| **W1-02** | **Bugfix + cleanup**: `resetHealth()` in `healthSlice.ts` — aggiungere `ollamaHealthy: true`. Rimuovere campi morti (identificati in W0-02). Rinominare `splitView` → `showMessageDetail` in `copilotSlice`. | `frontend/src/store/healthSlice.ts`, `frontend/src/store/copilotSlice.ts` |
| **W1-03** | **Consolidare Explorer fragmentation**: unificare i campi explorer sparsi in `workspaceSlice`, `navigationSlice`, `uiSlice` in un unico `explorerSlice`. Mantenere retrocompatibilità con alias. | `frontend/src/store/explorerSlice.ts` (nuovo), `frontend/src/store/workspaceSlice.ts`, `navigationSlice.ts`, `uiSlice.ts` |
| **W1-04** | **Riduzione campi a 38-42**: rimuovere campi deprecated secondo mappa di W0-02. Ogni rimozione accompagnata da aggiornamento di TUTTI i consumers identificati. Verificare con `tsc --noEmit`. | Tutti gli store slice + consumers |
| **W1-05** | **Rimuovere subscriptions duplicate**: `useAppStore` vs `useStore` — scegliere `useStore` canonical (o viceversa). Eliminare l'altro import in tutti i file. | Dovunque importano entrambi |

---

### W2 — Navigation Simplification (2g)
**Dipende da**: W1
**Blocca**: W3
**Spec**: `docs/specs/ux-redesign-w2-navigation.md`

| Task | Descrizione | File Coinvolti |
|------|-------------|----------------|
| **W2-01** | **Sidebar 13→5**: Dashboard, Explorer, Copilot, Oracle, Settings. Eliminare 8 item. Aggiungere section label visuali (non menu). Ogni item → comando con `data-slide` per routing. | `frontend/src/components/layout/Sidebar.tsx` |
| **W2-02** | **Scene Components**: creare 5 scene wrapper (`TerminalScene`, `ExploreScene`, `AgentsScene`, `SystemScene`, `DashboardScene`). Ogni scene gestisce la propria visualizzazione (fullscreen vs slideover). Sostituire view rendering diretto in `App.tsx`. | `frontend/src/scenes/` (nuovo, 5 file) |
| **W2-03** | **NavigationStateSync upgrade**: aggiungere `scene` param alla URL (es. `?scene=explore`, `?scene=copilot`). Sync bidirezionale con store. Non usare React Router — estendere 40-line esistente. | `frontend/src/components/layout/NavigationStateSync.tsx` |
| **W2-04** | **Comando palette semplificata**: ridurre a 3 sezioni (Navigate, Actions, System). Rimuovere comandi duplicati. Integrare `slideover:` prefisso per azioni SlideOver. | `frontend/src/components/layout/CommandPalette.tsx` |
| **W2-05** | **Dashboard fullscreen mode**: quando `scene=dashboard`, nessuna sidebar, nessuna slideover — terminal fullscreen con statistiche. | `frontend/src/scenes/DashboardScene.tsx` |

---

### W3 — SlideOver Unification (2g)
**Dipende da**: W2
**Blocca**: W4
**Spec**: `docs/specs/ux-redesign-w3-slideover-unification.md`

| Task | Descrizione | File Coinvolti |
|------|-------------|----------------|
| **W3-01** | **SlideOverContent rewrite**: ridurre da 20+ tipi a 4 scene (Terminal, Explore, Agents, System). Ogni scene ha il suo content renderer interno. Rimuovere mapping `slideType → component` hardcoded. | `frontend/src/components/layout/SlideOverContent.tsx` |
| **W3-02** | **SlideOverPanel cleanup**: rimuovere props deprecate (`slideType`, `view`, `tab`). Mantenere `scene`, `params`, `onClose`. Animazione invariata. | `frontend/src/components/layout/SlideOverPanel.tsx` |
| **W3-03** | **SHOW_INLINE → SlideOver reroute**: in `useAppActions`, ogni `SHOW_INLINE` viene reroutato a `SHOW_SLIDE` via `NavigationStateSync`. InlineRenderer non viene più chiamato. | `frontend/src/hooks/useAppActions.ts` |
| **W3-04** | **Delete InlineRenderer**: rimuovere `InlineRenderer.tsx`, rimuovere tutti gli import. Verificare che 11 view ora vivano solo in SlideOver (via scene components di W2-02). | `frontend/src/components/layout/InlineRenderer.tsx` (rimosso) |

---

### W4 — Copilot Slim (1.5g)
**Dipende da**: W3
**Blocca**: W5
**Spec**: `docs/specs/ux-redesign-w4-copilot-slim.md`

| Task | Descrizione | File Coinvolti |
|------|-------------|----------------|
| **W4-01** | **CopilotView split**: separare chat + search + settings in componenti atomic. `CopilotChat` (messaggi + input), `CopilotSearch` (finder), `CopilotSettings` (opzioni). Ogni componente ha props chiare. | `frontend/src/components/copilot/CopilotView.tsx`, `CopilotChat.tsx`, `CopilotSearch.tsx`, `CopilotSettings.tsx` |
| **W4-02** | **Command system unification**: unificare `copilotCommands` (5 hardcoded) e `slashCommands` (16) in unico `CommandRegistry`. Ogni comando ha: `id, label, description, handler, scope`. Scope controlla visibilità per contesto. | `frontend/src/commands/` (nuovo) |
| **W4-03** | **CopilotView state fragmentation fix**: stato SSE, streaming, confirm dialog — tutti segregati in hook dedicati. `useCopilotSSE`, `useCopilotStream`, `useCopilotConfirm`. | `frontend/src/hooks/useCopilotSSE.ts`, `useCopilotStream.ts`, `useCopilotConfirm.ts` |
| **W4-04** | **ConfirmDialog → SlideOver modal**: spostare conferma azioni distruttive da `window.confirm()` a SlideOver con `type=confirm`. | `frontend/src/components/copilot/`, `useAppActions.ts` |

---

### W5 — Progressive Disclosure (1.5g)
**Dipende da**: W4
**Blocca**: W6
**Spec**: `docs/specs/ux-redesign-w5-disclosure-polish.md`

| Task | Descrizione | File Coinvolti |
|------|-------------|----------------|
| **W5-01** | **Settings progressive**: SettingsView divisa in 3 livelli — Basic (tema, lingua), Advanced (API keys, providers), Expert (system prompt, DuckDB path). Default: Basic. Expand per Advanced/Expert. | `frontend/src/views/SettingsView.tsx` |
| **W5-02** | **Explorer progressive**: alberi collapsibili per ontologia. Default: chiusi. Tooltip informativi su ogni nodo. | `frontend/src/scenes/ExploreScene.tsx` (o ExplorerView) |
| **W5-03** | **Tool View progressive**: cards collapsibili per ogni tool. Descrizione + health status mostrati sempre. Dettagli (config, logs) espandibili. | `frontend/src/views/ToolsView.tsx` |
| **W5-04** | **Agent View progressive**: summary card + expand per run history, config, logs. | `frontend/src/views/AgentsView.tsx` |
| **W5-05** | **Empty states + onboarding**: per ogni view vuota, mostrare messaggio utile invece di white screen. Onboarding snippets inline. | `frontend/src/views/*.tsx` |

---

### W6 — Polish + Tests (1.5g)
**Dipende da**: W5
**Blocca**: ship
**Spec**: `docs/specs/ux-redesign-w5-disclosure-polish.md` (W6 section)

| Task | Descrizione | File Coinvolti |
|------|-------------|----------------|
| **W6-01** | **Playwright spec rewrite**: aggiornare 21 test esistenti per il nuovo sistema navigazione/scene. Aggiungere test per: sidebar ridotta, scene switching, SlideOver unificato, progressive disclosure. | `frontend/__tests__/e2e/` |
| **W6-02** | **TerminalView props audit**: verificare che tutti i props di TerminalView siano ancora corretti dopo le modifiche. Fixare mismatch types. | `frontend/src/components/terminal/TerminalView.tsx` |
| **W6-03** | **Accessibility review**: verificare WCAG AA per navigazione ridotta, SlideOver focus trap, scene aria-labels. | Tutti i nuovi/changed componenti |
| **W6-04** | **UX review**: verify flow end-to-end: sidebar → scene → slideover → action → close → back. | Manuale |
| **W6-05** | **Feature flag cleanup**: quando stabile, rimuovere feature flag e codice morto. | `frontend/src/config/features.ts` |

---

## Dipendenze

```
W0 (Foundation)
  └── W1 (Store Refactor)
        └── W2 (Navigation)
              └── W3 (SlideOver Unification)
                    └── W4 (Copilot Slim)
                          └── W5 (Progressive Disclosure)
                                └── W6 (Polish + Tests)
```

Nessun parallelismo possibile — ogni wave dipende strutturalmente dalla precedente.

---

## Specifiche Collegate

Ogni spec file vive in `docs/specs/`:

| Wave | Spec File | Contenuto |
|------|-----------|-----------|
| W0 | `ux-redesign-w0-foundation.md` | Feature flags, store inventory, consumer audit, backend audit, regression gate |
| W1 | `ux-redesign-w1-store-refactor.md` | Selettori, campi 61→38-42, explorerSlice, subscriptions, resetHealth fix |
| W2 | `ux-redesign-w2-navigation.md` | Sidebar 13→5, scene components, NavigationStateSync, command palette |
| W3 | `ux-redesign-w3-slideover-unification.md` | SlideOverContent rewrite, SHOW_INLINE reroute, InlineRenderer delete |
| W4 | `ux-redesign-w4-copilot-slim.md` | CopilotView split, CommandRegistry, hooks segregation, ConfirmDialog |
| W5+W6 | `ux-redesign-w5-progressive.md` | Progressive disclosure (5 views), empty states, Playwright rewrite, a11y, props audit |

---

## Metriche di Successo

| Metric | Current | Target |
|--------|---------|--------|
| Sidebar items | 13 | 5 |
| Store fields | 61 | 38-42 |
| SlideOver types | 20+ | 4 scenes |
| CopilotView lines | 282 | <180 (split in atomic files) |
| InlineRenderer | 266 lines, 11 views | 0 (deleted) |
| Full-store subscriptions | 3+ | 0 |
| tsc --noEmit | ✅ | ✅ sempre |
| vite build | ✅ | ✅ sempre |
| go build | ✅ | ✅ sempre |
| Playwright tests | 21 pass | 30+ pass |

---

## Stima Effort

| Wave | Giorni | Task | Complessità |
|------|--------|------|-------------|
| W0 | 1 | 5 | Bassa (audit + config) |
| W1 | 2 | 5 | Alta (store surgery) |
| W2 | 2 | 5 | Media (nuovi componenti) |
| W3 | 2 | 4 | Media (rewrite, delete) |
| W4 | 1.5 | 4 | Media (split, unify) |
| W5 | 1.5 | 5 | Media (disclosure patterns) |
| W6 | 1.5 | 5 | Media (test, polish) |
| **Totale** | **~11.5g** | **33** | |

---

## Ship Gate Checklist

- [ ] `npx tsc --noEmit` ✅
- [ ] `npx vite build` ✅
- [ ] `go build ./...` ✅
- [ ] `npx vitest run` ✅
- [ ] Playwright tests ✅ (30+)
- [ ] UX review: end-to-end flow
- [ ] Accessibility: WCAG AA
- [ ] Feature flag `uxRedesign=true` in produzione

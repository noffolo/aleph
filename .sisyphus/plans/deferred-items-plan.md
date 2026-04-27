# Piano Differiti Aleph-v2

## Ordine: W6-08 → W6-04 → W6-02 → W2-05

## W6-08: URL State (~1gg)

**Obiettivo:** Sincronizzare navigationSlice con URL tramite nuqs per copia/incolla bookmark funzionante, back/forward browser.

**Task:**
1. `frontend/src/hooks/useSyncNavigationState.ts` — hook che all'avvio legge `?view=&tab=&slide=` da URL e popola navigationSlice
2. `frontend/src/store/navigationSlice.ts` — dopo `setCurrentView/setActiveView/setSlideOverContent`, aggiornare URL via nuqs `useQueryState`
3. `frontend/src/components/AgentsView.tsx` — `useQueryState('q')` per search query filter
4. `frontend/src/components/ToolsView.tsx` — idem
5. `frontend/src/components/SkillsView.tsx` — idem
6. Verificare che URL copiabile ricarichi stato correttamente
7. `npx tsc --noEmit` + `npx vite build` ✅

## W6-04: Yjs Cleanup (~1gg)

**Obiettivo:** Rimuovere yjs + y-webrtc, sostituire con backend SSE esistente.

**Task:**
1. `internal/api/sse/sse.go` — aggiungere `BroadcastWorkspace(workspaceId, payload)` 
2. `frontend/src/store/workspaceSlice.ts` — rimuovere `import * as Y from 'yjs'`, `WebrtcProvider`, sostituire `yMap.observe` con SSE subscription
3. `frontend/src/store/useStore.ts` — rimuovere `WebrtcProvider` init
4. `npm uninstall yjs y-webrtc` da frontend
5. Aggiornare `workspaceSlice.test.ts`
6. `npx tsc --noEmit` + `npx vite build` ✅ (verifica bundle ridotto ~45KB)

## W6-02: i18n (~3gg)

**Obiettivo:** Sostituire 97+ stringhe ITA hardcoded con i18next + react-i18next, EN default + IT fallback.

**Task:**
1. `npm install i18next react-i18next`
2. `frontend/src/i18n/index.ts` — init i18next
3. `frontend/src/i18n/locales/en/common.json` + `it/common.json`
4. `frontend/src/i18n/locales/en/views.json` + `it/views.json`
5. Wrap App.tsx con Suspense + provider
6. Migrare 10+ componenti (AgentsView, AgentForm, SkillForm, ToolsView, CopilotView, SettingsView, EmptyState, StatusBar, 4 form SlideOver, DataHealthView, ToolManagementView)
7. Aggiungere `locale` a `uiSlice.ts` + localStorage persist
8. Selettore lingua in SettingsView
9. `npx tsc --noEmit` + `npx vite build` ✅

## W2-05: GNN Wire (~1gg)

**Obiettivo:** Wireare GNN link prediction esistente in DecisionEngine.

**Task:**
1. `internal/gnn/wire.go` — NewGNNEvaluator(embedder) che prende workspace embeddings e ritorna score
2. `internal/decision/decider.go` — GNNEvaluator opzionale
3. `internal/gnn/gnn_wire_test.go` — test integrazione
4. Config flag `--gnn-enabled` in app.go (default false)
5. `go build ./...` + `go test ./internal/gnn/` ✅

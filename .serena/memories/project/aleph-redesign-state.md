# Aleph-v2 Terminal Redesign — Session State

## DONE ✅

### Bug Fixes
1. main.go:44 — Graceful shutdown with `aleph.Close(ctx)`, added context import
2. engine.go:421 — `runURLFetch` uses `safeHTTPClient` (SSRF fix)
3. engine.go:408 — `validateHTTPRequest` validates actual URL not constructed one
4. probe.go:21 — Uses `safeHTTPClient` (SSRF fix)
5. nlp/main.py:55 — `sanitize_identifier()` for context_id before SQL interpolation
6. query.go:120 — Package-level `validName` regex for object name validation
7. query.go:216 — GetDataStats validates objName before use
8. auth/auth_service.go + test — REMOVED (dead code)

### Redesign Sprint 1 ✅
- design-tokens.json — dual dark/light theme (dark default)
- tailwind.config.js — CSS var colors, darkMode: 'class'
- index.css — theme vars + terminal utilities + blink keyframe
- terminal/TerminalPrompt.tsx — λ-prefix textarea
- terminal/TerminalOutput.tsx — TerminalLine[] renderer
- terminal/StatusBar.tsx — bottom health/status bar
- terminal/index.ts — barrel export

### Redesign Sprint 2 ✅
- Sidebar.tsx — 48px icon rail, sections with dividers
- App.tsx — terminal layout (bg-background text-text font-mono), removed header, used StatusBar, terminal-styled modals + error bar, preserved all business logic
- CopilotView.tsx — terminal chat with TerminalPrompt + TerminalOutput, agent selector
- slashCommands.ts — 14 commands, parseCommand, getTabCompletion, executeCommand

### Redesign Sprint 3 ✅
- DataPanel.tsx — slide-in panel, ESC close, mono, terminal colors
- DetailPanel.tsx — terminal-styled
- ExplorerView.tsx — pills, mono search, compact toggles
- AlephTable.tsx — ASCII-style, compact, font-mono
- AlephGraph.tsx — dark D3 colors

## TODO ⏳
- Sprint 4: Dual theme toggle + remaining views styling
- Sprint 5: Advanced slash commands + animations + shortcuts
- Verification: Frontend build tsc + vite, Go build+test, docker-compose, e2e
- Install deps: react-markdown, react-syntax-highlighter, framer-motion

## Key Files Changed
```
frontend/src/styles/design-tokens.json
frontend/tailwind.config.js
frontend/src/index.css
frontend/src/components/terminal/TerminalPrompt.tsx
frontend/src/components/terminal/TerminalOutput.tsx
frontend/src/components/terminal/StatusBar.tsx
frontend/src/components/terminal/index.ts
frontend/src/components/terminal/slashCommands.ts
frontend/src/components/Sidebar.tsx (rewrite)
frontend/src/App.tsx (rewrite)
frontend/src/components/CopilotView.tsx (rewrite)
frontend/src/components/DataPanel.tsx (new)
frontend/src/components/DetailPanel.tsx (rewrite)
frontend/src/components/ExplorerView.tsx (rewrite)
frontend/src/lib/AlephTable.tsx (rewrite)
frontend/src/lib/AlephGraph.tsx (rewrite)
```

## Agent Model Mapping
| Role | Model |
|------|-------|
| Orchestrator (Kimi K2.6) | kimi-k2.6:cloud |
| Vision | gemma4:27b-cloud |
| Router | ministral-3:8b-cloud |
| Worker 1 | qwen3.5:35b-a3b:cloud |
| Worker 2 | nemotron-3-super:cloud |
| Coder | GLM-5.1 |
| Oracle | GLM-5.1 |
| Revisore | GLM-5.1 |
| Planner | GLM-5.1 |

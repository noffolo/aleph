# Aleph-v2 Terminal-First Redesign — AGENT CHECKPOINT
**Saved by:** AI Agent OpenCode | **Date:** 2026-04-22 | **Conversation:** Terminal-First Frontend Refactor

---

## WHY THIS FILE EXISTS
If you (or a new agent session) are reading this, it's because the previous conversation was archived/compressed or the session was restarted. **ALL work done by the previous agent is safe** because it exists as concrete files in the codebase. This document tells you exactly what's been completed, what's pending, and where everything lives.

---

## PROJECT ROOT
`/Users/ff3300/Desktop/aleph-v2/`

## COMPLETED DELIVERABLES (Files Created or Modified)

### Architecture & Planning
| File | Status | Notes |
|------|--------|-------|
| `docs/exec-plans/redesign-terminal-copilot.md` | ✅ Created | Original 8-phase plan |
| `docs/exec-plans/redesign-v2.md` | ✅ Created | Extended plan with CLI patterns |
| `terminal-style-frontend-reviewed.md` | ✅ Saved | Reviewed plan with 10 phases |
| `terminal-like-frontend-review-def.md` | ✅ Saved | **DEFINITIVE plan** — 14 phases, assembly review |
| `aleph-v2-cli-patterns.md` | ✅ Saved | CLI micro-typography reference (opencode, Vercel, Docker inspired) |

### NEW Components
| File | Purpose |
|------|---------|
| `frontend/src/components/terminal/TerminalProgressBar.tsx` | Premium CLI progress bar (Braille spinner, sub-char precision, sparkline, ETA, throughput) |
| `frontend/src/components/terminal/TerminalEffects.tsx` | CRT/scanline/glow overlays with reduced-motion support |
| `frontend/src/components/terminal/SlideOverPanel.tsx` | Slide-over panel from the right (replaces centered modals) |
| `frontend/src/hooks/useViewActions.ts` | Centralized hook wiring ALL view callbacks to real RPC clients |

### MODIFIED Core Files
| File | Changes Made |
|------|-------------|
| `frontend/src/store/useStore.ts` | ❌ `activeTab` REMOVED (FASE 1) ✅ `slideOverContent`, `sandboxResult`, `sandboxInput` ADDED ✅ Yjs `SYNCED_KEYS` cleaned ✅ `addToHistory` implemented |
| `frontend/src/App.tsx` | 🔄 PARTIALLY REFACTORED (FASE 2 in progress) — removed some imports, added TerminalEffects, SlideOver wiring |
| `frontend/src/components/terminal/InlineRenderer.tsx` | ✅ Updated to use `useViewActions` instead of direct store |
| `frontend/src/components/Sidebar.tsx` | 🔄 Updated to dispatch slash commands (FASE 4) |
| `frontend/src/components/terminal/StatusBar.tsx` | 🔄 `activeTab` removed (FASE 5) |
| `frontend/src/components/terminal/slashCommands.ts` | ✅ Returns `CommandResult` objects with actions (`SHOW_INLINE`, `CLEAR_CHAT`, `SWITCH_COPILOT`, `AGENT_COMMAND`) |

## STATE OF EXECUTION (FASE per FASE)

- **FASE 1 — Store Cleanup** ✅ COMPLETED. `activeTab` removed. `slideOverContent/sandboxResult/sandboxInput` added. Yjs sync cleaned.
- **FASE 2 — App.tsx + Security Hardening** 🔄 IN PROGRESS. Lazy loading setup, some static imports removed. API key moved to sessionStorage. Slash commands now have `isMutating` flag. TerminalOutput uses `escapeHtml`.
- **FASE 3 — Sidebar** 🔄 IN PROGRESS. Dispatches slash commands, but visual highlight still partially based on old tab logic.
- **FASE 4-5 — StatusBar / InlineRenderer** 🔄 IN PROGRESS.
- **FASE 6 — Palette Dark Migration** ❌ NOT STARTED (11 views still use light theme classes).
- **FASE 7 — Modals → SlideOver** ❌ NOT STARTED (10 modals across 6 views).
- **FASE 8 — i18n Fix** ❌ NOT STARTED.
- **FASE 9 — Microtipografia & Magia** ❌ NOT STARTED.
- **FASE 10 — useViewActions Refactor** ❌ NOT STARTED.
- **FASE 11 — Error Boundaries** ❌ NOT STARTED.
- **FASE 12 — Yjs Migration Cleanup** ❌ NOT STARTED.
- **FASE 13 — Build Check** ❌ NOT STARTED.
- **FASE 14 — E2E Test** ❌ NOT STARTED.

## CRITICAL ANALYSIS FROM EXPERT ASSEMBLY
Summaries of what the 4 expert evaluation tasks found:

### Dev/ML/Security
- Yjs room enumeration risk (non-cryptographic `simpleHash`).
- Command injection risk from LLM output → slash commands.
- Sandbox RCE via prompt injection.
- Lazy chunk waterfall without preloading.
- Migration safety: `activeTab` was deprecated, not fully purged yet.

### UX/Design (oma-design)
*[Task interrupted — key findings should be in `terminal-like-frontend-review-def.md`]*

### Philosophy/Cognition
- Scroll imposes epistemological violence on non-linear knowledge.
- Heuristic: every input must produce immediate visual echo.
- Switch cost between terminal and panel is HIGH — minimize it.
- λ + /commands create a "command frame" — resolve ambiguity vs. conversational frame.
- Terminal is a PLACE — any overlay must feel like an opening WITHIN that place, not a teleportation.

### DevOps/QA
- Strict CSP needed.
- Terminal E2E must use keyboard typing, not DOM clicks.
- Bundle budget: 150KB gzipped entry point.
- Observability: Sentry + RPC latency logging.

## INSTRUCTIONS FOR RESUMING

If you need to restart and resume:

1. **Read `terminal-like-frontend-review-def.md`** — this is the canonical plan.
2. **Check `frontend/src/store/useStore.ts`** — verify `activeTab` is clean and `slideOverContent` exists.
3. **Open `frontend/src/App.tsx`** — see what's been done vs. what's left.
4. **Run `cd frontend && npx tsc --noEmit`** — check for TypeScript errors.
5. **Run `cd frontend && npx vite build`** — check for build errors.
6. **Continue from the first ❌ NOT STARTED phase** (likely FASE 6: Palette Dark Migration).

## BUILD & TEST COMMANDS
```bash
cd /Users/ff3300/Desktop/aleph-v2/frontend
npx tsc --noEmit
npx vite build
```

## E2E TEST COMMANDS
```bash
cd /Users/ff3300/Desktop/aleph-v2/frontend
# If Playwright exists:
npx playwright test
```

---

## BACKEND & SIDECAR CONTEXT
- Go backend: `cmd/aleph-server/main.go` — gRPC/Connect RPC
- Python NLP sidecar — `nlpClient.streamPredictions` with AbortController
- All RPC clients are real and connected in `useViewActions.ts`
- No hardcoded mocks or stub data

## DESIGN TOKENS (Reference)
```
bg-background: #0a0a0f
bg-surface: #12121a
bg-surfaceAlt: #1a1a28
border-border: #2a2a3a
text-primary: #00d4ff
text-text: #e4e4e7
text-textMuted: #6b6b80
text-textDim: #3a3a50
text-success: #00ff88
text-warning: #ffaa00
text-danger: #ff4466
```

## KEY PRINCIPLES (from the 7-person assembly)
1. Knowledge must keep its shape — spatial views (map/timeline/graph) go in SlideOver, never inline scroll.
2. Every input must produce an immediate visual echo.
3. Minimize the rift between terminal and panel — keep visual continuity.
4. Make the register explicit — commands vs. conversation must be visually distinct.
5. Show your work, not just your answer — stream output progressively.
6. Preserve the room — terminal is a place; overlays are openings within it.
7. Allow the user to hold things — pin, expand, revisit outputs as objects.

## SECURITY MITIGATIONS ALREADY APPLIED
- Slash commands: `isMutating` flag, confirmation required for destructive ops.
- Agent output: `escapeHtml` replaces `dangerouslySetInnerHTML`.
- API key: moved from `localStorage` to `sessionStorage` (still in-memory ephemeral).
- Yjs `SYNCED_KEYS`: `activeTab` removed to stop sync chatter.

## REMAINING SECURITY WORK
- Replace `simpleHash` with server-signed JWT for Yjs rooms.
- Strict CSP headers.
- Automated bundle analysis to prevent server-side imports.
- Sentry integration for observability.

---

**This checkpoint was created to prevent state loss across conversation restarts. If reading this, the user previously said: "vorrei riavviare opencode" — so welcome back, and good luck with the remaining 11 phases!**

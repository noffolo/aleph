# UX W6-04: End-to-End UX Review Checklist

> **Context:** Aleph-v2 post-UX-W5 (Progressive Disclosure) review
> **Views:** Settings, Tools, Agents, Oracle, Library, Components, Explorer
> **Pattern:** Sidebar → Scene → SlideOver → Action → Close → Back

---

## 1. Navigation Consistency

- [ ] Sidebar items load correct scene when clicked
- [ ] Active sidebar item is highlighted
- [ ] CommandPalette opens via Ctrl+K (or Cmd+K)
- [ ] SlideOver opens with correct content per scene
- [ ] SlideOver closes via backdrop click
- [ ] SlideOver closes via Escape key
- [ ] Focus returns to trigger element after SlideOver close

## 2. Progressive Disclosure (GlassPanel Collapsible)

- [ ] Settings: Quick Summary expanded by default
- [ ] Settings: All Settings collapses/open correctly
- [ ] Settings: Advanced Developer hidden behind gear toggle
- [ ] Tools: Overview collapsed starts expanded
- [ ] Tools: Tools grid collapsible
- [ ] Tools: Tool Details collapsible
- [ ] Agents: Grid section collapsible
- [ ] Oracle: Predictions section collapsible
- [ ] Oracle: Sentiment section collapsible
- [ ] Oracle: Advanced Analytics collapsible
- [ ] Library: Asset grid collapsible
- [ ] Components: Grid collapsible
- [ ] Collapse/expand state persists per session (Zustand expandedSections)

## 3. Empty States

- [ ] Tools view: "Nessun tool configurato" when empty
- [ ] Agents view: "Nessun agente configurato" when empty
- [ ] Library view: "Nessun asset" when empty
- [ ] Components view: "Nessun componente" when empty
- [ ] Oracle view: renders loading state before data
- [ ] Settings: "Nessuna chiave API" when no keys
- [ ] Settings: "Nessun canale" when no notifications

## 4. Data Display

- [ ] Tool cards show name, status indicator, description
- [ ] Agent grid shows name, model, system prompt preview
- [ ] Oracle predictions show loading → data
- [ ] Oracle sentiment shows loading → data
- [ ] Library asset grid renders cards with icons/names
- [ ] Component grid renders categories
- [ ] Settings toggle switches (scanlines/glow/flicker) work independently

## 5. SlideOver Panel

- [ ] Opens with correct scene content
- [ ] Title/header matches scene context
- [ ] glass-panel styling applied (backdrop-filter blur, border)
- [ ] Close button in header
- [ ] Fullscreen toggle works
- [ ] Content area scrollable
- [ ] z-index prevents interaction behind panel

## 6. Terminal / Copilot

- [ ] TerminalView renders CopilotView correctly
- [ ] Input accepts text
- [ ] Enter triggers send
- [ ] Agent selector dropdown works
- [ ] SSE status indicator shows current connection state
- [ ] Command palette (/commands) filter works
- [ ] Chat messages render with proper formatting

## 7. Visual Polish

- [ ] Dark palette consistent (#080810/#0e0e18/#141420)
- [ ] Font: JetBrains Mono applied throughout
- [ ] Glass panel border matches design tokens
- [ ] No layout shifts during transition
- [ ] Scrollbar styling consistent
- [ ] Hover states on interactive elements

## 8. Error Resilience

- [ ] InlineError or ErrorBoundary shows on component crash
- [ ] Loading skeleton shows during async operations
- [ ] Empty state shows when no data available
- [ ] Toast notifications appear for success/error actions

---

**Review Method:** Manual walkthrough of each view + SlideOver combination.
**After review:** Fix any issues found, then proceed to Phase 6 (E2E verification).

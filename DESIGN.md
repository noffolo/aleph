# Aleph-v2 Frontend Design System

> Aleph Data OS — a terminal-native operating system interface for data
>
> **Stack**: React 18.3 · TypeScript 5.5 · Vite 8 · Zustand 4.5 · ConnectRPC-Web · Tailwind CSS 3.4 · D3 7.9 · Leaflet 1.9 · Zod 3.23
>
> **Last updated**: 2026-05-11

---

## Design Philosophy

### Aleph is an Operating System for Data, Not a Website

Aleph rejects the metaphor of a "web application." Every design decision reinforces the idea that the user is operating a system — issuing commands, inspecting state, managing processes — not browsing pages. The browser is merely the display terminal.

### Terminal Aesthetic with Modern UX Patterns

The interface is a hybrid: the primary interaction mode is a terminal with a blinking cursor (`█`) and a prompt (`λ`), but panels slide in with glassmorphism surfaces for structured data exploration. This bridges the gap between power-user efficiency and information density.

### Dark-First, Glassmorphism Design

Dark mode is the default and only visual mode. The palette anchors on near-black `#080810` with cyan `#00d4ff` as the interaction color. Panels use glassmorphism (`backdrop-filter: blur(12px)` over semi-transparent `rgba(14,14,24,0.7)`) to create depth without breaking the terminal illusion.

### Volatility Visual Hierarchy

A four-layer volatility system categorizes every UI element by its dynamism:

| Layer | Class | Behavior | Examples |
|-------|-------|----------|---------|
| **Static** | `.vol-static` | No transitions, no animations | Background, borders, structural layout |
| **Structural** | `.vol-structural` | Fade-in on mount (250ms) | Panels, sidebars, the shell itself |
| **Interactive** | `.vol-interactive` | Hover transitions (150ms) | Buttons, list items, clickable surfaces |
| **Signal** | `.vol-signal` | Pulse animation (50ms, 3 cycles) | Status changes, transient alerts |

### Code as Interface

The terminal is the primary interaction mode. Slash commands (`/explore`, `/agent`, `/predict`, `/tools`, `/library`) drive navigation. Structured views (agent forms, data explorers, health dashboards) are secondary — accessed through the terminal or sidebar, never as the default.

---

## Visual Design System

### Color Palette

All values are extracted from `frontend/src/styles/design-tokens.json` and mapped to CSS custom properties on `:root`.

#### Dark (Default)

| Token | CSS Variable | Hex | Usage |
|-------|-------------|-----|-------|
| Background | `--color-background` | `#080810` | App shell, terminal area |
| Surface | `--color-surface` | `#0e0e18` | Panels, sidebar, status bar |
| Surface Alt | `--color-surfaceAlt` | `#141420` | Hover states, elevated surfaces |
| Border | `--color-border` | `#2a2a3a` | Dividers, panel borders |
| Primary | `--color-primary` | `#00d4ff` | Accent, links, active indicators |
| Primary Muted | `--color-primaryMuted` | `#0099bb` | Hover borders, muted accents |
| Success | `--color-success` | `#00ff88` | Data health: good, operational |
| Warning | `--color-warning` | `#ffaa00` | Data health: degraded, caution |
| Danger | `--color-danger` | `#ff4466` | Data health: critical, errors |
| Text | `--color-text` | `#e4e4e7` | Body text |
| Text Muted | `--color-textMuted` | `#6b6b80` | Secondary text, metadata |
| Text Dim | `--color-textDim` | `#3a3a50` | Placeholder, disabled, timestamps |

#### Light (Unused — Structural Only)

A light theme exists in tokens and CSS (`.theme-light`) for API compatibility but is **not togglable** in the UI. All surfaces assume dark mode.

- **Primary contrast ratio**: 11.7:1 against `#080810` (exceeds WCAG AAA)
- **Success contrast ratio**: 10.3:1 against `#080810`
- **Text contrast ratio**: 13.2:1 against `#080810`

### Typography System

| Token | Value | Usage |
|-------|-------|-------|
| Primary font family | `Inter, -apple-system, BlinkMacSystemFont, sans-serif` | Data panels, forms, structured views |
| Monospace font family | `JetBrains Mono, Menlo, monospace` | Terminal output, code input, status bars |
| Body size | `13px` / `1.25` leading (`.text-body`) | Default text in terminal views |
| Meta size | `11px` / `1.25` leading (`.text-meta`) | Timestamps, status indicators, secondary info |
| Tabular nums | `font-variant-numeric: tabular-nums` | Numbers, data columns, metrics |
| Ligatures | `font-variant-ligatures: none` | Disabled for code clarity |

**Weights**: 400 (normal) · 500 (medium) · 600 (semibold) · 700 (bold) · 800 (extrabold) · 900 (black)

**Principle**: Terminal views (chat, output, command input) use JetBrains Mono exclusively. Panel views (AgentsView, SkillsView, data tables) use the system font stack for readability at small sizes. The 13px body size is a deliberate departure from the 16px web standard — it matches terminal emulator conventions and maximizes information density.

### Spacing & Rhythm

- **Base unit**: `4px`
- **Scale**: 4px · 8px · 12px · 16px · 20px · 24px · 32px · 40px · 48px
- **Grid**: 8px gap is the standard rhythm (`.grid-8`)
- **Padding**: Terminal views use `px-4 py-3` (16px horizontal, 12px vertical); panels use `px-5` (20px)
- **Status bar**: 28px (`h-7`), typography at 10px with tracking-widest

### Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| Terminal | `0` (`.radius-terminal`) | All terminal surfaces — zero radius |
| Card | `8px` (`.radius-card`) | Panel content, dashboard widgets |
| `--radius-sm` | `0.5rem` (8px) | Buttons, inputs |
| `--radius-md` | `0.75rem` (12px) | Dropdowns, dialogs |
| `--radius-lg` | `1rem` (16px) | Modals, slide panels |
| `--radius-xl` | `1.5rem` (24px) | Large containers |
| `--radius-2xl` | `2rem` (32px) | Hero elements |

**Principle**: Terminal areas are strictly rectangular (radius: 0). Non-terminal surfaces (panels, dialogs, dashboard cards) use subtle rounding (8px). This draws a visual boundary between "system" chrome and "application" content.

### Elevation & Shadows

| Token | Value | Usage |
|-------|-------|-------|
| Flat | `0 0 0 0 transparent` | Default state |
| Low | `0 1px 2px rgba(0,0,0,0.3)` | Subtle surface separation |
| Mid | `0 4px 12px rgba(0,0,0,0.4)` | Slide panels, dropdowns |
| High | `0 8px 24px rgba(0,0,0,0.5)` | Modals, command palette |

### Glassmorphism

Defined in `index.css` as the `.glass-panel` utility class:

```css
.glass-panel {
  background: rgba(14, 14, 24, 0.7);    /* --color-surface at 70% opacity */
  backdrop-filter: blur(12px);            /* Frosted glass effect */
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(42, 42, 58, 0.5); /* --color-border at 50% opacity */
}
```

Used for: `SlideOverPanel`, dashboard cards, overlay panels. The semi-transparency allows the terminal background to show through, maintaining spatial context. A solid variant (`.glass-panel-solid`) exists for performance-sensitive surfaces.

### Terminal Aesthetic Details

- **Blinking block cursor**: `█` via `.terminal-cursor` with `blink 1s step-end infinite`
- **Terminal prompt**: Cyan `λ` symbol with `.terminal-glow` text shadow (`0 0 8px #00d4ff`)
- **Scanline overlay**: Repeating linear gradient at 3% opacity, animated via `scanline-scroll` (togglable in UI settings)
- **CRT flicker**: `flicker` animation at 0.15s interval (togglable, off by default)
- **Global glow**: Radial gradient cyan wash at 3% opacity with `mix-blend-mode: screen` (togglable)

All effects respect `prefers-reduced-motion: reduce` — durations collapse to 0.01ms.

---

## Component Architecture

### Overview

44 components live in `frontend/src/components/`, organized into:

```
components/
├── __tests__/                 # Component test files
├── terminal/                  # Terminal shell components (11 files)
│   ├── TerminalView.tsx       # Shell wrapper with agent bar
│   ├── TerminalOutput.tsx     # Scrollable output with type-based styling
│   ├── TerminalPrompt.tsx     # Input prompt with command parsing
│   ├── TerminalProgressBar.tsx
│   ├── TerminalEffects.tsx    # Scanline/glow/flicker overlays
│   ├── SlideOverPanel.tsx     # Right-slide panel with focus trap
│   ├── SlideOverContent.tsx   # Lazy-loaded view router for panels
│   ├── StatusBar.tsx          # Bottom status bar
│   ├── InlineRenderer.tsx     # Inline terminal content renderer
│   ├── slashCommands.ts       # Slash command definitions
│   └── index.ts
├── ui/                        # Primitive components (9 files)
│   ├── button.tsx             # Base UI button with CVA variants
│   ├── dialog.tsx
│   ├── input.tsx
│   ├── select.tsx
│   ├── switch.tsx
│   ├── tooltip.tsx
│   ├── EmptyState.tsx         # Rotating ghost prompts
│   ├── InlineError.tsx
│   └── ToastError.tsx
├── forms/                     # Slide-over form components
│   ├── AgentFormSlideOver.tsx
│   ├── SkillFormSlideOver.tsx
│   ├── ToolFormSlideOver.tsx
│   ├── DataSourceFormSlideOver.tsx
│   ├── ComponentFormSlideOver.tsx
│   ├── ComponentDetailSlideOver.tsx
│   ├── AssetDetailSlideOver.tsx
│   ├── DetailSlideOver.tsx
│   ├── SandboxResultSlideOver.tsx
│   ├── SkillExecuteSlideOver.tsx
│   └── ToolExecuteSlideOver.tsx
├── *.tsx                      # View and feature components (remaining)
│   ├── App.tsx                # (aliased from src/)
│   ├── AgentsView.tsx
│   ├── SkillsView.tsx
│   ├── ToolsView.tsx
│   ├── DataSourcesView.tsx
│   ├── ComponentsView.tsx
│   ├── DashboardView.tsx
│   ├── ExplorerView.tsx
│   ├── OntologyView.tsx
│   ├── OracleView.tsx
│   ├── SettingsView.tsx
│   ├── LibraryView.tsx
│   ├── CopilotView.tsx
│   ├── Sidebar.tsx
│   ├── CommandPalette.tsx
│   ├── GenericCommandPalette.tsx
│   ├── Toast.tsx
│   ├── AlephErrorBoundary.tsx
│   ├── InlineErrorBoundary.tsx
│   └── ... (views, forms, utilities)
```

### App Shell

`App.tsx` is the root component (~206 lines). The architectural pattern:

```
App.tsx
├── AlephErrorBoundary
│   ├── NavigationStateSync (SSE listener + route sync)
│   ├── CommandPalette (Cmd+K overlay)
│   ├── Sidebar (icon nav, 13 items)
│   ├── <main>
│   │   ├── TerminalView (always present, z-0)
│   │   └── DashboardView (z-10, absolute overlay, lazy)
│   ├── SlideOverPanel (z-90, lazy SlideOverContent)
│   ├── StatusBar (bottom, health + mode indicators)
│   └── ToastContainer (fixed bottom-right, z-100)
```

**Key characteristics**:
- No React Router — view switching is purely Zustand-driven via `slideOverContent.type` and `currentView`
- TerminalView is always mounted (the "desktop" of the OS metaphor)
- DashboardView overlays the terminal as an absolute-positioned layer
- SlideOverPanel slides in from the right with a `max-w-2xl` (672px) width, expandable to fullscreen
- Sidebar is 48px wide (`w-12`), always visible

### View Routing Pattern

Navigation flows through a single chain:

1. **Sidebar click** → sets `slideOverContent: { type, title }` via `useStore.getState().setSlideOverContent()`
2. **Command palette** → same pattern (from `useCommandPalette` or `/slash` commands)
3. **SlideOverContent** → reads `slideOverContent.type` and renders the corresponding lazy-loaded view (20+ cases in a switch statement)
4. **Dashboard** → special case: `slideOverContent.type === 'dashboard'` renders an inline overlay instead of a slide panel

### UI Primitives (components/ui/)

Built with **Base UI** (`@base-ui/react`) + **class-variance-authority** (CVA):

| Component | Base | Variants |
|-----------|------|----------|
| `button` | `@base-ui/react/button` | default · outline · secondary · ghost · destructive · link |
| `dialog` | `@base-ui/react/dialog` | Standard dialog pattern |
| `input` | `@base-ui/react/input` | With focus ring styles |
| `select` | `@base-ui/react/select` | Dropdown selector |
| `switch` | `@base-ui/react/switch` | Toggle control |
| `tooltip` | `@base-ui/react/tooltip` | Hover tooltip |
| `EmptyState` | (custom) | Rotating ghost prompts for empty terminal |
| `InlineError` | (custom) | Inline error display |
| `ToastError` | (custom) | Error variant of Toast |

The `button` component defines 6 variants and 7 sizes (`default` · `xs` · `sm` · `lg` · `icon` · `icon-xs` · `icon-sm` · `icon-lg`), all with Tailwind classes via CVA.

### Slide-Over Form Pattern

Forms (Agent, Skill, Tool, DataSource, Component) follow a consistent pattern:

1. User triggers form (from view toolbar or inline action)
2. `setSlideOverContent({ type: 'agent-form' | 'skill-form' | ... })` is called
3. `SlideOverContent` renders the corresponding `*FormSlideOver` component
4. Form validates client-side with Zod schemas (`AgentFormSchema`, `SkillFormSchema`, etc.)
5. On submit: calls API action from domain hooks (`useAgentActions`, `useSkillActions`, etc.)
6. On close: `setSlideOverContent(null)` → panel slides out

---

## State Management

### Store Architecture

A single Zustand store is composed from 6 slices, each a standalone `StateCreator`:

```
useStore (composite)
├── authSlice          — projectID, apiKeys, projects, notificationChannels, registryComponents
├── navigationSlice    — currentView, inlineContent, slideOverContent, showInlinePanel, isCommandPaletteOpen
├── uiSlice            — showOnboarding, showWizard, scanline/glow/flicker toggles, toastMessages, inputMode
├── healthSlice        — ollamaHealthy, nlpHealthy, dataHealthStats, lastError, ollamaModels
├── workspaceSlice     — sandboxInput, searchQuery, selectedObject, activeView, data, agents, skills, tools, ...
└── copilotSlice       — chat[], input, isStreaming, streamAbortController, pendingConfirmation, selectedAgent
```

### Cross-Slice Communication

All slices share the same store — cross-slice access uses direct `getState()` calls:

```typescript
// From useStore.ts — cross-slice reset on project switch:
setProjectContext: (projectID) => {
  const set = a[0]
  const state = a[1]()
  state.resetAuth()
  state.resetCopilot()
  state.resetWorkspace()
  state.resetHealth()
  state.resetUI()
  state.resetNavigation()
  set({ projectID })
}
```

This is a deliberate pattern: `setProjectContext` resets all 6 slices atomically when switching projects. Each slice exposes its own `reset*` function for clean teardown.

### useStore Hook Pattern

Components subscribe to individual fields via selector functions:

```typescript
// Efficient — only re-renders on projectID change
const projectID = useStore(s => s.projectID)

// Bulk access when necessary
const { handleError, loadProjectData } = useAppActions()
```

For cross-slice writes, `useStore.getState()` is used inside event handlers / callbacks to avoid stale closures:

```typescript
const onSend = useCallback(async () => {
  const store = useStore.getState()
  // ... use store.input, store.chat, store.selectedAgent
}, [])
```

### Active View Routing

There is **no URL-based routing**. The active view is determined entirely by Zustand state:

| State | Effect |
|-------|--------|
| `currentView === 'copilot'` | TerminalView is visible (default) |
| `currentView === 'inline'` | Inline panel (replaced by slide-over in current version) |
| `slideOverContent.type === 'dashboard'` | DashboardView overlays terminal |
| `slideOverContent.type` (other) | SlideOverPanel slides in from right |
| `showOnboarding === true` | WorkspaceOnboarding replaces entire app |
| `showWizard === true` | SetupWizard replaces entire app |

The sidebar reads `inlineContent?.type || slideOverContent?.type` to determine active state, highlighted with a cyan left border (`border-l-2 border-primary`).

---

## Data Flow

### API Communication

**ConnectRPC-Web** is the primary transport, using binary protobuf over HTTP/2:

```typescript
// api/factory.ts — Client instantiation
export const transport = createConnectTransport({
  baseUrl: "",
  credentials: "include",
})

export const queryClient = createPromiseClient(QueryService, transport)
export const projectClient = createPromiseClient(ProjectService, transport)
export const agentClient = createPromiseClient(AgentService, transport)
// ... 11 total clients
```

**11 service clients** defined in `api/factory.ts`:

| Client | Service | Purpose |
|--------|---------|---------|
| `registryClient` | `RegistryService` | Component registry |
| `sandboxClient` | `SandboxService` | Code execution sandbox |
| `queryClient` | `QueryService` | Data queries, chat, predictions |
| `projectClient` | `ProjectService` | Project CRUD |
| `agentClient` | `AgentService` | Agent CRUD + model listing |
| `ingestionClient` | `IngestionService` | Data ingestion tasks |
| `libraryClient` | `LibraryService` | Asset library |
| `authClient` | `AuthService` | API key management |
| `skillClient` | `SkillService` | Skill CRUD |
| `toolClient` | `ToolService` | Tool CRUD |
| `nlpClient` | `NLPService` | NLP analysis (sentiment, etc.) |
| `notificationClient` | `NotificationService` | Notification channels |

**Session auth**: Uses httpOnly cookies (`credentials: "include"` via `POST /api/v1/auth/session`). The `apiGet`/`apiPost`/`apiPatch` helpers in `client.ts` provide legacy REST fallback for endpoints not yet migrated to ConnectRPC.

### Real-Time Events (SSE)

A custom `useSSE` hook manages Server-Sent Events from `/api/v1/events`:

```typescript
function useSSE(handlers: SSEHandlers) {
  // Automatic reconnection with exponential backoff (1s → 30s)
  // Last-Event-ID tracking for resumption
  // AbortController-based cleanup
  // Returns: { connect, disconnect, status, reconnectCount }
}
```

**Event types** (parsed via SSE `event:` field):

| Event | Payload | Handler |
|-------|---------|---------|
| `tool_status` | `ToolStatusPayload` | `onToolStatus` → Toast notifications |
| `notification` | `NotificationPayload` | `onNotification` → Toast notifications |
| `ingestion_progress` | `IngestionProgressPayload` | `onIngestionProgress` → progress tracking |
| `system_alert` | `SystemAlertPayload` | `onSystemAlert` → alert system |

Two pre-built hooks compose `useSSE`:
- `useToolStatusSSE()` — monitors tool execution, shows success/error toasts
- `useNotificationSSE()` — shows notification toasts

### Zod Validation & Mapping

**19 Zod schemas** in `schemas/index.ts` define the type contract for all API data:

| Schema | Type | Notes |
|--------|------|-------|
| `ApiKeySchema` | `ApiKey` | id, label, key, createdAt |
| `ProjectSchema` | `Project` | id, name |
| `AgentSchema` | `Agent` | **passthrough** — allows extra fields |
| `SkillSchema` | `Skill` | **passthrough** — allows extra fields |
| `ToolSchema` | `Tool` | **passthrough** — allows extra fields |
| `ChatMessageSchema` | `ChatMessage` | role (enum), content, toolCall |
| `QueryDataSchema` | `QueryData` | columns, rows, sql |
| `RegistryComponentSchema` | `RegistryComponent` | 23 fields incl. metrics |
| `SandboxResultSchema` | `SandboxResult` | **passthrough** |
| `PredictionSchema` | `Prediction` | entityId, probability, state |
| `IngestionTaskSchema` | `IngestionTask` | id, name, sourceType, status, progress |
| `AssetSchema` | `Asset` | id, name, type, createdAt |
| `ColumnStatsSchema` | `ColumnStats` | count/uniqueCount support bigint |
| `NotificationChannelSchema` | `NotificationChannel` | type, configJson |
| `PendingConfirmationSchema` | `PendingConfirmation` | projectId, agentId |
| `AgentFormSchema` | `AgentFormData` | w/ URL format validation |
| `SkillFormSchema` | `SkillFormData` | min-length validations |
| `ToolFormSchema` | `ToolFormData` | category optional |
| `DataSourceFormSchema` | `DataSourceFormData` | w/ JSON parse validation |

**Mapper pattern**: Each schema has a corresponding `fromProtoTo*` function in `schemas/mappers.ts`:

```typescript
export function fromProtoToAgent(proto: unknown): Agent {
  return fromProto(AgentSchema, proto)
}
```

The `fromProto` function simply calls `schema.parse(data)` — it throws `ZodError` on shape mismatch, providing runtime type safety for protobuf deserialization.

### Data Loading Pattern

`useAppActions.loadProjectData()` fires on mount and project change:

```typescript
const loadProjectData = useCallback(() => {
  // Fires 11 parallel requests:
  projectClient.getOntology(...)
  agentClient.listAgents(...)
  ingestionClient.listTasks(...)
  libraryClient.listAssets(...)
  skillClient.listSkills(...)
  toolClient.listTools(...)
  agentClient.listModels(...)
  nlpClient.analyzeSentiment(...)
  authClient.listApiKeys(...)
  notificationClient.listChannels(...)
  registryClient.listComponents(...)
}, [projectID])
```

Each response writes directly to the Zustand store via `useStore.getState().set*()`. All requests use a shared `AbortController` — when `projectID` changes, in-flight requests are cancelled.

---

## Animation Principles

### Terminal Views

- **Minimal animation**: Terminal output appears instantly. No fade-in, no stagger.
- **Cursor blink**: 1s `step-end` infinite — deterministic, no easing.
- **Streaming text**: Real-time updates via `requestAnimationFrame` in `TerminalOutput`, rendering at display refresh rate.
- **Mode indicator**: `[CMD]` / `[INPUT]` labels in status bar toggle instantly.

### Dashboard & Panel Views

- **Slide-in panels**: `slideInRight` keyframe — 250ms, `cubic-bezier(0.16, 1, 0.3, 1)` (custom terminal easing).
- **Fade-in overlays**: Dashboard view fades in over 300ms with `ease-out`.
- **Toast notifications**: Slide-in from bottom-right, 300ms `cubic-bezier(0.16, 1, 0.3, 1)`.
- **Confirm dialogs**: Scale from 95% + translateY(8px) → 100%, 250ms.

### Glassmorphism Elements

- **Hover transitions**: 150ms on background-color and border-color for interactive elements (`.vol-interactive`).
- **No hover on terminal content**: Terminal output lines do not animate on hover — only the cursor position indicator changes.

### Loading States

- **SkeletonLoader**: Shimmer placeholders, rows with alternating opacity.
- **Progress bars**: CSS-only width transitions, used in `TerminalProgressBar`.
- **Modal/Slide loading**: `SkeletonLoader` inside `Suspense` fallback for lazy-loaded views.

### Motion Safety

All animations respect `prefers-reduced-motion: reduce` — `animation-duration` and `transition-duration` collapse to `0.01ms`:

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

Terminal effects (scanline, flicker) also check the `prefers-reduced-motion` media query and disable themselves accordingly.

### Easing Curve

All motion uses a single custom easing curve:

```
cubic-bezier(0.16, 1, 0.3, 1)
```

This "terminal ease" is a fast-arriving deceleration curve — elements snap into position quickly (16% of duration to start) then settle smoothly. No bounce, no spring, no elastic overshoot. Stored as `--terminal-ease: cubic-bezier(0.16, 1, 0.3, 1)` and `transition-timing-function: terminal` in Tailwind.

---

## Accessibility

### WCAG AA Compliance Target

All color pairs meet WCAG AA contrast requirements (4.5:1 for normal text, 3:1 for large text). Verified ratios:

| Pair | Ratio | Level |
|------|-------|-------|
| `#00d4ff` on `#080810` | 11.7:1 | AAA |
| `#e4e4e7` on `#080810` | 13.2:1 | AAA |
| `#6b6b80` on `#0e0e18` | 5.8:1 | AA |
| `#6b6b80` on `#141420` | 4.6:1 | AA |

### Keyboard Navigation

- **Skip link**: `#main-content` skip-to-content link is the first focusable element
- **Command palette**: `Cmd+K` / `Ctrl+K` opens the command palette, fully keyboard-operable
- **Sidebar navigation**: Arrow keys can navigate sidebar items; each item is a `<button>` with `aria-current="page"`
- **Slide panels**: Full focus trap — Tab/Shift+Tab cycles through panel elements; Escape closes
- **All views**: Clickable surfaces are `<button>` or `<a>` elements, never divs with onClick

### Screen Reader Support

- **Terminal output**: Each line is a `<div>` with semantic type styling; streaming state uses `aria-live="polite"`
- **Status bar**: `role="status"` with `aria-live="polite"` — health changes are announced
- **Toast messages**: `role="status"` in a `aria-live="polite"` region
- **Slide panel**: `role="dialog"` with `aria-modal="true"`, `aria-label`, and `aria-describedby`
- **Inline content**: `sr-only` descriptions for icon-only buttons

### Focus Management

- **Slide panel open**: Focus moves to first focusable element inside the panel; `#main-content` gets `inert` attribute
- **Slide panel close**: Focus returns to the trigger element (stored in `triggerRef`)
- **Sibling elements**: Hidden from assistive technology (`aria-hidden="true"`) while panel is open
- **Empty states**: Ghost prompts rotate every 4 seconds, with fade-out/ fade-in transitions that respect motion preferences

### Terminal-Centric Considerations

- Terminal output escapes HTML entities (`escapeHtml`) to prevent XSS in rendered content
- Monospace font at 13px is legible at default browser zoom levels
- Text color (`#e4e4e7`) on background (`#080810`) provides 13.2:1 contrast — exceeds all requirements
- Warning/error states use color + icon/prefix (`λ`, `→/`, `⚙`) to avoid color-only signaling

---

## CSS Architecture

### Foundation

- **Tailwind CSS 3.4** is the primary styling tool — utility-first, with custom theme extensions
- **Custom properties** (`--color-*`) are the single source of truth, sourced from `design-tokens.json`
- **`index.css`** defines base layer (`@layer base`), utilities (`@layer utilities`), and keyframes
- **`tailwind.config.js`** maps design tokens to Tailwind theme extensions (colors, fontFamily, fontSize, animation, keyframes, transitionTimingFunction, transitionDuration)

### Token-to-Tailwind Mapping

The Tailwind config imports `design-tokens.json` directly and maps it:

```javascript
// tailwind.config.js
import tokens from './src/styles/design-tokens.json';

colors: {
  background: 'var(--color-background)',
  surface: { DEFAULT: 'var(--color-surface)', alt: 'var(--color-surfaceAlt)' },
  border: 'var(--color-border)',
  primary: { DEFAULT: 'var(--color-primary)', muted: 'var(--color-primaryMuted)' },
  success: 'var(--color-success)',
  warning: 'var(--color-warning)',
  danger: 'var(--color-danger)',
  text: { DEFAULT: 'var(--color-text)', muted: 'var(--color-textMuted)', dim: 'var(--color-textDim)' },
},
fontFamily: {
  sans: tokens.typography.fontFamily.main,
  mono: tokens.typography.fontFamily.mono,
},
fontSize: {
  body: ['13px', { lineHeight: '1.25' }],
  meta: ['11px', { lineHeight: '1.25' }],
},
```

### Custom Utility Classes

**Terminal utilities** (in `index.css` `@layer utilities`):

| Class | Purpose |
|-------|---------|
| `.no-scrollbar` | Hide scrollbars (sidebar, terminal output) |
| `.custom-scrollbar` | Thin 6px scrollbar (data tables, panels) |
| `.terminal-glow` | Cyan text glow `0 0 8px var(--color-primary)` |
| `.terminal-cursor` | Blinking block cursor animation |
| `.terminal-prompt` | Cyan monospace prompt text |
| `.terminal-input` | Transparent background, cyan caret |
| `.terminal-border` | Border color matching terminal chrome |
| `.terminal-surface` | Surface background for terminal elements |

**Glassmorphism utilities**:

| Class | Effect |
|-------|--------|
| `.glass-panel` | 70% opacity surface + 12px blur + semi-transparent border |
| `.glass-panel-solid` | Solid surface background (fallback) |

**Volatility layers** (4-tier system):

| Class | Behavior |
|-------|----------|
| `.vol-static` | No transitions, no animations |
| `.vol-structural` | Fade-in 250ms on mount |
| `.vol-interactive` | 150ms hover transitions on bg/border/text |
| `.vol-signal` | 3-cycle pulse (50ms each) |

### Dark-Only Mode

- CSS custom properties on `:root` define the dark palette exclusively
- `.theme-light` exists as a class but is **never applied** — it exists for API consumers and testing
- `darkMode: 'class'` in Tailwind config is present but unused
- All surfaces assume `#080810` base; there is no light mode toggle in the UI

### Component Scoping

- **Terminal CSS** is separated from panel CSS by component boundaries — terminal components use `font-mono` directly, panel components use `font-sans`
- Tailwind utilities handle 95%+ of styling; no CSS modules
- Animation keyframes are defined globally in `index.css` (not per-component) to avoid duplication
- Component-specific overrides (e.g., SlideOverPanel animation, TerminalOutput line height) are inline Tailwind classes
- The `InlineRenderer.tsx` component renders HTML from the backend — it applies terminal styling classes explicitly to sanitized output

---

## Key Files

| File | Purpose |
|------|---------|
| `frontend/src/styles/design-tokens.json` | Single source of truth — 65 tokens for color, typography, spacing, radius, elevation, shadow, transition, border |
| `frontend/src/index.css` | CSS custom properties, utilities, keyframes, animations, glassmorphism, volatility layers |
| `frontend/tailwind.config.js` | Token → Tailwind mapping, extended theme |
| `frontend/src/App.tsx` | Root component, Zustand-driven view switching |
| `frontend/src/store/useStore.ts` | Composite Zustand store (6 slices) |
| `frontend/src/components/terminal/SlideOverPanel.tsx` | Right-slide panel with focus trap, inert, aria management |
| `frontend/src/components/terminal/SlideOverContent.tsx` | Lazy view router (20+ view types) |
| `frontend/src/hooks/useSSE.ts` | SSE hook with reconnection, Last-Event-ID |
| `frontend/src/hooks/useAppActions.ts` | Main action orchestrator (send, confirm, load) |
| `frontend/src/schemas/index.ts` | 19 Zod schemas for API contracts |
| `frontend/src/schemas/mappers.ts` | 17 fromProto mapper functions |
| `frontend/src/api/factory.ts` | 11 ConnectRPC service clients |

---

*This document describes the Aleph-v2 frontend as of May 2026. The system evolves through planned waves (W0–W6); see `/plans/aleph-reconciliation-plan.md` for the full roadmap.*

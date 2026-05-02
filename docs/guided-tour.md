# Guided Tour — Aleph Interface Walkthrough

This guide describes every panel in the Aleph user interface. Use it to understand where to find features and how each section works.

---

## Interface Overview

```
┌─────────────────────────────────────────────────────┐
│ Status Bar │ Project: demo │ Health: ✅ │ UTC       │
├──────────┬──────────────────────────────────────────┤
│          │                                          │
│ Sidebar  │            Main Content Area             │
│          │                                          │
│  Agents  │   (Chat / Slide-Over / View Content)     │
│  Skills  │                                          │
│  Tools   │                                          │
│  Data    │                                          │
│  Library │                                          │
│  Comps   │                                          │
│          │                                          │
├──────────┴──────────────────────────────────────────┤
│  Command Input  ─────────────────────────────────── │
└─────────────────────────────────────────────────────┘
```

---

## 1. Status Bar (Top Edge)

A thin bar at the very top of the screen showing:

| Element | Description |
|---------|-------------|
| Current project | Name of the active project (e.g., "demo") |
| Health indicator | Green ✅ = all services healthy, Red ❌ = issues detected |
| Connection status | Shows if the backend SSE (Server-Sent Events) connection is active |
| Timestamp | Current UTC time |

Clicking the health indicator opens a detailed health overview of backend subsystems.

---

## 2. Sidebar (Left Panel)

The sidebar is the primary navigation. It contains six main sections:

### Agents
- Lists all agents in the current project
- Each entry shows agent name, provider, and model
- **+** button to create a new agent
- Click an agent to edit its configuration
- Agents can be assigned to conversations via the chat interface

### Skills
- Lists composable capabilities (tool combinations + prompts)
- Each skill shows associated tools and description
- **+** button to create a new skill from existing tools
- Drag tools onto skills to compose them

### Tools
- Lists executable functions available in the system
- Each tool shows category, version, and health status
- Tools come from: built-in packages (finance, OSINT, human ecosystems, adaptation), external imports, custom code
- **+** button to import or create tools
- `/tool health <id>` — check a tool's operational status
- `/tool diagnose <id>` — run diagnostics on a tool

### Data Sources
- Lists imported datasets per project
- Supports file-based (CSV, JSON) and connected sources (feeds, APIs, GitHub)
- **+** button opens the multi-step data source form:
  1. **Upload** — Drop files or select from file system
  2. **Database** — Configure DuckDB or PostgreSQL connection
  3. **URL** — Add RSS feeds, GitHub repos, sitemaps, or Google Sheets
- Shows ingestion progress and status for each source
- Click a source to see its schema preview

### Library
- Versioned, reusable components: tools, skills, agents, and datasource configurations
- Components can be shared across projects
- **+** button to save a component to the library
- Supports versioning and rollback
- Used by the Genesis auto-suggestion pipeline to propose new capabilities

### Components
- Active registry components with trust scores and health metrics
- Shows: category, status (active/inactive), avg Brier score, trust percentage
- Used by the Decision Engine (PAORA) to evaluate tool reliability
- Click a component to see its full metadata and score history

---

## 3. Main Content Area (Center)

The main area shows one of the following, depending on context:

### Chat Interface (Default)
- Terminal-style conversation with the active agent
- Messages appear as bubbles: user (left) and agent (right)
- Agent responses include structured analysis, data visualizations, and tool call results
- SSE (Server-Sent Events) streams agent responses token by token
- Tool call results are displayed inline with formatted output

### Slide-Over Panels
These panels slide in from the right when you:
- Create/edit an agent, skill, tool, or data source
- View component details
- Open library entries

Each panel has:
- **Header** — Title and close button
- **Form / Content** — The main editable fields or detail view
- **Footer** — Action buttons (Save, Cancel, Delete)

### View Screens
Clicking sidebar items shows dedicated list views:
- **Agents View** — Table of agents with actions
- **Skills View** — Grid of skills with composition editor
- **Tools View** — Categorized tool browser
- **Data Sources View** — Source management dashboard
- **Library View** — Component browser with version history
- **Components View** — Registry dashboard with trust metrics

---

## 4. Command Input (Bottom Bar)

A persistent text input at the bottom of the screen:

- **Chat mode** (default) — Type messages to the active agent
- **Command mode** — Type `/` to enter commands:
  - `/project list | create | delete`
  - `/agent list | create | delete | update`
  - `/skill list | create | delete`
  - `/tool list | install | health | diagnose`
  - `/data list | ingest`
  - `/help` — Show all available commands

Press `Tab` for command autocompletion. Press `↑` for command history.

### Command Mode vs Input Mode

The input bar has two visual modes:
- **Input Mode** — Normal text input (no prefix)
- **Command Mode** — Activated by typing `/` — shows command suggestions

When in command mode, available arguments are shown as you type.

---

## 5. Notification System

Alerts and notifications appear in two locations:

### Toast Notifications
- Brief, non-blocking popups in the top-right corner
- Types: Success (green), Warning (yellow), Error (red), Info (blue)
- Auto-dismiss after 5 seconds
- Click to view details

### Notification Panel
- Accessible from the bell icon in the status bar
- Persistent list of system notifications
- Includes: tool health changes, ingestion completion, system alerts
- Each notification has a timestamp and action link

---

## 6. Setup Wizard

On the very first load (when no projects exist in the system):

1. A **Welcome Screen** greets you with a brief overview
2. The system checks if demo data was auto-created (see `onboarding-guide.md`)
3. The interface launches directly into the demo project chat

The wizard is non-blocking — you can dismiss it and start exploring immediately.

---

## 7. Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd+K` / `Ctrl+K` | Open command palette |
| `Cmd+,` / `Ctrl+,` | Open settings |
| `Esc` | Close slide-over / cancel input |
| `↑` / `↓` | Command history in input |
| `Tab` | Autocomplete command |
| `Cmd+Enter` | Submit multi-line input |
| `/` | Enter command mode (at start of input) |

---

## 8. Error Boundaries

Each view screen is wrapped in an error boundary (`AlephErrorBoundary`). If a view crashes:

1. The boundary catches the error gracefully
2. A fallback UI is shown with a description of the problem
3. A **Retry** button reloads the view
4. The error is logged to the console and (if configured) Sentry

The main app also has a global error boundary. Only the crashed view is affected — other panels continue working.

---

## Next Steps

- Complete the [Onboarding Guide](./onboarding-guide.md) for hands-on first steps
- Read `docs/manuale-tecnico.md` for an in-depth technical overview
- Check `docs/api-reference.md` for API endpoint documentation

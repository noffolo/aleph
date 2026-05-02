# Onboarding Guide — First Steps in Aleph

Welcome to **Aleph**, the Decision Intelligence system. This guide walks you through your first interaction after installation.

---

## 1. First Run: What Happens Automatically

The first time Aleph starts with an empty database, it detects "first run" and automatically creates:

| Resource | Value | Description |
|----------|-------|-------------|
| Project | `demo` | A ready-to-use project with sample sales data |
| CSV data | `sample.csv` | 30 rows of weekly sales by product, category, region |
| Ontology | `core.aleph` | A basic ontology mapping dimensions and measures |
| Agent | `Analista Demo` | An Ollama-based agent with system prompt for data analysis |

You can verify this happened by checking the server logs for:

```
[Onboarding] First run detected — setting up demo project with sample data
[Onboarding] Demo project created: id=demo agent=agent-demo-... csv=30 rows
```

> **Note**: No LLM provider is required for the setup itself. The agent is created with Ollama as provider — if Ollama is not configured, the agent will still exist but will fall back to degraded mode (no LLM responses).

---

## 2. Open the Interface

Navigate to `http://localhost:5173` (or the port configured for the frontend).

### What you'll see:

1. **Welcome Screen** — The main terminal-like interface
2. **Sidebar** — Navigation panel on the left with sections: Agents, Skills, Tools, Data Sources, Library, Components
3. **Command Input** — A text input at the bottom where you can type queries

---

## 3. Explore the Demo Project

### Via the command palette:

Open the command palette with `Cmd+K` (Mac) or `Ctrl+K` (Linux/Windows), then type:

- `/project list` — see the demo project
- `/data list` — see the sample CSV file
- `/agent list` — see the Analista Demo agent

### Via the sidebar:

Click on **Data Sources** in the sidebar to see the `sample.csv` file uploaded to the demo project.

Click on **Agents** in the sidebar to inspect the `Analista Demo` agent configuration.

---

## 4. Chat with the Demo Agent

From the main chat interface:

1. Ensure the demo project is selected (the current project indicator is in the top bar)
2. Type a question like:
   - "Quali sono i trend di revenue per prodotto?"
   - "Confronta le vendite per regione"
   - "Quale prodotto ha la miglior performance?"
3. The demo agent will analyze the sample data and respond

### What the demo agent can do:

- Analyze sales trends across weeks
- Compare performance between products (Widget Alpha, Gadget Beta, Component Gamma, Tool Delta)
- Break down revenue by region (North, South, East, West)
- Calculate derived metrics (profit margin, unit economics)
- Identify patterns and anomalies in the data

---

## 5. Create Your Own Project

Once you're comfortable with the demo:

1. Open the command palette with `Cmd+K` / `Ctrl+K`
2. Type: `/project create <your-project-name>`
3. The system creates the project directories and DuckDB schema

Alternatively, use the **sidebar** → click the **+** icon next to "Data Sources" to add data directly.

### Upload your own data:

| Method | How |
|--------|-----|
| Drag-and-drop | Drop CSV/JSON files into the project's `raw/` directory |
| File picker | Use the Data Sources form in the interface |
| Command | `/ingest add <path>` to import a file |
| Auto-watch | The file system watcher automatically picks up new files in `data/projects/<project>/raw/` |

---

## 6. Create Custom Agents

After adding data to your project, create an agent tailored to your analysis needs:

1. Open **Agents** in the sidebar
2. Click **+ New Agent**
3. Fill in:
   - **Name**: A descriptive name (e.g., "Revenue Analyst")
   - **Provider**: Ollama (default), OpenAI, or Anthropic
   - **Model**: A model available in your provider
   - **System Prompt**: Instructions that define the agent's role and behavior
4. Click **Create**

You can also create agents via command palette: `/agent create --name "..." --provider ollama`

---

## 7. Next Steps

Once you've explored the basic flow:

| Topic | Resource |
|-------|----------|
| UI panel-by-panel walkthrough | `docs/guided-tour.md` |
| Full technical documentation | `docs/manuale-tecnico.md` |
| API reference | `docs/api-reference.md` |
| Architecture overview | `docs/ARCHITECTURE.md` |
| Release notes | `docs/CHANGELOG.md` |

### Key concepts to understand:

- **Projects** — Isolated workspaces with their own DuckDB schema and data
- **Agents** — LLM-powered analysts with system prompts and tool access
- **Skills** — Composable capabilities (tools + prompts) that agents can use
- **Data Sources** — Imported data (CSV, JSON, feeds, APIs) organized per project
- **Tools** — Executable functions that agents can invoke (analysis, OSINT, finance, etc.)
- **Library** — Versioned, reusable components (tools, skills, agents) that can be shared across projects

---

## Troubleshooting

| Problem | Likely cause | Solution |
|---------|-------------|----------|
| No demo data on first run | PostgreSQL not initialized | Run migrations: see `docs/CI-CD-README.md` |
| Agent shows "degraded mode" | No Ollama running | Start Ollama: `ollama serve` |
| CSV not visible | Project schema not created | `/project list` then `/data ingest <project>/raw/sample.csv` |
| "First run" log not appearing | Projects already exist | Delete rows from `system_projects` table to reset |
| Agent "Provider: nil" in logs | LLM provider config missing | Set `OLLAMA_BASE_URL` in `.env` (defaults to `http://localhost:11434`) |

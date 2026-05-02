# Release Announcement Draft

**Aleph v2.0.0 — Decision Intelligence in Production**

*May 2026*

---

## The Short Version

Aleph v2 is out. It is the first production-ready release of our Decision Intelligence system. You can install it with Docker Compose in under five minutes, and it now ships with a full decision engine, auto-repair, memory, monitoring, and a completely redesigned interface.

---

## What is Aleph?

Aleph is a Decision Intelligence system. It does not make decisions for you. It builds a better map of your uncertainty so you can decide with more awareness.

It combines data ingestion, predictive models, agent workflows, and a decision cycle called PAORA (Plan, Act, Observe, Reflect, Admit) into a single integrated environment. Everything runs locally via Docker Compose. No cloud lock-in, no external API required unless you want one.

---

## What is New in v2.0

### PAORA Decision Engine

Every chat interaction now follows a full decision cycle. The system plans tool calls, executes them in an isolated sandbox, observes the results, reflects on what happened, and admits the outcome or retries. This is built into the Go backend. No external orchestration needed.

### Auto-Repair Engine

Data is messy. Aleph now detects and fixes seven common anomaly types automatically: null values, outliers, duplicates, constraint violations, type errors, timestamp mismatches, and broken correlations. Every fix is tracked and reversible.

### Genesis Auto-Suggestion

The system watches how you work and proposes new tools and skills. Each proposal is sandboxed for safety and held in a veto registry with a TTL before activation. You stay in control.

### Memory Store (VSS)

Aleph remembers context. Using DuckDB vector similarity search, it retrieves relevant past information and injects it into the decision cycle. Namespaces keep projects isolated.

### Data Ingestion Pipeline

Seven fetchers are production-ready: RSS/Atom, GitHub, CSV/JSON upload, XML sitemap, Google Sheets, and IMAP email. Each one includes SSRF-safe validation and sanitization.

### File System Watcher

Drop a file into a watched directory and Aleph ingests it automatically. Debounced at 500ms so bulk drops do not overwhelm the system.

### Monitoring Stack

Prometheus, Grafana, and Alertmanager are now first-class citizens in the Docker Compose setup. Metrics and alerts ship out of the box.

### Redesigned Interface

The frontend has been rebuilt around a terminal-as-workspace metaphor. Command palette, glassmorphism panels, design tokens, and React.lazy code splitting make the UI fast and consistent.

### Security Hardening

- AES-256-GCM encryption for API keys at rest (mandatory)
- Argon2id hashing for password-equivalent storage
- Hardened CSP with no `unsafe-inline`
- SSRF guard with DNS validation and private IP blocking
- Sandbox blocklist for dangerous Go packages
- Audit logging for every tool operation

---

## Who This is For

- **Analysts and researchers** who need to explore scenarios, not just dashboards.
- **Strategists** who want to compare hypotheses and monitor weak signals.
- **Developers** who want a modular, extensible platform for building agentic workflows.
- **Teams** who need self-hosted Decision Intelligence without cloud dependencies.

---

## How to Try It

```bash
git clone https://github.com/noffolo/aleph.git
cd aleph
cp .env.example .env
# Edit .env and set KEY_ENCRYPTION_KEY
docker compose up --build -d
```

The UI is at `http://localhost:5173`.

---

## Social Media Posts

### Twitter / X

Aleph v2.0 is live. Production-ready Decision Intelligence: PAORA engine, auto-repair, memory, 7 data fetchers, monitoring stack, and a redesigned terminal UI. Self-hosted via Docker Compose. No cloud lock-in.

https://github.com/noffolo/aleph

#DecisionIntelligence #OpenSource #SelfHosted

### LinkedIn

We are releasing Aleph v2.0, our production-ready Decision Intelligence system.

What that means in practice:
- A full decision cycle (PAORA) for every AI interaction
- Automatic data repair with 7 strategies
- Context memory via vector similarity search
- 7 production data fetchers with SSRF-safe validation
- Prometheus + Grafana monitoring out of the box
- Hardened security: AES-256-GCM, Argon2id, CSP, sandbox blocklists

Everything runs locally via Docker Compose. No cloud required.

If you work with complex data, ambiguous signals, or scenario planning, Aleph is built for you.

Repository: https://github.com/noffolo/aleph

---

## What's Next

The v2.0 release completes the core platform. Upcoming areas of focus:

- GNN link prediction for epistemic trust scoring
- Multi-language UI (i18n framework is in place)
- Plugin marketplace for community tools
- Kubernetes Helm chart

---

*Aleph v2.0.0 — May 2026*

# Aleph-v2 Documentation

Welcome to the Aleph-v2 documentation hub. This directory contains guides, references, and operational manuals for users, developers, and operators.

---

## Quick Start

**New user?** Read the [User Guide (English)](./user-guide-en.md) or [Guida Utente (Italiano)](./user-guide-it.md) first.

**Deploying to production?** Jump to the [Deployment Guide](./deployment-guide.md).

**Joining the dev team?** Start with [Developer Onboarding](./developer-onboarding.md).

**On-call engineer?** Keep the [Runbook](./runbook.md) open.

**Integrating with the API?** See the [API Reference](./api-reference.md).

---

## Documentation Index

### User Documentation

| Document | Audience | Language | Description |
|----------|----------|----------|-------------|
| [User Guide — English](./user-guide-en.md) | End users | EN | Terminal interface, workflows, agents, tools, troubleshooting |
| [Guida Utente — Italiano](./user-guide-it.md) | End users | IT | Interfaccia terminale, workflow principali, concetti chiave |
| [API Reference](./api-reference.md) | Integrators | EN | All ConnectRPC services, REST endpoints, SSE streaming, error codes |

### Developer Documentation

| Document | Audience | Description |
|----------|----------|-------------|
| [Developer Onboarding](./developer-onboarding.md) | New contributors | Setup, architecture, testing, contribution guidelines |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Contributors | Local setup, build workflow, PR process, coding standards |
| [Manuale Tecnico](./manuale-tecnico.md) | Engineers | Full technical manual in Italian (stack, architecture, security, testing) |
| [API.md](./API.md) | Integrators | Legacy API reference (REST endpoints and ConnectRPC overview) |

### Operations Documentation

| Document | Audience | Description |
|----------|----------|-------------|
| [Deployment Guide](./deployment-guide.md) | DevOps / SRE | Docker Compose deployment, SSL, Ollama, backup, monitoring |
| [Runbook](./runbook.md) | On-call engineers | Per-subsystem procedures, incident response, data recovery |
| [CI-CD-README.md](./CI-CD-README.md) | DevOps | CI/CD pipeline details, GitHub Actions workflows |
| [Release Checklist](./release-checklist.md) | Release manager | Pre-release verification steps |

### Project Documentation

| Document | Description |
|----------|-------------|
| [CHANGELOG.md](./CHANGELOG.md) | Release history and wave completion status |
| [i18n-evaluation.md](./i18n-evaluation.md) | Internationalization evaluation report |
| [plans/piano-final-integrato.md](./plans/piano-final-integrato.md) | Integrated final plan |
| [specs/](./specs/) | Technical specifications and analysis reports |

---

## Architecture at a Glance

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React/TS)                       │
│  TerminalView · CopilotView · SlideOver · Cmd+K Palette     │
│  Zustand Composite Store · SSE Streaming · ConnectRPC        │
└────────────────────────┬────────────────────────────────────┘
                         │ ConnectRPC (HTTP/2) + SSE + REST
┌────────────────────────┴────────────────────────────────────┐
│                    Backend Go                                │
│  QueryHandler · ChatSession · DecisionEngine (PAORA)        │
│  13 ConnectRPC Services · Sandbox · Health · Diagnostic      │
│  7 Middleware · Genesis · Tools Registry · Audit             │
└──────────┬────────────────────────────────┬────────────────┘
           │ gRPC (HTTP/2 cleartext)        │ DuckDB (read-only)
┌──────────┴──────────┐     ┌───────────────┴────────────────┐
│  Python NLP Sidecar  │     │         PostgreSQL 16          │
│  Sentiment · ONNX    │     │    API Keys · Audit · Chat     │
│  Ensemble Prophet/GBM│     └────────────────────────────────┘
│  DuckDB read-only    │
└──────────────────────┘
```

**Stack:** Go 1.25 · React 18 · TypeScript 5 · DuckDB · PostgreSQL 16 · Python 3.12 · Ollama

---

## Getting Help

1. Check the relevant guide above
2. Search this repository for error codes or log messages
3. Open an issue with:
   - What you were trying to do
   - What you expected
   - What happened instead
   - Relevant logs or screenshots

---

*Last updated: April 2026 · Aleph-v2 v2.0.0*

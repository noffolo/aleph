# Aleph-v2 Documentation Production Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce world-class documentation suite for Aleph-v2 Data OS — README, architecture, deploy, user guides, agents map, changelog, and community docs.

**Method:** Multi-agent content production with cross-review cycle. Each document is drafted by a specialized agent, then cross-reviewed by 2-3 other agents for consistency, tone, and completeness. Final review by Momus (critic), Metis (consultant), and Oracle.

**Tech Stack:** Markdown, docs/ directory, GitHub-flavored markdown, mermaid for diagrams.

---

## Communication Strategy — Developer Outreach

Parallel task in Phase 1. Reddit Community Builder + Growth Hacker write a joint strategy document.

**Output file:** `docs/community-outreach-strategy.md`

**Goal:** Attract developers to contribute code and beta testers to try Aleph-v2 before public release.

### Task D11: Community Outreach Strategy — Reddit Community Builder + Growth Hacker

**Scope:** /Users/ff3300/Desktop/aleph-v2/docs/community-outreach-strategy.md

Content:
- Target audience segments (Go developers, data engineers, AI tinkerers, early adopters)
- Channels strategy: Reddit (r/golang, r/dataengineering, r/selfhosted), Hacker News, X/Twitter, dev.to
- Pitch angles per channel (what resonates with each audience)
- Contributor funnel: "see → try → contribute → become maintainer"
- Beta tester onboarding flow
- Success metrics (stars, contributors, issues, beta signups)
- Timeline: pre-launch, launch, post-launch phases

Tone: Authentic, developer-first, transparent. No marketing fluff.

## Document Inventory
|---|------|---------------|-----------|----------|
| D1 | `README.md` | Technical Writer | Content Creator, Growth Hacker, Reddit Builder | New visitors, GitHub browsers |
| D2 | `AGENTS.md` | Senior Developer | Technical Writer, DevOps Automator | AI agent developers |
| D3 | `CONTRIBUTING.md` | Senior Developer | Technical Writer, DevOps Automator | Open source contributors |
| D4 | `CHANGELOG.md` | Ad Creative Strategist | Content Creator, Technical Writer | All users, release watchers |
| D5 | `docs/ARCHITECTURE.md` | Workflow Architect | Senior Developer, Oracle | Engineers, architects |
| D6 | `docs/DEPLOY.md` | DevOps Automator | Senior Developer, Technical Writer | DevOps, sysadmins |
| D7 | `docs/user-guide-en.md` | Anthropologist | Content Creator, Reddit Builder | End users (English) |
| D8 | `docs/user-guide-it.md` | Anthropologist | Content Creator | End users (Italian) |
| D9 | `docs/README.md` (docs index) | Content Creator | Technical Writer | Documentation visitors |
| D10 | `docs/release-announcement.md` | Ad Creative Strategist | Growth Hacker, Reddit Builder | Community, media |

---

## Phase 1: Drafting (parallel — 6 agents, ~2min each)

### Task D1: README.md — Technical Writer

**Scope:** /Users/ff3300/Desktop/aleph-v2/README.md

Write README.md for Aleph-v2 Data OS. Must be self-contained, <300 lines, English.

Structure:
- Tagline + badges (CI, Go, React, Docker)
- "What is Aleph" — 2 paragraphs
- Quick Start (3 commands)
- Architecture overview (ASCII diagram)
- Key Features (6-8 bullet points)
- Documentation links
- License

Tone: Professional, confident, inspiring. Make someone who lands on the page want to try it in <30 seconds.

### Task D2+D3: AGENTS.md + CONTRIBUTING.md — Senior Developer

**Scope:** /Users/ff3300/Desktop/aleph-v2/AGENTS.md + CONTRIBUTING.md

AGENTS.md:
- Table of build agents (Sisyphus, Sisyphus-Junior)
- Subagent types (explore, librarian, oracle, metis, momus) with purpose
- Categories (visual-engineering, ultrabrain, deep, quick)
- How to add a new agent

CONTRIBUTING.md:
- Setup dev env (Go 1.24+, Node 22+, Docker)
- Branch strategy (main + feat/ prefix)
- Conventional commits
- Tests before PR
- Code review process
- Style guide

### Task D4+D10: CHANGELOG.md + release-announcement.md — Ad Creative Strategist

**Scope:** /Users/ff3300/Desktop/aleph-v2/CHANGELOG.md + docs/release-announcement.md

CHANGELOG.md:
- Keep a Changelog format
- Start from v2.0.0
- Categories: Added, Changed, Fixed, Security
- Feature highlights: PAORA, sandbox, RBAC, NLP

release-announcement.md:
- Enthusiastic + professional tone
- Problem → solution structure
- 3-4 use cases
- Call to action

### Task D6: DEPLOY.md — DevOps Automator

**Scope:** /Users/ff3300/Desktop/aleph-v2/docs/DEPLOY.md

Content:
- Hardware requirements
- Docker Compose quick start
- Environment variables reference
- Secrets management (gosecrets)
- TLS/SSL via nginx
- Backup & restore
- Rollback strategy
- Healthcheck endpoints

### Task D7+D8: user-guide-en.md + user-guide-it.md — Anthropologist

**Scope:** /Users/ff3300/Desktop/aleph-v2/docs/user-guide-en.md + user-guide-it.md

Content (EN + IT translation):
- What is Aleph (simple explanation)
- Getting started: create project, generate API key, connect data
- Features: chat with data, agents, tools, sandbox
- Real use examples
- Troubleshooting

Tone: Accessible, clear, never condescending. Use metaphors but be precise.

### Task D9: docs/README.md (index) — Content Creator

**Scope:** /Users/ff3300/Desktop/aleph-v2/docs/README.md

Landing page for docs/ directory:
- Brief project description
- Table of all documentation files with descriptions
- Quick navigation by role (developer, devops, user, contributor)
- Links to external resources

---

## Phase 2: Cross-Review (parallel — 4 reviewers)

Each document is reviewed by 2-3 agents from the pool:

| Document | Reviewers |
|----------|-----------|
| README.md | Content Creator, Growth Hacker, Reddit Community Builder |
| AGENTS.md | Technical Writer, DevOps Automator |
| CONTRIBUTING.md | Technical Writer, DevOps Automator |
| CHANGELOG.md | Content Creator, Technical Writer |
| ARCHITECTURE.md | Senior Developer (already complete — verify) |
| DEPLOY.md | Senior Developer, Technical Writer |
| user-guide-en.md | Content Creator, Reddit Community Builder |
| user-guide-it.md | Content Creator |
| docs/README.md | Technical Writer, Anthropologist |
| release-announcement.md | Growth Hacker, Reddit Community Builder |

Review checklist for EACH reviewer:
- [ ] Tone consistent with target audience
- [ ] No contradictions with other docs
- [ ] Technical accuracy
- [ ] No broken links
- [ ] File path in sync with project structure

---

## Phase 3: Final Review (serial — 3 agents)

- [ ] **Momus** — Plan critic: review all docs for clarity, verifiability, completeness
- [ ] **Metis** — Pre-planning consultant: identify hidden gaps, ambiguities
- [ ] **Oracle** — Architecture/debug consulting: verify technical accuracy across all docs

Each reviewer returns structured feedback with:
- Critical issues (must fix before release)
- Recommendations (nice to have)
- Already correct (what to keep)

---

## Phase 4: Fix & Polish

- [ ] Apply all Critical fixes from Phase 3
- [ ] Apply high-priority Recommendations
- [ ] Final consistency pass: verify all cross-references between docs
- [ ] Commit and push

---

## Execution Order

```
Phase 1 (parallel)
  ├── Technical Writer → README.md
  ├── Senior Developer → AGENTS.md + CONTRIBUTING.md
  ├── Ad Creative Strategist → CHANGELOG.md + release-announcement.md
  ├── DevOps Automator → DEPLOY.md
  ├── Anthropologist → user-guide-en.md + user-guide-it.md
  └── Content Creator → docs/README.md
       │
       ▼ (all complete — ~2min)
Phase 2 (parallel)
  ├── Content Creator, Growth Hacker, Reddit Builder → cross-review all docs
       │
       ▼ (all complete — ~1min)
Phase 3 (serial)
  ├── Momus → structural review
  ├── Metis → gap analysis
  └── Oracle → technical accuracy
       │
       ▼ (all complete — ~2min)
Phase 4
  └── Apply fixes, final pass, commit
```

Expected total: ~5-7 minutes.

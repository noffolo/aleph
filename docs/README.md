# Aleph-v2 Documentation

Aleph-v2 is a Data Operating System that turns unstructured data into actionable intelligence. It brings together AI agents, sandboxed execution, and DuckDB analytics inside a single Go and React stack. Every operation follows the PAORA cycle: Plan, Act, Observe, Reflect, Admit.

---

## Start Here

| If you are... | Start here |
|---|---|
| New user | [user-guide-en.md](./user-guide-en.md) |
| Developer | [README.md](../README.md), [AGENTS.md](../AGENTS.md), [CONTRIBUTING.md](./CONTRIBUTING.md) |
| DevOps | [DEPLOY.md](./DEPLOY.md), [ARCHITECTURE.md](../ARCHITECTURE.md) |
| Contributor | [CONTRIBUTING.md](./CONTRIBUTING.md), [AGENTS.md](../AGENTS.md) |
| Just curious | [README.md](../README.md), [release-announcement.md](./release-announcement.md) |

---

## Full Document List

### Project Root

| File | Description | Audience | Link |
|---|---|---|---|
| README.md | Project overview, quick start, features, and architecture summary. | Everyone | [../README.md](../README.md) |
| AGENTS.md | Agent system map, build agents, subagent types, and skill reference. | Developers | [../AGENTS.md](../AGENTS.md) |
| ARCHITECTURE.md | System design, backend structure, frontend layout, and data flow. | Developers, Architects | [../ARCHITECTURE.md](../ARCHITECTURE.md) |
| CHANGELOG.md | Release history and wave completion status. | Everyone | [../CHANGELOG.md](../CHANGELOG.md) |
| SECURITY.md | Security model, authentication, authorization, and data protection. | DevOps, Security engineers | [../SECURITY.md](../SECURITY.md) |

### User Guides (docs/)

| File | Description | Audience | Link |
|---|---|---|---|
| user-guide-en.md | English user guide for the terminal interface and workflows. | End users | [./user-guide-en.md](./user-guide-en.md) |
| user-guide-it.md | Italian user guide (Guida Utente). | Italian-speaking end users | [./user-guide-it.md](./user-guide-it.md) |
| guided-tour.md | Interactive walkthrough of features and UI elements. | New users | [./guided-tour.md](./guided-tour.md) |
| onboarding-guide.md | Step-by-step onboarding for new team members and users. | New users | [./onboarding-guide.md](./onboarding-guide.md) |
| release-announcement.md | Public release announcement and value proposition. | New users, Community | [./release-announcement.md](./release-announcement.md) |
| version-history.md | Version history and archived release notes. | Everyone | [./version-history.md](./version-history.md) |
| migration-v1-v2.md | Migration steps and breaking changes from Aleph v1 to v2. | Existing users | [./migration-v1-v2.md](./migration-v1-v2.md) |

### Developer & Contributor (docs/)

| File | Description | Audience | Link |
|---|---|---|---|
| CONTRIBUTING.md | Local setup, build workflow, pull request process, and coding standards. | Contributors | [./CONTRIBUTING.md](./CONTRIBUTING.md) |
| developer-onboarding.md | From zero to a working development environment. | New contributors | [./developer-onboarding.md](./developer-onboarding.md) |
| manuale-tecnico.md | Full technical manual in Italian covering stack and security. | Italian-speaking engineers | [./manuale-tecnico.md](./manuale-tecnico.md) |
| CHANGELOG.md | Detailed release history maintained in the docs folder. | Everyone | [./CHANGELOG.md](./CHANGELOG.md) |
| api-reference.md | Full API reference covering ConnectRPC, REST, and SSE protocols. | Integrators | [./api-reference.md](./api-reference.md) |
| API.md | Legacy API reference with endpoint examples and protobuf summary. | Integrators | [./API.md](./API.md) |
| CI-CD-README.md | Continuous integration and delivery pipeline details. | DevOps | [./CI-CD-README.md](./CI-CD-README.md) |
| i18n-evaluation.md | Internationalization evaluation findings. | Developers | [./i18n-evaluation.md](./i18n-evaluation.md) |
| accessibility.md | Accessibility audit results and WCAG compliance notes. | Frontend developers | [./accessibility.md](./accessibility.md) |
| css-purge-audit.md | Audit of CSS usage and removal of unused styles. | Frontend developers | [./css-purge-audit.md](./css-purge-audit.md) |

### Operations & Deployment (docs/)

| File | Description | Audience | Link |
|---|---|---|---|
| DEPLOY.md | Deployment and rollback procedures with a decision matrix. | DevOps, SRE | [./DEPLOY.md](./DEPLOY.md) |
| deployment-guide.md | Docker Compose production deployment guide with SSL and monitoring. | DevOps | [./deployment-guide.md](./deployment-guide.md) |
| runbook.md | On-call procedures, incident response, and troubleshooting steps. | SRE, On-call engineers | [./runbook.md](./runbook.md) |
| backup-recovery.md | Backup strategies and disaster recovery procedures. | DevOps, SRE | [./backup-recovery.md](./backup-recovery.md) |
| release-checklist.md | Release checklist with verification steps and sign-offs. | Release managers | [./release-checklist.md](./release-checklist.md) |
| pre-release-check.md | Pre-release verification checklist and acceptance criteria. | Release managers | [./pre-release-check.md](./pre-release-check.md) |

### Security & Compliance (docs/ + audit/ + deploy/)

| File | Description | Audience | Link |
|---|---|---|---|
| data-residency.md | Data residency policies and regional storage compliance. | Compliance officers, DevOps | [./data-residency.md](./data-residency.md) |
| privacy-impact-assessment.md | Privacy impact assessment findings and mitigations. | Compliance officers | [./privacy-impact-assessment.md](./privacy-impact-assessment.md) |
| audit/gosec-report.md | Go security checker results and findings. | Security engineers, Developers | [../audit/gosec-report.md](../audit/gosec-report.md) |
| audit/govulncheck-report.md | Go vulnerability database scan results. | Security engineers, Developers | [../audit/govulncheck-report.md](../audit/govulncheck-report.md) |
| audit/security-audit-2026-05-02.md | Full security audit findings dated May 2026. | Security engineers | [../audit/security-audit-2026-05-02.md](../audit/security-audit-2026-05-02.md) |
| audit/trivy-report.md | Container image vulnerability scan by Trivy. | Security engineers, DevOps | [../audit/trivy-report.md](../audit/trivy-report.md) |
| audit/zap-config.md | OWASP ZAP configuration and web application scan results. | Security engineers | [../audit/zap-config.md](../audit/zap-config.md) |
| deploy/bare-metal/SECURITY.md | Security hardening steps for bare metal installations. | Security engineers, DevOps | [../deploy/bare-metal/SECURITY.md](../deploy/bare-metal/SECURITY.md) |

### Deployment Infrastructure (deploy/)

| File | Description | Audience | Link |
|---|---|---|---|
| deploy/docker-secrets-readme.md | Docker secrets setup and management guide. | DevOps | [../deploy/docker-secrets-readme.md](../deploy/docker-secrets-readme.md) |
| deploy/bare-metal/README.md | Bare metal deployment instructions and server requirements. | DevOps | [../deploy/bare-metal/README.md](../deploy/bare-metal/README.md) |
| deploy/helm/README.md | Helm chart usage and Kubernetes deployment notes. | DevOps | [../deploy/helm/README.md](../deploy/helm/README.md) |
| deploy/helm/ARCHITECTURE.md | Helm chart architecture and design decisions. | DevOps, Architects | [../deploy/helm/ARCHITECTURE.md](../deploy/helm/ARCHITECTURE.md) |
| deploy/load-tests/README.md | Load testing setup, execution, and result interpretation. | QA, DevOps | [../deploy/load-tests/README.md](../deploy/load-tests/README.md) |
| scripts/pg-restore.md | PostgreSQL restore procedures and script usage. | DevOps, SRE | [../scripts/pg-restore.md](../scripts/pg-restore.md) |

### Frontend (frontend/)

| File | Description | Audience | Link |
|---|---|---|---|
| frontend/README.md | Frontend-specific setup, scripts, and dependency notes. | Frontend developers | [../frontend/README.md](../frontend/README.md) |

### Specifications & Plans (docs/specs/ + docs/plans/ + docs/reports/ + docs/superpowers/)

| File | Description | Audience | Link |
|---|---|---|---|
| specs/wave0-auth-spec.md | Authentication system specification for Wave 0. | Developers, Security engineers | [./specs/wave0-auth-spec.md](./specs/wave0-auth-spec.md) |
| specs/wave0-secrets-spec.md | Secrets management and encryption specification. | Developers, Security engineers | [./specs/wave0-secrets-spec.md](./specs/wave0-secrets-spec.md) |
| specs/wave1-injection-spec.md | Data ingestion pipeline specification for Wave 1. | Developers | [./specs/wave1-injection-spec.md](./specs/wave1-injection-spec.md) |
| specs/wave1-sandbox-spec.md | Sandboxed execution environment specification. | Developers | [./specs/wave1-sandbox-spec.md](./specs/wave1-sandbox-spec.md) |
| specs/wave2-database-spec.md | Database design and DuckDB concurrency specification. | Developers | [./specs/wave2-database-spec.md](./specs/wave2-database-spec.md) |
| specs/wave3-api-spec.md | API layer and ConnectRPC specification for Wave 3. | Developers, Integrators | [./specs/wave3-api-spec.md](./specs/wave3-api-spec.md) |
| specs/wave3-frontend-spec.md | Frontend architecture and component specification. | Frontend developers | [./specs/wave3-frontend-spec.md](./specs/wave3-frontend-spec.md) |
| specs/wave4-concurrency-spec.md | Concurrency and parallel agent execution specification. | Backend developers | [./specs/wave4-concurrency-spec.md](./specs/wave4-concurrency-spec.md) |
| specs/wave4-infra-spec.md | Infrastructure and deployment specification for Wave 4. | DevOps, Architects | [./specs/wave4-infra-spec.md](./specs/wave4-infra-spec.md) |
| specs/piano-operativo-specs.md | Operational plan specifications in Italian. | Project managers | [./specs/piano-operativo-specs.md](./specs/piano-operativo-specs.md) |
| specs/relazione-opportunita.md | Opportunity analysis report in Italian. | Stakeholders | [./specs/relazione-opportunita.md](./specs/relazione-opportunita.md) |
| specs/relazione-criticita.md | Criticality analysis report in Italian. | Stakeholders, Architects | [./specs/relazione-criticita.md](./specs/relazione-criticita.md) |
| plans/audit-remediation.md | Security audit remediation plan and tracking. | Security engineers, DevOps | [./plans/audit-remediation.md](./plans/audit-remediation.md) |
| plans/piano-final-integrato.md | Integrated final operational plan in Italian. | Project managers | [./plans/piano-final-integrato.md](./plans/piano-final-integrato.md) |
| plans/archive/piano-operativo-90gg.md | 90-day operational plan archive in Italian. | Project managers | [./plans/archive/piano-operativo-90gg.md](./plans/archive/piano-operativo-90gg.md) |
| reports/report-1.md | Internal report #1 with findings and recommendations. | Project leads | [./reports/report-1.md](./reports/report-1.md) |
| superpowers/plans/2026-05-02-content-production-plan.md | Content production schedule and deliverables. | Content creators | [./superpowers/plans/2026-05-02-content-production-plan.md](./superpowers/plans/2026-05-02-content-production-plan.md) |

---

## External Resources

| Resource | Link |
|---|---|
| GitHub Repository | https://github.com/noffolo/aleph |
| Issue Tracker | https://github.com/noffolo/aleph/issues |
| Docker Hub | https://hub.docker.com/r/noffolo/aleph |

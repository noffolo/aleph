# Privacy Impact Assessment — Aleph-v2

## 1. Overview

This document describes the data protection measures, retention policies, and privacy controls implemented in Aleph-v2 to comply with the General Data Protection Regulation (GDPR) and other applicable privacy frameworks.

## 2. Data Categories

### 2.1 Metadata (PostgreSQL)

| Category | Description | Retention |
|---|---|---|
| Project records | Project IDs, names, creation timestamps | Until project deletion |
| Agent records | Agent names, models, encrypted API keys | Until project deletion |
| Agent chat history | Messages, tool calls, timestamps | 365 days (configurable) |
| Task records | Ingestion task configs, progress, status | Until project deletion |
| Skill records | Skill definitions, tool associations | Until project deletion |
| API key records | Hashed API keys (argon2id), labels | Until project deletion |
| Notification channels | Channel configs (no credentials) | Until project deletion |
| Audit logs | User actions, timestamps, diffs | 730 days (configurable) |

### 2.2 Analytical Data (DuckDB)

| Category | Description | Retention |
|---|---|---|
| Project schemas | User-created data tables | Until project deletion |
| System tables | Ontology versions, vector embeddings | Until project deletion |

### 2.3 Filesystem Data

| Category | Description | Retention |
|---|---|---|
| Raw uploads | CSV, JSON, ontology files | Until project deletion |
| Ontology backups | `.bak` snapshots of `core.aleph` | Until project deletion |

## 3. Data Deletion

### 3.1 Cascade Deletion

When a project is deleted, all associated data is removed:

1. **DuckDB schema** — `DROP SCHEMA ... CASCADE` removes all user tables and data
2. **PostgreSQL metadata** — Transactional DELETE covers:
   - `system_agents`
   - `system_skills`
   - `system_tasks`
   - `system_api_keys`
   - `system_notification_channels`
   - `system_chat_history`
   - `system_chat_sessions`
   - `system_ontology_versions`
   - `system_projects`
3. **Filesystem** — Project directory (uploads, ontologies, backups)

### 3.2 Grace Period

Deleted project data in the `project` retention class has a 30-day grace period before permanent purge. This allows recovery in case of accidental deletion.

### 3.3 Retention Policy Table

Retention rules are stored in `data_retention_policy`:

| Resource Type | Default Retention | Configurable |
|---|---|---|
| `chat_history` | 365 days | Yes |
| `audit_log` | 730 days | Yes |
| `project` (purge) | 30 days | Yes |

## 4. Data Encryption

| Layer | Mechanism |
|---|---|
| API keys at rest | AES-256-GCM with `KEY_ENCRYPTION_KEY` |
| Hashed API keys | Argon2id (legacy SHA-256 detected and migrated) |
| Transport | TLS for HTTP and gRPC (if configured) |
| DuckDB files | Filesystem-level encryption (deployer responsibility) |

## 5. Access Controls

| Mechanism | Scope |
|---|---|
| API key authentication | Per-request via `X-Aleph-Api-Key` header |
| Role-based access | Admin, User, ReadOnly roles |
| Session cookies | httpOnly, Secure for web UI |
| CORS | Restricted to explicit origins |
| CSP | No `unsafe-inline` |
| SSRF guard | DNS resolution, redirect re-validation, private IP blocking |

## 6. Audit Logging

All mutating operations are logged to the `audit_log` table:

- **Actions tracked**: create, update, delete for agents, tools, skills, tasks, API keys, projects
- **Fields recorded**: user/project ID, action type, resource type, resource ID, timestamp, JSON diff
- **Retention**: 730 days (configurable)
- **Indexed queries**: by resource, by user, by timestamp, by project ID

## 7. Data Residency

See [data-residency.md](data-residency.md) for detailed storage locations, backup policies, and regional deployment considerations.

## 8. Data Subject Rights

The following mechanisms support GDPR data subject requests:

- **Right to access**: Audit logs provide a record of all operations; `ListAgents`, `ListAPIKeys`, `ListTasks` etc. expose user data
- **Right to deletion**: `DeleteProject` implements cascade deletion across all storage layers
- **Right to rectification**: `UpdateAgent`, `SaveOntology` etc. allow data correction
- **Right to data portability**: Data can be exported via API or direct DuckDB/PostgreSQL queries
- **Right to restrict processing**: Sandbox validation and SSRF guards prevent unauthorized processing

## 9. Data Processing Register

| Processing Activity | Purpose | Legal Basis | Data Categories |
|---|---|---|---|
| Chat history storage | Conversation continuity | Legitimate interest | User messages, tool calls |
| API key storage | Authentication | Contractual necessity | Hashed keys, labels |
| Project data storage | User workspace | Contractual necessity | User-uploaded data |
| Audit logging | Security monitoring | Legal obligation | Operation metadata |
| Vector embeddings | Similarity search | Legitimate interest | Embedded text |

## 10. Third-Party Data Processing

| Service | Data Shared | Safeguards |
|---|---|---|
| Ollama (local) | Prompts, context | Local-only, no data leaves host |
| OpenAI / Anthropic (optional) | Prompts, context | Configurable per agent; no PII in transit |
| SMTP / IMAP (optional) | Email credentials | Environment variables, zero temp files |

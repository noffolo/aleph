# Helm Chart for Aleph-v2 — Architecture Decision

> Status: **Draft / Skeleton** — This directory will become active after v2.0 GA.

## Why Kubernetes is Scheduled Post-v2.0

### 1. Current Architecture is a Tight Monolith

Aleph-v2 today runs as a single Go binary with co-located DuckDB (in-process) and an optional Python NLP sidecar. The backend code is structured as packages under `internal/`, but these are not independently deployable services. The Go binary owns the full request lifecycle: HTTP/Connect RPC handlers, middleware stack, DuckDB analytics, PostgreSQL metadata persistence, gRPC calls to the NLP sidecar, and the React SPA frontend served (usually through a reverse proxy or the Go static handler).

Kubernetes works best with well-defined service boundaries, clear network contracts, and horizontal scaling units. Our current monolith has none of these. Moving it directly to K8s would give us none of the platform benefits (independent scaling, rolling updates per service, circuit breaking at the service mesh layer) while adding significant operational complexity (pod networking, ConfigMap/Secret management, readiness/liveness probes, PVC handling for DuckDB).

### 2. DuckDB and Persistent Volumes Create Friction

DuckDB is an embedded, single-process analytical database. It is not designed to be shared across pods. In a K8s deployment, DuckDB data lives on a PersistentVolume and can only be mounted by a single pod at a time. This means the analytics layer cannot horizontally scale without fundamental architectural changes (e.g., introducing a remote DuckDB instance or switching to a scalable OLAP engine). Those are not v2.0 goals.

### 3. The Service Split Must Happen First

Before a production-ready Helm chart makes sense, the following decomposition work must be completed:

1. **Extract the NLP sidecar** into a true independent service with its own deployment lifecycle.
2. **Frontend** should be built, containerized, and served from its own pod (or a CDN / S3 bucket) rather than bundled with the Go backend.
3. **Observation pipeline** (telemetry, health, audit) should be externalized to dedicated collectors.
4. **DuckDB** decision made: either stick to embedded (single-replica only) or move to a remote queryable layer.

### 4. Docker Compose Fills the Gap Today

`docker-compose.yml` covers the v2.0 deployment needs:

- Single-machine development and staging
- Simple service dependency wiring (`depends_on`, `networks`)
- Volume-based persistence for DuckDB and PostgreSQL
- No need for pod scheduling, HPA, ingress controllers, or network policies

### 5. What This Skeleton Provides

This directory contains a **placeholder Helm chart** designed to guide the post-v2.0 migration. It defines the expected service topology, resource defaults, and migration path. When the service split is complete, the placeholders in `templates/` will be replaced with real Deployment, Service, Ingress, and NetworkPolicy manifests.

## Target State (Post-v2.0)

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Ingress   │────▶│  Frontend   │────▶│   Backend   │
│  Controller │     │   (React)    │     │   (Go API)  │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                                │
                    ┌─────────────┐     ┌────────┴────────┐
                    │   Grafana   │◀────│   Prometheus    │
                    └─────────────┘     └─────────────────┘
                                                │
                     ┌────────────┐    ┌────────┴────────┐
                     │   Ollama   │    │  NLP Sidecar    │
                     │ (Optional) │    │   (Python gRPC) │
                     └────────────┘    └─────────────────┘
                                                │
                                          ┌─────┴─────┐
                                          │ PostgreSQL│
                                          │  (Metadata│
                                          │  + Audit) │
                                          └───────────┘
                                                │
                                          ┌─────┴─────┐
                                          │   DuckDB  │
                                          │ (PVC, 1x) │
                                          └───────────┘
```

## Recommended Timeline

1. **v2.0.x** — Stabilize Docker Compose, harden PostgreSQL/DuckDB backup story.
2. **v2.1** — Service extraction. NLP sidecar and frontend become independently deployable.
3. **v2.2** — Helm chart goes live. Add HPA, PDB, and service mesh annotations.
4. **v2.3** — Multi-cluster considerations, ArgoCD GitOps setup.

## References

- `../docker-compose.yml` — Current deployment topology
- `../../ARCHITECTURE.md` — System architecture overview
- `../../docs/CI-CD-README.md` — Build and release pipeline

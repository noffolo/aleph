# Aleph-v2 Helm Chart (Skeleton)

> **Chart version:** 0.1.0  
> **App version:** 2.0.0  
> **Status:** Skeleton — deployment templates are placeholders. See [ARCHITECTURE.md](ARCHITECTURE.md) for the rationale.

This directory contains the Helm chart scaffolding for Aleph-v2. The chart does not contain live Deployment or Service manifests yet, because the backend is still a monolith with an embedded DuckDB instance that does not lend itself to horizontal scaling. Full K8s support is a post-v2.0 objective (target: v2.1).

## Prerequisites

- Kubernetes cluster **1.28+**
- Helm **3.12+**
- (Optional) **cert-manager** for automatic TLS certificate provisioning
- (Optional) **NVIDIA GPU operator** if you enable Ollama or GPU-based NLP inference
- (Optional) **NGINX Ingress Controller** for external traffic routing

## Quick Start (Not for Production)

**1. Install the chart**

```bash
helm install aleph-v2 ./deploy/helm \
  --namespace aleph-v2 \
  --create-namespace
```

**2. Expose via port-forward**

```bash
kubectl port-forward svc/aleph-v2-backend 8080:8080 -n aleph-v2
kubectl port-forward svc/aleph-v2-frontend    80:80 -n aleph-v2
```

**Important:** The chart currently lacks Deployment manifests (intentionally). Running `helm install` will install only ConfigMaps and Secrets created by `_helpers.tpl`. Use Docker Compose for any v2.0 deployment.

## Service Configuration

All services are defined in `values.yaml` as placeholder blocks. Each block mirrors what the eventual Deployment manifest will need.

| Service   | Type                 | Scaling Note                                  |
|-----------|----------------------|-----------------------------------------------|
| Backend   | Go (Connect RPC)     | Single replica only until DuckDB bottleneck is resolved |
| Frontend  | React + Vite         | Can scale once backed by a CDN or static asset server |
| NLP       | Python gRPC          | Requires GPU node pool when model inference is enabled |
| Postgres  | Metadata persistence | Use managed Postgres in production             |
| Ollama    | Optional LLM local   | StatefulSet with large model cache PVC         |
| Prometheus| Metrics storage      | Persistent volume for time-series data         |
| Grafana   | Dashboards + alerts  | Lightweight; can run as single replica         |

## Migration Path from Docker Compose

If you are currently running Aleph-v2 via `docker-compose.yml` and want to move to Kubernetes in the future, follow this sequence.

### Step 1: Data Export

cd aleph-v2/

docker compose exec -T postgres pg_dump -U aleph aleph > aleph_pg_backup.sql
docker compose cp backend:/data/aleph.duckdb ./aleph.duckdb.backup

### Step 2: Service Split (Pre-requisite)

Kubernetes is not a drop-in replacement for Compose. Before migration, complete these engineering milestones.

1. **Extract the NLP sidecar.** Move `nlp/` into its own container image, helm chart sub-chart, and gRPC service. The Go backend should discover it via DNS (`nlp-sidecar:50051`) rather than localhost.
2. **Containerize the frontend independently.** The React build (`frontend/Dockerfile`) must produce a static image served by nginx or a CDN, not bundled inside the Go binary.
3. **Resolve DuckDB placement.** Decide between:
   - Keeping DuckDB embedded (single pod, no horizontal scaling)
   - Moving to a remote DuckDB or MotherDuck setup
   - Replacing with a horizontally scalable OLAP engine

### Step 3: Helm Installation with Real Data

After service extraction, use the skeleton values and add your own overrides.

```bash
# example-overrides.yaml
backend:
  env:
    DATABASE_URL: "postgres://user@your-managed-pg:5432/aleph"
    DUCKDB_PATH: "/data/aleph.duckdb"

postgres:
  enabled: false  # use managed Postgres

ingress:
  enabled: true
  hosts:
    - host: aleph.yourdomain.com
      paths:
        - path: /
          service: frontend
        - path: /api
          service: backend

grafana:
  admin:
    existingSecret: "my-grafana-secret"
```

```bash
helm install aleph-v2 ./deploy/helm \
  --namespace aleph-v2 \
  --create-namespace \
  -f example-overrides.yaml
```

### Step 4: Data Import

```bash
# Restore Postgres (run a temporary pod with the pg_dump file)
kubectl run pg-restore --rm -i --restart=Never \
  --image=postgres:16-alpine \
  -- psql -h $(kubectl get svc aleph-v2-postgres -o jsonpath='{.spec.clusterIP}') \
         -U aleph aleph < aleph_pg_backup.sql

# Restore DuckDB (upload backup to the backend PVC via kubectl cp)
kubectl cp aleph.duckdb.backup \
  $(kubectl get pod -l app.kubernetes.io/name=aleph-v2 -o jsonpath='{.items[0].metadata.name}'):/data/aleph.duckdb -n aleph-v2
```

## Values Reference

| Key                        | Default        | Description                          |
|----------------------------|----------------|--------------------------------------|
| `backend.enabled`          | `true`         | Enable Go backend placeholder        |
| `backend.image.repository` | `aleph-v2/backend` | Container image name             |
| `backend.resources.limits` | `cpu: 2000m, memory: 2Gi` | Backend resource ceiling          |
| `frontend.enabled`         | `true`         | Enable frontend placeholder          |
| `nlp.enabled`              | `true`         | Enable NLP sidecar placeholder       |
| `nlp.resources.limits.nvidia.com/gpu` | `1` | GPU required for inference            |
| `postgres.enabled`         | `true`         | Enable Postgres placeholder          |
| `ollama.enabled`           | `false`        | Disable Ollama by default            |
| `prometheus.enabled`       | `true`         | Enable Prometheus placeholder        |
| `grafana.enabled`          | `true`         | Enable Grafana placeholder           |
| `ingress.enabled`          | `false`        | No ingress until domain is configured|
| `autoscaling.enabled`      | `false`        | Disabled until backend can scale out |

## Files Overview

| File                                   | Purpose                                 |
|----------------------------------------|-----------------------------------------|
| `Chart.yaml`                           | Chart metadata (name, version, appVersion) |
| `values.yaml`                          | Default configuration for all services  |
| `templates/_helpers.tpl`               | Reusable Helm template helpers          |
| `templates/NOTES.txt`                    | Post-install instructions for users     |
| `ARCHITECTURE.md`                      | ADR explaining K8s postponement         |

## Roadmap

- **v2.0.x** — Stabilize Docker Compose; no Helm changes
- **v2.1** — Service extraction; add real Deployment, Service, Ingress, HPA manifests
- **v2.2** — Helm chart goes GA with autoscaling, PDB, network policies
- **v2.3** — GitOps (ArgoCD) and multi-cluster federation support

## Contributing

The skeleton is open for PRs in these areas.

- Adding real Deployment templates (requires a service split first)
- Helm tests (`templates/tests/`)
- JSON schema for `values.yaml`
- OPA / Kyverno policies for security hardening

For full architecture context, see the project root `ARCHITECTURE.md`.

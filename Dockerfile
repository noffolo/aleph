# --- Stage 1: Frontend Build ---
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# --- Stage 2: Backend Build ---
FROM golang:1.24-bullseye AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy frontend build from previous stage so it can be embedded
COPY --from=frontend-builder /app/frontend/dist ./dist
RUN go build -o aleph main.go

# --- Stage 3: Python NLP Environment ---
FROM python:3.12-slim-bullseye AS python-builder
WORKDIR /app/nlp
COPY nlp/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# --- Stage 4: Final Hardened Image ---
FROM python:3.12-slim-bullseye
WORKDIR /app

# Install system dependencies (for DuckDB VSS, health checks, and general stability)
RUN apt-get update && apt-get install -y \
    libstdc++6 \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy backend binary
COPY --from=backend-builder /app/aleph .
# Copy swagger docs
COPY internal/api/proto/aleph_api.swagger.json ./internal/api/proto/

# Copy Python NLP code and pre-installed packages
COPY nlp/ ./nlp/
COPY --from=python-builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages

# Environment variables
ENV PORT=8080
ENV DATA_ROOT=/app/data/raw
ENV POSTGRES_DSN="postgres://postgres:postgres@db:5432/aleph?sslmode=disable"
ENV DUCKDB_PATH="/app/data/aleph.duckdb"

# Create data directories
RUN mkdir -p /app/data/projects /app/data/raw /app/data/ontologies

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD curl -f http://localhost:8080/api/v1/healthz || exit 1

# Start script to run both if needed, but per Master Plan, Go is the orchestrator
# Go app will manage the sidecar if started via shell or it assumes sidecar is running elsewhere.
# For Docker Compose simplicity, we might run them separately, but here we provide a unified image.
CMD ["./aleph"]

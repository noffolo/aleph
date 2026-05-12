# --- Stage 1: Frontend Build ---
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY frontend/ ./
RUN --mount=type=cache,target=/root/.npm npm run build

# --- Stage 2: Go Backend Build ---
FROM golang:1.26-bookworm AS backend-builder
RUN apt-get update && apt-get install -y --no-install-recommends gcc g++ libc6-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod/ go env -w GOPROXY=https://proxy.golang.org,direct && go mod download
COPY . .
# Embed frontend dist from stage 1
COPY --from=frontend-builder /app/frontend/dist ./dist
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod/ CGO_ENABLED=1 go build -ldflags="-s -w" -o aleph main.go

# --- Stage 3: Python NLP Dependencies ---
FROM python:3.12-slim-bookworm AS python-builder
RUN apt-get update && apt-get install -y --no-install-recommends gcc g++ libc6-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app/nlp
COPY nlp/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# --- Stage 4: Production (debian:bookworm-slim) ---
FROM debian:bookworm-slim
WORKDIR /app

# Install only runtime essentials including gosu for privilege dropping
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip ca-certificates wget curl libstdc++6 gosu && \
    rm -rf /var/lib/apt/lists/*

# Copy Go binary
COPY --from=backend-builder /app/aleph .
# Copy entrypoint script
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh
# Copy DB migrations
COPY migrations/ ./migrations/
# Copy Swagger docs
COPY internal/api/proto/aleph_api.swagger.json ./internal/api/proto/

# NLP source code
COPY nlp/ ./nlp/
# Python packages from builder
COPY --from=python-builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages

ENV PORT=8080
ENV DATA_ROOT=/app/data/raw
ENV DUCKDB_PATH="/app/data/aleph.duckdb"

# Create user and directories
RUN groupadd -r aleph && useradd -r -g aleph -d /app aleph \
    && mkdir -p /app/data/projects /app/data/raw /app/data/ontologies /app/data/backups/duckdb

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1

EXPOSE 8080

ENTRYPOINT ["./docker-entrypoint.sh"]
CMD ["./aleph"]

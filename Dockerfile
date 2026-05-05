# --- Stage 1: Frontend Build ---
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- Stage 2: Go Backend Build ---
FROM golang:1.26-bookworm AS backend-builder
RUN apt-get update && apt-get install -y --no-install-recommends gcc g++ libc6-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org,direct && go mod download
COPY . .
# Embed frontend dist from stage 1
COPY --from=frontend-builder /app/frontend/dist ./dist
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o aleph main.go

# --- Stage 3: Python NLP Dependencies ---
FROM python:3.12-slim-bookworm AS python-builder
RUN apt-get update && apt-get install -y --no-install-recommends gcc g++ libc6-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app/nlp
COPY nlp/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# --- Stage 4: Production (debian:bookworm-slim) ---
FROM debian:bookworm-slim
WORKDIR /app

# Install only runtime essentials
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip ca-certificates wget libstdc++6 && \
    rm -rf /var/lib/apt/lists/*

# Copy Go binary
COPY --from=backend-builder /app/aleph .
# Copy entrypoint script
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh
# Copy Swagger docs
COPY internal/api/proto/aleph_api.swagger.json ./internal/api/proto/

# Copy NLP source code
COPY nlp/ ./nlp/
# Copy pre-built Python packages from bookworm-compatible builder
COPY --from=python-builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages

# Environment defaults
ENV PORT=8080
ENV DATA_ROOT=/app/data/raw
ENV DUCKDB_PATH="/app/data/aleph.duckdb"

# Create non-root user & data directories
RUN groupadd -r aleph && useradd -r -g aleph -d /app aleph \
    && mkdir -p /app/data/projects /app/data/raw /app/data/ontologies \
    && chown -R aleph:aleph /app

USER aleph
EXPOSE 8080

# Healthcheck uses /healthz (registered by app.go via mux)
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/healthz || exit 1

CMD ["./aleph"]

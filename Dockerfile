# --- Stage 1: Frontend Build ---
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- Stage 2: Go Backend Build ---
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Embed frontend dist from stage 1
COPY --from=frontend-builder /app/frontend/dist ./dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o aleph main.go

# --- Stage 3: Python NLP Dependencies ---
FROM python:3.12-alpine AS python-builder
WORKDIR /app/nlp
COPY nlp/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# --- Stage 4: Production (alpine:3.20, <80MB target) ---
FROM alpine:3.20
WORKDIR /app

# Install only runtime essentials: Python for sandbox tool execution, ca-certs, wget for healthcheck
RUN apk add --no-cache \
    python3 \
    py3-pip \
    ca-certificates \
    wget \
    libstdc++

# Copy Go binary (CGO_ENABLED=0, static, ~15MB)
COPY --from=backend-builder /app/aleph .
# Copy entrypoint script
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh
# Copy Swagger docs
COPY internal/api/proto/aleph_api.swagger.json ./internal/api/proto/

# Copy NLP source code
COPY nlp/ ./nlp/
# Copy pre-built Python packages from alpine-compatible builder
COPY --from=python-builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages

# Environment defaults
ENV PORT=8080
ENV DATA_ROOT=/app/data/raw
ENV DUCKDB_PATH="/app/data/aleph.duckdb"

# Create non-root user & data directories
RUN adduser -D -h /app appuser \
    && mkdir -p /app/data/projects /app/data/raw /app/data/ontologies \
    && chown -R appuser:appuser /app

USER appuser
EXPOSE 8080

# Healthcheck uses /healthz (registered by app.go via mux)
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/healthz || exit 1

CMD ["./aleph"]

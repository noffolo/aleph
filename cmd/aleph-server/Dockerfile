FROM golang:1.25-bookworm AS builder
LABEL stage="builder"
ARG APP_VERSION="dev"
ARG GIT_COMMIT=""
ENV CGO_ENABLED=1

WORKDIR /app
COPY go.mod go.sum ./
RUN apt-get update && apt-get install -y git build-essential
RUN go mod download

COPY . .
# Aggiungiamo go mod tidy per assicurarci che il modulo sia allineato al toolchain
RUN go mod tidy
RUN go build \
    -ldflags="-w -s -X main.version=${APP_VERSION} -X main.commit=${GIT_COMMIT}" \
    -o /aleph-server ./cmd/aleph-server/main.go

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /aleph-server /app/aleph-server
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --retries=3 CMD wget --quiet --fail-early http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/app/aleph-server"]

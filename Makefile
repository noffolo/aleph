.PHONY: build run frontend-dev nlp-dev build-models proto-python clean
.PHONY: test test-go test-frontend test-e2e test-nlp test-integration test-all
.PHONY: dev dev-backend dev-frontend dev-nlp
.PHONY: docker-up docker-down docker-build docker-dev docker-devcontainer

# ── Build ───────────────────────────────────────────────────────

build:
	cd frontend && npm run build
	go build -o aleph main.go

run: build
	./aleph

# ── Development Servers ─────────────────────────────────────────

dev-backend:
	air

dev-frontend:
	cd frontend && npm run dev

dev-nlp:
	cd nlp && . .venv/bin/activate && python3 main.py

dev: dev-backend

frontend-dev: dev-frontend
nlp-dev: dev-nlp

# ── Tests ───────────────────────────────────────────────────────

test-go:
	go test -count=1 -race ./...

test-frontend:
	cd frontend && npx vitest run

test-e2e:
	cd frontend && npx playwright test

test-nlp:
	cd nlp && . .venv/bin/activate && python3 -m pytest

test-integration:
	go test -count=1 -tags=integration ./internal/integration/...

test-contract:
	docker compose up -d aleph-python-sidecar
	sleep 3
	go test -count=1 -tags=contract ./internal/nlp/...
	docker compose down

test: test-go test-frontend

test-all: test-go test-frontend test-nlp test-e2e

# ── Coverage ─────────────────────────────────────────────────────

.PHONY: test-go-cover
test-go-cover:
	go test -race -count=1 -coverprofile=coverage.out ./internal/...
	@echo "── Go Coverage (excluding proto) ──"
	@go tool cover -func=coverage.out | grep -v '.pb.go' | grep -v '_grpc.pb.go' | tail -1
	@echo "── Per-package ──"
	@go tool cover -func=coverage.out | grep -v '.pb.go' | grep -v '_grpc.pb.go' | grep -v 'total:' | awk '{file=$$1; sub(/:[0-9]+:/, "", file); cov=$$NF; print cov, file}' | sort -rn | head -20

.PHONY: test-fe-cover
test-fe-cover:
	cd frontend && npx vitest run --coverage

# ── Docker ──────────────────────────────────────────────────────

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-dev:
	docker compose -f docker-compose.yml -f .devcontainer/docker-compose.dev.yml up -d aleph-dev

docker-devcontainer:
	@echo "Open in VS Code: Remote-Containers: Reopen in Container"
	@echo "Or via CLI: devcontainer open ."

# ── NLP ─────────────────────────────────────────────────────────

build-models:
	cd nlp && . .venv/bin/activate && python3 convert_onnx.py

proto-python:
	cd nlp && python3 -m grpc_tools.protoc -I../api/proto --python_out=. --grpc_python_out=. ../api/proto/aleph/nlp/v1/nlp.proto

# ── Cleanup ─────────────────────────────────────────────────────

clean:
	rm -rf dist aleph
	cd frontend && rm -rf node_modules dist
	cd nlp && rm -rf .venv
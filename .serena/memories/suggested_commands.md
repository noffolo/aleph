# Suggested Commands

## Build
- `go build -o aleph main.go` — Build standalone binary (requires `dist/` from frontend)
- `cd frontend && npm run build` — Build frontend
- `cd frontend && npx tsc --noEmit` — TypeScript typecheck

## Run
- `docker compose up --build -d` — Full stack via Docker
- `./aleph` — Run standalone (needs PostgreSQL running)
- `cd frontend && npm run dev` — Frontend dev server

## Test
- `go test ./...` — Run Go tests
- `cd internal/dsl && go test` — DSL parser/compiler tests

## Lint
- `go vet ./...` — Go vet (currently fails on unused imports in auth)
- `cd frontend && npx eslint .` — ESLint

## Dependencies
- `go mod tidy` — Clean Go modules
- `cd frontend && npm install` — Frontend deps

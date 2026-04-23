# Piano di Esecuzione — Aleph-v2 (GLM-5.1)

## Track 1: Compilabilita e Avviabilita

### T1.1 — `internal/auth/auth_service.go`
- Rimuovere 4 import inutilizzati
- Mantenere validazione mock: accetta qualsiasi stringa che inizi con `aleph_` e ritorna project ID hardcoded `"default"`

### T1.2 — `internal/middleware/auth_middleware.go`
- Rinominare `NewAuthMiddleware` -> `NewAuthInterceptor`
- Cambiare parametro da `*auth.AuthService` a `*repository.MetadataRepository`
- Validazione: leggere `X-Aleph-Api-Key` header -> SHA256 hash -> query `system_api_keys` nel metaRepo -> estrarre project_id
- Se la key non e valida, ritornare `CodeUnauthenticated`
- Se la key e vuota/missing, skip (per permettere AuthService stesso di funzionare)

### T1.3 — `internal/app/app.go`
- `middleware.NewAuthInterceptor(a.metaRepo)` — ora matcha
- Non registrare l'interceptor su `AuthService`

### T1.4 — `go.mod` e Dockerfile
- `go 1.25.0` -> `go 1.24.0`
- `golang:1.25-bookworm` -> `golang:1.24-bookworm`

### T1.5 — `docker-compose.yml`
- Aggiungere servizio `postgres:16-alpine`
- `aleph-backend` depends_on postgres con `condition: service_healthy`
- Rimuovere `aleph-registry-db` (alpine dummy non serve piu)

## Track 2: Handlers mancanti — Implementazione reale

### T2.1 — Registry: GetComponent e UpdateComponentStatus
- Aggiungere `GetComponentByID` e `UpdateComponentStatus` al `DuckDBRegistry`
- Implementare i due handler nel `registry_handler.go`

### T2.2 — Sandbox: ExecuteTool e RunSkill — Esecuzione reale
- ExecuteTool: leggere tool da system_tools, eseguire con timeout 30s, capturare stdout/stderr/exit code
- RunSkill: leggere skill, risolvere tool_ids, eseguire in sequenza
- Implementare `ExecSandbox.ExecuteTool` e `ExecSandbox.RunSkill`

### T2.3 — Rimuovere directory legacy
- Eliminare `internal/api/middleware/`
- Aggiungere `.gitkeep` a `internal/workflow/` e `internal/service/library/`

## Track 3: Bug Logici

### T3.1 — Fix parametri `?` -> `$1, $2` in `query.go:79`
### T3.2 — Fix double `rows.Close()` in `project.go:185,188`
### T3.3 — Fix Circuit Breaker NLP
- Rimuovere StreamPredictions dal CircuitBreakerClient
- Gestire degrado direttamente in NLPHandler
- Aggiungere flag `isSidecarHealthy` atomico

### T3.4 — Sanitizzazione `task.Id` in `engine.go:78`
- Validare con regex `^[a-zA-Z0-9_-]+$`

### T3.5 — DuckDB VSS error handling
- Loggare warning se VSS fallisce
- Aggiungere flag `hasVSS` al DuckDB struct

## Track 4: DSL e Compilatore

### T4.1 — Aggiungere `filter` al DSL
- Sintassi: `filter <field> <op> <value>`
- Operatori: eq, neq, gt, gte, lt, lte, like
- FilterDefinition nell'AST
- Compilatore genera WHERE clause

### T4.2 — Aggiungere `aggregate` al DSL
- Sintassi: `aggregate <function>(<field>) as <alias>`
- Function: count, sum, avg, min, max
- AggregateDefinition nell'AST
- Compilatore genera SELECT aggregati + GROUP BY

### T4.3 — Migliorare errori del parser
- Wrappare errori di participle con posizione
- Aggiungere messaggi leggibili

### T4.4 — Test DSL estesi
- parser_test.go: filtri, aggregazioni, errori
- compiler_test.go: SQL con filtri, aggregazioni, combinazioni

## Track 5: NLP Sidecar

### T5.1 — Fix `nlp/main.py`
- Aggiungere `import time`
- Fix RecordFeedback return type
- Fix path feedback_log.jsonl
- Aggiungere try/except nel model loading

### T5.2 — Rigenerare proto Python
- Aggiungere target `proto-python` al Makefile
- Sostituire file generati a mano

## Track 6: Frontend — Razionalizzazione

### T6.1 — Unificare design tokens
- Aggiornare design-tokens.json con palette blue
- Estendere tailwind.config.js
- Eliminare design-system.styles.ts

### T6.2 — Eliminare App.css
### T6.3 — Fix SetupWizard.tsx (import order)
### T6.4 — Fix LibraryView.tsx (import order)

### T6.5 — Migrare stato in Zustand store
- Estendere useStore.ts con tutto lo stato
- Aggiungere custom hooks per data fetching
- Rimuovere stato locale da App.tsx

### T6.6 — CMD+K per CommandPalette
### T6.7 — Migliorare AlephErrorBoundary
### T6.8 — Tailwind config esteso

## Track 7: Test e Finalizzazione

### T7.1 — Test Go
- auth_service_test.go
- auth_middleware_test.go
- parser_test.go (esteso)
- compiler_test.go (esteso)
- duckdb_registry_test.go
- duckdb_test.go

### T7.2 — .env e .gitignore
### T7.3 — Verifica finale

## Ordine di Esecuzione
1. T1.1 -> T1.4 (compilabilita Go)
2. T3.1 -> T3.5 (bug logici)
3. T1.5 (Docker/PostgreSQL)
4. T2.1 -> T2.3 (handlers mancanti)
5. T4.1 -> T4.4 (DSL)
6. T5.1 -> T5.2 (NLP sidecar)
7. T6.1 -> T6.8 (Frontend)
8. T7.1 -> T7.3 (Test e verifica)

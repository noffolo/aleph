# Error Glossary — Aleph-v2

Technical errors (gRPC, logs) use English. User-facing messages use Italian with English technical terms.

## Error Codes

| Code | Italian Message | Technical Description |
|------|----------------|---------------------|
| ERR_NOT_FOUND | Risorsa non trovata | Resource does not exist |
| ERR_UNAUTHORIZED | Autenticazione richiesta | Authentication required |
| ERR_FORBIDDEN | Permessi insufficienti | Insufficient permissions |
| ERR_INTERNAL | Errore interno del sistema | Unexpected internal error |
| ERR_VALIDATION | Dati inseriti non validi | Invalid input data |
| ERR_UNAVAILABLE | Servizio temporaneamente non disponibile | Service temporarily down |
| ERR_DEADLINE_EXCEEDED | Operazione scaduta | Operation timed out |
| ERR_FAILED_PRECONDITION | Condizione preliminare non soddisfatta | Required condition not met |
| ERR_INVALID_ARGUMENT | Argomento non valido | Invalid argument provided |

## Translation Glossary (W6-02)

| English Term | Italian Translation | Notes |
|-------------|-------------------|-------|
| Authentication | Autenticazione | |
| Permission | Permesso | |
| Token | Token | Keep English |
| Database | Database | Keep English |
| Query | Query | Keep English |
| Timeout | Timeout | Keep English |
| Validation | Validazione | |
| Resource | Risorsa | |
| Service | Servizio | |
| Error | Errore | |
| Unavailable | Non disponibile | |
| Expired | Scaduto | |
| Invalid | Non valido | |
| Not found | Non trovato | |

## APIError Structure

```go
type APIError struct {
    Code    string            // Machine-readable error code (e.g., ERR_NOT_FOUND)
    Message string            // Italian user-facing message
    Details map[string]string // Additional context (English, technical)
    Err     error            // Wrapped underlying error (English, technical)
}
```

## Usage Pattern

Technical layer (logs, gRPC):
```go
slog.Error("query execution failed", "code", errors.ErrInternal, "error", err)
```

User-facing (API responses):
```go
errors.NewNotFound("", err) // Empty userMsg = use default Italian message
errors.NewNotFound("Progetto non trovato", err) // Custom Italian message
```
# SPEC-02: Secrets Management — gosecrets, Credential Passing, Frontend Cleanup

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 0, tasks W0-1, W0-2, W0-7  
**Findings addressed**: L1-L5, L8 (credential leakage), INF1 (plaintext secrets), R7 (apiKey in Zustand)  
**Related specs**: `docs/specs/wave0-auth-spec.md` (auth middleware depends on secrets), `docs/specs/wave3-frontend-spec.md` (Zustand cleanup, apiKey removal)  
**Status**: ✅ Approved — ready for execution

---

## 1. gosecrets Configuration

### Tool

**Package**: `github.com/bilustek/gosecrets`  
**Pattern**: Rails-style encrypted credentials — secrets in repo (encrypted), keys outside repo.

### Directory Structure

```
aleph-v2/
├── secrets/
│   ├── development.enc     # git-tracked (encrypted)
│   ├── production.enc      # git-tracked (encrypted)
│   └── development.key     # .gitignore (plaintext)
├── .gitignore              # + secrets/*.key
└── .github/
    └── workflows/
        └── deploy.yml      # reads GOSECRETS_PRODUCTION_KEY from GitHub Secrets
```

### Secrets Inventory

| Secret | Current Location | New Location |
|--------|-----------------|--------------|
| `database.url` | .env / os.Getenv | secrets/development.enc |
| `smtp.password` | .env / os.Getenv | secrets/development.enc |
| `aleph.api_key_secret_backend` | .env / os.Getenv | secrets/development.enc |
| `jwt.secret` | .env / os.Getenv | secrets/development.enc |
| `key_encryption_key` | .env / os.Getenv | secrets/development.enc |
| `ollama.base_url` | .env / config | secrets/development.enc |
| `postgres.dsn` | .env / os.Getenv | secrets/development.enc |
| `nlp.sidecar_url` | .env / config | secrets/development.enc |

### Go Loading Code

```go
// internal/config/config.go
import "github.com/bilustek/gosecrets"

func LoadConfig() (*Config, error) {
    secrets, err := gosecrets.Load()
    if err != nil {
        return nil, fmt.Errorf("gosecrets: %w", err)
    }

    return &Config{
        DBUrl:               secrets.String("database.url", "postgres://localhost:5432/aleph?sslmode=disable"),
        SMTPPassword:        secrets.String("smtp.password", ""),
        APIKeySecretBackend: secrets.MustString("aleph.api_key_secret_backend"),
        JWTSecret:           secrets.MustString("jwt.secret"),
        KeyEncryptionKey:    secrets.MustString("key_encryption_key"),
        OllamaBaseURL:       secrets.String("ollama.base_url", "http://localhost:11434"),
        PostgresDSN:         secrets.String("postgres.dsn", "postgres://aleph:aleph@localhost:5432/aleph?sslmode=disable"),
        NLPSidecarURL:       secrets.String("nlp.sidecar_url", "http://localhost:8001"),
    }, nil
}
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
- name: Decrypt secrets
  run: |
    echo "${{ secrets.GOSECRETS_PRODUCTION_KEY }}" > secrets/production.key
  env:
    GOSECRETS_ENV: production
```

### Development Setup

```bash
# First-time developer setup
gosecrets init                         # Creates secrets/development.key + .enc
gosecrets set database.url "postgres://localhost:5432/aleph?sslmode=disable"
gosecrets set jwt.secret "$(openssl rand -hex 32)"
# ... set all required secrets

# Running the app
GOSECRETS_ENV=development go run ./cmd/aleph/
```

---

## 2. Subprocess Credential Contract

### Current Problem

`internal/ingestion/engine.go` `runEmailFetch` (line 882-960):
- Constructs inline Python script with `imaplib`
- Passes email credentials via environment variables
- Environment variables visible in `/proc/<pid>/environ`

### Solution: Stdin Pipe Contract

```go
// Go side
type EmailCredentials struct {
    Server   string `json:"server"`
    Port     int    `json:"port"`
    Username string `json:"username"`
    Password string `json:"password"`
}

func runEmailFetch(ctx context.Context, cfg EmailConfig) error {
    cmd := exec.CommandContext(ctx, "python3", "-c", emailFetchScript)
    
    // START: Never on disk, never in env
    stdin, _ := cmd.StdinPipe()
    cmd.Start()
    
    json.NewEncoder(stdin).Encode(EmailCredentials{
        Server:   cfg.Server,
        Username: cfg.Username,
        Password: cfg.Password,  // This is still plaintext in Go memory — acceptable for now
    })
    stdin.Close()
    
    return cmd.Wait()
}
```

```python
# Python side
import sys, json, imaplib

creds = json.loads(sys.stdin.read())

mail = imaplib.IMAP4_SSL(creds["server"], creds["port"])
mail.login(creds["username"], creds["password"])
# ... process emails ...

# Clean up
del creds  # Remove from memory
mail.logout()
```

### Contract Rules

1. **Credentials NEVER in environment**: `cmd.Env` is minimal (`PATH=/usr/bin`, `LANG=en_US.UTF-8`)
2. **Credentials NEVER on disk**: stdin pipe only, no temp files
3. **Python side deletes after use**: `del creds` immediately after use
4. **No Python-to-Go reverse channel**: Python never sends credentials back to Go

### Affected Code Paths

| File | Current Method | Fix |
|------|---------------|-----|
| `internal/ingestion/engine.go:runEmailFetch()` | env vars | stdin pipe |
| `internal/dsl/compiler_tool.go:apiConnectorPythonTemplate` | urllib.request import | Remove from template; add SSRF-guarded HTTP client or stdin credentials |
| NLP sidecar Dockerfile | env vars | Add `--stdin-credentials` flag |

---

## 3. Frontend Secret Removal

### apiKey in Zustand — Removal Plan

**Current state**: `frontend/src/store/authSlice.ts` line 6, default `''`, set by `setProjectContext()`

**Target state**: Never stored in client memory. Server-side JWT cookie handles auth.

### Steps

1. **Remove from authSlice**:
```typescript
// Before
interface AuthSlice {
    apiKey: string;
    // ...
}

// After
interface AuthSlice {
    // apiKey REMOVED — use httpOnly cookie
    projectID: string;
    // ...
}
```

2. **Update `setProjectContext`**:
```typescript
// Before
setProjectContext: (projectID: string, apiKey: string) => { ... }

// After
setProjectContext: (projectID: string) => { ... }  // No apiKey param
```

3. **Update `useSSE.ts`**:
```typescript
// Before: reads apiKey from Zustand
const apiKey = useStore(s => s.apiKey);

// After: relies on httpOnly cookie (credentials: "include" already set)
// No explicit apiKey needed — browser sends cookie automatically
```

4. **Remove from `SetupWizard.tsx`** (line 164):
```tsx
// Before
<div>{apiKey}</div>
<button onClick={() => navigator.clipboard.writeText(apiKey)}>Copy</button>

// After
// Don't show the API key at all — user creates keys via SettingsView
// SetupWizard should prompt user to save the key immediately (shown once, then hidden)
<div>API key created. Save it now — it won't be shown again.</div>
<button onClick={copyAndMask}>Copy to clipboard</button>
```

5. **Fix `SettingsView.tsx`** masking direction:
```tsx
// Before: shows last 4 chars (security risk)
{k.key ? '...' + k.key.slice(-4) : '••••••••'}

// After: shows first 4 chars (prefix), suffixed with dots
{k.key ? k.key.slice(0, 4) + '••••••••••••••••' : '••••••••••••••••'}
```

6. **Verify `client.ts` session flow**:
```typescript
// Already correct — uses credentials: "include"
export const transport = createConnectTransport({
    baseUrl: API_BASE_URL,
    credentials: "include",
});
```

### Verification

```bash
# After W0-2: apiKey should not exist in frontend state
grep -rn "apiKey" frontend/src/store/ frontend/src/components/SetupWizard.tsx
# → Only in SettingsView.tsx (display-only, from server response, not Zustand)

# Browser DevTools: window.__ALEPH_STORE__ (if dev mode) should have no apiKey field
```

---

## 4. Environment Variable Cleanup

### .env → gosecrets Migration Checklist

- [ ] Remove from `.env`: `MASTER_KEY`, `JWT_SECRET`, `KEY_ENCRYPTION_KEY`, `ALEPH_API_KEY_SECRET_BACKEND`
- [ ] Remove from `.env`: `DB_PASSWORD`, `SMTP_PASSWORD`, `POSTGRES_DSN`
- [ ] Remove from `.env`: `OLLAMA_BASE_URL`, `NLP_SIDECAR_URL`
- [ ] Keep in `.env`: `ALEPH_ENV=development`, `ALEPH_PORT=8080` (non-sensitive config)
- [ ] Add to `.gitignore`: `secrets/*.key`, `.env` (double-check — ensure not tracked)
- [ ] Add to `CONTRIBUTING.md`: gosecrets setup instructions
- [ ] Add to `docs/DEPLOY.md`: production secrets rotation procedure

---

## 5. Verification

### Test Coverage

- [ ] `secrets_test.go`: `gosecrets.Load()` succeeds when key present; fails with clear error when key missing
- [ ] `authSlice.test.ts`: `apiKey` field removed from state
- [ ] `subprocess_creds_test.go`: Python script reads credentials from stdin only; env is clean

### Manual Verification

```bash
# No secrets in .env
grep -E "(SECRET|PASSWORD|KEY)" .env
# → Only non-sensitive defaults

# gosecrets works in dev
GOSECRETS_ENV=development go run ./cmd/aleph/ 2>&1 | head -5
# → Application starts without credential errors

# Python subprocess can't read env
# (Run ingestion test and verify /proc/<pid>/environ has no passwords)

# Zustand store has no apiKey
# (Open browser DevTools, check __ALEPH_STORE__ state)
```

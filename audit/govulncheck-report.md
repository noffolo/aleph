# Govulncheck Report

**Date:** 2 May 2026
**Tool:** govulncheck v1.3.0
**Scope:** `./...` (all Go packages)

---

## Result: ✅ CLEAN — 0 Vulnerabilities Affecting Your Code

```
No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 2 vulnerabilities in packages you import and 0
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
```

---

## Imported Package Vulnerabilities (Not Called by Your Code)

The following vulnerabilities exist in packages you import, but your code
does not call the affected functions:

| Package | Vulnerability | Severity | Fixed In | Notes |
|---------|--------------|----------|----------|-------|
| `golang.org/x/net` | Various (GV) | — | latest | Network library transitive dependency |
| `google.golang.org/grpc` | Various (GV) | — | latest | gRPC library transitive dependency |

These are standard transitive dependencies. No action required unless you
introduce code that calls the affected functions.

---

## Recommendations

1. Run `govulncheck ./...` regularly (weekly or per-release)
2. Keep `go.mod` dependencies updated with `go get -u` for patch versions
3. For critical security releases, subscribe to Go Security announcements:
   https://groups.google.com/g/golang-announce

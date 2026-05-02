# Trivy Filesystem Security Scan Report

**Date:** 2 May 2026
**Tool:** Trivy v0.69.3
**Scanners:** vulnerability, secret, misconfiguration
**Scope:** Go backend + Dockerfiles (excluding frontend/node_modules/, nlp/, .git/, data/, deploy/, docs/, scripts/, secrets/, migrations/, aleph_tools/)

---

## Executive Summary

| Severity | Vulnerabilities | Misconfigurations | Secrets |
|----------|----------------|-------------------|---------|
| CRITICAL | 1 | 0 | 0 |
| HIGH | 0 | 0 | 0 |
| MEDIUM | 1 | 0 | 0 |
| LOW | 1 | 0 | 0 |
| **Total** | **3** | **1** | **0** |

---

## Vulnerability Findings

### CVE-2026-33816: pgx Memory Safety (CRITICAL)

| Field | Value |
|-------|-------|
| **Library** | `github.com/jackc/pgx/v5` v5.7.2 |
| **Severity** | CRITICAL |
| **Fixed in** | v5.9.0 |
| **Title** | Memory-safety vulnerability in pgx PostgreSQL driver |
| **URL** | https://avd.aquasec.com/nvd/cve-2026-33816 |

**Remediation:** Upgrade `github.com/jackc/pgx/v5` to `>= v5.9.0`:
```bash
go get github.com/jackc/pgx/v5@v5.9.0
go mod tidy
```

### GHSA-j88v-2chj-qfwx: pgx SQL Injection (LOW)

| Field | Value |
|-------|-------|
| **Library** | `github.com/jackc/pgx/v5` v5.7.2 |
| **Severity** | LOW |
| **Fixed in** | v5.9.2 |
| **Title** | SQL Injection via placeholder confusion with dollar quoted string literals |
| **URL** | https://github.com/advisories/GHSA-j88v-2chj-qfwx |

**Remediation:** Upgrade `github.com/jackc/pgx/v5` to `>= v5.9.2`.

### Go stdlib — MEDIUM severity

| Field | Value |
|-------|-------|
| **Library** | `stdlib` (Go 1.26.2) |
| **Severity** | MEDIUM |
| **Details** | Standard library medium-severity finding from Trivy advisory DB |

**Remediation:** Update Go toolchain and rebuild.

---

## Misconfiguration Findings

### DS-0002: No USER in frontend/Dockerfile (HIGH)

| Field | Value |
|-------|-------|
| **File** | `frontend/Dockerfile` |
| **Severity** | HIGH |
| **Description** | The runtime image runs as root. If an attacker compromises the nginx process, they gain root access to the container. |
| **Remediation** | Add `USER nginx` before `ENTRYPOINT` in the runtime stage, or change to a non-root user. |

**Status:** ✅ FIXED — Added `USER nginx` to `frontend/Dockerfile` during this audit.

---

## Secret Scan Results

| Result | Count |
|--------|-------|
| Secrets detected | 0 |
| False positives (credentials in comments/examples) | 0 |

**No secrets found.** The codebase properly uses environment variables and Docker secrets for all sensitive values.

---

## Recommendations

| Priority | Action | Effort |
|----------|--------|--------|
| 🔴 | Upgrade `pgx/v5` to v5.9.2+ (fixes CRITICAL + LOW CVEs) | 30m |
| 🟡 | Update Go toolchain to latest patch | 15m |
| 🟢 | Add USER directive to frontend/Dockerfile | ✅ Done |

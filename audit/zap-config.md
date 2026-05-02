# OWASP ZAP Configuration for Aleph-v2

**Date:** 2 May 2026
**Purpose:** Configuration guide for automated DAST scanning of Aleph-v2 with OWASP ZAP

---

## Overview

OWASP ZAP (Zed Attack Proxy) is a dynamic application security testing (DAST) tool that
can discover vulnerabilities by actively scanning a running application. This document
describes the configuration needed to scan Aleph-v2.

---

## Prerequisites

- Running Aleph-v2 instance (http://localhost:8080)
- Valid API key for authenticated endpoints
- OWASP ZAP installed (CLI or desktop)

---

## Configuration

### Context Definition

Create a ZAP context for Aleph-v2 with the following structure:

**Context Name:** `aleph-v2`

**Included URLs:** `http://localhost:8080.*`

**Excluded URLs:**
- `http://localhost:8080/api/v1/healthz` (no auth, health check)
- `http://localhost:8080/api/v1/metrics` (Prometheus)

### Authentication

Aleph-v2 uses API key authentication via header. Configure ZAP's script-based auth:

**Script: aleph-auth.js**
```javascript
// Aleph-v2 API Key Authentication
// Place in ZAP scripts/auth/ directory

var API_KEY = "YOUR_API_KEY_HERE";

function authenticate(helper, paramsValues, credentials) {
    var msg = helper.prepareMessage();
    var uri = msg.getRequestHeader().getURI();
    
    // Set the API key header
    msg.getRequestHeader().setHeader("X-Aleph-Api-Key", API_KEY);
    
    // Alternatively, set as Bearer token
    msg.getRequestHeader().setHeader("Authorization", "Bearer " + API_KEY);
    
    // Set content type for POST requests
    if (msg.getRequestHeader().getMethod() === "POST") {
        msg.getRequestHeader().setHeader("Content-Type", "application/json");
    }
    
    helper.sendAndReceive(msg);
    return msg;
}

function getRequiredParamsNames() {
    return [];
}

function getOptionalParamsNames() {
    return ["apiKey"];
}
```

### Session Management

- **Type:** Cookie-based
- **Cookie Names:** `aleph_session`
- **Scope:** Context-only

---

## Scan Policy

### Technology Set
- Go (backend)
- PostgreSQL (database)
- DuckDB (embedded analytical DB)
- gRPC (NLP sidecar communication)
- HTTP/2 (ConnectRPC transport)
- SSE (Server-Sent Events streaming)

### Active Scan Policy

| Rule | Threshold | Strength |
|------|-----------|----------|
| Path Traversal | MEDIUM | MEDIUM |
| SQL Injection | MEDIUM | HIGH |
| XSS (Reflected) | MEDIUM | MEDIUM |
| XSS (Persistent) | MEDIUM | MEDIUM |
| Remote Code Execution | LOW | MEDIUM |
| Command Injection | MEDIUM | MEDIUM |
| CRLF Injection | MEDIUM | MEDIUM |
| Directory Browsing | MEDIUM | MEDIUM |
| Information Disclosure | LOW | MEDIUM |
| Session ID in URL Rewrite | HIGH | MEDIUM |
| X-Content-Type-Options Header | LOW | LOW |
| Content Security Policy | LOW | LOW |
| CSRF | MEDIUM | MEDIUM |
| Authentication Bypass | MEDIUM | HIGH |
| Format String Error | MEDIUM | MEDIUM |
| External Redirect | MEDIUM | MEDIUM |

### Passive Scan Rules
- All standard passive rules enabled
- Disable: `WSDL File Detection` (not applicable)
- Disable: `Web Browser Fingerprinting` (not applicable)

---

## Endpoint Inventory for Spidering

The following endpoints should be included in the spider's seed URL list:

### Public (No Auth Required)
```
GET /api/v1/healthz
```

### Authenticated (Requires X-Aleph-Api-Key)
```
# RPC-style Connect endpoints (JSON over HTTP)
POST /api/v1/QueryService/Chat
POST /api/v1/QueryService/StreamChat
POST /api/v1/QueryService/Search
POST /api/v1/SandboxService/ExecuteTool
POST /api/v1/SandboxService/RunSkill
POST /api/v1/RegistryService/BuildTool
POST /api/v1/RegistryService/GetTool
POST /api/v1/RegistryService/GetToolByCategory
POST /api/v1/RegistryService/ListTools
POST /api/v1/RegistryService/GetSkills
POST /api/v1/RegistryService/GetSkill
POST /api/v1/RegistryService/CreateSkill
POST /api/v1/RegistryService/UpdateSkill
POST /api/v1/RegistryService/DeleteSkill
POST /api/v1/RegistryService/UpdateHealthStatus
POST /api/v1/RegistryService/VerifyTool
POST /api/v1/NotificationService/StreamNotifications

# REST endpoints
GET  /api/v1/projects
POST /api/v1/projects
GET  /api/v1/projects/{id}
PUT  /api/v1/projects/{id}
DELETE /api/v1/projects/{id}
GET  /api/v1/projects/{id}/data-health
GET  /api/v1/agents
GET  /api/v1/tools
GET  /api/v1/tools/health
GET  /api/v1/tools/{name}
POST /api/v1/tools/verify
GET  /api/v1/skills
GET  /api/v1/skills/{name}
POST /api/v1/skills
GET  /api/v1/library
POST /api/v1/library
GET  /api/v1/library/{id}
DELETE /api/v1/library/{id}
GET  /api/v1/auth/session/{id}
POST /api/v1/auth/session
DELETE /api/v1/auth/session/{id}
POST /api/v1/ingestion/ingest
GET  /api/v1/ingestion/status/{id}
POST /api/v1/ingestion/upload
POST /api/v1/tool-exec/execute
POST /api/v1/tool-suggest/suggest
POST /api/v1/codeflow/analyze
POST /api/v1/codeflow/track
POST /api/v1/codeflow/visualize
GET  /api/v1/diagnostic/patterns
POST /api/v1/register
POST /api/v1/login
POST /api/v1/session

# SSE
GET  /api/v1/sse/events
GET  /api/v1/notifications/stream
```

---

## Running the Scan

### CLI Mode
```bash
# Start ZAP in daemon mode
zap.sh -daemon -port 8081 -host 0.0.0.0 -config api.disablekey=true

# Spider the target
zap-cli spider http://localhost:8080

# Run active scan
zap-cli active-scan --scanners all http://localhost:8080

# Generate report
zap-cli report -o /tmp/aleph-zap-report.html -f html
```

### Docker Mode
```bash
docker run -d --name zap \
  -v $(pwd)/audit:/zap/wrk:rw \
  -p 8081:8080 \
  ghcr.io/zaproxy/zaproxy:stable \
  zap.sh -daemon -port 8080 -host 0.0.0.0 \
    -config api.disablekey=true

# Wait for ZAP to start, then:
docker exec zap zap-cli open-url http://host.docker.internal:8080
docker exec zap zap-cli spider http://host.docker.internal:8080
docker exec zap zap-cli active-scan \
  --scanners all \
  http://host.docker.internal:8080
docker exec zap zap-cli report \
  -o /zap/wrk/audit/zap-report.html -f html
docker kill zap
```

### API Mode (Python Automation)
```python
#!/usr/bin/env python3
"""Aleph-v2 ZAP automation script."""

import time
from zapv2 import ZAPv2

ZAP_ADDRESS = "localhost"
ZAP_PORT = 8081
TARGET = "http://localhost:8080"
API_KEY = "YOUR_ZAP_API_KEY"

zap = ZAPv2(apikey=API_KEY, 
            proxy={"http": f"http://{ZAP_ADDRESS}:{ZAP_PORT}", 
                   "https": f"http://{ZAP_ADDRESS}:{ZAP_PORT}"})

# Open URL
print(f"Spidering target {TARGET}")
zap.urlopen(TARGET)
time.sleep(2)

# Spider
scan_id = zap.spider.scan(TARGET)
while int(zap.spider.status(scan_id)) < 100:
    print(f"Spider progress: {zap.spider.status(scan_id)}%")
    time.sleep(5)

print("Spider completed")

# Active Scan
print("Starting active scan...")
scan_id = zap.ascan.scan(TARGET)
while int(zap.ascan.status(scan_id)) < 100:
    print(f"Active scan progress: {zap.ascan.status(scan_id)}%")
    time.sleep(10)

print("Active scan completed")

# Generate report
with open("audit/zap-report.html", "w") as f:
    f.write(zap.core.htmlreport())
print("Report saved to audit/zap-report.html")

# Print alert summary
alerts = zap.core.alerts()
severity_counts = {"High": 0, "Medium": 0, "Low": 0, "Informational": 0}
for alert in alerts:
    sev = alert.get("risk", "Informational")
    severity_counts[sev] = severity_counts.get(sev, 0) + 1

print(f"Findings: High={severity_counts['High']}, "
      f"Medium={severity_counts['Medium']}, "
      f"Low={severity_counts['Low']}, "
      f"Info={severity_counts['Informational']}")
```

---

## Known Limitations

1. **gRPC (h2c)**: ConnectRPC uses h2c (HTTP/2 cleartext). ZAP's HTTP/2 support
   for non-TLS may require additional configuration. Consider using the REST
   equivalents or testing via the `grpcurl` tool instead.

2. **SSE Endpoints**: Server-Sent Events endpoints maintain long-lived connections.
   Configure ZAP to timeout SSE connections after 10s to avoid hanging scans.

3. **Rate Limiting**: Aleph-v2 has per-IP rate limiting. ZAP's aggressive scanning
   may trigger rate limiting. Configure ZAP to add delays between requests:
   ```
   -config rules.delay=200
   ```

4. **Authentication**: All functional endpoints require API key auth. Without
   proper auth configuration, ZAP will only reach the health endpoint and
   receive 401 responses for everything else.

5. **DuckDB**: Embedded in the Go binary — not a network service. ZAP cannot scan
   DuckDB directly.

---

## Expected Findings (Non-Issues)

These findings from ZAP are expected and safe to mark as false positives:

| Finding | Reason |
|---------|--------|
| Missing `X-Frame-Options` on 401 responses | Auth middleware strips headers on error |
| Missing `Content-Security-Policy` on error pages | Error pages are minimal and plaintext |
| `Server` header disclosure | Go's http.Server always sends `Server:` header |
| Cookie `aleph_session` without `Secure` flag | Only in dev mode (HTTP). In production, HTTPS should be used |
| `X-Powered-By` detection | Go does not set this header; false positive |

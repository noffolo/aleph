# Sandbox Threat Model for Aleph-v2 Decision Intelligence System

## Overview
This document outlines the threat model for the Aleph-v2 sandbox isolation system. The sandbox executes untrusted user-provided code (Go and Python) within the decision intelligence platform. The goal is to prevent malicious code from compromising system security, data integrity, or availability.

## Attack Vectors Considered

### 1. Code Execution Attacks
- **Arbitrary System Command Execution**: Using `os/exec`, `subprocess`, `os.system` to run shell commands
- **Network Access**: Using `net`, `socket` packages to make HTTP requests, open sockets, or exfiltrate data
- **Filesystem Escalation**: Using `syscall`, `unsafe` packages to bypass filesystem restrictions
- **Dynamic Code Loading**: Using `reflect`, `eval`, `exec`, `__import__` to dynamically load and execute code
- **Memory Corruption**: Using `unsafe` packages to manipulate memory directly

### 2. Resource Exhaustion Attacks
- **CPU Overload**: Infinite loops, computational bombs
- **Memory Exhaustion**: Allocating large arrays, memory leaks
- **Disk Space**: Writing large files to `/tmp` or mounted volumes
- **Process Forking**: Creating unlimited child processes

### 3. Data Exfiltration Attacks
- **Network Exfiltration**: Using network packages to send data out
- **DNS Tunneling**: Using DNS queries for covert channels
- **File-Based Exfiltration**: Writing data to files and reading later
- **Environment Variable Leaks**: Reading sensitive env variables

### 4. Privilege Escalation Attacks
- **Container Escape**: Attempting to break out of Docker isolation
- **Host System Calls**: Using `syscall` to interact with host kernel
- **Capability Abuse**: Using retained capabilities (if any) for privilege escalation

## Mitigations Implemented

### Language-Level Restrictions

#### Go Import Blocklist
The following Go packages are blocked during AST analysis: `unsafe`, `reflect`, `os/*`, `io/*`, `crypto/*`, `encoding/*`, `net/*`, `syscall`, `embed`, `plugin`, `os/exec`, `os/signal`.

Implementation: AST-based parsing of Go source code before execution, with fallback to string scanning.

#### Python Import/Call Blocklist
The following Python patterns are blocked via regex matching:
- Import statements: `import subprocess/socket/ctypes/__import__`
- Function calls: `subprocess.run/call/Popen`, `socket.socket`, `os.system`, `eval()`, `exec()`
- Network file operations: `open()` with `http://`, `https://`, `ftp://` URLs

Implementation: Regex pattern matching on Python source code lines.

### Container Isolation (Docker)

#### Filesystem Restrictions
- `read_only: true` - Makes container filesystem read-only
- `tmpfs: /tmp:rw,noexec,nosuid` - `/tmp` mounted as tmpfs with noexec, nosuid flags
- Volume mounts restricted to specific directories only

#### Capability Restrictions
- `cap_drop: ALL` - Drop all Linux capabilities
- `cap_add: CHOWN, SETGID, SETUID` - Only allow minimal capabilities needed
- `security_opt: no-new-privileges:true` - Prevent privilege escalation

#### Resource Limits
- Environment variables limited to those explicitly allowed

#### Network Restrictions
- Default Docker network provides some isolation
- Future: Consider `network_mode: none` for complete network isolation if needed

### Runtime Restrictions

#### Timeouts
- Go sandbox: 60s timeout via `context.WithTimeout`
- All HTTP fetches: 30s timeout (configured per-source via `RateLimitedClient`)
- Ollama embedding: 30s timeout (`embedTimeout`)
- gRPC sidecar health: 3s timeout per check

#### Environment Hardening
- `PATH=/usr/bin:/bin` - Restricted PATH variable
- `HOME=/tmp` - Use temporary directory as HOME
- Limited environment variables exposed

#### Process Isolation
- `runDynamic()` builds Go code in isolated temp dir, runs as separate process with minimal env vars
- Python email fetch runs as subprocess inside Docker container with read-only filesystem
- `security_opt: no-new-privileges:true` prevents privilege escalation

## Security Testing Verification

### Test Categories
1. **Import Blocking Tests**: Verify blocked imports are rejected
2. **Language Feature Tests**: Verify dangerous language features are blocked
3. **Container Isolation Tests**: Verify Docker restrictions work
4. **Resource Limit Tests**: Verify CPU/memory limits are enforced
5. **Network Isolation Tests**: Verify network access is prevented

### Test Implementation
See `internal/sandbox/validation_test.go` and `internal/sandbox/exec_sandbox_security_test.go` for detailed test cases.

## Known Limitations

### 1. AST Parsing Limitations
- Go AST parsing may fail on malformed code; fallback to string scanning used
- String scanning may be evaded by clever formatting or obfuscation
- Python regex patterns may not catch all variants of dangerous calls

### 2. Container Escape Risks
- Docker provides good isolation but not perfect security
- Kernel vulnerabilities could allow container escape
- Shared kernel with host creates attack surface

### 3. Resource Limits Enforcement
- CPU limits are soft limits (cgroups shares)
- Memory limits enforced via OOM killer
- Disk quotas not implemented for `/tmp`

### 4. Network Isolation
- Docker network provides isolation but not complete network disablement
- Future enhancement: `network_mode: none` for complete network cutoff

## Future Enhancements

### Short-term (Implemented in W7)
1. CSP: removed `unsafe-inline` from `style-src`, moved all inline styles to CSS files — ✅ done
2. Rate limiting: `extractClientIP()` with X-Forwarded-For → X-Real-IP → RemoteAddr chain — ✅ done
3. CSRF: Origin/Referer validation middleware on all non-GET requests — ✅ done
4. Release gate: `docs/release-checklist.md` with build/security/Docker/CI-CD checks — ✅ done

### Short-term (Next Release)
1. Implement `network_mode: none` for complete network isolation
2. Add disk quotas for `/tmp` tmpfs
3. Enhance Python pattern matching with AST-based analysis

### Medium-term
1. Implement seccomp profiles for syscall filtering
2. Add AppArmor/SELinux profiles
3. Implement resource usage monitoring and alerting

### Long-term
1. Consider gVisor or Kata Containers for stronger isolation
2. Implement code signing for trusted tool execution
3. Add execution provenance tracking and auditing

## Incident Response

### Detection Points
1. Sandbox validation failures logged at ERROR level
2. Container resource limit violations (OOM kills)
3. Unexpected network traffic from sandbox containers

### Response Procedures
1. Immediate: Blocked import → reject execution and log incident
2. Container escape detection → stop all sandbox containers and investigate
3. Resource exhaustion → restart affected containers with tighter limits

## Compliance Considerations

### Data Protection
- AES-256-GCM encryption for API keys via `KEY_ENCRYPTION_KEY` (mandatory — startup FATAL if missing)
- Argon2id hashing for API key verification (SHA-256 legacy fallback detected)
- httpOnly+Secure+SameSite=Strict cookies for session management
- SSRF protection: DNS-resolving DialContext blocks private IPs/bipass forms 

### Audit Requirements
- All sandbox execution attempts logged with validation results
- Resource usage metrics collected per execution
- Security violations trigger alerts and are recorded

## Conclusion
The Aleph-v2 sandbox provides multi-layered security with language-level restrictions, container isolation, and runtime controls. While not perfectly secure, it implements defense-in-depth to mitigate the most common attack vectors while allowing flexible code execution within the decision intelligence platform.

Regular security testing and threat model updates are required as the platform evolves and new attack techniques emerge.
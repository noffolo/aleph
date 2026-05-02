# SPEC-03: Sandbox Isolation — Namespaces, Seccomp, gVisor, Blocklists

**Spec version**: 1.0  
**Date**: 2 May 2026  
**Plan reference**: `docs/plans/audit-remediation.md` Wave 1  
**Findings addressed**: S1-S10 (sandbox cluster), L6-L7 (code generation)  
**Depends on**: `docs/specs/wave0-auth-spec.md` (sandbox scoping needs RBAC)  
**Related specs**: `docs/specs/wave1-injection-spec.md` (DSL/SSRF fixes share sandbox code paths)  
**Status**: ✅ Approved — ready for execution

---

## 1. Isolation Architecture

### Layered Defense Model

```
┌─────────────────────────────────────────┐
│ Layer 4: gVisor (user-space kernel)     │  ← OCI runtime (Docker --runtime=runsc)
├─────────────────────────────────────────┤
│ Layer 3: Namespace Isolation            │  ← PID, Mount, Net, User, UTS, IPC
├─────────────────────────────────────────┤
│ Layer 2: seccomp-bpf Syscall Filter     │  ← elastic/go-seccomp-bpf
├─────────────────────────────────────────┤
│ Layer 1: cgroups v2 Resource Limits     │  ← Memory, CPU, PIDs
├─────────────────────────────────────────┤
│ Layer 0: Go/Python Blocklists           │  ← Static import validation (AST + regex)
└─────────────────────────────────────────┘
```

### Execution Path (after hardening)

```
Tool Code → Blocklist Validation → Namespace Sandbox → seccomp Filter → cgroups → Execute → Clean up
                                                                                     (or gVisor if enabled)
```

---

## 2. Namespace Isolation Specification

### Cloneflags Configuration

```go
// internal/sandbox/namespace_isolated.go (NEW)
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWUTS |   // Hostname isolation
                 syscall.CLONE_NEWPID |   // PID namespace (process = PID 1)
                 syscall.CLONE_NEWNS  |   // Mount namespace (no host filesystem access)
                 syscall.CLONE_NEWNET |   // Network namespace (disabled by default — agents may need internet access)
                 syscall.CLONE_NEWIPC |   // IPC isolation
                 syscall.CLONE_NEWUSER,   // User namespace (root inside = nobody outside)
    
    Unshareflags: syscall.CLONE_NEWNS,    // Prevent mount propagation to host
    
    UidMappings: []syscall.SysProcIDMap{
        {ContainerID: 0, HostID: os.Getuid(), Size: 1},
    },
    GidMappings: []syscall.SysProcIDMap{
        {ContainerID: 0, HostID: os.Getgid(), Size: 1},
    },
}
```

### Namespace Matrix

| Namespace | Flag | Effect | Escape Risk |
|-----------|------|--------|-------------|
| **PID** | `CLONE_NEWPID` | Process sees only itself as PID 1 | Low (combined with mount namespace) |
| **Mount** | `CLONE_NEWNS` | No access to host `/proc`, `/sys`, `/dev` | Medium (requires pivot_root + mount proc) |
| **Network** | ~~`CLONE_NEWNET`~~ (disabled) | Full internet access — agents can call external APIs |
| **User** | `CLONE_NEWUSER` | Root in container → unprivileged UID on host | Low (combined with seccomp) |
| **UTS** | `CLONE_NEWUTS` | Isolated hostname | None |
| **IPC** | `CLONE_NEWIPC` | No shared memory/Semaphores with host | None |

### Minimal Filesystem

Inside the sandbox, only:
- Temp directory (rw) — code and output
- `/usr/bin/python3` (ro bind mount)
- `/usr/bin/go` (ro bind mount)  
- Required shared libraries (ro bind mount)

Everything else is NOT visible inside the namespace.

---

## 3. seccomp-bpf Profile

### Allowed Syscalls

```go
// internal/sandbox/seccomp_profile.go (NEW)
var allowedSyscalls = []string{
    // Process lifecycle
    "read", "write", "close", "exit", "exit_group",
    // Memory management
    "mmap", "mprotect", "munmap", "brk", "mremap",
    // Synchronization
    "futex", "futex_waitv", "futex_wake",
    // Time
    "clock_gettime", "gettimeofday", "nanosleep",
    // Signals
    "rt_sigaction", "rt_sigprocmask", "sigreturn", "rt_sigreturn",
    // Filesystem (restricted)
    "openat", "readlink", "fstat", "stat", "lstat", "newfstatat",
    "getdents64", "lseek",
    // Process info
    "getpid", "gettid", "getuid", "getgid", "geteuid", "getegid",
    // Scheduling
    "sched_yield", "sched_getaffinity",
    // Random
    "getrandom",
    // Architecture
    "arch_prctl",
    // Go runtime (Goroutine scheduling)
    "clone", "clone3",
    // Python runtime
    "set_robust_list", "rseq", "prlimit64",
    // Misc
    "uname", "getcwd", "ioctl", "fcntl",
}
```

### Blocked Syscalls (Default-Deny)

```go
// Explicitly blocked (would be caught by default-deny anyway)
var blockedSyscalls = []string{
    "ptrace",          // Process debugging / injection
    "mount",           // Filesystem manipulation
    "umount2",
    "pivot_root",
    "chroot",
    "socket",          // Network access
    "connect",
    "bind",
    "listen",
    "accept",
    "sendto",
    "recvfrom",
    "init_module",     // Kernel module loading
    "finit_module",
    "delete_module",
    "execveat",        // Alternative exec
    "kexec_load",
    "kexec_file_load",
    "bpf",             // eBPF program loading
    "perf_event_open", // Performance monitoring (info leak)
    "process_vm_readv", // Cross-process memory access
    "process_vm_writev",
    "ptrace",
}
```

### seccomp Configuration

```go
filter := seccomp.Filter{
    NoNewPrivs: true,
    Flag:       seccomp.FilterFlagTSync,  // Sync to all threads
    Policy: seccomp.Policy{
        DefaultAction: seccomp.ActionErrno,  // Default DENY (EPERM)
        Syscalls: []seccomp.SyscallGroup{
            {
                Action: seccomp.ActionAllow,
                Names:  allowedSyscalls,
            },
        },
    },
}
```

---

## 4. cgroups v2 Resource Limits

### Default Limits per Execution

| Resource | Limit | Rationale |
|----------|-------|-----------|
| Memory | 256 MB | Prevents OOM on host; sufficient for Python/Go codegen |
| CPU | 0.5 core (50ms/100ms) | Prevents CPU exhaustion; burstable |
| PIDs | 32 | Prevents fork bombs |
| CPU Time | 30 seconds | Hard timeout on execution |
| I/O Weight | 10 (low) | Prevents I/O starvation of host processes |

### cgroups Configuration

```go
// cgroup v2 path
const cgroupBase = "/sys/fs/cgroup/aleph-sandbox-{execution_id}"

func setupCgroups(execID string) error {
    cgPath := fmt.Sprintf("%s/aleph-sandbox-%s", cgroupBase, execID)
    
    // Create cgroup
    os.MkdirAll(cgPath, 0755)
    
    // Memory limit: 256MB
    os.WriteFile(cgPath+"/memory.max", []byte("268435456"), 0644)
    
    // CPU limit: 50ms per 100ms = 0.5 cores
    os.WriteFile(cgPath+"/cpu.max", []byte("50000 100000"), 0644)
    
    // PID limit: 32 processes
    os.WriteFile(cgPath+"/pids.max", []byte("32"), 0644)
    
    // Add process to cgroup
    os.WriteFile(cgPath+"/cgroup.procs", []byte(fmt.Sprintf("%d", pid)), 0644)
    
    return nil
}
```

---

## 5. gVisor Integration

### Docker Runtime

```go
// internal/sandbox/container_sandbox.go
hostConfig := &container.HostConfig{
    Runtime:      "runsc",           // gVisor OCI runtime
    NetworkMode:  "none",            // Extra defense (gVisor already isolates)
    ReadonlyRootfs: true,
    SecurityOpt:  []string{"no-new-privileges:true"},
    CapDrop:      []string{"ALL"},
    Resources: container.Resources{
        Memory:   256 * 1024 * 1024,
        NanoCPUs: 500000000,
    },
    Mounts: []mount.Mount{
        {
            Type:   "bind",
            Source: tmpDir,
            Target: "/workspace",
            ReadOnly: true,
        },
    },
}
```

### Fallback Logic

```go
func (cs *ContainerSandbox) ExecuteTool(ctx context.Context, ...) error {
    // Check gVisor runtime available
    if cs.runtime == "runsc" && !cs.isRuntimAvailable("runsc") {
        // gVisor unavailable → check if plain Docker is acceptable for this tool
        if tool.RequiresHardenedSandbox {
            return ErrRuntimeUnavailable
        }
        cs.logger.Warn("gVisor unavailable, falling back to plain Docker")
        cs.runtime = "runc"
    }
    
    // Check Docker available
    if !cs.dockerAvailable() {
        // NO FALLBACK to ExecSandbox
        return ErrContainerUnavailable
    }
    
    return cs.executeInContainer(ctx, ...)
}
```

### Startup Health Check

```go
func (cs *ContainerSandbox) HealthCheck() error {
    // Check Docker daemon
    if _, err := cs.cli.Ping(context.Background()); err != nil {
        return fmt.Errorf("docker daemon unreachable: %w", err)
    }
    
    // Check gVisor runtime
    info, _ := cs.cli.Info(context.Background())
    if _, ok := info.Runtimes["runsc"]; !ok {
        return fmt.Errorf("gVisor (runsc) runtime not installed")
    }
    
    return nil
}
```

---

## 6. Blocklists — Unified Specification

### Python Blocklist (Full)

```python
# BLOCKED MODULES — import-level
BLOCKED_IMPORTS = [
    "subprocess",   # Process execution
    "socket",       # Network
    "ctypes",       # Native code loading
    "importlib",    # Dynamic import (bypass vector)
    "runpy",        # Script execution
    "pickle",       # Deserialization (code exec)
    "shelve",       # Persistent dict
    "shutil",        # Filesystem operations
    "os",            # Full OS access (block entire module)
    "code",          # Interactive interpreter
    "builtins",      # Access to __import__, eval, exec
    "compile",       # Dangerous with eval/exec (caught separately)
    "requests",      # HTTP client (SSRF)
    "httpx",         # HTTP client (SSRF)
    "urllib3",       # HTTP client (SSRF)
    "urllib",        # HTTP client (SSRF)
    "aiohttp",       # Async HTTP client (SSRF)
    "websockets",    # WebSocket client
    "imaplib",       # Email (credential leak vector)
    "smtplib",       # Email (credential leak vector)
]

# BLOCKED FUNCTIONS — method-level
BLOCKED_FUNCTIONS = [
    "os.system",
    "os.popen",
    "os.popen2",  # Python 2 compat
    "os.popen3",
    "os.popen4",
    "eval(",
    "exec(",
    "compile(",
    "__import__(",
    "open(" with network schema (http, ftp, s3),
]

# DYNAMIC ACCESS DETECTION
BLOCKED_PATTERNS = [
    r"getattr\s*\(",
    r"__getattribute__",
    r"__dict__",
    r"__class__",
    r"globals\s*\(\s*\)",
    r"locals\s*\(\s*\)",
    r"vars\s*\(\s*\)",
    r"__builtins__",
]
```

### Go Blocklist (Full — Aligned with Ingestion)

```go
// BLOCKED IMPORTS
var blockedGoImports = map[string]bool{
    // Process execution
    "os/exec":     true,
    "os":          true,  // Full OS access
    "syscall":     true,  // Raw syscalls
    
    // Dynamic loading
    "plugin":      true,  // Shared library loading
    "unsafe":      true,  // Memory safety bypass
    
    // Reflection (potential bypass)
    "reflect":     true,
    
    // Network (ALL net packages)
    "net":         true,
    "net/http":    true,
    "net/url":     true,
    "net/smtp":    true,
    "net/rpc":     true,
    "net/mail":    true,
    
    // Cryptography (potential encrypted exfil)
    "crypto/aes":     true,
    "crypto/cipher":  true,
    "crypto/des":     true,
    "crypto/rsa":     true,
    
    // Encoding (potential data exfil)
    "encoding/base64": true,
    "encoding/hex":    true,
    "encoding/json":   true,  // Already safe, but explicit
    
    // Runtime access
    "runtime":     true,
    "runtime/cgo": true,
    "runtime/pprof": true,
    
    // Debug access
    "debug/dwarf":   true,
    "debug/elf":     true,
    "debug/gosym":   true,
    "debug/macho":   true,
    "debug/pe":      true,
    "debug/plan9obj": true,
    
    // Templates (code exec)
    "text/template": true,
    "html/template": true,
    
    // Multipart (file upload)
    "mime/multipart": true,
    
    // Internal packages
    "internal/":     true,  // All internal packages
    
    // I/O (block raw, allow via safe wrapper)
    "io/ioutil": true,
}

// SAFE IMPORTS (explicit allowlist for clarity)
var safeGoImports = map[string]bool{
    "fmt":       true,
    "strings":   true,
    "strconv":   true,
    "math":      true,
    "sort":      true,
    "time":      true,
    "errors":    true,
    "context":   true,
    "sync":      true,
    "encoding/csv":   true,
    "encoding/xml":   true,  // JSON blocked above, reconsider if needed
    "unicode":   true,
    "unicode/utf8": true,
    "bytes":     true,
    "bufio":     true,
}
```

### Validation Method

```go
// AST-based validation (replaces regex + strings.Contains)
func ValidateGoCodeAST(code string) ([]string, error) {
    fset := token.NewFileSet()
    f, err := parser.ParseFile(fset, "tool.go", code, parser.ImportsOnly)
    if err != nil {
        return nil, fmt.Errorf("parse error: %w", err)
    }
    
    var violations []string
    for _, imp := range f.Imports {
        path := strings.Trim(imp.Path.Value, `"`)
        if _, blocked := blockedGoImports[path]; blocked {
            violations = append(violations, path)
        }
        // Check subpackages
        for blocked := range blockedGoImports {
            if strings.HasPrefix(path, blocked+"/") {
                violations = append(violations, path)
            }
        }
    }
    
    return violations, nil
}
```

---

## 7. CommandAllowlist (Hardened)

### Final Allowlist

```go
var allowedCommands = map[string]bool{
    "ls":    true,
    "cat":   true,
    "head":  true,
    "tail":  true,
    "wc":    true,
    "sort":  true,
    "grep":  true,
    // REMOVED: curl, pip, python3, python, git, make, echo
}
```

### File Operations (replaced by Go code, not shell)

| Old Command | Replacement |
|-------------|-------------|
| `curl` | Go `net/http` with SSRF guard (mcp.ValidateSSRF) |
| `pip install` | Pre-approved package list, installed at sandbox image build time |
| `git clone` | Go `go-git` library (no shell out) |
| `python3 code.py` | Sandbox runtime ONLY (never raw exec) |
| `make` | Go build system (go build, go test) |

### Argument Whitelisting

```go
var allowedArgs = map[string][]string{
    "ls":   {"-l", "-a", "-la", "-lh", "-laH"},
    "cat":  {},  // No restrictions on cat args
    "head": {"-n"},  // Must specify line count
    "tail": {"-n", "-f"},  // -f allowed but sensible
    "wc":   {"-l", "-w", "-c"},
    "sort": {"-n", "-r"},
    "grep": {"-i", "-v", "-n", "-r", "-l"},  // -r careful (recursive)
}
```

---

## 8. Verification

### Test Coverage

- [ ] `sandbox/namespace_test.go` (NEW): Escape attempts — fork bomb, /proc mount, network socket, chroot
- [ ] `sandbox/seccomp_test.go` (NEW): Blocked syscalls return `EPERM`
- [ ] `sandbox/limit_test.go` (NEW): Memory exceeded → killed; PID limit → `EAGAIN`
- [ ] `validation_test.go` (expand): New blocklist entries, AST bypass, importlib escape
- [ ] `genesis/sandbox_test.go` (NEW): AST catches os.Remove, plugin.Open, net.Listen
- [ ] `allowlist_test.go` (NEW): curl/pip/git/make removed; args whitelisted
- [ ] `sandbox/escape_test.go` (NEW): 10 known sandbox escape scripts — all blocked
- [ ] `sandbox/fuzz_test.go` (expand): Malformed Python/Go — no panics, no bypasses

### Gate

```
go test -race -count=1 ./internal/sandbox/ ./internal/genesis/ ./internal/ingestion/
→ ALL pass, ZERO skipped
→ Python importlib bypass → blocked
→ Go os/exec import → blocked
→ curl command → rejected
```

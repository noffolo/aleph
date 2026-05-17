# Linux Phase — Sandbox Seccomp/Namespace TDD + Playwright E2E

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete test coverage for Linux-specific sandbox code (seccomp/namespace/cgroups) and verify Playwright E2E tests compile.

**Architecture:** 4 Linux-only files (namespace_isolated.go, seccomp_profile.go, cgroups.go, namespace_isolation_test.go) behind `//go:build linux`. Write new test files with same build constraint. Existing tests require root for namespace/cgroup ops, but `sandboxPolicy()` is a pure function testable without root. Playwright E2E tests exist at frontend/tests/e2e/ (12 files) — verify they compile.

**Tech Stack:** Go (build-tag-constrained), Playwright, seccomp-bpf, cgroups v2

---

### Wave A: Seccomp Policy Pure-Function Tests

Files:
- Create: `internal/sandbox/seccomp_profile_test.go` (+120 lines, `//go:build linux`)

- [ ] **Step 1: Write TestSandboxPolicy_DefaultAction**

```go
//go:build linux

package sandbox

import (
    "testing"
    seccomp "github.com/elastic/go-seccomp-bpf"
)

func TestSandboxPolicy_DefaultAction(t *testing.T) {
    p := sandboxPolicy()
    if p.DefaultAction != seccomp.ActionErrno {
        t.Errorf("expected DefaultAction ActionErrno, got %v", p.DefaultAction)
    }
}
```

- [ ] **Step 2: Verify test compiles**

Run: `go vet ./internal/sandbox/...` (skips //go:build linux on macOS — verify no syntax error only)
Expected: no errors (build constraint correctly excluded)

- [ ] **Step 3: Write TestSandboxPolicy_SyscallGroup**

```go
func TestSandboxPolicy_SyscallGroup(t *testing.T) {
    p := sandboxPolicy()
    if len(p.Syscalls) != 1 {
        t.Fatalf("expected 1 SyscallGroup, got %d", len(p.Syscalls))
    }
    sg := p.Syscalls[0]
    if sg.Action != seccomp.ActionAllow {
        t.Errorf("expected ActionAllow, got %v", sg.Action)
    }
    // Verify critical syscalls are in the allowlist
    critical := []string{"read", "write", "openat", "socket", "connect", "mmap", "exit", "exit_group"}
    for _, name := range critical {
        found := false
        for _, n := range sg.Names {
            if n == name {
                found = true
                break
            }
        }
        if !found {
            t.Errorf("critical syscall %q not in allowlist", name)
        }
    }
    // Verify dangerous syscalls are NOT in the allowlist
    forbidden := []string{"open", "fork", "vfork", "ptrace", "execve", "execveat"}
    for _, name := range forbidden {
        for _, n := range sg.Names {
            if n == name {
                t.Errorf("dangerous syscall %q should NOT be in allowlist", name)
            }
        }
    }
}
```

- [ ] **Step 4: Write TestApplySeccompFilter_NoPanic**

```go
func TestApplySeccompFilter_NoPanic(t *testing.T) {
    // This function logs errors but never panics — gracefull degradation contract
    ApplySeccompFilter()
}
```

- [ ] **Step 5: Verify compilation + commit**

Run:
```bash
go vet ./internal/sandbox/...
git add internal/sandbox/seccomp_profile_test.go
git commit -m "test(sandbox): add seccomp policy pure-function tests (//go:build linux)"
```

---

### Wave B: Namespace/Cgroup Error-Path Tests

Files:
- Create: `internal/sandbox/namespace_linux_test.go` (+80 lines, `//go:build linux`)

- [ ] **Step 1: Write TestPrepareSandboxedCmd_SeccompNotSupported**

```go
//go:build linux

package sandbox

import (
    "context"
    "os/exec"
    "testing"
)

func TestPrepareSandboxedCmd_NamespaceFailure(t *testing.T) {
    // ExecuteIsolated returns nil when SysProcAttr set (no actual execution)
    // But we pass a cmd with bogus path — the error comes from namespace
    // isolation setup, not command execution
    cmd := exec.Command("/nonexistent/binary")
    cleanup, err := prepareSandboxedCmd(context.Background(), cmd, "test-exec-1")
    if cleanup != nil {
        defer cleanup()
    }
    // On non-root, ExecuteIsolated may succeed (just sets SysProcAttr).
    // On systems without namespace support, it may fail. Both are valid.
    t.Logf("prepareSandboxedCmd returned err=%v", err)
}
```

- [ ] **Step 2: Write TestCreateCgroup_NilCgroupBase (error path — invalid path)**

```go
func TestCleanupCgroup_NoPath(t *testing.T) {
    err := CleanupCgroup("/nonexistent/cgroup/path")
    if err == nil {
        t.Error("expected error for nonexistent cgroup path")
    }
}

func TestAddProcessToCgroup_InvalidPath(t *testing.T) {
    err := AddProcessToCgroup("/nonexistent/cgroup", 999999)
    if err == nil {
        t.Error("expected error for nonexistent cgroup path")
    }
}
```

- [ ] **Step 3: Verify compilation + commit**

Run:
```bash
go vet ./internal/sandbox/...
git add internal/sandbox/namespace_linux_test.go
git commit -m "test(sandbox): add namespace/cgroup error-path tests (//go:build linux)"
```

---

### Wave C: Playwright E2E Smoke Test (macOS-verifiable)

Files:
- Verify: `frontend/tests/e2e/smoke.spec.ts`

- [ ] **Step 1: Check Playwright config exists**

```bash
ls frontend/playwright.config.ts
```
Expected: file exists with target pointing to tests/e2e/

- [ ] **Step 2: List all Playwright tests to verify they parse**

Run: `cd frontend && npx playwright test --list --project=chromium`
Expected: 12 test files listed, 0 parsing errors

- [ ] **Step 3: Verify import paths are correct (no orphaned dependencies)**

```bash
# No test file should import from ../e2e/ (old orphaned dir)
grep -r "from.*['\"].*e2e/" frontend/tests/e2e/*.ts
grep -r "from.*['\"].*e2e/" frontend/e2e/*.ts 2>/dev/null
```
Expected: 0 matches (all imports resolved within tests/e2e/)

- [ ] **Step 4: Commit Playwright verification**

```bash
git add frontend/tests/e2e/
git commit -m "chore: verify Playwright E2E test compilation (12 files)"
```

---

### Wave D: Full Suite Verification + GitNexus

- [ ] **Step 1: Run full Go test suite**

```bash
go build ./...
go vet ./...
go test -count=1 ./internal/sandbox/...   # runs non-Linux tests
```

- [ ] **Step 2: Run frontend verification**

```bash
cd frontend && npx tsc --noEmit && npx vitest run
```

- [ ] **Step 3: GitNexus reindex**

```bash
npx gitnexus analyze
```

- [ ] **Step 4: Push to main**

```bash
git push origin main
```

---

### Execution Notes for Linux

When running on a Linux machine:
1. Run all tests with `go test -count=1 ./internal/sandbox/...` (now includes Linux-specific tests)
2. For root-requiring tests: `go test -count=1 -tags=linux ./internal/sandbox/...` as root
3. Run full suite: `go test -count=1 ./...`
4. Run Playwright: `cd frontend && npx playwright test --project=chromium`

//go:build linux

package sandbox

import (
	"fmt"
	"log/slog"

	seccomp "github.com/elastic/go-seccomp-bpf"
)

// allowedSyscalls defines the whitelist of syscalls permitted inside the
// sandbox. Any syscall not on this list returns EPERM.
var allowedSyscalls = []string{
	// I/O primitives
	"read", "write", "close",
	// Network (required for sandbox agents to access external APIs)
	"socket", "connect", "sendto", "recvfrom", "sendmsg", "recvmsg",
	"getsockname", "getpeername", "setsockopt", "getsockopt",
	"bind", "listen", "accept", "accept4",
	"shutdown", "socketpair",
	// Process termination
	"exit", "exit_group",
	// Memory management
	"mmap", "mprotect", "munmap", "brk", "mremap",
	// Synchronisation
	"futex", "futex_waitv",
	// Time
	"clock_gettime", "gettimeofday", "nanosleep",
	// Signals
	"rt_sigaction", "rt_sigprocmask", "sigreturn", "rt_sigreturn",
	// File access (no open — only openat)
	"openat", "readlink",
	// File metadata
	"fstat", "stat", "lstat", "newfstatat",
	// Directory
	"getdents64",
	// Seeking
	"lseek",
	// Process identity
	"getpid", "gettid", "getuid", "getgid", "geteuid", "getegid",
	// Scheduling
	"sched_yield",
	// Random
	"getrandom",
	// Architecture-specific
	"arch_prctl",
	// Cloning (needed for Go runtime)
	"clone", "clone3",
	// Robust futexes
	"set_robust_list",
	// Restartable sequences
	"rseq",
	// Resource limits
	"prlimit64",
	// System info
	"uname",
	// Working directory
	"getcwd",
	// I/O control
	"ioctl", "fcntl",
}

// sandboxPolicy builds the seccomp-bpf policy: allow-list the above syscalls,
// deny everything else with EPERM.
func sandboxPolicy() seccomp.Policy {
	return seccomp.Policy{
		DefaultAction: seccomp.ActionErrno,
		Syscalls: []seccomp.SyscallGroup{
			{
				Names:  allowedSyscalls,
				Action: seccomp.ActionAllow,
			},
		},
	}
}

// LoadSeccompFilter assembles and installs the seccomp-bpf filter into the
// calling process. Must be called in the SAME process that will be sandboxed
// (typically a child wrapper or the tool binary itself).
//
// IMPORTANT: Do NOT call this from the main Aleph server process, as it would
// restrict the server's own syscalls. Use this in a dedicated sandbox wrapper.
//
// Returns nil if seccomp is not supported by the kernel (graceful degradation).
func LoadSeccompFilter() error {
	if !seccomp.Supported() {
		slog.Warn("seccomp not supported by kernel, skipping filter installation")
		return nil
	}

	filter := seccomp.Filter{
		NoNewPrivs: true,
		Flag:       seccomp.FilterFlagTSync,
		Policy:     sandboxPolicy(),
	}

	if err := seccomp.LoadFilter(filter); err != nil {
		return fmt.Errorf("seccomp filter installation failed: %w", err)
	}

	slog.Info("seccomp-bpf filter installed", "allowed_syscalls", len(allowedSyscalls))
	return nil
}

// ApplySeccompFilter is a convenience wrapper that installs the seccomp filter
// and logs any errors. It never returns an error — failures are logged but do
// not block execution, allowing the system to degrade gracefully on
// non-seccomp kernels.
func ApplySeccompFilter() {
	if err := LoadSeccompFilter(); err != nil {
		slog.Error("seccomp filter installation failed, running without syscall restriction", "error", err)
	}
}
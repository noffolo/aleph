package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var shellMetaRE = regexp.MustCompile(`[;&|` + "`" + `$(){}\<\>]`)

// maxOutputBytes is the maximum output size from an allowed command.
// 10MB limit prevents memory exhaustion from runaway output.
const maxOutputBytes = 10 * 1024 * 1024

type CommandAllowlist struct {
	allowedCommands    map[string]bool
	allowedCommandArgs map[string][]string
	allowedArgSet      map[string]map[string]bool
	blockedFlags       map[string]bool
	timeout            time.Duration
}

// NewCommandAllowlist creates a CommandAllowlist with strict defaults.
//
// Only read-only file inspection commands are allowed. All execution,
// network, and package-management commands (python, pip, git, make,
// curl, echo) are removed — they must run through the sandbox runtime
// (ExecSandbox or ContainerSandbox).
func NewCommandAllowlist() *CommandAllowlist {
	allowed := map[string]bool{
		"ls":   true,
		"cat":  true,
		"head": true,
		"tail": true,
		"wc":   true,
		"sort": true,
		"grep": true,
	}

	allowedArgs := map[string][]string{
		"ls":   {"-l", "-a", "-la", "-lh"},
		"head": {"-n"},
		"tail": {"-n"},
		"wc":   {"-l", "-w", "-c"},
		"sort": {"-n", "-r"},
		"grep": {"-i", "-v", "-n", "-r", "-rn", "-l"},
	}

	// Build lookup sets for O(1) arg validation
	argSet := make(map[string]map[string]bool, len(allowedArgs))
	for cmd, args := range allowedArgs {
		set := make(map[string]bool, len(args)+1)
		for _, a := range args {
			set[a] = true
		}
		argSet[cmd] = set
	}

	return &CommandAllowlist{
		allowedCommands:    allowed,
		allowedCommandArgs: allowedArgs,
		allowedArgSet:      argSet,
		blockedFlags: map[string]bool{
			"--pty":         true,
			"--interactive": true,
			"--tty":         true,
			"-i":            true,
			"-t":            true,
		},
		timeout: 30 * time.Second,
	}
}

// Validate checks that the command is on the allowlist, all arguments are
// in the per-command whitelist (if defined), no blocked flags are present,
// and no shell metacharacters are detected.
//
// Validation order: 1) command allowlist, 2) blocked flags, 3) arg whitelist,
// 4) shell metacharacters. Blocked flags are checked before arg whitelist so
// that flags like --pty are always rejected regardless of command context.
func (a *CommandAllowlist) Validate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided")
	}

	// 0. Check all args for shell metacharacters FIRST (prevent injection before any other check)
	for _, arg := range args {
		if shellMetaRE.MatchString(arg) {
			return fmt.Errorf("shell metacharacter detected in argument: %s", arg)
		}
	}

	cmd := args[0]

	// 1. Command must be allowed
	if !a.allowedCommands[cmd] {
		return fmt.Errorf("command not allowed: %s", cmd)
	}

	// 2. Check blocked flags (rejected unless in this command's arg allowlist)
	allowedSet, hasAllowedSet := a.allowedArgSet[cmd]
	for _, arg := range args[1:] {
		if a.blockedFlags[arg] {
			if hasAllowedSet && allowedSet[arg] {
				continue
			}
			return fmt.Errorf("blocked flag: %s", arg)
		}
	}

	// 3. If per-command arg whitelist exists, validate each arg
	if allowedSet, ok := a.allowedArgSet[cmd]; ok {
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "-") && !allowedSet[arg] {
				return fmt.Errorf("argument not allowed for %s: %s", cmd, arg)
			}
		}
	}

	return nil
}

func (a *CommandAllowlist) ExecCommandContext(ctx context.Context, args ...string) ([]byte, error) {
	cmd := args[0]
	cmdArgs := args[1:]

	if err := a.Validate(args); err != nil {
		return nil, err
	}

	execCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	output, err := exec.CommandContext(execCtx, cmd, cmdArgs...).CombinedOutput()

	// Enforce output size limit (10MB)
	if len(output) > maxOutputBytes {
		return nil, fmt.Errorf("command output exceeds %d bytes (got %d)", maxOutputBytes, len(output))
	}

	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}
	return output, nil
}

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

type CommandAllowlist struct {
	allowedCommands map[string]bool
	blockedFlags    map[string]bool
	timeout         time.Duration
}

func NewCommandAllowlist() *CommandAllowlist {
	return &CommandAllowlist{
		allowedCommands: map[string]bool{
			"python3": true,
			"python":  true,
			"pip":     true,
			"git":     true,
			"make":    true,
			"curl":    true,
			"ls":      true,
			"cat":     true,
			"echo":    true,
			"head":    true,
			"tail":    true,
			"wc":      true,
			"sort":    true,
			"grep":    true,
		},
		blockedFlags: map[string]bool{
			"--pty":         true,
			"-i":            true,
			"--interactive": true,
			"--tty":         true,
			"-t":            true,
		},
		timeout: 30 * time.Second,
	}
}

func (a *CommandAllowlist) Validate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided")
	}
	if !a.allowedCommands[args[0]] {
		return fmt.Errorf("command not allowed: %s", args[0])
	}
	for _, arg := range args[1:] {
		if a.blockedFlags[arg] {
			return fmt.Errorf("blocked flag: %s", arg)
		}
	}
	joined := strings.Join(args, " ")
	if shellMetaRE.MatchString(joined) {
		return fmt.Errorf("shell metacharacters detected")
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
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}
	return output, nil
}

package sandbox

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandAllowlist_AllowedCommands(t *testing.T) {
	a := NewCommandAllowlist()

	// Only read-only inspection commands should be allowed
	allowed := []string{"ls", "cat", "head", "tail", "wc", "sort", "grep"}
	for _, cmd := range allowed {
		assert.True(t, a.allowedCommands[cmd], "expected %s to be allowed", cmd)
	}

	// Dangerous commands must NOT be on the allowlist
	blocked := []string{"python3", "python", "pip", "git", "make", "curl", "echo", "sh", "bash", "rm", "wget", "nc"}
	for _, cmd := range blocked {
		assert.False(t, a.allowedCommands[cmd], "expected %s to be blocked", cmd)
	}
}

func TestCommandAllowlist_Validate_CommandBlocked(t *testing.T) {
	a := NewCommandAllowlist()
	blockedCommands := []string{"python3", "python", "pip", "git", "make", "curl", "echo", "sh", "bash", "rm", "wget", "nc", "docker", "sudo"}

	for _, cmd := range blockedCommands {
		err := a.Validate([]string{cmd})
		assert.Error(t, err, "expected error for blocked command: %s", cmd)
		assert.Contains(t, err.Error(), "not allowed", "error for %s should say 'not allowed'", cmd)
	}
}

func TestCommandAllowlist_Validate_AllowedCommands(t *testing.T) {
	a := NewCommandAllowlist()
	tests := []struct {
		name string
		args []string
	}{
		{"ls", []string{"ls"}},
		{"ls -l", []string{"ls", "-l"}},
		{"ls -la", []string{"ls", "-la"}},
		{"cat file", []string{"cat", "file.txt"}},
		{"head file", []string{"head", "file.txt"}},
		{"head -n 10", []string{"head", "-n", "10", "file.txt"}},
		{"tail file", []string{"tail", "file.txt"}},
		{"tail -n 5", []string{"tail", "-n", "5", "file.txt"}},
		{"wc -l file", []string{"wc", "-l", "file.txt"}},
		{"sort file", []string{"sort", "file.txt"}},
		{"sort -n file", []string{"sort", "-n", "file.txt"}},
		{"grep pattern file", []string{"grep", "pattern", "file.txt"}},
		{"grep -i pattern", []string{"grep", "-i", "pattern", "file.txt"}},
		{"grep -rn pattern dir", []string{"grep", "-rn", "pattern", "dir"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := a.Validate(tc.args)
			assert.NoError(t, err, "expected %v to be allowed", tc.args)
		})
	}
}

func TestCommandAllowlist_Validate_ArgWhitelist(t *testing.T) {
	a := NewCommandAllowlist()

	// Allowed args
	allowedArgs := [][]string{
		{"ls", "-l"},
		{"ls", "-la"},
		{"ls", "-lh"},
		{"head", "-n", "10"},
		{"tail", "-n", "5"},
		{"wc", "-l"},
		{"wc", "-w"},
		{"wc", "-c"},
		{"sort", "-n"},
		{"sort", "-r"},
		{"grep", "-i"},
		{"grep", "-v"},
		{"grep", "-rn"},
	}
	for _, args := range allowedArgs {
		err := a.Validate(args)
		assert.NoError(t, err, "expected args %v to be allowed", args)
	}

	// Blocked args for ls
	blockedArgs := [][]string{
		{"ls", "-R"},     // recursive listing
		{"ls", "--exec"}, // dangerous flag
		{"grep", "-e"},   // not in allowlist
		{"sort", "-k"},   // not in allowlist
		{"wc", "-L"},     // not in allowlist
	}
	for _, args := range blockedArgs {
		err := a.Validate(args)
		assert.Error(t, err, "expected args %v to be blocked", args)
	}
}

func TestCommandAllowlist_Validate_BlockedFlags(t *testing.T) {
	a := NewCommandAllowlist()
	blockedFlags := []string{"--pty", "-i", "--interactive", "--tty", "-t"}

	for _, flag := range blockedFlags {
		err := a.Validate([]string{"ls", flag})
		assert.Error(t, err, "expected flag %s to be blocked", flag)
		assert.Contains(t, err.Error(), "blocked flag", "error for %s should mention blocked flag", flag)
	}
}

func TestCommandAllowlist_Validate_ShellMetacharacters(t *testing.T) {
	a := NewCommandAllowlist()
	dangerousInputs := []string{
		"ls; rm -rf /",
		"cat `cat /etc/passwd`",
		"ls $(whoami)",
		"grep {a,b}",
		"ls > /etc/passwd",
		"cat < /etc/shadow",
		"ls | cat /etc/passwd",
		"ls & rm -rf /",
	}
	for _, input := range dangerousInputs {
		args := strings.Split(input, " ")
		err := a.Validate(args)
		assert.Error(t, err, "expected shell metacharacters to be blocked: %s", input)
		assert.Contains(t, err.Error(), "shell metacharacter", "error should mention shell metacharacters for: %s", input)
	}
}

func TestCommandAllowlist_Validate_EmptyCommand(t *testing.T) {
	a := NewCommandAllowlist()
	err := a.Validate([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no command provided")
}

func TestCommandAllowlist_Validate_NoArgs(t *testing.T) {
	a := NewCommandAllowlist()
	err := a.Validate([]string{"ls"})
	assert.NoError(t, err)
}

func TestCommandAllowlist_Validate_NonFlagArgsAllowed(t *testing.T) {
	a := NewCommandAllowlist()
	// Non-flag args (file paths, patterns) should be allowed since they don't start with "-"
	err := a.Validate([]string{"grep", "pattern", "file.txt"})
	assert.NoError(t, err)

	err = a.Validate([]string{"cat", "/etc/hosts"})
	assert.NoError(t, err)

	err = a.Validate([]string{"wc", "-l", "file.txt"})
	assert.NoError(t, err)
}

func TestCommandAllowlist_ExecCommandContext_BlockedCommand(t *testing.T) {
	a := NewCommandAllowlist()
	ctx := context.Background()

	_, err := a.ExecCommandContext(ctx, "curl", "http://evil.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestCommandAllowlist_ExecCommandContext_Timeout(t *testing.T) {
	a := NewCommandAllowlist()
	ctx := context.Background()

	// ls should succeed with timeout
	output, err := a.ExecCommandContext(ctx, "ls", "/tmp")
	if err != nil {
		// May fail in CI without /tmp, but should not be a validation error
		assert.NotContains(t, err.Error(), "not allowed")
		assert.NotContains(t, err.Error(), "shell metacharacter")
	} else {
		assert.NotNil(t, output)
	}
}

func TestCommandAllowlist_DefaultTimeout(t *testing.T) {
	a := NewCommandAllowlist()
	assert.Equal(t, 30*time.Second, a.timeout)
}

func TestCommandAllowlist_OutputSizeLimit(t *testing.T) {
	// Verify the constant is set correctly
	assert.Equal(t, int64(10*1024*1024), int64(maxOutputBytes))
}

func TestCommandAllowlist_ExecCommandContext_CancelledContext(t *testing.T) {
	a := NewCommandAllowlist()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.ExecCommandContext(ctx, "ls")
	require.Error(t, err)
}

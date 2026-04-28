package genesis

import (
	"context"
	"strings"
	"time"
)

type Sandbox struct {
	timeout time.Duration
}

func NewSandbox(timeout time.Duration) *Sandbox {
	return &Sandbox{
		timeout: timeout,
	}
}

func (s *Sandbox) Validate(ctx context.Context, suggestion Suggestion) (bool, error) {
	if suggestion.Code == "" {
		return true, nil
	}
	return s.validateCode(ctx, suggestion.Code)
}

func (s *Sandbox) validateCode(ctx context.Context, code string) (bool, error) {
	dangerous := []string{
		"os/exec", "syscall", "unsafe", "reflect",
		"os.Remove", "os.RemoveAll", "os.Chmod",
		"net.Listen", "net.Dial",
	}
	for _, pattern := range dangerous {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		if strings.Contains(code, pattern) {
			return false, nil
		}
	}
	return true, nil
}

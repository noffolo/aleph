package genesis

import (
	"context"
	"testing"
	"time"
)

func TestGenesisEngine_Reject(t *testing.T) {
	ctx := context.Background()
	veto := NewVetoRegistry(ctx, 1*time.Hour)
	defer veto.Shutdown()
	engine := NewGenesisEngine(NewSuggester(), NewSandbox(5*time.Second), veto)

	s := Suggestion{ID: "r1", Name: "reject-me", Status: "pending"}
	veto.Register(s)

	err := engine.Reject(ctx, "r1")
	if err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}

	pending, _ := veto.ListPending(ctx)
	if len(pending) != 0 {
		t.Error("expected no pending after reject")
	}
}

func TestGenesisEngine_ListPending(t *testing.T) {
	ctx := context.Background()
	veto := NewVetoRegistry(ctx, 1*time.Hour)
	defer veto.Shutdown()
	engine := NewGenesisEngine(NewSuggester(), NewSandbox(5*time.Second), veto)

	veto.Register(Suggestion{ID: "p1", Name: "pending1", Status: "pending"})
	veto.Register(Suggestion{ID: "p2", Name: "pending2", Status: "pending"})

	pending, err := engine.ListPending(ctx)
	if err != nil {
		t.Fatalf("ListPending returned error: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("expected 2 pending, got %d", len(pending))
	}
}

func TestSandbox_checkDangerousPatternsFallback(t *testing.T) {
	s := NewSandbox(5 * time.Second)

	t.Run("os/exec import blocked", func(t *testing.T) {
		blocked := s.checkDangerousPatternsFallback(`import "os/exec"`)
		if len(blocked) == 0 {
			t.Error("expected os/exec to be blocked by fallback")
		}
	})

	t.Run("net/http matches via substring", func(t *testing.T) {
		blocked := s.checkDangerousPatternsFallback(`import "net/http"`)
		if len(blocked) == 0 {
			t.Error("expected substring match for net in net/http")
		}
	})

	t.Run("os.Remove call blocked", func(t *testing.T) {
		blocked := s.checkDangerousPatternsFallback("os.Remove(\"/etc/passwd\")")
		if len(blocked) == 0 {
			t.Error("expected os.Remove blocked by fallback")
		}
	})

	t.Run("net.Dial call blocked", func(t *testing.T) {
		blocked := s.checkDangerousPatternsFallback("net.Dial(\"tcp\", \"evil.com:666\")")
		if len(blocked) == 0 {
			t.Error("expected net.Dial blocked by fallback")
		}
	})
}

func Test_firstNLines(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"shorter than n", "a\nb", 5, "a\nb"},
		{"exactly n lines", "a\nb\nc", 3, "a\nb\nc"},
		{"more than n lines", "a\nb\nc\nd", 2, "a\nb..."},
		{"single line less than n", "hello", 3, "hello"},
		{"empty string", "", 1, ""},
		{"n is zero", "a\nb", 0, "..."},
		{"single line with n=1", "hello", 1, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstNLines(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("firstNLines(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

func Test_clampRisk(t *testing.T) {
	tests := []struct {
		name string
		v    float64
		want float64
	}{
		{"within range", 0.5, 0.5},
		{"zero", 0.0, 0.0},
		{"one", 1.0, 1.0},
		{"above one", 1.5, 1.0},
		{"way above one", 99.0, 1.0},
		{"negative", -0.5, -0.5},
		{"way negative", -99.0, -99.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampRisk(tt.v)
			if got != tt.want {
				t.Errorf("clampRisk(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

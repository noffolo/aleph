package decision

import (
	"context"
	"testing"
)

func TestDefaultAdmitter_Admit_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		results     []*ActResult
		maxAttempts int
		wantAdmit   bool
	}{
		{
			name:        "no results — continue",
			results:     []*ActResult{},
			maxAttempts: 5,
			wantAdmit:   false,
		},
		{
			name: "maxAttempts is zero — treat as unlimited, continue",
			results: []*ActResult{
				{Output: "ok"},
			},
			maxAttempts: 0,
			wantAdmit:   false,
		},
		{
			name: "maxAttempts is negative — continue",
			results: []*ActResult{
				{Output: "ok"},
			},
			maxAttempts: -1,
			wantAdmit:   false,
		},
		{
			name: "exactly at maxAttempts — admit",
			results: []*ActResult{
				{Output: "r1"},
				{Output: "r2"},
				{Output: "r3"},
			},
			maxAttempts: 3,
			wantAdmit:   true,
		},
		{
			name: "over maxAttempts — admit",
			results: []*ActResult{
				{Output: "r1"},
				{Output: "r2"},
				{Output: "r3"},
				{Output: "r4"},
			},
			maxAttempts: 3,
			wantAdmit:   true,
		},
		{
			name: "one below maxAttempts — continue",
			results: []*ActResult{
				{Output: "r1"},
				{Output: "r2"},
			},
			maxAttempts: 3,
			wantAdmit:   false,
		},
		{
			name: "last result has error — admit",
			results: []*ActResult{
				{Output: "ok"},
				{Error: "something broke"},
			},
			maxAttempts: 5,
			wantAdmit:   true,
		},
		{
			name: "error in middle result, last is ok — continue",
			results: []*ActResult{
				{Output: "ok"},
				{Error: "transient error"},
				{Output: "recovered"},
			},
			maxAttempts: 5,
			wantAdmit:   false,
		},
		{
			name:        "single successful result — continue",
			results:     []*ActResult{{Output: "ok"}},
			maxAttempts: 5,
			wantAdmit:   false,
		},
		{
			name:        "maxAttempts=1 with one result — admit",
			results:     []*ActResult{{Output: "done"}},
			maxAttempts: 1,
			wantAdmit:   true,
		},
		{
			name:        "maxAttempts=1 with no results — continue",
			results:     []*ActResult{},
			maxAttempts: 1,
			wantAdmit:   false,
		},
		{
			name: "maxAttempts=1 with error result — admit",
			results: []*ActResult{
				{Error: "fatal"},
			},
			maxAttempts: 1,
			wantAdmit:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adm := NewDefaultAdmitter()
			got, err := adm.Admit(context.Background(), tt.results, tt.maxAttempts)
			if err != nil {
				t.Fatalf("Admit returned error: %v", err)
			}
			if got != tt.wantAdmit {
				t.Errorf("Admit = %v, want %v", got, tt.wantAdmit)
			}
		})
	}
}

func TestDefaultAdmitter_NewDefaultAdmitter(t *testing.T) {
	adm := NewDefaultAdmitter()
	if adm == nil {
		t.Fatal("expected non-nil DefaultAdmitter")
	}
}

func TestDefaultAdmitter_ImplementsInterface(t *testing.T) {
	var a Admitter = NewDefaultAdmitter()
	if a == nil {
		t.Fatal("DefaultAdmitter must satisfy Admitter interface")
	}
}

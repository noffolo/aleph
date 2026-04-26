package cursor

import "testing"

func TestEncodeDecodePosition(t *testing.T) {
	tests := []struct {
		name   string
		offset int
	}{
		{"zero", 0},
		{"small", 10},
		{"large", 99999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodePosition(tt.offset)
			if encoded == "" {
				t.Fatal("expected non-empty cursor")
			}
			decoded := DecodePosition(encoded)
			if decoded != tt.offset {
				t.Fatalf("DecodePosition() = %d, want %d", decoded, tt.offset)
			}
		})
	}
}

func TestEncodeDecodeID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"standard", "agent-1234"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeID(tt.id)
			if encoded == "" {
				t.Fatal("expected non-empty cursor")
			}
			decoded := DecodeID(encoded)
			if decoded != tt.id {
				t.Fatalf("DecodeID() = %q, want %q", decoded, tt.id)
			}
		})
	}
}

func TestDecodeEmptyCursor(t *testing.T) {
	if n := DecodePosition(""); n != 0 {
		t.Fatalf("DecodePosition(\"\") = %d, want 0", n)
	}
	if s := DecodeID(""); s != "" {
		t.Fatalf("DecodeID(\"\") = %q, want empty", s)
	}
}

func TestDecodeInvalidCursor(t *testing.T) {
	if n := DecodePosition("not-base64!!"); n != 0 {
		t.Fatalf("expected 0 for invalid cursor, got %d", n)
	}
	if s := DecodeID("!!!invalid!!!"); s != "" {
		t.Fatalf("expected empty for invalid cursor, got %q", s)
	}
}

func TestCrossTypeDecode(t *testing.T) {
	posCur := EncodePosition(42)
	if id := DecodeID(posCur); id != "" {
		t.Fatalf("DecodeID on position cursor should return empty, got %q", id)
	}
	idCur := EncodeID("agent-1")
	if pos := DecodePosition(idCur); pos != 0 {
		t.Fatalf("DecodePosition on id cursor should return 0, got %d", pos)
	}
}

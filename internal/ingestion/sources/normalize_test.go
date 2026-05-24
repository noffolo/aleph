package sources

import (
	"testing"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MAGI", "Magi"},
		{"RICCARDO", "Riccardo"},
		{"DE LUCIA", "De Lucia"},
		{"D'AGOSTO", "D'Agosto"},
		{"D'AMICO", "D'Amico"},
		{"MACCHIA", "Macchia"},
		{"BLO'", "Blo'"},
		{"CALABRO'", "Calabro'"},
		{"NICCOLO'", "Niccolo'"},
		{"STEFANO  MARIA", "Stefano Maria"},
		{"SALVATORE ", "Salvatore"},
		{" ALESSANDRO", "Alessandro"},
		{"DE ANGELIS", "De Angelis"},
		{"DELL'ANNA", "Dell'Anna"},
		{"DE ROSA", "De Rosa"},
		{"LA ROSA", "La Rosa"},
		{"DI GIOVANNI", "Di Giovanni"},
		{"DEL MONTE", "Del Monte"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeName(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeFullName(t *testing.T) {
	tests := []struct {
		cognome  string
		nome     string
		expected string
	}{
		{"MAGI", "RICCARDO", "Magi Riccardo"},
		{"DE LUCIA", "MARIA", "De Lucia Maria"},
		{"D'AGOSTO", "LUIGI", "D'Agosto Luigi"},
	}

	for _, tt := range tests {
		t.Run(tt.cognome+"_"+tt.nome, func(t *testing.T) {
			got := NormalizeFullName(tt.cognome, tt.nome)
			if got != tt.expected {
				t.Errorf("NormalizeFullName(%q, %q) = %q, want %q", tt.cognome, tt.nome, got, tt.expected)
			}
		})
	}
}

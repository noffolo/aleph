package handler

import (
	"net/http"
	"testing"
)

func TestParsePagePagination_Defaults(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test", nil)
	params := ParsePagePagination(r)
	if params.Page != 1 {
		t.Fatalf("expected default Page=1, got %d", params.Page)
	}
	if params.PerPage != 50 {
		t.Fatalf("expected default PerPage=50, got %d", params.PerPage)
	}
}

func TestParsePagePagination_CustomPage(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test?page=3", nil)
	params := ParsePagePagination(r)
	if params.Page != 3 {
		t.Fatalf("expected Page=3, got %d", params.Page)
	}
	if params.PerPage != 50 {
		t.Fatalf("expected default PerPage=50, got %d", params.PerPage)
	}
}

func TestParsePagePagination_CustomPerPage(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test?per_page=20&page=2", nil)
	params := ParsePagePagination(r)
	if params.Page != 2 {
		t.Fatalf("expected Page=2, got %d", params.Page)
	}
	if params.PerPage != 20 {
		t.Fatalf("expected PerPage=20, got %d", params.PerPage)
	}
}

func TestParsePagePagination_PerPageCappedAt100(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test?per_page=200", nil)
	params := ParsePagePagination(r)
	if params.PerPage != 100 {
		t.Fatalf("expected PerPage capped at 100, got %d", params.PerPage)
	}
}

func TestParsePagePagination_ZeroPageIgnored(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test?page=0", nil)
	params := ParsePagePagination(r)
	if params.Page != 1 {
		t.Fatalf("expected page=0 to fallback to 1, got %d", params.Page)
	}
}

func TestParsePagePagination_NegativePageIgnored(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test?page=-5", nil)
	params := ParsePagePagination(r)
	if params.Page != 1 {
		t.Fatalf("expected negative page to fallback to 1, got %d", params.Page)
	}
}

func TestParsePagePagination_NegativePerPageIgnored(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test?per_page=-10", nil)
	params := ParsePagePagination(r)
	if params.PerPage != 50 {
		t.Fatalf("expected negative per_page to fallback to 50, got %d", params.PerPage)
	}
}

func TestParsePagePagination_NonNumericIgnored(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test?page=abc&per_page=xyz", nil)
	params := ParsePagePagination(r)
	if params.Page != 1 {
		t.Fatalf("expected non-numeric page to fallback to 1, got %d", params.Page)
	}
	if params.PerPage != 50 {
		t.Fatalf("expected non-numeric per_page to fallback to 50, got %d", params.PerPage)
	}
}

func TestPagePaginationParams_Offset(t *testing.T) {
	tests := []struct {
		name   string
		page   int
		perPage int
		want   int
	}{
		{"page 1", 1, 50, 0},
		{"page 2", 2, 50, 50},
		{"page 3 perPage 20", 3, 20, 40},
		{"page 1 perPage 100", 1, 100, 0},
		{"page 5 perPage 10", 5, 10, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PagePaginationParams{Page: tt.page, PerPage: tt.perPage}
			if got := p.Offset(); got != tt.want {
				t.Errorf("Offset() = %d, want %d", got, tt.want)
			}
		})
	}
}

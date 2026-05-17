package sources

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSheetConfig_Defaults(t *testing.T) {
	cfg := SheetConfig{
		SpreadsheetID: "1AbCdEfGhIjKlMnOpQrStUvWxYz",
		Range:         "Sheet1!A:Z",
		SheetName:     "Sheet1",
	}
	assert.Equal(t, "1AbCdEfGhIjKlMnOpQrStUvWxYz", cfg.SpreadsheetID)
	assert.Equal(t, "Sheet1!A:Z", cfg.Range)
	assert.Equal(t, "Sheet1", cfg.SheetName)
}

func TestParseSheetIDFromURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantID  string
		wantErr bool
	}{
		{
			name:   "standard URL",
			url:    "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz/edit",
			wantID: "1AbCdEfGhIjKlMnOpQrStUvWxYz",
		},
		{
			name:   "with trailing slash",
			url:    "https://docs.google.com/spreadsheets/d/12345/",
			wantID: "12345",
		},
		{
			name:   "with gid parameter",
			url:    "https://docs.google.com/spreadsheets/d/abc123/edit#gid=0",
			wantID: "abc123",
		},
		{
			name:   "with edit query params",
			url:    "https://docs.google.com/spreadsheets/d/XYZ789/edit?usp=sharing",
			wantID: "XYZ789",
		},
		{
			name:    "d segment present but no id follows",
			url:     "https://docs.google.com/spreadsheets/d/",
			wantErr: true,
		},
		{
			name:    "not a sheets URL",
			url:     "https://example.com/data.csv",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "://invalid",
			wantErr: true,
		},
		{
			name:    "double slash after d",
			url:     "https://docs.google.com/spreadsheets/d//edit",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseSheetIDFromURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestCheckSheetsHTTPError_AllCodes(t *testing.T) {
	tests := []struct {
		statusCode int
		wantErr    bool
	}{
		{http.StatusOK, false},
		{http.StatusNotModified, false},
		{http.StatusBadRequest, true},
		{http.StatusUnauthorized, true},
		{http.StatusForbidden, true},
		{http.StatusNotFound, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
	}
	for _, tt := range tests {
		err := checkSheetsHTTPError(tt.statusCode, []byte(`{"error":"test"}`))
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestCheckSheetsHTTPError_TruncatesLongBody(t *testing.T) {
	longBody := make([]byte, 1000)
	for i := range longBody {
		longBody[i] = 'x'
	}
	err := checkSheetsHTTPError(http.StatusForbidden, longBody)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 403")
}

func TestNewSheetsIngester_Variants(t *testing.T) {
	t.Run("without API key", func(t *testing.T) {
		s := NewSheetsIngester("")
		assert.NotNil(t, s)
		assert.NotNil(t, s.client)
		assert.Equal(t, "", s.apiKey)
	})
	t.Run("with API key", func(t *testing.T) {
		s := NewSheetsIngester("test-api-key")
		assert.NotNil(t, s)
		assert.NotNil(t, s.client)
		assert.Equal(t, "test-api-key", s.apiKey)
	})
}

func TestSheetsAPIHost_Constant(t *testing.T) {
	assert.Equal(t, "https://sheets.googleapis.com/v4/spreadsheets", sheetsAPIHost)
}

func TestSheetConfig_ZeroValue(t *testing.T) {
	cfg := SheetConfig{}
	assert.Equal(t, "", cfg.SpreadsheetID)
	assert.Equal(t, "", cfg.Range)
	assert.Equal(t, "", cfg.SheetName)
}

func TestSheetConfig_EmptyRange(t *testing.T) {
	cfg := SheetConfig{
		SpreadsheetID: "id",
		Range:         "",
		SheetName:     "default",
	}
	assert.Equal(t, "", cfg.Range)
	assert.Equal(t, "id", cfg.SpreadsheetID)
}

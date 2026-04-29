package mcp

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ff3300/aleph-v2/internal/ssrf"
)

// ValidateSSRF checks a URL against SSRF protection rules.
// It delegates to the consolidated ssrf package.
func ValidateSSRF(rawURL string) error {
	return ssrf.ValidateURL(rawURL)
}

// ParseMCPURI parses an mcp:// URI into components.
// Example: mcp://localhost:8080/tools -> (mcp, localhost, 8080, /tools)
func ParseMCPURI(uri string) (scheme, host, port, path string, err error) {
	if !strings.HasPrefix(uri, "mcp://") {
		return "", "", "", "", fmt.Errorf("invalid MCP URI: must start with mcp://, got %q", uri)
	}

	// Convert mcp:// to http:// for URL parsing
	httpURL := "http://" + strings.TrimPrefix(uri, "mcp://")
	u, err := url.Parse(httpURL)
	if err != nil {
		return "", "", "", "", fmt.Errorf("invalid MCP URI format: %w", err)
	}

	scheme = "mcp"
	host = u.Hostname()

	port = u.Port()
	if port == "" {
		port = "8080" // default MCP port
	}

	path = u.Path
	if path == "" {
		path = "/"
	}

	return scheme, host, port, path, nil
}

// ValidatePrivateRanges remains for backward compatibility.
// The ssrf package now handles all CIDR initialization.
func ValidatePrivateRanges() error {
	return nil
}

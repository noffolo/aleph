package mcp

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateSSRF checks a URL against SSRF protection rules.
// It blocks private/internal IPs and disallowed schemes.
func ValidateSSRF(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("disallowed scheme %q: only http and https are permitted", u.Scheme)
	}

	// Resolve hostname and check for private IPs
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname in URL")
	}

	// Block obvious private/internal hostnames
	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" {
		return fmt.Errorf("localhost addresses are not permitted")
	}
	if strings.HasSuffix(lowerHost, ".local") || strings.HasSuffix(lowerHost, ".internal") {
		return fmt.Errorf("internal DNS names are not permitted")
	}

	// Resolve IP and check for private ranges
	ips, err := net.LookupIP(host)
	if err != nil {
		// If we can't resolve, block it — fails closed
		return fmt.Errorf("cannot resolve hostname %q: %w", host, err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("private IP address %s is not permitted", ip)
		}
	}

	return nil
}

// isPrivateIP checks if an IP is in a private/reserved range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("100.64.0.0/10")},   // Carrier-grade NAT
		{mustParseCIDR("169.254.0.0/16")},   // Link-local
		{mustParseCIDR("127.0.0.0/8")},      // Loopback
		{mustParseCIDR("::1/128")},           // IPv6 loopback
		{mustParseCIDR("fc00::/7")},          // IPv6 unique local
		{mustParseCIDR("fe80::/10")},          // IPv6 link-local
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR %q: %v", s, err))
	}
	return network
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
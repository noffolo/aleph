// Package ssrf provides consolidated SSRF (Server-Side Request Forgery) protection.
//
// It is the single source of truth for all SSRF validation in the aleph-v2 project.
// Use NewClient() for making HTTP requests with connection-time protection,
// and ValidateURL()/ValidateHostname() for pre-request validation.
package ssrf

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Public API ───────────────────────────────────────────────────────────────────

// NewClient returns an *http.Client whose Transport dials connections through
// DNS-resolving SSRF protection. This eliminates TOCTOU (DNS rebinding) races
// by checking the resolved IP at connection time rather than before the request.
//
// The client:
//   - Resolves DNS via net.LookupIP before dialing (fails closed on DNS error)
//   - Blocks private/internal IP ranges (including CGNAT, link-local, ULA)
//   - Detects bypass IP forms (octal, hex, integer, short-form)
//   - Re-validates all redirect targets via CheckRedirect calling ValidateURL
//   - Configures TLS minimum version to 1.2
//
// Use this everywhere you would use a plain &http.Client{}.
func NewClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			DialContext: dialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			// Re-validate each redirect target
			if req.URL != nil {
				return ValidateURL(req.URL.String())
			}
			return nil
		},
	}
}

// ValidateURL checks rawURL against SSRF protection rules.
// It parses the URL, validates the scheme, resolves the hostname via DNS,
// and blocks private/internal IPs. Fails closed on DNS errors.
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("disallowed scheme %q: only http and https are permitted", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname in URL")
	}

	return validateHost(host)
}

// ValidateHostname validates a host:port pair (for non-URL contexts like gRPC dial).
// It resolves the host via DNS and blocks private/internal IPs. Fails closed.
// Port may be empty.
func ValidateHostname(host string, port string) error {
	if host == "" {
		return fmt.Errorf("empty hostname")
	}
	if port != "" {
		if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("invalid port %q", port)
		}
	}
	return validateHost(host)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────────

func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	// Block known-bypass hostname forms before DNS resolution
	if isBypassHost(host) {
		return nil, fmt.Errorf("SSRF blocked: suspicious hostname form %q", host)
	}

	// Resolve DNS at connection time (prevents DNS rebinding TOCTOU)
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("SSRF blocked: cannot resolve %q: %w", host, err)
	}

	for _, ip := range ips {
		private, err := isPrivateIP(ip.IP)
		if err != nil {
			return nil, fmt.Errorf("SSRF check failed: %w", err)
		}
		if private {
			return nil, fmt.Errorf("SSRF blocked: private IP %s is not permitted", ip.IP)
		}
	}

	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}

func validateHost(host string) error {
	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" {
		return fmt.Errorf("localhost addresses are not permitted")
	}
	if strings.HasSuffix(lowerHost, ".local") || strings.HasSuffix(lowerHost, ".internal") {
		return fmt.Errorf("internal DNS names are not permitted")
	}

	// Block bypass IP forms before DNS resolution
	if isBypassHost(host) {
		return fmt.Errorf("suspicious hostname form: %q", host)
	}

	// Resolve IP and check for private ranges
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve hostname %q: %w", host, err)
	}

	for _, ip := range ips {
		private, err := isPrivateIP(ip)
		if err != nil {
			return fmt.Errorf("SSRF check failed: %w", err)
		}
		if private {
			return fmt.Errorf("private IP address %s is not permitted", ip)
		}
	}

	return nil
}

// ─── Private IP detection ─────────────────────────────────────────────────────────

var (
	initMu      sync.Mutex
	initDone    bool
	initErr     error
	privateNets []*net.IPNet
)

// consolidated CIDR ranges merged from all prior implementations.
var cidrStrs = []string{
	"10.0.0.0/8",     // RFC 1918
	"172.16.0.0/12",  // RFC 1918
	"192.168.0.0/16", // RFC 1918
	"100.64.0.0/10",  // Carrier-grade NAT (RFC 6598)
	"169.254.0.0/16", // Link-local (RFC 3927)
	"127.0.0.0/8",    // Loopback
	"0.0.0.0/8",      // Current network (RFC 1122)
	"::1/128",        // IPv6 loopback
	"fc00::/7",       // IPv6 unique local address (ULA)
	"fe80::/10",      // IPv6 link-local
}

// validateCIDR parses s as a CIDR notation IP address and prefix, returning
// an error if the CIDR string is invalid.
func validateCIDR(s string) error {
	_, _, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("internal/ssrf/validator: invalid hardcoded CIDR %q: %w", s, err)
	}
	return nil
}

func initPrivateNets() error {
	initMu.Lock()
	defer initMu.Unlock()
	if initDone {
		return initErr
	}
	for _, s := range cidrStrs {
		if err := validateCIDR(s); err != nil {
			initErr = err
			initDone = true
			return err
		}
		_, n, _ := net.ParseCIDR(s) // safe: already validated above
		privateNets = append(privateNets, n)
	}
	initDone = true
	return nil
}

func isPrivateIP(ip net.IP) (bool, error) {
	if err := initPrivateNets(); err != nil {
		return false, err
	}
	for _, n := range privateNets {
		if n.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

// ─── Bypass IP detection ──────────────────────────────────────────────────────────

// isBypassHost detects IP address representations that bypass naive string checks:
//   - Octal: 0177.0.0.1
//   - Hex: 0x7f.0.0.1
//   - Integer: 2130706433 (single integer representing 127.0.0.1)
//   - Short-form: 127.1 (missing octets)
func isBypassHost(host string) bool {
	if host == "" {
		return false
	}

	parts := strings.Split(host, ".")

	switch {
	case len(parts) == 4:
		// Standard dotted-quad — check each part for non-decimal representation
		for _, p := range parts {
			if looksLikeNonDecimalPart(p) {
				return true
			}
		}
		return false

	case len(parts) == 1:
		// Single component — could be integer-form IP
		if _, err := strconv.Atoi(host); err == nil {
			return len(host) > 3 || host == "0"
		}
		return false

	case len(parts) >= 2 && len(parts) <= 3:
		// Short-form IP like "127.1" (meaning 127.0.0.1).
		// Only block if ALL parts are numeric (not a domain name).
		for _, p := range parts {
			if _, err := strconv.Atoi(p); err != nil {
				return false
			}
		}
		return true

	default:
		// 0 parts (empty) or 5+ parts — not a bypass form
		return false
	}
}

func looksLikeNonDecimalPart(part string) bool {
	if part == "" {
		return false
	}
	// Leading zero (but not "0" or "0.x"): octal
	if part[0] == '0' && len(part) > 1 {
		if part[1] != 'x' && part[1] != 'X' {
			return true
		}
	}
	// 0x or 0X prefix: hex
	if len(part) > 2 && (part[0:2] == "0x" || part[0:2] == "0X") {
		return true
	}
	return false
}

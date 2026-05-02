package osint

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/ssrf"
)

// ─── IP Lookup Tool ────────────────────────────────────────────────────────────────

// IPLookupResult represents the result of an IP geolocation lookup via ip-api.com.
type IPLookupResult struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country,omitempty"`
	CountryCode string  `json:"country_code,omitempty"`
	Region      string  `json:"region,omitempty"`
	City        string  `json:"city,omitempty"`
	ISP         string  `json:"isp,omitempty"`
	Org         string  `json:"org,omitempty"`
	Latitude    float64 `json:"lat,omitempty"`
	Longitude   float64 `json:"lon,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
	Status      string  `json:"status"`
	Error       string  `json:"error,omitempty"`
	Source      string  `json:"source"`
}

// IPLookupTool performs IP geolocation lookups using the free ip-api.com API.
type IPLookupTool struct {
	client *http.Client
}

// NewIPLookupTool creates an IPLookupTool with a 5-second timeout.
func NewIPLookupTool() *IPLookupTool {
	client := ssrf.NewClient()
	client.Timeout = 5 * time.Second
	return &IPLookupTool{client: client}
}

// Lookup performs an IP geolocation lookup against ip-api.com (free tier, 45 req/min).
func (t *IPLookupTool) Lookup(ctx context.Context, ip string) (IPLookupResult, error) {
	if ip == "" {
		return IPLookupResult{}, fmt.Errorf("ip is required")
	}

	target := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	if err := ssrf.ValidateURL(target); err != nil {
		return IPLookupResult{}, fmt.Errorf("SSRF validation failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return IPLookupResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Aleph-v2-OSINT/1.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return IPLookupResult{IP: ip, Status: "error", Source: "ip-api.com",
			Error: fmt.Sprintf("HTTP request failed: %v", err)}, nil
	}
	defer resp.Body.Close()

	var raw struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		Region      string  `json:"regionName"`
		City        string  `json:"city"`
		ISP         string  `json:"isp"`
		Org         string  `json:"org"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Timezone    string  `json:"timezone"`
		Query       string  `json:"query"`
		Message     string  `json:"message,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return IPLookupResult{IP: ip, Status: "error", Source: "ip-api.com",
			Error: fmt.Sprintf("decode response: %v", err)}, nil
	}

	result := IPLookupResult{
		Source: "ip-api.com",
	}

	if raw.Status == "fail" {
		result.IP = ip
		result.Status = "fail"
		result.Error = raw.Message
		return result, nil
	}

	result.IP = raw.Query
	result.Status = raw.Status
	result.Country = raw.Country
	result.CountryCode = raw.CountryCode
	result.Region = raw.Region
	result.City = raw.City
	result.ISP = raw.ISP
	result.Org = raw.Org
	result.Latitude = raw.Lat
	result.Longitude = raw.Lon
	result.Timezone = raw.Timezone

	return result, nil
}

// Execute implements the JSON→JSON tool interface for IPLookupTool.
func (t *IPLookupTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.IP == "" {
		return "", fmt.Errorf("ip is required")
	}
	result, err := t.Lookup(ctx, args.IP)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

// Register registers the IP lookup tool in the metadata repository.
func (t *IPLookupTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_ip_lookup",
		Name:         "osint_ip_lookup",
		Description:  "IP geolocation lookup using ip-api.com (free, 45 req/min)",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

// ─── DNS Resolution Tool ───────────────────────────────────────────────────────────

// DNSResolutionResult contains A, MX, and TXT records for a domain.
type DNSResolutionResult struct {
	Domain string   `json:"domain"`
	A      []string `json:"a_records,omitempty"`
	MX     []string `json:"mx_records,omitempty"`
	TXT    []string `json:"txt_records,omitempty"`
	Error  string   `json:"error,omitempty"`
	Source string   `json:"source"`
}

// DNSResolutionTool performs DNS resolution using Go's net package.
type DNSResolutionTool struct{}

// NewDNSResolutionTool creates a new DNS resolution tool.
func NewDNSResolutionTool() *DNSResolutionTool {
	return &DNSResolutionTool{}
}

// Resolve performs DNS resolution for the given domain (A, MX, TXT records).
func (t *DNSResolutionTool) Resolve(ctx context.Context, domain string) (DNSResolutionResult, error) {
	if domain == "" {
		return DNSResolutionResult{}, fmt.Errorf("domain is required")
	}

	resolver := &net.Resolver{}
	result := DNSResolutionResult{
		Domain: domain,
		Source: "go_net_dns",
	}

	// Resolve A records (IPv4) and AAAA records (IPv6) via LookupHost.
	ips, err := resolver.LookupHost(ctx, domain)
	if err != nil {
		result.Error = fmt.Sprintf("LookupHost failed: %v", err)
		return result, nil
	}
	result.A = ips

	// Resolve MX records (non-fatal if absent).
	mxRecords, err := resolver.LookupMX(ctx, domain)
	if err == nil {
		for _, mx := range mxRecords {
			result.MX = append(result.MX, fmt.Sprintf("%s (priority %d)", mx.Host, mx.Pref))
		}
	}

	// Resolve TXT records (non-fatal if absent).
	txtRecords, err := resolver.LookupTXT(ctx, domain)
	if err == nil {
		result.TXT = txtRecords
	}

	return result, nil
}

// Execute implements the JSON→JSON tool interface for DNSResolutionTool.
func (t *DNSResolutionTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Domain == "" {
		return "", fmt.Errorf("domain is required")
	}
	result, err := t.Resolve(ctx, args.Domain)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

// Register registers the DNS resolution tool in the metadata repository.
func (t *DNSResolutionTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_dns_resolution",
		Name:         "osint_dns_resolution",
		Description:  "DNS resolution (A, MX, TXT) using Go net.Lookup",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

// ─── WHOIS Lookup Tool ─────────────────────────────────────────────────────────────

// WhoisResult contains the raw WHOIS data for a domain.
type WhoisResult struct {
	Domain    string `json:"domain"`
	WhoisRaw  string `json:"whois_raw"`
	Server    string `json:"server"`
	TLD       string `json:"tld"`
	Error     string `json:"error,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Source    string `json:"source"`
}

// WhoisLookupTool performs WHOIS lookups using the protocol on TCP port 43.
type WhoisLookupTool struct{}

// NewWhoisLookupTool creates a new WHOIS lookup tool.
func NewWhoisLookupTool() *WhoisLookupTool {
	return &WhoisLookupTool{}
}

// Lookup performs a WHOIS lookup for the given domain.
func (t *WhoisLookupTool) Lookup(ctx context.Context, domain string) (WhoisResult, error) {
	if domain == "" {
		return WhoisResult{}, fmt.Errorf("domain is required")
	}

	tld := extractTLD(domain)

	// Step 1: query IANA to find the authoritative WHOIS server for this TLD.
	whoisServer, err := t.queryIANA(ctx, tld)
	if err != nil {
		return WhoisResult{
			Domain: domain, TLD: tld, Source: "whois_protocol",
			Error: fmt.Sprintf("IANA query failed: %v", err),
		}, nil
	}

	// Step 2: query the authoritative WHOIS server for the domain.
	raw, truncated, err := t.queryServer(ctx, whoisServer, domain)
	if err != nil {
		return WhoisResult{
			Domain: domain, TLD: tld, Server: whoisServer, Source: "whois_protocol",
			Error: fmt.Sprintf("WHOIS query to %s failed: %v", whoisServer, err),
		}, nil
	}

	return WhoisResult{
		Domain:    domain,
		WhoisRaw:  raw,
		Server:    whoisServer,
		TLD:       tld,
		Truncated: truncated,
		Source:    "whois_protocol",
	}, nil
}

// queryIANA queries whois.iana.org for the TLD's authoritative WHOIS server.
func (t *WhoisLookupTool) queryIANA(ctx context.Context, tld string) (string, error) {
	const ianaServer = "whois.iana.org:43"

	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", ianaServer)
	if err != nil {
		return "", fmt.Errorf("connect to IANA: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", fmt.Errorf("set deadline: %w", err)
	}

	// Send the TLD query.
	if _, err := fmt.Fprintf(conn, "%s\r\n", tld); err != nil {
		return "", fmt.Errorf("send query: %w", err)
	}

	// Read response (limit to 8KB).
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 8192), 8192)
	var responseLines []string
	for scanner.Scan() {
		responseLines = append(responseLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read IANA response: %w", err)
	}

	// Look for the "whois:" field in the response.
	for _, line := range responseLines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "whois:") {
			// Split on ":" to get the value, handling extra whitespace.
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				server := strings.TrimSpace(parts[1])
				if server != "" {
					return server, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no WHOIS server found for TLD %q in IANA response", tld)
}

// queryServer queries a WHOIS server for the given domain.
func (t *WhoisLookupTool) queryServer(ctx context.Context, server, domain string) (string, bool, error) {
	// Ensure server has port suffix.
	whoisAddr := strings.TrimSpace(server)
	if !strings.Contains(whoisAddr, ":") {
		whoisAddr = server + ":43"
	}

	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", whoisAddr)
	if err != nil {
		return "", false, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", false, fmt.Errorf("set deadline: %w", err)
	}

	if _, err := fmt.Fprintf(conn, "%s\r\n", domain); err != nil {
		return "", false, fmt.Errorf("send query: %w", err)
	}

	// Read up to 64KB.
	const maxRead = 65536
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, maxRead), maxRead)
	var responseLines []string
	for scanner.Scan() {
		responseLines = append(responseLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("read response: %w", err)
	}

	raw := strings.Join(responseLines, "\n")
	// Check if the response was truncated by looking at the total read length.
	truncated := len(raw) >= maxRead-1024
	return raw, truncated, nil
}

// Execute implements the JSON→JSON tool interface for WhoisLookupTool.
func (t *WhoisLookupTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Domain == "" {
		return "", fmt.Errorf("domain is required")
	}
	result, err := t.Lookup(ctx, args.Domain)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

// Register registers the WHOIS lookup tool in the metadata repository.
func (t *WhoisLookupTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_whois_lookup",
		Name:         "osint_whois_lookup",
		Description:  "WHOIS domain lookup via TCP port 43 (IANA→authoritative)",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

// ─── Threat Intel Check Tool ───────────────────────────────────────────────────────

// ThreatIntelResult contains a basic threat intelligence assessment for an IP.
type ThreatIntelResult struct {
	IP          string   `json:"ip"`
	IsPrivate   bool     `json:"is_private"`
	IsLoopback  bool     `json:"is_loopback"`
	IsRFC1918   bool     `json:"is_rfc1918"`
	RiskLevel   string   `json:"risk_level"` // "safe", "low", "medium", "high"
	Warnings    []string `json:"warnings,omitempty"`
	Source      string   `json:"source"`
}

// ThreatIntelCheckTool performs basic threat intel checks using heuristics only.
type ThreatIntelCheckTool struct{}

// NewThreatIntelCheckTool creates a new threat intel check tool.
func NewThreatIntelCheckTool() *ThreatIntelCheckTool {
	return &ThreatIntelCheckTool{}
}

// Check performs a basic threat intelligence assessment of the given IP address.
// Uses RFC 1918/localhost heuristics — NOT a full threat intel platform.
func (t *ThreatIntelCheckTool) Check(ctx context.Context, ip string) (ThreatIntelResult, error) {
	if ip == "" {
		return ThreatIntelResult{}, fmt.Errorf("ip is required")
	}

	result := ThreatIntelResult{
		IP:       ip,
		RiskLevel: "safe",
		Source:   "heuristic_analysis",
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		result.RiskLevel = "medium"
		result.Warnings = append(result.Warnings, "Invalid IP address format")
		return result, nil
	}

	// Check for loopback (127.0.0.0/8, ::1).
	if parsedIP.IsLoopback() {
		result.IsLoopback = true
		result.IsPrivate = true
		result.RiskLevel = "low"
		result.Warnings = append(result.Warnings, "IP is loopback (localhost)")
		return result, nil
	}

	// Check for RFC 1918 private ranges.
	if isRFC1918(parsedIP) {
		result.IsPrivate = true
		result.IsRFC1918 = true
		result.RiskLevel = "safe"
		result.Warnings = append(result.Warnings, "IP is RFC 1918 private address — internal use only")
		return result, nil
	}

	// Check for link-local (169.254.x.x).
	if parsedIP.IsLinkLocalUnicast() {
		result.IsPrivate = true
		result.RiskLevel = "low"
		result.Warnings = append(result.Warnings, "IP is link-local (169.254.x.x)")
		return result, nil
	}

	// Check for multicast.
	if parsedIP.IsMulticast() {
		result.RiskLevel = "low"
		result.Warnings = append(result.Warnings, "IP is multicast address")
		return result, nil
	}

	// Public IP — low default risk with heuristic notes.
	result.RiskLevel = "low"
	result.Warnings = append(result.Warnings, "IP is a public address")

	return result, nil
}

// isRFC1918 checks if an IP falls within private RFC 1918 ranges.
func isRFC1918(ip net.IP) bool {
	privateRanges := []struct {
		start net.IP
		end   net.IP
	}{
		{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	for _, r := range privateRanges {
		if bytesCompare(ip4, r.start.To4()) >= 0 && bytesCompare(ip4, r.end.To4()) <= 0 {
			return true
		}
	}
	return false
}

func bytesCompare(a, b net.IP) int {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// Execute implements the JSON→JSON tool interface for ThreatIntelCheckTool.
func (t *ThreatIntelCheckTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.IP == "" {
		return "", fmt.Errorf("ip is required")
	}
	result, err := t.Check(ctx, args.IP)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

// Register registers the threat intel check tool in the metadata repository.
func (t *ThreatIntelCheckTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_threat_intel_check",
		Name:         "osint_threat_intel_check",
		Description:  "Basic threat intel assessment via heuristics (RFC1918, localhost, multicast)",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

// ─── Shared helpers ────────────────────────────────────────────────────────────────

// extractTLD returns the last registered part of a domain.
// For "www.example.com" it returns "com".
func extractTLD(domain string) string {
	parts := strings.Split(strings.TrimRight(domain, "."), ".")
	if len(parts) <= 1 {
		return domain
	}
	return parts[len(parts)-1]
}

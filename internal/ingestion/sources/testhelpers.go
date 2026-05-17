package sources

import "net/http"

func NewTestRateLimitedClient() *RateLimitedClient {
	c := NewRateLimitedClient(RateLimitConfig{RequestsPerSecond: 100, Burst: 100})
	c.client = &http.Client{}
	return c
}

func (j *JSONAPIIngester) UseNonSSRFClient() { j.client = NewTestRateLimitedClient() }
func (s *SitemapIngester) UseNonSSRFClient()  { s.client = NewTestRateLimitedClient() }
func (g *GitHubIngester) UseNonSSRFClient()   { g.client = NewTestRateLimitedClient() }
func (sh *SheetsIngester) UseNonSSRFClient()  { sh.client = NewTestRateLimitedClient() }

// Package sources implements W3 ingestion methods for GitHub repositories.
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

// =============================================================================
// W3-06: GitHub Repo Ingestion
// =============================================================================

// GitHubIngester fetches data from the GitHub REST API (issues, PRs, commits).
type GitHubIngester struct {
	client *RateLimitedClient
	token  string
}

// NewGitHubIngester creates a GitHubIngester with the given personal access token.
// If token is empty, requests are unauthenticated (60 req/hr limit).
func NewGitHubIngester(token string) *GitHubIngester {
	return &GitHubIngester{
		client: NewRateLimitedClient(GitHubRate),
		token:  token,
	}
}

// parseLinkHeader extracts rel→URL mappings from an HTTP Link header.
// Standard format: <https://api.github.com/...>; rel="next", <https://...>; rel="last"
func parseLinkHeader(header string) map[string]string {
	result := make(map[string]string)
	if header == "" {
		return result
	}
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)

		urlStart := strings.Index(part, "<")
		urlEnd := strings.Index(part, ">")
		if urlStart < 0 || urlEnd < 0 || urlEnd <= urlStart {
			continue
		}
		url := part[urlStart+1 : urlEnd]

		relTag := `rel="`
		relStart := strings.Index(part, relTag)
		if relStart < 0 {
			continue
		}
		relValue := part[relStart+len(relTag):]
		relEnd := strings.Index(relValue, `"`)
		if relEnd < 0 {
			continue
		}
		rel := relValue[:relEnd]
		result[rel] = url
	}
	return result
}

// defaultHeaders returns the standard GitHub API headers.
func (g *GitHubIngester) defaultHeaders() map[string]string {
	h := map[string]string{
		"Accept": "application/vnd.github.v3+json",
	}
	if g.token != "" {
		h["Authorization"] = "Bearer " + g.token
	}
	return h
}

// fetchPaginated fetches a paginated GitHub API endpoint and merges all pages
// into a single JSON array. Pagination is driven by the Link header (rel="next").
func (g *GitHubIngester) fetchPaginated(ctx context.Context, url string) ([]byte, error) {
	var allResults []json.RawMessage
	headers := g.defaultHeaders()

	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request for %s: %w", url, err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("GET %s: %w", url, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read body from %s: %w", url, err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP %d from %s: %s",
				resp.StatusCode, url, string(body[:min(len(body), 500)]))
		}

		var page []json.RawMessage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode JSON from %s: %w", url, err)
		}
		allResults = append(allResults, page...)

		linkHeader := resp.Header.Get("Link")
		links := parseLinkHeader(linkHeader)
		url = links["next"]
	}

	out, err := json.Marshal(allResults)
	if err != nil {
		return nil, fmt.Errorf("marshal combined results: %w", err)
	}
	return out, nil
}

// FetchIssues fetches all issues (state=all) for the given repository.
func (g *GitHubIngester) FetchIssues(ctx context.Context, owner, repo string) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=all&per_page=100", owner, repo)
	data, err := g.fetchPaginated(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch issues %s/%s: %w", owner, repo, err)
	}
	return data, nil
}

// FetchPRs fetches all pull requests (state=all) for the given repository.
func (g *GitHubIngester) FetchPRs(ctx context.Context, owner, repo string) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=all&per_page=100", owner, repo)
	data, err := g.fetchPaginated(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch pulls %s/%s: %w", owner, repo, err)
	}
	return data, nil
}

// FetchCommits fetches all commits for the given repository.
func (g *GitHubIngester) FetchCommits(ctx context.Context, owner, repo string) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=100", owner, repo)
	data, err := g.fetchPaginated(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch commits %s/%s: %w", owner, repo, err)
	}
	return data, nil
}

// FetchAll fetches issues, PRs, and commits concurrently.
// Returns a map of "issues" → raw JSON, "pulls" → raw JSON, "commits" → raw JSON.
func (g *GitHubIngester) FetchAll(ctx context.Context, owner, repo string) (map[string][]byte, error) {
	results := make(map[string][]byte)
	var mu sync.Mutex
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		data, err := g.FetchIssues(ctx, owner, repo)
		if err != nil {
			return err
		}
		mu.Lock()
		results["issues"] = data
		mu.Unlock()
		return nil
	})

	eg.Go(func() error {
		data, err := g.FetchPRs(ctx, owner, repo)
		if err != nil {
			return err
		}
		mu.Lock()
		results["pulls"] = data
		mu.Unlock()
		return nil
	})

	eg.Go(func() error {
		data, err := g.FetchCommits(ctx, owner, repo)
		if err != nil {
			return err
		}
		mu.Lock()
		results["commits"] = data
		mu.Unlock()
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

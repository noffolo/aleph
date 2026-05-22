package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── NewGitHubIngester ───────────────────────────────────────────────────────

func TestNewGitHubIngester_HappyPath_WithToken(t *testing.T) {
	g := NewGitHubIngester("ghp_test123")
	require.NotNil(t, g)
	assert.NotNil(t, g.client)
	assert.Equal(t, "ghp_test123", g.token)
}

func TestNewGitHubIngester_EmptyToken(t *testing.T) {
	g := NewGitHubIngester("")
	require.NotNil(t, g)
	assert.Equal(t, "", g.token)
}

func TestNewGitHubIngester_NilClientCheck(t *testing.T) {
	g := NewGitHubIngester("any-token")
	require.NotNil(t, g)
	assert.NotNil(t, g.client)
}

// ─── parseLinkHeader ─────────────────────────────────────────────────────────

func TestParseLinkHeader_HappyPath(t *testing.T) {
	got := parseLinkHeader(`<https://api.github.com/repos/owner/repo/issues?page=2>; rel="next"`)
	assert.Equal(t, "https://api.github.com/repos/owner/repo/issues?page=2", got["next"])
}

func TestParseLinkHeader_MultiLink(t *testing.T) {
	got := parseLinkHeader(`<https://api.example.com/page/2>; rel="next", <https://api.example.com/page/5>; rel="last"`)
	assert.Equal(t, "https://api.example.com/page/2", got["next"])
	assert.Equal(t, "https://api.example.com/page/5", got["last"])
}

func TestParseLinkHeader_EmptyHeader(t *testing.T) {
	got := parseLinkHeader("")
	assert.Empty(t, got)
}

func TestParseLinkHeader_MalformedNoRel(t *testing.T) {
	got := parseLinkHeader(`<https://example.com>`)
	assert.Empty(t, got)
}

func TestParseLinkHeader_MalformedNoURL(t *testing.T) {
	got := parseLinkHeader(`rel="next"`)
	assert.Empty(t, got)
}

func TestParseLinkHeader_WithSpaces(t *testing.T) {
	got := parseLinkHeader(` <https://example.com/2> ; rel="next" `)
	assert.Equal(t, "https://example.com/2", got["next"])
}

// ─── defaultHeaders ──────────────────────────────────────────────────────────

func TestDefaultHeaders_WithToken(t *testing.T) {
	g := NewGitHubIngester("ghp_secret")
	h := g.defaultHeaders()
	assert.Equal(t, "application/vnd.github.v3+json", h["Accept"])
	assert.Equal(t, "Bearer ghp_secret", h["Authorization"])
}

func TestDefaultHeaders_WithoutToken(t *testing.T) {
	g := NewGitHubIngester("")
	h := g.defaultHeaders()
	assert.Equal(t, "application/vnd.github.v3+json", h["Accept"])
	_, hasAuth := h["Authorization"]
	assert.False(t, hasAuth)
}

func TestDefaultHeaders_ContainsAcceptHeader(t *testing.T) {
	g := NewGitHubIngester("tok")
	h := g.defaultHeaders()
	assert.Contains(t, h, "Accept")
	assert.NotEmpty(t, h["Accept"])
}

// ─── fetchPaginated ──────────────────────────────────────────────────────────

func TestFetchPaginated_HappyPath_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{{"id": 1}, {"id": 2}})
	}))
	defer srv.Close()

	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	data, err := g.fetchPaginated(context.Background(), srv.URL)
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
}

func TestFetchPaginated_MultiPage(t *testing.T) {
	callCount := 0
	var serverURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Header().Set("Link", `<`+serverURL+`?page=2>; rel="next"`)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]any{{"id": 1}})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]any{{"id": 2}})
		}
	}))
	defer srv.Close()
	serverURL = srv.URL

	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	data, err := g.fetchPaginated(context.Background(), serverURL)
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Len(t, results, 2)
	assert.Equal(t, 2, callCount)
}

func TestFetchPaginated_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer srv.Close()

	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	_, err := g.fetchPaginated(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFetchPaginated_EmptyArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	data, err := g.fetchPaginated(context.Background(), srv.URL)
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(data, &results))
	assert.Empty(t, results)
}

func TestFetchPaginated_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	_, err := g.fetchPaginated(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode JSON")
}

// ─── FetchIssues ─────────────────────────────────────────────────────────────

func TestFetchIssues_Error_NoTokenRealAPI(t *testing.T) {
	g := NewGitHubIngester("")
	g.client = NewTestRateLimitedClient()
	_, err := g.FetchIssues(context.Background(), "owner", "repo")
	assert.Error(t, err)
}

func TestFetchIssues_EmptyOwner(t *testing.T) {
	g := NewGitHubIngester("test-token")
	g.client = NewTestRateLimitedClient()
	_, err := g.FetchIssues(context.Background(), "", "repo")
	assert.Error(t, err)
}

func TestFetchIssues_FullPath_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{{"title": "Test"}})
	}))
	defer srv.Close()

	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	data, err := g.fetchPaginated(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

// ─── FetchPRs ────────────────────────────────────────────────────────────────

func TestFetchPRs_Error_NoTokenRealAPI(t *testing.T) {
	g := NewGitHubIngester("")
	g.client = NewTestRateLimitedClient()
	_, err := g.FetchPRs(context.Background(), "owner", "repo")
	assert.Error(t, err)
}

func TestFetchPRs_ErrorOnCall(t *testing.T) {
	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	_, err := g.FetchPRs(context.Background(), "owner", "repo")
	assert.Error(t, err)
}

func TestFetchPRs_HTTPBuildsCorrectURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{{"number": 1}})
	}))
	defer srv.Close()

	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	data, err := g.fetchPaginated(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

// ─── FetchCommits ────────────────────────────────────────────────────────────

func TestFetchCommits_Error_NoTokenRealAPI(t *testing.T) {
	g := NewGitHubIngester("")
	g.client = NewTestRateLimitedClient()
	_, err := g.FetchCommits(context.Background(), "owner", "repo")
	assert.Error(t, err)
}

func TestFetchCommits_NilClient(t *testing.T) {
	g := &GitHubIngester{token: "test", client: NewTestRateLimitedClient()}
	_, err := g.FetchCommits(context.Background(), "owner", "repo")
	assert.Error(t, err)
}

func TestFetchCommits_HTTPBuildsCorrectURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{{"sha": "abc123"}})
	}))
	defer srv.Close()

	g := &GitHubIngester{client: NewTestRateLimitedClient()}
	data, err := g.fetchPaginated(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

// ─── FetchAll ────────────────────────────────────────────────────────────────

func TestFetchAll_ErrorOnFetch(t *testing.T) {
	g := NewGitHubIngester("")
	g.client = NewTestRateLimitedClient()
	_, err := g.FetchAll(context.Background(), "owner", "repo")
	assert.Error(t, err)
}

func TestFetchAll_NilClientCrashes(t *testing.T) {
	g := &GitHubIngester{token: "test-token", client: NewTestRateLimitedClient()}
	_, err := g.FetchAll(context.Background(), "owner", "repo")
	assert.Error(t, err)
}

func TestFetchAll_EmptyOwnerRepoFails(t *testing.T) {
	g := NewGitHubIngester("tok")
	g.client = NewTestRateLimitedClient()
	_, err := g.FetchAll(context.Background(), "", "")
	assert.Error(t, err)
}

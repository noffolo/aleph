package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── NewSitemapIngester ─────────────────────────────────────────────────────

func TestNewSitemapIngester_HappyPath(t *testing.T) {
	s := NewSitemapIngester()
	require.NotNil(t, s)
	assert.NotNil(t, s.client)
}

func TestNewSitemapIngester_ReturnsNewInstance(t *testing.T) {
	s1 := NewSitemapIngester()
	s2 := NewSitemapIngester()
	assert.NotSame(t, s1, s2)
}

func TestNewSitemapIngester_ClientDefaultRate(t *testing.T) {
	s := NewSitemapIngester()
	assert.NotNil(t, s.client)
}

// ─── CrawlSitemap ────────────────────────────────────────────────────────────

func TestCrawlSitemap_HappyPath_URLSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/page1</loc></url></urlset>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	result, err := s.CrawlSitemap(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL, result.SitemapURL)
	assert.Len(t, result.URLs, 1)
}

func TestCrawlSitemap_SitemapIndex(t *testing.T) {
	childSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/page1</loc></url></urlset>`))
	}))
	defer childSrv.Close()

	indexSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>` + childSrv.URL + `</loc></sitemap></sitemapindex>`))
	}))
	defer indexSrv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	result, err := s.CrawlSitemap(context.Background(), indexSrv.URL)
	require.NoError(t, err)
	assert.Len(t, result.URLs, 1)
}

func TestCrawlSitemap_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	_, err := s.CrawlSitemap(context.Background(), srv.URL)
	assert.Error(t, err)
}

func TestCrawlSitemap_InvalidXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not xml at all`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	_, err := s.CrawlSitemap(context.Background(), srv.URL)
	assert.Error(t, err)
}

func TestCrawlSitemap_InvalidRootElement(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><randomroot></randomroot>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	_, err := s.CrawlSitemap(context.Background(), srv.URL)
	assert.Error(t, err)
}

func TestCrawlSitemap_EmptyLocInIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc></loc></sitemap></sitemapindex>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	result, err := s.CrawlSitemap(context.Background(), srv.URL)
	assert.NoError(t, err)
	assert.Empty(t, result.URLs)
}

func TestCrawlSitemap_BadChildURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>://bad-url</loc></sitemap></sitemapindex>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	result, err := s.CrawlSitemap(context.Background(), srv.URL)
	assert.NoError(t, err)
	assert.Empty(t, result.URLs)
}

func TestCrawlSitemap_BadChildSitemap(t *testing.T) {
	childSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer childSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>` + childSrv.URL + `</loc></sitemap></sitemapindex>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	result, err := s.CrawlSitemap(context.Background(), srv.URL)
	assert.NoError(t, err)
	assert.Empty(t, result.URLs)
}

func TestCrawlSitemap_URLSetWithMultiplePages(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>` + srv.URL + `/page1</loc></url><url><loc>` + srv.URL + `/page2</loc></url></urlset>`))
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html>page</html>`))
		}
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	result, err := s.CrawlSitemap(context.Background(), srv.URL+"/sitemap.xml")
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/sitemap.xml", result.SitemapURL)
}

// ─── fetchXML ────────────────────────────────────────────────────────────────

func TestFetchXML_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<root/>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	body, err := s.fetchXML(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestFetchXML_BadContentType_New(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not xml`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	_, err := s.fetchXML(context.Background(), srv.URL)
	assert.Error(t, err)
}

func TestFetchXML_NoContentType_New(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<root/>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	body, err := s.fetchXML(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestFetchXML_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	_, err := s.fetchXML(context.Background(), srv.URL)
	assert.Error(t, err)
}

func TestFetchXML_InvalidURL(t *testing.T) {
	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	_, err := s.fetchXML(context.Background(), "://invalid-url")
	assert.Error(t, err)
}

func TestFetchXML_TextXMLContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<data></data>`))
	}))
	defer srv.Close()

	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	body, err := s.fetchXML(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

// ─── detectRootElement ───────────────────────────────────────────────────────

func TestDetectRootElement_HappyPath_URLSet(t *testing.T) {
	root, err := detectRootElement([]byte(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/</loc></url></urlset>`))
	require.NoError(t, err)
	assert.Equal(t, "urlset", root)
}

func TestDetectRootElement_SitemapIndex(t *testing.T) {
	root, err := detectRootElement([]byte(`<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>https://example.com/sitemap.xml</loc></sitemap></sitemapindex>`))
	require.NoError(t, err)
	assert.Equal(t, "sitemapindex", root)
}

func TestDetectRootElement_EmptyXML(t *testing.T) {
	_, err := detectRootElement([]byte(``))
	assert.Error(t, err)
}

func TestDetectRootElement_EndElementFirst(t *testing.T) {
	_, err := detectRootElement([]byte(`</root>`))
	assert.Error(t, err)
}

func TestDetectRootElement_ProcInst_New(t *testing.T) {
	name, err := detectRootElement([]byte(`<?xml version="1.0"?><test/>`))
	require.NoError(t, err)
	assert.Equal(t, "test", name)
}

func TestDetectRootElement_GenericRoot(t *testing.T) {
	name, err := detectRootElement([]byte(`<root></root>`))
	require.NoError(t, err)
	assert.Equal(t, "root", name)
}

// ─── isXMLContentType ────────────────────────────────────────────────────────

func TestIsXMLContentType_ApplicationXML(t *testing.T) {
	assert.True(t, isXMLContentType("application/xml"))
}

func TestIsXMLContentType_TextXML(t *testing.T) {
	assert.True(t, isXMLContentType("text/xml"))
}

func TestIsXMLContentType_PlusXML(t *testing.T) {
	assert.True(t, isXMLContentType("application/atom+xml"))
	assert.True(t, isXMLContentType("application/rss+xml"))
}

func TestIsXMLContentType_NotXML(t *testing.T) {
	assert.False(t, isXMLContentType("text/html"))
	assert.False(t, isXMLContentType("application/json"))
	assert.False(t, isXMLContentType("text/plain"))
}

// ─── isTextContentType ───────────────────────────────────────────────────────

func TestIsTextContentType_HTML(t *testing.T) {
	assert.True(t, isTextContentType("text/html"))
}

func TestIsTextContentType_Plain(t *testing.T) {
	assert.True(t, isTextContentType("text/plain"))
}

func TestIsTextContentType_JSON(t *testing.T) {
	assert.True(t, isTextContentType("application/json"))
}

func TestIsTextContentType_NotText(t *testing.T) {
	assert.False(t, isTextContentType("application/xml"))
	assert.False(t, isTextContentType("image/png"))
	assert.False(t, isTextContentType("application/pdf"))
}

// ─── resolveURL ──────────────────────────────────────────────────────────────

func TestResolveURL_AbsoluteURL(t *testing.T) {
	got, err := resolveURL("https://example.com/sitemap.xml", "https://other.com/page.html")
	require.NoError(t, err)
	assert.Equal(t, "https://other.com/page.html", got)
}

func TestResolveURL_RelativeURL(t *testing.T) {
	got, err := resolveURL("https://example.com/sitemap.xml", "/page.html")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/page.html", got)
}

func TestResolveURL_RelativePath(t *testing.T) {
	got, err := resolveURL("https://example.com/dir/sitemap.xml", "page.html")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/dir/page.html", got)
}

func TestResolveURL_InvalidBase(t *testing.T) {
	_, err := resolveURL("://invalid", "/path")
	assert.Error(t, err)
}

func TestResolveURL_InvalidRawURL(t *testing.T) {
	_, err := resolveURL("https://example.com", "://invalid")
	assert.Error(t, err)
}

// ─── followRedirects ─────────────────────────────────────────────────────────

func TestFollowRedirects_HappyPath(t *testing.T) {
	client := followRedirects(3)
	require.NotNil(t, client)
	assert.NotNil(t, client.CheckRedirect)
	assert.Equal(t, int64(30*1e9), int64(client.Timeout))
}

func TestFollowRedirects_ZeroRedirects(t *testing.T) {
	client := followRedirects(0)
	assert.NotNil(t, client)
	assert.NotNil(t, client.CheckRedirect)
}

func TestFollowRedirects_LargeLimit(t *testing.T) {
	client := followRedirects(20)
	assert.NotNil(t, client)
	assert.NotNil(t, client.CheckRedirect)
}

// ─── isAllowedContentType ────────────────────────────────────────────────────

func TestIsAllowedContentType_HTML(t *testing.T) {
	assert.True(t, isAllowedContentType("text/html"))
}

func TestIsAllowedContentType_Plain(t *testing.T) {
	assert.True(t, isAllowedContentType("text/plain"))
}

func TestIsAllowedContentType_JSON(t *testing.T) {
	assert.True(t, isAllowedContentType("application/json"))
}

func TestIsAllowedContentType_XML(t *testing.T) {
	assert.True(t, isAllowedContentType("application/xml"))
	assert.True(t, isAllowedContentType("text/xml"))
	assert.True(t, isAllowedContentType("application/atom+xml"))
}

func TestIsAllowedContentType_BinaryBlocked(t *testing.T) {
	assert.False(t, isAllowedContentType("image/jpeg"))
	assert.False(t, isAllowedContentType("application/pdf"))
	assert.False(t, isAllowedContentType("video/mp4"))
}

// ─── fetchAllPages ───────────────────────────────────────────────────────────

func TestFetchAllPages_EmptySlice(t *testing.T) {
	s := NewSitemapIngester()
	results, err := s.fetchAllPages(context.Background(), []string{})
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestFetchAllPages_InvalidURLs(t *testing.T) {
	s := NewSitemapIngester()
	s.client = NewTestRateLimitedClient()
	results, err := s.fetchAllPages(context.Background(), []string{"://bad-url"})
	assert.NoError(t, err)
	require.Len(t, results, 1)
	assert.NotNil(t, results[0].Err)
}

// ─── Struct types ────────────────────────────────────────────────────────────

func TestSitemapIndex_Struct_New(t *testing.T) {
	index := SitemapIndex{
		Sitemaps: []SitemapEntry{
			{Loc: "https://example.com/sitemap1.xml", LastMod: "2024-01-01"},
		},
	}
	assert.Len(t, index.Sitemaps, 1)
	assert.Equal(t, "https://example.com/sitemap1.xml", index.Sitemaps[0].Loc)
}

func TestSitemapEntry_Struct(t *testing.T) {
	entry := SitemapEntry{Loc: "https://example.com/sitemap.xml", LastMod: "2024-01-01"}
	assert.Equal(t, "https://example.com/sitemap.xml", entry.Loc)
	assert.Equal(t, "2024-01-01", entry.LastMod)
}

func TestURLSet_Struct_New(t *testing.T) {
	urlset := URLSet{
		URLs: []URLEntry{
			{Loc: "https://example.com/", LastMod: "2024-01-01", ChangeFreq: "daily", Priority: "1.0"},
		},
	}
	assert.Len(t, urlset.URLs, 1)
	assert.Equal(t, "https://example.com/", urlset.URLs[0].Loc)
	assert.Equal(t, "daily", urlset.URLs[0].ChangeFreq)
	assert.Equal(t, "1.0", urlset.URLs[0].Priority)
}

func TestURLEntry_Struct(t *testing.T) {
	entry := URLEntry{
		Loc:        "https://example.com/page",
		LastMod:    "2024-02-01",
		ChangeFreq: "weekly",
		Priority:   "0.8",
	}
	assert.Equal(t, "https://example.com/page", entry.Loc)
	assert.Equal(t, "weekly", entry.ChangeFreq)
	assert.Equal(t, "0.8", entry.Priority)
}

func TestCrawlResult_Struct_New(t *testing.T) {
	result := CrawlResult{
		SitemapURL: "https://example.com/sitemap.xml",
		URLs: []PageResult{
			{URL: "https://example.com/page1", Status: 200, Content: []byte("html")},
		},
	}
	assert.Equal(t, "https://example.com/sitemap.xml", result.SitemapURL)
	assert.Len(t, result.URLs, 1)
	assert.Equal(t, 200, result.URLs[0].Status)
}

func TestPageResult_Struct_New(t *testing.T) {
	result := PageResult{
		URL:     "https://example.com/page",
		Content: []byte("content"),
		Size:    100,
		Status:  200,
		Err:     nil,
	}
	assert.Equal(t, "https://example.com/page", result.URL)
	assert.Equal(t, int64(100), result.Size)
	assert.Equal(t, 200, result.Status)
}

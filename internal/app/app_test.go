package app

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/registry"
)

// ── newH2CClient ──────────────────────────────────────────────────────────

func TestNewH2CClient_Basic(t *testing.T) {
	client := newH2CClient()
	require.NotNil(t, client)
	assert.Equal(t, 30*time.Second, client.Timeout)

	transport, ok := client.Transport.(*http2.Transport)
	require.True(t, ok, "transport should be *http2.Transport")
	assert.True(t, transport.AllowHTTP, "AllowHTTP should be true for h2c")
}

func TestNewH2CClient_SSRFRejectsLinkLocal(t *testing.T) {
	client := newH2CClient()
	transport := client.Transport.(*http2.Transport)

	_, err := transport.DialTLSContext(context.Background(), "tcp", "169.254.169.254:80", &tls.Config{})
	assert.Error(t, err, "link-local should be rejected by SSRF")
	assert.Contains(t, err.Error(), "SSRF")
}

// ── newTLSClient ──────────────────────────────────────────────────────────

func TestNewTLSClient_Basic(t *testing.T) {
	client := newTLSClient()
	require.NotNil(t, client)
	assert.Equal(t, 30*time.Second, client.Timeout)

	transport, ok := client.Transport.(*http2.Transport)
	require.True(t, ok, "transport should be *http2.Transport")
	assert.NotNil(t, transport.DialTLSContext, "DialTLSContext should be set")
}

func TestNewTLSClient_SSRFRejectsPrivate(t *testing.T) {
	client := newTLSClient()
	transport := client.Transport.(*http2.Transport)

	_, err := transport.DialTLSContext(context.Background(), "tcp", "192.168.1.1:443", &tls.Config{MinVersion: tls.VersionTLS13})
	assert.Error(t, err, "private IP should be rejected by SSRF")
	assert.Contains(t, err.Error(), "SSRF")
}

// ── makeSentimentHelper ───────────────────────────────────────────────────

func TestMakeSentimentHelper_NilNLPHandler(t *testing.T) {
	a := &AlephApp{nlpHandler: nil}
	helper := a.makeSentimentHelper()

	result, err := helper(context.Background(), "test text")
	assert.NoError(t, err)
	assert.Contains(t, result, "neutral")
	assert.Contains(t, result, "NLP sidecar unavailable")
}

func TestMakeSentimentHelper_ReturnsClosure(t *testing.T) {
	a := &AlephApp{nlpHandler: nil}
	helper := a.makeSentimentHelper()
	require.NotNil(t, helper)
}

// ── makeTrustScoreHelper ──────────────────────────────────────────────────

func TestMakeTrustScoreHelper_NilRegistry(t *testing.T) {
	a := &AlephApp{}
	helper := a.makeTrustScoreHelper(nil)

	result, err := helper(context.Background(), "entity-1")
	assert.NoError(t, err)
	assert.Contains(t, result, "registry unavailable")
}

func TestMakeTrustScoreHelper_ReturnsClosure(t *testing.T) {
	a := &AlephApp{}
	helper := a.makeTrustScoreHelper(nil)
	require.NotNil(t, helper)
}

// ── makeComponentByIDHelper ───────────────────────────────────────────────

func TestMakeComponentByIDHelper_NilRegistry(t *testing.T) {
	a := &AlephApp{}
	helper := a.makeComponentByIDHelper(nil)

	result, err := helper(context.Background(), "comp-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry unavailable")
	assert.Nil(t, result)
}

// ── Type existence ────────────────────────────────────────────────────────

func TestStructInitialization(t *testing.T) {
	a := &AlephApp{cancel: func() {}}
	require.NotNil(t, a)
}

// Prevent unused import errors from registry/decision types referenced
// in production code signatures.
var _ = &registry.DuckDBRegistry{}
var _ = &decision.ComponentMetadata{}
var _ = connect.NewRequest(&nlpv1.AnalyzeSentimentRequest{})

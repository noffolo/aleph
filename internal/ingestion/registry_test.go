package ingestion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFetcher struct{ sourceType string }

func (m *mockFetcher) SourceType() string { return m.sourceType }
func (m *mockFetcher) Validate() error     { return nil }

func mockFactory(sourceType string) Fetcher { return &mockFetcher{sourceType: sourceType} }

func TestRegistryRegistration(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)

	r.Register("mock_one", mockFactory)
	r.Register("mock_two", mockFactory)

	fetcher, err := r.Create("mock_one")
	require.NoError(t, err)
	assert.Equal(t, "mock_one", fetcher.SourceType())

	_, err = r.Create("nonexistent")
	assert.ErrorIs(t, err, ErrSourceTypeNotFound)
}

func TestRegistryListAll(t *testing.T) {
	r := NewRegistry()
	r.Register("a", mockFactory)
	r.Register("b", mockFactory)

	types := r.List()
	assert.Len(t, types, 2)
	assert.Contains(t, types, "a")
	assert.Contains(t, types, "b")
}

package memory

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/storage"
)

func TestArrayLiteral_Empty(t *testing.T) {
	ms := &MemoryStore{dim: 768}
	got := ms.arrayLiteral([]float32{})
	expected := "CAST(ARRAY[] AS FLOAT[768])"
	if got != expected {
		t.Errorf("arrayLiteral empty: expected %q, got %q", expected, got)
	}
}

func TestArrayLiteral_Single(t *testing.T) {
	ms := &MemoryStore{dim: 4}
	got := ms.arrayLiteral([]float32{0.5})
	expected := "[0.5]::FLOAT[4]"
	if got != expected {
		t.Errorf("arrayLiteral single: expected %q, got %q", expected, got)
	}
}

func TestArrayLiteral_Multiple(t *testing.T) {
	ms := &MemoryStore{dim: 3}
	got := ms.arrayLiteral([]float32{0.1, 0.2, 0.3})
	expected := "[0.1,0.2,0.3]::FLOAT[3]"
	if got != expected {
		t.Errorf("arrayLiteral multiple: expected %q, got %q", expected, got)
	}
}

func TestArrayLiteral_Large(t *testing.T) {
	ms := &MemoryStore{dim: 768}
	emb := make([]float32, 768)
	for i := range emb {
		emb[i] = float32(i) * 0.001
	}
	got := ms.arrayLiteral(emb)
	require.Contains(t, got, "::FLOAT[768]")
	require.Contains(t, got, "[")
	require.Contains(t, got, "]")
}

func TestClose_Noop(t *testing.T) {
	ms := &MemoryStore{}
	if err := ms.Close(); err != nil {
		t.Errorf("Close: expected nil, got %v", err)
	}
}

func TestClose_MultipleCalls(t *testing.T) {
	ms := &MemoryStore{}
	if err := ms.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ms.Close(); err != nil {
		t.Fatal("second Close should also be noop")
	}
}

func TestNewMemoryStore_NegativeDim(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = NewMemoryStore(db, "", -1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "embedding dimension must be positive")
}

func TestNewMemoryStore_WarnsOnNonStandardDim(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	store, err := NewMemoryStore(db, "", 384)
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestNewMemoryStore_StandardDim(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	store, err := NewMemoryStore(db, "", 768)
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestStore_EmptyEmbedding(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)

	err = ms.Store(context.Background(), "empty-emb-key", []byte("value"), []float32{})
	require.Error(t, err, "DuckDB rejects empty embedding arrays in INSERT")
}

func TestStore_NilValue(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)

	err = ms.Store(context.Background(), "nil-value-key", nil, []float32{0.1, 0.2, 0.3, 0.4})
	require.NoError(t, err)

	val, ok := ms.Get(context.Background(), "nil-value-key")
	require.True(t, ok)
	require.Equal(t, []byte{}, val, "DuckDB turns nil BLOB into empty byte slice")
}

func TestStore_RetryOnConstraint(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("retry-key-%d", i)
		err := ms.Store(ctx, key, []byte("retry-value"), []float32{0.1, 0.2, 0.3, 0.4})
		require.NoError(t, err)
	}

	entries, err := ms.List(ctx, 20, 0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 10)
}

func TestSearch_LowerLimit(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("lim-key-%d", i)
		_ = ms.Store(ctx, key, []byte("data"), []float32{0.1, 0.2, 0.3, 0.4})
	}

	results, err := ms.Search(ctx, []float32{0.1, 0.2, 0.3, 0.4}, 2)
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestSearch_EmptyStore(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)

	results, err := ms.Search(context.Background(), []float32{0.1, 0.2, 0.3, 0.4}, 10)
	require.NoError(t, err)
	require.Len(t, results, 0)
}

func TestSearchText_LowerLimit(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("text-key-%d", i)
		_ = ms.Store(ctx, key, []byte("match-"+"text-key-"+fmt.Sprint(i)), []float32{0.1, 0.2, 0.3, 0.4})
	}

	results, err := ms.SearchText(ctx, "text-key", 2)
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestSearchText_EmptyStore(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)

	results, err := ms.SearchText(context.Background(), "query", 10)
	require.NoError(t, err)
	require.Len(t, results, 0)
}

func TestDelete_Nonexistent(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)

	err = ms.Delete(context.Background(), "never-stored-key")
	require.NoError(t, err)
}

func TestList_EmptyStore(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)

	entries, err := ms.List(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 0)
}

func TestGet_EmptyStore(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	ms, err := NewMemoryStore(db, "", 4)
	require.NoError(t, err)

	_, ok := ms.Get(context.Background(), "nonexistent")
	require.False(t, ok)
}

func TestExpectedEmbedDim(t *testing.T) {
	if expectedEmbedDim != 768 {
		t.Errorf("expectedEmbedDim should be 768, got %d", expectedEmbedDim)
	}
}

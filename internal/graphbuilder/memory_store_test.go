package graphbuilder

import (
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/gnn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// embeddedRow represents a row read back from the memory_store table.
type embeddedRow struct {
	key       string
	nodeType  string
	embedding []float64
	metadata  []byte
}

// readMemoryStore reads all rows from the memory_store table for verification.
func readMemoryStore(t *testing.T, db *sql.DB) []embeddedRow {
	t.Helper()
	rows, err := db.Query("SELECT key, node_type, metadata, embedding FROM memory_store ORDER BY key")
	require.NoError(t, err)
	defer rows.Close()

	var results []embeddedRow
	for rows.Next() {
		var key, nodeType string
		var metadata []byte
		var rawEmbedding []any
		if err := rows.Scan(&key, &nodeType, &metadata, &rawEmbedding); err != nil {
			t.Fatalf("scan memory_store row: %v", err)
		}
		emb := make([]float64, len(rawEmbedding))
		for i, v := range rawEmbedding {
			switch val := v.(type) {
			case float64:
				emb[i] = val
			case float32:
				emb[i] = float64(val)
			default:
				t.Fatalf("unexpected embedding element type %T at index %d", v, i)
			}
		}
		results = append(results, embeddedRow{
			key:       key,
			nodeType:  nodeType,
			embedding: emb,
			metadata:  metadata,
		})
	}
	require.NoError(t, rows.Err())
	return results
}

func buildTestGraph(t *testing.T) (*gnn.Graph, *gnn.GNNModel) {
	t.Helper()

	g := gnn.NewGraph()
	g.AddNode(&gnn.WorkflowNode{ID: "party:partito-democratico", Type: "party"})
	g.AddNode(&gnn.WorkflowNode{ID: "donor:mario-rossi", Type: "donor"})
	g.AddNode(&gnn.WorkflowNode{ID: "person:giorgia-bianchi", Type: "person"})
	g.AddNode(&gnn.WorkflowNode{ID: "election:camera:2022", Type: "election"})

	g.AddEdge(gnn.Edge{Source: "donor:mario-rossi", Target: "party:partito-democratico", Weight: 1.0})
	g.AddEdge(gnn.Edge{Source: "person:giorgia-bianchi", Target: "party:partito-democratico", Weight: 0.8})
	g.AddEdge(gnn.Edge{Source: "party:partito-democratico", Target: "election:camera:2022", Weight: 0.19})

	nodeIndex := g.BuildNodeIndex()
	model := gnn.NewGNNModel(g.NumNodes(), 16, 42)
	model.BuildAdjacency(nodeIndex, g.Edges)
	return g, model
}

func TestStoreEmbeddings(t *testing.T) {
	t.Run("stores embeddings for all nodes", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		require.NoError(t, err)
		defer db.Close()

		graph, model := buildTestGraph(t)

		// Run one forward pass so embeddings are initialized
		embeddings := model.Forward()

		err = StoreEmbeddings(db, graph, model)
		require.NoError(t, err)

		rows := readMemoryStore(t, db)
		assert.Len(t, rows, 4, "should have 4 stored embeddings (one per node)")

		// Verify each node has an entry with correct key, node_type, and embedding length
		nodeKeys := make(map[string]bool)
		for _, r := range rows {
			nodeKeys[r.key] = true
			assert.Len(t, r.embedding, 16, "embedding should have expected dimension")
			assert.NotEmpty(t, r.nodeType, "node_type should not be empty")
			assert.NotNil(t, r.metadata, "metadata should not be nil")
		}
		assert.True(t, nodeKeys["party:partito-democratico"])
		assert.True(t, nodeKeys["donor:mario-rossi"])
		assert.True(t, nodeKeys["person:giorgia-bianchi"])
		assert.True(t, nodeKeys["election:camera:2022"])

		// Verify a specific embedding matches model.Forward() output
		nodeIndex := graph.BuildNodeIndex()
		idx := nodeIndex["party:partito-democratico"]
		fwd := embeddings[idx]
		var stored []float64
		for _, r := range rows {
			if r.key == "party:partito-democratico" {
				stored = r.embedding
				break
			}
		}
		require.Len(t, stored, len(fwd))
		for i := range fwd {
			assert.InDelta(t, fwd[i], stored[i], 1e-6, "stored embedding element %d should match", i)
		}
	})

	t.Run("overwrites existing entries on second call", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		require.NoError(t, err)
		defer db.Close()

		graph, model := buildTestGraph(t)
		_ = model.Forward()

		err = StoreEmbeddings(db, graph, model)
		require.NoError(t, err)

		// Train briefly to change embeddings
		posEdges := [][2]int{{2, 0}, {1, 0}} // donor→party, person→party
		negEdges := [][2]int{{0, 3}, {1, 3}} // some negative samples
		trainer := gnn.NewTrainer(model, 0.1)
		trainer.Train(posEdges, negEdges, 5)

		embeddings := model.Forward()
		// Store again (should overwrite)
		err = StoreEmbeddings(db, graph, model)
		require.NoError(t, err)

		rows := readMemoryStore(t, db)
		assert.Len(t, rows, 4, "should still have 4 rows after overwrite")

		nodeIndex := graph.BuildNodeIndex()
		idx := nodeIndex["party:partito-democratico"]
		fwd := embeddings[idx]
		var stored []float64
		for _, r := range rows {
			if r.key == "party:partito-democratico" {
				stored = r.embedding
				break
			}
		}
		require.Len(t, stored, len(fwd))
		// Should match the NEW embeddings, not the old ones
		for i := range fwd {
			assert.InDelta(t, fwd[i], stored[i], 1e-6, "overwritten embedding element %d should match new value", i)
		}
	})

	t.Run("handles empty graph gracefully", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		require.NoError(t, err)
		defer db.Close()

		emptyGraph := gnn.NewGraph()
		emptyModel := gnn.NewGNNModel(0, 16, 42)

		err = StoreEmbeddings(db, emptyGraph, emptyModel)
		require.NoError(t, err)

		rows := readMemoryStore(t, db)
		assert.Len(t, rows, 0, "empty graph should produce no rows")
	})

	t.Run("idempotent – multiple stores produce the same data", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		require.NoError(t, err)
		defer db.Close()

		graph, model := buildTestGraph(t)
		embeddings := model.Forward()

		err = StoreEmbeddings(db, graph, model)
		require.NoError(t, err)

		err = StoreEmbeddings(db, graph, model)
		require.NoError(t, err)

		rows := readMemoryStore(t, db)
		assert.Len(t, rows, 4)

		nodeIndex := graph.BuildNodeIndex()
		idx := nodeIndex["party:partito-democratico"]
		fwd := embeddings[idx]
		var stored []float64
		for _, r := range rows {
			if r.key == "party:partito-democratico" {
				stored = r.embedding
				break
			}
		}
		require.Len(t, stored, len(fwd))
		for i := range fwd {
			assert.InDelta(t, fwd[i], stored[i], 1e-6)
		}
	})

	t.Run("metadata contains JSON with node_type", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		require.NoError(t, err)
		defer db.Close()

		graph, model := buildTestGraph(t)
		_ = model.Forward()

		err = StoreEmbeddings(db, graph, model)
		require.NoError(t, err)

		rows := readMemoryStore(t, db)
		for _, r := range rows {
			var meta map[string]interface{}
			err := json.Unmarshal(r.metadata, &meta)
			require.NoError(t, err, "metadata should be valid JSON for key %s", r.key)
			assert.Equal(t, r.nodeType, meta["node_type"], "node_type column should match metadata.node_type for key %s", r.key)
		}
	})
}

func TestStoreEmbeddingsNilInputs(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	defer db.Close()

	t.Run("nil db returns error", func(t *testing.T) {
		graph, model := buildTestGraph(t)
		err := StoreEmbeddings(nil, graph, model)
		assert.Error(t, err)
	})

	t.Run("nil graph returns error", func(t *testing.T) {
		_, model := buildTestGraph(t)
		err := StoreEmbeddings(db, nil, model)
		assert.Error(t, err)
	})

	t.Run("nil model returns error", func(t *testing.T) {
		graph, _ := buildTestGraph(t)
		err := StoreEmbeddings(db, graph, nil)
		assert.Error(t, err)
	})
}

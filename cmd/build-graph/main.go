package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/ff3300/aleph-v2/internal/graphbuilder"
)

func main() {
	dbPath := flag.String("db", "elections.duckdb", "DuckDB database path")
	dim := flag.Int("dim", 64, "GNN embedding dimension")
	epochs := flag.Int("epochs", 50, "GNN training epochs")
	outDir := flag.String("out", "./graph_output", "Output directory for JSON exports")
	seed := flag.Int64("seed", 42, "Random seed for GNN training")
	flag.Parse()

	log.Printf("Opening DuckDB: %s", *dbPath)
	db, err := sql.Open("duckdb", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open DuckDB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("DuckDB ping failed: %v", err)
	}

	log.Println("Building political knowledge graph...")
	builder := graphbuilder.NewPoliticalGraphBuilder(db)
	builder.Seed = *seed
	if err := builder.Build(); err != nil {
		log.Fatalf("Graph build failed: %v", err)
	}
	log.Printf("Graph built: %d nodes, %d edges", builder.Graph.NumNodes(), builder.Graph.NumEdges())

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	graphExport := builder.ExportGraph()
	graphPath := filepath.Join(*outDir, "graph.json")
	if err := writeJSON(graphPath, graphExport); err != nil {
		log.Fatalf("Failed to write graph.json: %v", err)
	}
	log.Printf("Exported graph to %s", graphPath)

	trends, err := builder.AnalyzeTrends()
	if err != nil {
		log.Fatalf("Trend analysis failed: %v", err)
	}
	trendsPath := filepath.Join(*outDir, "trends.json")
	if err := writeJSON(trendsPath, trends); err != nil {
		log.Fatalf("Failed to write trends.json: %v", err)
	}
	log.Printf("Exported trends to %s", trendsPath)

	log.Printf("Training GNN (dim=%d, epochs=%d)...", *dim, *epochs)
	trainResult, err := builder.TrainGNN(*dim, *epochs)
	if err != nil {
		log.Printf("GNN training skipped: %v", err)
	} else {
		log.Printf("Training complete: AUC=%.4f, MRR=%.4f, loss=%.4f",
			trainResult.AUC, trainResult.MRR, trainResult.FinalLoss)

		embeddings := trainResult.Model.Forward()
		nodeIndex := builder.Graph.BuildNodeIndex()
		predictions := builder.ExportPredictions(trainResult.Model, embeddings, nodeIndex, trainResult.AUC, trainResult.MRR)
		predPath := filepath.Join(*outDir, "predictions.json")
		if err := writeJSON(predPath, predictions); err != nil {
			log.Fatalf("Failed to write predictions.json: %v", err)
		}
		log.Printf("Exported %d predictions to %s", len(predictions.Predictions), predPath)

		log.Print("Storing node embeddings to DuckDB memory_store...")
		if err := graphbuilder.StoreEmbeddings(db, builder.Graph, trainResult.Model); err != nil {
			log.Printf("Embedding storage failed: %v", err)
		} else {
			log.Print("Embeddings stored successfully.")
		}
	}

	log.Println("Done.")
}

func writeJSON(path string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

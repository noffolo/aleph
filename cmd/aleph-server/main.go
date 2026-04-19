package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/rs/cors"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/ff3300/aleph-v2/internal/storage"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	registryManager, err := registry.NewDuckDBRegistry("./aleph_registry.duckdb", logger)
	if err != nil {
		logger.Error("Registry init failed", "error", err)
		os.Exit(1)
	}

	storageDB, err := storage.NewDuckDB("./aleph_registry.duckdb")
	if err != nil {
		logger.Error("Storage init failed", "error", err)
		os.Exit(1)
	}
	
	sandboxManager := sandbox.NewExecSandbox(logger, registryManager, "python3", "go") 

	mux := http.NewServeMux()

	// Root handler for health check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Aleph Backend OK"))
	})
	
	registryPath, registryHandler := v1connect.NewRegistryServiceHandler(handler.NewRegistryServiceHandler(registryManager, logger))
	mux.Handle(registryPath, registryHandler)

	sandboxPath, sandboxHandler := v1connect.NewSandboxServiceHandler(handler.NewSandboxServiceHandler(sandboxManager, logger))
	mux.Handle(sandboxPath, sandboxHandler)

	// Register ProjectService
	projectManager := handler.NewProjectHandler("./data/projects", storageDB)
	projectPath, projectHandler := v1connect.NewProjectServiceHandler(projectManager)
	mux.Handle(projectPath, projectHandler)

	// Setup CORS - Configurato per essere estremamente permissivo per gRPC-web
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "x-grpc-web", "x-user-agent", "x-aleph-api-key"},
		AllowCredentials: true,
		Debug:            true,
	})

	server := &http.Server{Addr: ":8080", Handler: c.Handler(mux)}
	logger.Info("Aleph Backend starting on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

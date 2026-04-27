package routes

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
	"github.com/ff3300/aleph-v2/internal/api/sse"
	"github.com/ff3300/aleph-v2/internal/diagnostic"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
)

// RegisterConfig carries all dependencies needed to register routes.
type RegisterConfig struct {
	MetaRepo          *repository.MetadataRepository
	SSEBroker         *sse.Broker
	SSEHandler        *handler.SSEHandler
	DiagnosticMonitor *diagnostic.DiagnosticMonitor
	Frontend          embed.FS
	CodeFlow          *codeflow.CodeFlow

	// Connect RPC handlers
	QueryHandler        *handler.QueryHandler
	ProjectHandler      *handler.ProjectHandler
	AgentHandler        *handler.AgentHandler
	SkillHandler        *handler.SkillHandler
	LibraryHandler      *handler.LibraryHandler
	ToolHandler         *handler.ToolHandler
	NLPHandler          *handler.NLPHandler
	NotificationHandler *handler.NotificationHandler
	AuthHandler         *handler.AuthHandler
	IngestionHandler    *handler.IngestionHandler
	SandboxHandler      *handler.SandboxServiceHandler
	RegistryHandler     *handler.RegistryServiceHandler

	// Raw HTTP handlers
	ToolExecHandler   *handler.ToolExecuteHandler
	CodeFlowHandler   *handler.CodeFlowHandler
	SuggestPipeline   http.Handler

	Interceptors []connect.HandlerOption
}

// RegisterRoutes registers all HTTP routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, cfg RegisterConfig) {
	// Unauthenticated health check endpoint (for Docker HEALTHCHECK, load balancers, etc.)
	mux.HandleFunc("/api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Connect RPC Routes
	mux.Handle(v1connect.NewQueryServiceHandler(cfg.QueryHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewProjectServiceHandler(cfg.ProjectHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewAgentServiceHandler(cfg.AgentHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewSkillServiceHandler(cfg.SkillHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewToolServiceHandler(cfg.ToolHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewLibraryServiceHandler(cfg.LibraryHandler, cfg.Interceptors...))
	mux.Handle(nlpconnect.NewNLPServiceHandler(cfg.NLPHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewNotificationServiceHandler(cfg.NotificationHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewAuthServiceHandler(cfg.AuthHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewIngestionServiceHandler(cfg.IngestionHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewSandboxServiceHandler(cfg.SandboxHandler, cfg.Interceptors...))
	mux.Handle(v1connect.NewRegistryServiceHandler(cfg.RegistryHandler, cfg.Interceptors...))

	// Raw HTTP routes (protected by AuthMiddleware)
	authMW := func(next http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(cfg.MetaRepo, next)
	}

	mux.Handle("/api/v1/tools/intelligence", authMW(cfg.ToolHandler.ServeHTTP))
	mux.Handle("/api/v1/tools/recommendations", authMW(cfg.ToolHandler.ServeHTTP))
	mux.Handle("/api/v1/tools/health", authMW(cfg.ToolHandler.ServeHTTP))
	mux.Handle("/api/v1/tools/verify", authMW(cfg.ToolHandler.HandleVerify))
	mux.Handle("/api/v1/tools/", authMW(cfg.ToolHandler.HandleHealthHistory))
	mux.Handle("/api/v1/tools", authMW(cfg.ToolHandler.ServeHTTP))

	// Tool suggestion workflow
	mux.Handle("/api/v1/tools/suggest", middleware.AuthMiddleware(cfg.MetaRepo, cfg.SuggestPipeline))
	mux.Handle("/api/v1/tools/suggest/approve", middleware.AuthMiddleware(cfg.MetaRepo, cfg.SuggestPipeline))

	// Tool execution routes
	mux.Handle("/api/v1/tools/categories", authMW(cfg.ToolExecHandler.HandleListCategories))
	mux.Handle("/api/v1/tools/execute/{category}/{name}", authMW(cfg.ToolExecHandler.ServeHTTP))
	mux.Handle("/api/v1/tools/call", authMW(cfg.ToolExecHandler.HandleCallTool))
	mux.Handle("/api/v1/tools/register", authMW(cfg.ToolExecHandler.HandleRegister))

	// CodeFlow routes
	mux.Handle("/api/v1/codeflow/graph", authMW(cfg.CodeFlowHandler.HandleGetGraph))
	mux.Handle("/api/v1/codeflow/metrics", authMW(cfg.CodeFlowHandler.HandleGetMetrics))
	mux.Handle("/api/v1/codeflow/executions", authMW(cfg.CodeFlowHandler.HandleListExecutions))
	mux.Handle("/api/v1/codeflow/engines", authMW(cfg.CodeFlowHandler.HandleListEngines))

	// Diagnostic patterns
	mux.Handle("/api/v1/diagnostic/patterns", authMW(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		patterns := cfg.DiagnosticMonitor.GetPatterns()
		jsonData, err := json.Marshal(patterns)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
	}))

	// SSE endpoint
	mux.HandleFunc("/api/v1/events", cfg.SSEHandler.Stream)

	// API Documentation
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "internal/api/proto/aleph_api.swagger.json")
	})

	// Frontend SPA Hosting with SPA Routing (fallback to index.html)
	subFS, _ := fs.Sub(cfg.Frontend, "dist")
	fileServer := http.FileServer(http.FS(subFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/aleph.v1.") || r.URL.Path == "/swagger.json" {
			mux.ServeHTTP(w, r)
			return
		}
		f, err := subFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		index, err := subFS.Open("index.html")
		if err != nil {
			http.Error(w, "frontend not found", http.StatusNotFound)
			return
		}
		index.Close()
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// CORSHandler wraps the given handler with CORS middleware.
func CORSHandler(next http.Handler, logger interface{ Warn(msg string, args ...any) }) http.Handler {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173,http://localhost:3000"
	}
	originMap := map[string]bool{}
	for _, o := range strings.Split(allowedOrigins, ",") {
		trimmed := strings.TrimSpace(o)
		if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
			logger.Warn("skipping invalid CORS origin", "origin", trimmed)
			continue
		}
		originMap[trimmed] = true
	}

	return corsMiddleware(next, originMap)
}

func corsMiddleware(next http.Handler, origins map[string]bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Aleph-Api-Key, X-Request-Id, X-Project-Id")
		w.Header().Set("Access-Control-Expose-Headers", "Grpc-Status, Grpc-Message")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ProjectsRoot returns the absolute path to the projects data directory.
func ProjectsRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	return filepath.Join(wd, "data", "projects"), nil
}

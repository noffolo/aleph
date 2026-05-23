package routes

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
	"github.com/ff3300/aleph-v2/internal/api/sse"
	"github.com/ff3300/aleph-v2/internal/diagnostic"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/telemetry"
	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
)

// isDraining tracks graceful shutdown state for readiness checks
var isDraining = &atomic.Bool{}

// SetDraining sets the draining flag for graceful shutdown
func SetDraining(draining bool) {
	isDraining.Store(draining)
}

// RegisterConfig carries all dependencies needed to register routes.
type RegisterConfig struct {
	MetaRepo          *repository.MetadataRepository
	JWTSecret         []byte
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
	ToolExecHandler *handler.ToolExecuteHandler
	CodeFlowHandler *handler.CodeFlowHandler
	SuggestPipeline http.Handler
	SessionHandler  *handler.SessionHandler
	AuthRateLimiter *middleware.AuthRateLimiter

	IngestionHealthHandler http.Handler

	Interceptors []connect.HandlerOption

	// HealthCheckFunc is an optional function for DB-backed liveness/health probes.
	// When set, /livez and /api/v1/healthz will call it with a 2s timeout and
	// return 503 if it returns an error.
	HealthCheckFunc func(ctx context.Context) error
}

// RegisterRoutes registers all HTTP routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, cfg RegisterConfig) {
	// Readiness probe: returns 503 during graceful drain
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if isDraining.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"draining"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Liveness probe: lightweight check with DB health
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if cfg.HealthCheckFunc != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := cfg.HealthCheckFunc(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status":"unhealthy","reason":"` + err.Error() + `"}`))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	})

	// Unauthenticated health check endpoint (for Docker HEALTHCHECK, load balancers, etc.)
	mux.HandleFunc("/api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if cfg.HealthCheckFunc != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := cfg.HealthCheckFunc(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status":"unhealthy","reason":"` + err.Error() + `"}`))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.Handle("/metrics", telemetry.MetricsHandler())

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

	// Session management (unauthenticated — validates credentials then sets cookie)
	// Rate-limited: 5 req/min per IP to prevent brute-force attacks
	sessionHandler := cfg.SessionHandler
	sessionMux := http.NewServeMux()
	sessionMux.HandleFunc("/api/v1/auth/session", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			cfg.AuthRateLimiter.RateLimitHTTPFunc("session_create", sessionHandler.HandleCreateSession)(w, r)
		case http.MethodGet:
			sessionHandler.HandleValidateSession(w, r)
		case http.MethodDelete:
			sessionHandler.HandleDeleteSession(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.Handle("/api/v1/auth/session", sessionMux)

	// Raw HTTP routes (protected by AuthMiddleware + RBAC)
	authMW := func(next http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(cfg.MetaRepo, cfg.JWTSecret, next)
	}

	adminOnly := middleware.RequireRoleHTTP(middleware.RoleAdmin)
	readWrite := middleware.RequireRoleHTTP(middleware.RoleAdmin, middleware.RoleUser)
	readAny := middleware.RequireRoleHTTP(middleware.RoleAdmin, middleware.RoleUser, middleware.RoleReadOnly)

	mux.Handle("/api/v1/tools/intelligence", readAny(authMW(cfg.ToolHandler.ServeHTTP)))
	mux.Handle("/api/v1/tools/recommendations", readAny(authMW(cfg.ToolHandler.ServeHTTP)))
	mux.Handle("/api/v1/tools/health", readAny(authMW(cfg.ToolHandler.ServeHTTP)))
	mux.Handle("/api/v1/tools/verify", readWrite(authMW(cfg.ToolHandler.HandleVerify)))
	mux.Handle("/api/v1/tools/", readAny(authMW(cfg.ToolHandler.HandleHealthHistory)))
	mux.Handle("/api/v1/tools", readAny(authMW(cfg.ToolHandler.ServeHTTP)))

	// Tool suggestion workflow
	suggestHandler := middleware.AuthMiddleware(cfg.MetaRepo, cfg.JWTSecret, cfg.SuggestPipeline)
	mux.Handle("/api/v1/tools/suggest", readWrite(suggestHandler))
	mux.Handle("/api/v1/tools/suggest/approve", adminOnly(suggestHandler))

	// Tool execution routes
	mux.Handle("/api/v1/tools/categories", readAny(authMW(cfg.ToolExecHandler.HandleListCategories)))
	mux.Handle("/api/v1/tools/execute/{category}/{name}", readWrite(authMW(cfg.ToolExecHandler.ServeHTTP)))
	mux.Handle("/api/v1/tools/call", readWrite(authMW(cfg.ToolExecHandler.HandleCallTool)))
	mux.Handle("/api/v1/tools/register", adminOnly(authMW(cfg.ToolExecHandler.HandleRegister)))

	// CodeFlow routes
	mux.Handle("/api/v1/codeflow/graph", readAny(authMW(cfg.CodeFlowHandler.HandleGetGraph)))
	mux.Handle("/api/v1/codeflow/metrics", readAny(authMW(cfg.CodeFlowHandler.HandleGetMetrics)))
	mux.Handle("/api/v1/codeflow/executions", readAny(authMW(cfg.CodeFlowHandler.HandleListExecutions)))
	mux.Handle("/api/v1/codeflow/engines", readAny(authMW(cfg.CodeFlowHandler.HandleListEngines)))

	// Ontology Negotiation routes (W2C-01)
	mux.Handle("/api/v1/ontology/propose", readWrite(authMW(cfg.ProjectHandler.NegotiatePropose)))
	mux.Handle("/api/v1/ontology/accept", readWrite(authMW(cfg.ProjectHandler.NegotiateAccept)))
	mux.Handle("/api/v1/ontology/reject", readWrite(authMW(cfg.ProjectHandler.NegotiateReject)))
	mux.Handle("/api/v1/ontology/versions", readAny(authMW(cfg.ProjectHandler.NegotiateList)))

	// Diagnostic patterns
	mux.Handle("/api/v1/diagnostic/patterns", readAny(authMW(func(w http.ResponseWriter, r *http.Request) {
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
	})))

	// SSE endpoint
	mux.HandleFunc("/api/v1/events", cfg.SSEHandler.Stream)

	if cfg.IngestionHealthHandler != nil {
		mux.Handle("/api/health/ingestion", cfg.IngestionHealthHandler)
	}

	// API Documentation
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "internal/api/proto/aleph_api.swagger.json")
	})

	// Frontend SPA Hosting with SPA Routing (fallback to index.html)
	subFS, _ := fs.Sub(cfg.Frontend, "dist")
	fileServer := http.FileServer(http.FS(subFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// NOTE: gRPC (aleph.v1.*) and /swagger.json are handled by more specific
		// mux patterns above. If a request reaches here, it's genuinely unmatched
		// (not an RPC or swagger path), so we treat it as a frontend SPA request.
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
func CORSHandler(next http.Handler, allowedOrigins []string, logger interface{ Warn(msg string, args ...any) }) http.Handler {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:5173", "http://localhost:3000"}
	}
	originMap := map[string]bool{}
	for _, o := range allowedOrigins {
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

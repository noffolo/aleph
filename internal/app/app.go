package app

import (
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	_ "github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/config"
	"github.com/ff3300/aleph-v2/internal/diagnostic"
	"github.com/ff3300/aleph-v2/internal/health"
	"github.com/ff3300/aleph-v2/internal/ingestion"
	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/migrate"
	"github.com/ff3300/aleph-v2/internal/nlp_adapter"
	"github.com/ff3300/aleph-v2/internal/predict"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/routes"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	"github.com/ff3300/aleph-v2/internal/service/notification"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/ff3300/aleph-v2/internal/telemetry"
	"github.com/ff3300/aleph-v2/internal/tools/adaptation"
	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
	"github.com/ff3300/aleph-v2/internal/tools/humanecosystems"
	"github.com/ff3300/aleph-v2/internal/tools/osint"
	"log/slog"
)

type AlephApp struct {
	db           *storage.DuckDB
	pg           *storage.Postgres
	cfg          *config.Config
	eng          *ingestion.Engine
	metaRepo     *repository.MetadataRepository
	frontend     embed.FS
	server       *http.Server
	logger       *slog.Logger
	brierMonitor *predict.BrierMonitor
	nlpHandler   *handler.NLPHandler
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewAlephApp(cfg *config.Config, frontend embed.FS) (*AlephApp, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Data DB (DuckDB) - Analytic Engine
	db, err := storage.NewDuckDB(cfg.DuckDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %v", err)
	}
	db.Exec("PRAGMA memory_limit='80%'")

	// Run both DuckDB and PostgreSQL migrations before any table creation
	if err := migrate.RunAllMigrations(cfg.DuckDBPath, cfg.PostgresDSN); err != nil {
		log.Printf("Warning: Some migrations failed: %v", err)
		// Continue without migrations for backward compatibility
	}

	// System DB (PostgreSQL) - System Records & Consistency
	pg, err := storage.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %v", err)
	}

	metaRepo, err := repository.NewMetadataRepository(pg.DB())
	if err != nil {
		return nil, fmt.Errorf("failed to init metadata repo: %v", err)
	}

	wd, _ := os.Getwd()
	projectsRoot := filepath.Join(wd, "data", "projects")
	
	nlpAddr := cfg.NLPAddr
	if nlpAddr == "" { nlpAddr = "http://localhost:8001" }
	if !strings.HasPrefix(nlpAddr, "http") { nlpAddr = "http://" + nlpAddr }
	h2cClient := newH2CClient()
	nlpClient := nlpconnect.NewNLPServiceClient(h2cClient, nlpAddr, connect.WithGRPC())
	nlpHandler := handler.NewNLPHandler(logger, nlpClient, h2cClient)
	nlpAdapter := &nlp_adapter.Adapter{NLPHandler: nlpHandler}
	
	eng := ingestion.NewEngine(projectsRoot, metaRepo, db, nlpAdapter)
	
	brierMonitor := predict.NewBrierMonitor(logger)
	nlpHandler.SetBrierMonitor(brierMonitor)

	ctx, cancel := context.WithCancel(context.Background())

	return &AlephApp{
		db:           db,
		pg:           pg,
		cfg:          cfg,
		eng:          eng,
		metaRepo:     metaRepo,
		frontend:     frontend,
		logger:       logger,
		brierMonitor: brierMonitor,
		nlpHandler:   nlpHandler,
		ctx:          ctx,
		cancel:       cancel,
	}, nil
}


func (a *AlephApp) Serve(port int) error {
	projectsRoot, _ := routes.ProjectsRoot()

	// ── W3 Interceptors ──────────────────────────────────────────────────────
	errorHandler := middleware.NewErrorHandlerInterceptor()
	auditRepo := repository.NewAuditRepository(a.pg.DB())
	auditInterceptor := middleware.NewAuditInterceptor(auditRepo, a.logger)
	authInterceptor := middleware.NewAuthInterceptor(a.metaRepo)
	timeoutInterceptor := middleware.NewTimeoutInterceptor(nil) // defaults
	retryInterceptor := middleware.NewRetryInterceptor(nil)     // defaults
	bulkheadInterceptor := middleware.NewBulkheadInterceptor(nil) // defaults

	interceptors := []connect.HandlerOption{
		connect.WithInterceptors(
			errorHandler,
			auditInterceptor,
			authInterceptor,
			timeoutInterceptor,
			retryInterceptor,
			bulkheadInterceptor,
		),
	}

	// ── Subsystem Startups ──────────────────────────────────────────────────
	// Health checker
	healthChecker := health.NewHealthChecker(a.logger, a.metaRepo)
	go healthChecker.Start(a.ctx)

	// Diagnostic monitor
	diagHealthInt := &diagnostic.HealthIntegration{
		GetConsecutiveFailures: func(toolID string) int { return healthChecker.ConsecutiveFailures(toolID) },
		GetToolHealthStatus:    func(toolID string) string { return healthChecker.GetLatestStatus(toolID) },
	}
	diagnosticMonitor := diagnostic.NewDiagnosticMonitor(3, diagHealthInt)

	// MCP discovery engine
	discoveryConfig := mcp.DiscoveryConfig{
		ServerURIs:  []string{}, // populated from config in future
		HealthCheck: 5 * time.Minute,
	}
	discoveryEngine := mcp.NewDiscoveryEngine(a.logger, a.metaRepo, discoveryConfig)
	go discoveryEngine.Start(a.ctx)

	// ── Handlers ─────────────────────────────────────────────────────────────
	registryMgr, _ := registry.NewDuckDBRegistryFromDuckDB(a.db, a.logger)
	queryHandler := handler.NewQueryHandler(a.db, projectsRoot, a.metaRepo, a.nlpHandler, registryMgr)
	projectHandler := handler.NewProjectHandler(projectsRoot, a.db)
	agentHandler := handler.NewAgentHandler(projectsRoot, a.metaRepo, a.cfg.OllamaBaseURL)
	skillHandler := handler.NewSkillHandler(projectsRoot, a.metaRepo)
	toolHandler := handler.NewToolHandler(projectsRoot, a.metaRepo)
	libraryHandler := handler.NewLibraryHandler(projectsRoot)

	notificationSvc := notification.NewNotificationService()
	notificationHandler := handler.NewNotificationHandler(notificationSvc, a.metaRepo)

	authHandler := handler.NewAuthHandler(a.metaRepo)
	ingestionHandler := handler.NewIngestionHandler(projectsRoot, a.eng, a.metaRepo)
	sandboxManager := sandbox.NewExecSandbox(a.logger, nil, a.metaRepo, "python3", "go")
	sandboxHandler := handler.NewSandboxServiceHandler(sandboxManager, a.logger)
	registryHandler := handler.NewRegistryServiceHandler(registryMgr, a.logger)

	// ── Tool Execution & CodeFlow ─────────────────────────────────────────────
	codeFlow := codeflow.NewCodeFlow()
	shadowbroker := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)
	duckdbLayer := humanecosystems.NewDuckDBLayer(a.db)
	toolExecHandler := handler.NewToolExecuteHandler(a.metaRepo, shadowbroker, duckdbLayer)
	codeFlowHandler := handler.NewCodeFlowHandler(codeFlow)

	// ── Tool Suggestion Pipeline ──────────────────────────────────────────────
	suggestPipeline := adaptation.NewPipeline(a.metaRepo)
	toolSuggestHandler := handler.NewToolSuggestHandler(discoveryEngine, suggestPipeline, a.cfg.MCPServerURIs)

	// ── Routes ───────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux, routes.RegisterConfig{
		MetaRepo:          a.metaRepo,
		SSEBroker:         nil, // TODO: wire SSE broker
		SSEHandler:        nil, // TODO: wire SSE handler
		DiagnosticMonitor: diagnosticMonitor,
		Frontend:          a.frontend,
		CodeFlow:          codeFlow,
		QueryHandler:      queryHandler,
		ProjectHandler:    projectHandler,
		AgentHandler:      agentHandler,
		SkillHandler:      skillHandler,
		LibraryHandler:    libraryHandler,
		ToolHandler:       toolHandler,
		NLPHandler:        a.nlpHandler,
		NotificationHandler: notificationHandler,
		AuthHandler:       authHandler,
		IngestionHandler:  ingestionHandler,
		SandboxHandler:    sandboxHandler,
		RegistryHandler:   registryHandler,
		ToolExecHandler:   toolExecHandler,
		CodeFlowHandler:   codeFlowHandler,
		SuggestPipeline:   toolSuggestHandler,
		Interceptors:      interceptors,
	})

	corsHandler := routes.CORSHandler(mux, a.logger)
	telemetryHandler := telemetry.Middleware(corsHandler)

	go a.watchSidecar(a.nlpHandler)

	log.Printf("[Aleph] Data OS starting on :%d", port)
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h2c.NewHandler(telemetryHandler, &http2.Server{}),
	}

	return a.server.ListenAndServe()
}

func (a *AlephApp) Close(ctx context.Context) error {
	log.Println("[Aleph] Shutting down services...")
	if a.cancel != nil {
		a.cancel()
	}
	if a.server != nil {
		a.server.Shutdown(ctx)
	}
	if a.nlpHandler != nil {
		a.nlpHandler.Close()
	}
	a.eng.Close()
	a.pg.Close()
	return a.db.Close()
}

func newH2CClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
}

func (a *AlephApp) watchSidecar(nlpHandler *handler.NLPHandler) {
	addr := a.cfg.NLPAddr
	if strings.HasPrefix(addr, "http") {
		addr = strings.TrimPrefix(addr, "http://")
		addr = strings.TrimPrefix(addr, "https://")
	}
	
	slog.Info("avvio monitoraggio neurale", "addr", addr)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("connessione al sidecar fallita", "error", err)
		return
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			slog.Info("sidecar monitor stopped")
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "aleph.nlp.v1.NLPService"})
			cancel()
			if err != nil {
				slog.Warn("sidecar non risponde", "error", err)
			} else if resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
				nlpHandler.MarkHealthy()
				slog.Info("sidecar neurale operativo")
			} else {
				slog.Warn("sidecar non SERVING", "status", resp.GetStatus())
			}
		}
	}
}

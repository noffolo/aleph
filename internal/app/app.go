package app

import (
	"context"
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/ff3300/aleph-v2/internal/api/handler"
	alephv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/api/sse"
	"github.com/ff3300/aleph-v2/internal/config"
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/diagnostic"
	"github.com/ff3300/aleph-v2/internal/health"
	"github.com/ff3300/aleph-v2/internal/ingestion"
	"github.com/ff3300/aleph-v2/internal/mcp"
	"github.com/ff3300/aleph-v2/internal/memory"
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

	healthChecker    *health.HealthChecker
	discoveryEngine  *mcp.DiscoveryEngine
	notificationSvc  *notification.NotificationService
	sseBroker        *sse.Broker
	rlCleanup        func()
	memStore         *memory.MemoryStore
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
	metaRepo.SetEncryptionKey(cfg.EncryptionKey)

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
	a.healthChecker = health.NewHealthChecker(a.logger, a.metaRepo)
	go a.healthChecker.Start(a.ctx)

	// Diagnostic monitor
	diagHealthInt := &diagnostic.HealthIntegration{
		GetConsecutiveFailures: func(toolID string) int { return a.healthChecker.ConsecutiveFailures(toolID) },
		GetToolHealthStatus:    func(toolID string) string { return a.healthChecker.GetLatestStatus(toolID) },
	}
	diagnosticMonitor := diagnostic.NewDiagnosticMonitor(3, diagHealthInt)

	// MCP discovery engine
	discoveryConfig := mcp.DiscoveryConfig{
		ServerURIs:  []string{}, // populated from config in future
		HealthCheck: 5 * time.Minute,
	}
	a.discoveryEngine = mcp.NewDiscoveryEngine(a.logger, a.metaRepo, discoveryConfig)
	go a.discoveryEngine.Start(a.ctx)

	// ── Handlers ─────────────────────────────────────────────────────────────
	registryMgr, _ := registry.NewDuckDBRegistryFromDuckDB(a.db, a.logger)
	queryHandler := handler.NewQueryHandler(a.db, projectsRoot, a.metaRepo, a.nlpHandler, registryMgr)
	projectHandler := handler.NewProjectHandler(projectsRoot, a.db)
	agentHandler := handler.NewAgentHandler(projectsRoot, a.metaRepo, a.cfg.OllamaBaseURL)
	skillHandler := handler.NewSkillHandler(projectsRoot, a.metaRepo)
	toolHandler := handler.NewToolHandler(projectsRoot, a.metaRepo)
	libraryHandler := handler.NewLibraryHandler(projectsRoot)

	a.notificationSvc = notification.NewNotificationService()
	notificationHandler := handler.NewNotificationHandler(a.notificationSvc, a.metaRepo)

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

	// ── Memory Subsystem (W4W6) ──────────────────────────────────────────────
	memStore, mErr := memory.NewMemoryStore(a.db.DB(), a.cfg.DuckDBSchema, 768)
	if mErr != nil {
		a.logger.Warn("memory store init failed (degraded)", "err", mErr)
		memStore = nil
	}
	a.memStore = memStore

	// ── Decision Engine Wiring (W4W6) ───────────────────────────────────────
	decision.NewToolExecutor = func(
		executeQuery func(ctx context.Context, req *connect.Request[alephv1.ExecuteQueryRequest]) (*connect.Response[alephv1.ExecuteQueryResponse], error),
		analyzeSentiment func(ctx context.Context, text string) (string, error),
		getTrustScore func(ctx context.Context, entityID string) (string, error),
		getComponentByID func(id string) (*decision.ComponentMetadata, error),
	) decision.ToolExecutor {
		return handler.CreateToolExecutor(executeQuery, analyzeSentiment, getTrustScore, getComponentByID)
	}

	metaRepoAdapter := &decision.MetaRepoAdapter{Repo: a.metaRepo}
	registryAdapter := &decision.RegistryAdapter{Reg: registryMgr}

	helperExec := handler.CreateToolExecutor(
		queryHandler.ExecuteQuery,
		a.makeSentimentHelper(),
		a.makeTrustScoreHelper(registryMgr),
		a.makeComponentByIDHelper(registryMgr),
	)

	engineCfg := decision.EngineConfig{
		Provider:    nil,
		MetaRepo:    metaRepoAdapter,
		Executor:    helperExec,
		Registry:    registryAdapter,
		MaxAttempts: 5,
	}

	// ── GNN Link Predictor (optional, epistemic trust) ───────────────────
	gnnPredictor := decision.NewGNNLinkPredictor(100, 64, 0.01)
	engineCfg.LinkPredictor = gnnPredictor

	decisionEngine := decision.NewEngine(engineCfg)

	queryHandler.SetDecisionEngine(decisionEngine, helperExec)

	// ── Tool Suggestion Pipeline ──────────────────────────────────────────────
	suggestPipeline := adaptation.NewPipeline(a.metaRepo)
	toolSuggestHandler := handler.NewToolSuggestHandler(a.discoveryEngine, suggestPipeline, a.cfg.MCPServerURIs)

	// ── SSE Broker (W2-04) ──────────────────────────────────────────────────
	a.sseBroker = sse.NewBroker(30*time.Second, a.logger)
	sseHandler := handler.NewSSEHandler(a.sseBroker, a.logger)

	// ── Routes ───────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux, routes.RegisterConfig{
		MetaRepo:          a.metaRepo,
		SSEBroker:         a.sseBroker,
		SSEHandler:        sseHandler,
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

	corsHandler := routes.CORSHandler(mux, a.cfg.CORSAllowedOrigins, a.logger)
	telemetryHandler := telemetry.Middleware(corsHandler)
	recoveryHandler := middleware.Recovery(telemetryHandler)
	secureHandler := middleware.SecurityHeaders(recoveryHandler)
	ridHandler := middleware.RequestID(secureHandler)

	rateLimitCfg := middleware.RateLimitConfig{
		ChatLimit:    rate.Limit(a.cfg.RateLimitChat) / 60.0,
		HealthLimit:  rate.Limit(a.cfg.RateLimitHealth) / 60.0,
		DefaultLimit: rate.Limit(a.cfg.RateLimitDefault) / 60.0,
		ChatBurst:    5,
		HealthBurst:  20,
		DefaultBurst: 50,
	}
	rateLimitMw, rateLimitStop := middleware.RateLimitMiddleware(&rateLimitCfg)
	a.rlCleanup = rateLimitStop
	rateLimitedHandler := rateLimitMw(ridHandler)

	promHandler := telemetry.PrometheusMiddleware(rateLimitedHandler)

	go a.watchSidecar(a.nlpHandler)

	log.Printf("[Aleph] Data OS starting on :%d", port)
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h2c.NewHandler(promHandler, &http2.Server{}),
	}

	return a.server.ListenAndServe()
}

func (a *AlephApp) Close(ctx context.Context) error {
	log.Println("[Aleph] Shutting down services...")

	// Stop goroutine-backed services first (W2-01)
	if a.healthChecker != nil {
		a.healthChecker.Stop()
	}
	if a.discoveryEngine != nil {
		a.discoveryEngine.Stop()
	}
	if a.notificationSvc != nil {
		a.notificationSvc.Stop()
	}
	if a.sseBroker != nil {
		a.sseBroker.Close()
	}
	if a.rlCleanup != nil {
		a.rlCleanup()
	}
	if a.memStore != nil {
		a.memStore.Close()
	}

	// Cancel root context → stops watchSidecar, enrichment goroutines, etc.
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

func (a *AlephApp) makeSentimentHelper() func(ctx context.Context, text string) (string, error) {
	return func(ctx context.Context, text string) (string, error) {
		if a.nlpHandler == nil {
			slog.Warn("sentiment analysis unavailable — NLP sidecar not configured")
			return `{"score": 0, "label": "neutral", "error": "NLP sidecar non disponibile"}`, nil
		}
		resp, err := a.nlpHandler.AnalyzeSentiment(ctx, connect.NewRequest(&nlpv1.AnalyzeSentimentRequest{Text: text}))
		if err != nil {
			slog.Warn("sentiment analysis failed", "err", err)
			return "", fmt.Errorf("Errore analisi sentiment: %v", err)
		}
		result := map[string]interface{}{
			"score": resp.Msg.Score,
			"label": resp.Msg.Label,
		}
		jb, _ := json.Marshal(result)
		return string(jb), nil
	}
}

func (a *AlephApp) makeTrustScoreHelper(reg *registry.DuckDBRegistry) func(ctx context.Context, entityID string) (string, error) {
	return func(ctx context.Context, entityID string) (string, error) {
		if reg == nil {
			return `{"error": "registry non disponibile"}`, nil
		}
		comp, err := reg.GetComponentByID(ctx, entityID)
		if err != nil || comp == nil {
			return "", fmt.Errorf("entità %s non trovata", entityID)
		}
		result := map[string]interface{}{
			"entity_id":       entityID,
			"avg_brier_score": comp.AvgBrierScore,
			"trust_score":     comp.TrustScore,
		}
		jb, _ := json.Marshal(result)
		return string(jb), nil
	}
}

func (a *AlephApp) makeComponentByIDHelper(reg *registry.DuckDBRegistry) func(id string) (*decision.ComponentMetadata, error) {
	return func(id string) (*decision.ComponentMetadata, error) {
		if reg == nil {
			return nil, fmt.Errorf("registry non disponibile")
		}
		comp, err := reg.GetComponentByID(context.Background(), id)
		if err != nil || comp == nil {
			return nil, err
		}
		return &decision.ComponentMetadata{
			ID:       comp.ID,
			Name:     comp.Name,
			Category: comp.Category,
			Status:   comp.Status,
		}, nil
	}
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

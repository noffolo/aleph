package app

import (
	"context"
	"crypto/tls"
	"embed"
	"encoding/json"
	"errors"
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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/ff3300/aleph-v2/internal/api/handler"
	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/api/sse"
	"github.com/ff3300/aleph-v2/internal/config"
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/diagnostic"
	"github.com/ff3300/aleph-v2/internal/health"
	"github.com/ff3300/aleph-v2/internal/ingestion"
	"github.com/ff3300/aleph-v2/internal/llm"
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
	"github.com/ff3300/aleph-v2/internal/service/tracker"
	"github.com/ff3300/aleph-v2/internal/ssrf"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/ff3300/aleph-v2/internal/telemetry"
	"github.com/ff3300/aleph-v2/internal/tools/adaptation"
	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
	"github.com/ff3300/aleph-v2/internal/tools/humanecosystems"
	"github.com/ff3300/aleph-v2/internal/tools/osint"
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// sidecar health-check constants — extracted for testability.
var (
	sidecarMaxRestarts   = 3
	sidecarRestartWindow = 5 * time.Minute
	sidecarBackoffSteps  = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
	sidecarCheckInterval = 5 * time.Second
	sidecarCheckTimeout  = 3 * time.Second
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
	authRlCleanup    func()
	memStore         *memory.MemoryStore
	usageTracker     tracker.Tracker
}

func NewAlephApp(cfg *config.Config, frontend embed.FS) (*AlephApp, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:         dsn,
			Environment: "production",
			ServerName:  "aleph-backend",
			Release:     os.Getenv("APP_VERSION"),
		}); err != nil {
			slog.Warn("sentry init failed", "error", err)
		} else {
			slog.Info("sentry error monitoring initialized")
		}
	}

	// Data DB (DuckDB) - Analytic Engine
	db, err := storage.NewDuckDB(cfg.DuckDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}
	func() {
		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		defer cancel()
		db.Exec(ctx, "PRAGMA memory_limit='80%'")
	}()
	if cfg.SlowQueryThresholdMs > 0 {
		db.SetSlowQueryThreshold(time.Duration(cfg.SlowQueryThresholdMs) * time.Millisecond)
	}

	// Run both DuckDB and PostgreSQL migrations before any table creation
	if err := migrate.RunAllMigrations(cfg.DuckDBPath, cfg.PostgresDSN); err != nil {
		log.Printf("Warning: Some migrations failed: %v", err)
		// Continue without migrations for backward compatibility
	}

	// Usage Tracking (W1.5-04)
	usageTracker := tracker.NewDuckDBTracker(db.DB())

	// System DB (PostgreSQL) - System Records & Consistency
	pg, err := storage.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	metaRepo, err := repository.NewMetadataRepository(pg.DB())
	if err != nil {
		return nil, fmt.Errorf("failed to init metadata repo: %w", err)
	}
	metaRepo.SetEncryptionKey(cfg.EncryptionKey)

	llm.OllamaPort = cfg.OllamaPort

	wd, _ := os.Getwd()
	projectsRoot := filepath.Join(wd, "data", "projects")
	
	nlpAddr := cfg.NLPAddr

	var httpClient *http.Client
	if cfg.DevMode {
		if !strings.HasPrefix(nlpAddr, "http") { nlpAddr = "http://" + nlpAddr }
		httpClient = newH2CClient()
	} else {
		if !strings.HasPrefix(nlpAddr, "http") { nlpAddr = "https://" + nlpAddr }
		httpClient = newTLSClient()
	}
	nlpClient := nlpconnect.NewNLPServiceClient(httpClient, nlpAddr, connect.WithGRPC())
	nlpHandler := handler.NewNLPHandler(logger, nlpClient, httpClient)
	nlpAdapter := &nlp_adapter.Adapter{NLPHandler: nlpHandler}
	
	eng := ingestion.NewEngine(projectsRoot, metaRepo, db, nlpAdapter)
	
	brierMonitor := predict.NewBrierMonitor(logger)
	nlpHandler.SetBrierMonitor(brierMonitor)

	// context.Background is the root context for the Aleph application lifecycle.
	// All subsystem contexts derive from this via WithCancel during Start().
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
		usageTracker: usageTracker,
	}, nil
}


func (a *AlephApp) Serve(port int) error {
	projectsRoot, _ := routes.ProjectsRoot()

	// ── W3 Interceptors ──────────────────────────────────────────────────────
	errorHandler := middleware.NewErrorHandlerInterceptor()
	subsystemInterceptor := middleware.NewSubsystemInterceptor()
	auditRepo := repository.NewAuditRepository(a.pg.DB())
	auditInterceptor := middleware.NewAuditInterceptor(auditRepo, a.logger)
	authInterceptor := middleware.NewAuthInterceptor(a.metaRepo, a.cfg.JWTSecret)
	timeoutInterceptor := middleware.NewTimeoutInterceptor(nil) // defaults
	retryInterceptor := middleware.NewRetryInterceptor(nil)     // defaults
	bulkheadInterceptor := middleware.NewBulkheadInterceptor(nil) // defaults
	circuitBreakerInterceptor := middleware.NewCircuitBreakerInterceptor(5, 30*time.Second)
	trackingInterceptor := tracker.NewTrackingInterceptor(a.usageTracker)

	authRateLimiter := middleware.NewAuthRateLimiter(nil, middleware.DefaultAuthRateLimitConfig)
	a.authRlCleanup = authRateLimiter.Close
	authRateLimitInterceptor := authRateLimiter.RateLimitInterceptor()

	interceptors := []connect.HandlerOption{
		connect.WithInterceptors(
			subsystemInterceptor,
			errorHandler,
			auditInterceptor,
			authInterceptor,
			authRateLimitInterceptor,
			timeoutInterceptor,
			retryInterceptor,
			bulkheadInterceptor,
			circuitBreakerInterceptor,
			trackingInterceptor,
		),
	}

	// ── Subsystem Startups ──────────────────────────────────────────────────
	// Health checker
	a.healthChecker = health.NewHealthChecker(a.ctx, a.logger, a.metaRepo)
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

	// Security scanner — runs once at startup to audit tool code
	go a.runSecurityScan()

	// ── Handlers ─────────────────────────────────────────────────────────────
	registryMgr, _ := registry.NewDuckDBRegistryFromDuckDB(a.db, a.logger)
	queryHandler := handler.NewQueryHandler(a.db, projectsRoot, a.metaRepo, a.nlpHandler, registryMgr, time.Duration(a.cfg.LLMTimeoutSeconds)*time.Second)
	projectHandler := handler.NewProjectHandler(projectsRoot, a.db)
	projectHandler.SetMetaRepo(a.metaRepo)
	projectHandler.SetMaxProjects(a.cfg.MaxProjects)
	ontoRepo := repository.NewOntologyRepository(a.pg.DB())
	projectHandler.SetOntologyRepository(ontoRepo)
	agentHandler := handler.NewAgentHandler(projectsRoot, a.metaRepo, a.cfg.OllamaBaseURL)
	agentHandler.SetMaxAgentsPerProject(a.cfg.MaxAgentsPerProject)
	skillHandler := handler.NewSkillHandler(projectsRoot, a.metaRepo)
	toolHandler := handler.NewToolHandler(projectsRoot, a.metaRepo)
	libraryHandler := handler.NewLibraryHandler(projectsRoot)

	a.notificationSvc = notification.NewNotificationService()
	notificationHandler := handler.NewNotificationHandler(a.notificationSvc, a.metaRepo)

	authHandler := handler.NewAuthHandler(a.metaRepo)
	sessionHandler := handler.NewSessionHandler(a.metaRepo, a.cfg.JWTSecret).WithRevocationStore(authInterceptor.RevocationStore())
	ingestionHandler := handler.NewIngestionHandler(projectsRoot, a.eng, a.metaRepo)
	sandboxManager := sandbox.NewContainerSandbox(a.logger, nil, a.metaRepo, sandbox.DefaultContainerConfig(), nil)
	sandboxHandler := handler.NewSandboxServiceHandler(sandboxManager, a.logger)
	registryHandler := handler.NewRegistryServiceHandler(registryMgr, a.logger)

	// ── Tool Execution & CodeFlow ─────────────────────────────────────────────
	codeFlow := codeflow.NewCodeFlow()
	shadowbroker := osint.NewShadowbroker(osint.ShadowbrokerConfig{})
	duckdbLayer := humanecosystems.NewDuckDBLayer(a.db)
	toolExecHandler := handler.NewToolExecuteHandler(a.metaRepo, shadowbroker, duckdbLayer)
	codeFlowHandler := handler.NewCodeFlowHandler(codeFlow)

	// ── Memory Subsystem (W4W6) ──────────────────────────────────────────────
	memStore, mErr := memory.NewMemoryStore(a.db, a.cfg.DuckDBSchema, 768)
	if mErr != nil {
		a.logger.Warn("memory store init failed (degraded)", "err", mErr)
		memStore = nil
	}
	a.memStore = memStore

	// ── Decision Engine Wiring (W4W6) ───────────────────────────────────────
	metaRepoAdapter := &decision.MetaRepoAdapter{Repo: a.metaRepo}
	registryAdapter := &decision.RegistryAdapter{Reg: registryMgr}

	helperExec := handler.NewHandlerToolExecutor(
		queryHandler.ExecuteQuery,
		a.nlpHandler,
		registryMgr,
	)

	llmProvider, providerErr := llm.NewProvider("ollama", a.cfg.OllamaBaseURL, ssrf.NewClient(), time.Duration(a.cfg.LLMTimeoutSeconds)*time.Second)
	if providerErr != nil {
		a.logger.Warn("LLM provider init failed (degraded mode)", "error", providerErr)
		llmProvider = nil
	}

	engineCfg := decision.EngineConfig{
		Provider:    llmProvider,
		MetaRepo:    metaRepoAdapter,
		Executor:    helperExec,
		Registry:    registryAdapter,
		MaxAttempts: 5,
	}

	// ── GNN Link Predictor (optional, epistemic trust) ───────────────────
	gnnPredictor := decision.NewGNNLinkPredictor(100, 64, 0.01)
	engineCfg.LinkPredictor = gnnPredictor

	decisionEngine := decision.NewEngine(engineCfg)

	projectHandler.SetLLMProvider(engineCfg.Provider)

	queryHandler.SetDecisionEngine(decisionEngine, helperExec)

	if a.memStore != nil {
		queryHandler.SetMemoryStore(a.memStore)
	}

	// ── Tool Suggestion Pipeline ──────────────────────────────────────────────
	suggestPipeline := adaptation.NewPipeline(a.metaRepo)
	toolSuggestHandler := handler.NewToolSuggestHandler(a.discoveryEngine, suggestPipeline, a.cfg.MCPServerURIs)

	// ── DuckDB Auto-Backup (B14) ────────────────────────────────────────────
	{
		interval, err := time.ParseDuration(a.cfg.BackupInterval)
		if err != nil || interval <= 0 {
			interval = 24 * time.Hour
			a.logger.Warn("invalid BACKUP_INTERVAL, using default",
				"configured", a.cfg.BackupInterval, "fallback", interval)
		}
		backupDir := a.cfg.BackupDir
		if backupDir == "" {
			backupDir = filepath.Join(filepath.Dir(a.cfg.DataRoot), "backups", "duckdb")
		}
		if a.db != nil {
			go a.db.AutoBackup(a.ctx, interval, backupDir, a.cfg.BackupKeep)
		}
	}

	// ── SSE Broker (W2-04) ──────────────────────────────────────────────────
	a.sseBroker = sse.NewBroker(30*time.Second, a.logger)
	sseHandler := handler.NewSSEHandler(a.sseBroker, a.logger).WithJWTSecret(a.cfg.JWTSecret)

	// ── First-Run Onboarding (A15) ─────────────────────────────────────────
	if a.metaRepo != nil {
		go a.setupDemoData(projectsRoot)
	}

	// ── Routes ───────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	routes.RegisterRoutes(mux, routes.RegisterConfig{
		MetaRepo:          a.metaRepo,
		JWTSecret:         a.cfg.JWTSecret,
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
		SessionHandler:    sessionHandler,
		IngestionHandler:  ingestionHandler,
		SandboxHandler:    sandboxHandler,
		RegistryHandler:   registryHandler,
		ToolExecHandler:   toolExecHandler,
		CodeFlowHandler:   codeFlowHandler,
		SuggestPipeline:   toolSuggestHandler,
		Interceptors:      interceptors,
		AuthRateLimiter:   authRateLimiter,
	})

	corsHandler := routes.CORSHandler(mux, a.cfg.CORSAllowedOrigins, a.logger)
	telemetryHandler := telemetry.Middleware(corsHandler)
	recoveryHandler := middleware.Recovery(telemetryHandler)
	csrfHandler := middleware.CSRFProtection(a.cfg.CORSAllowedOrigins)(recoveryHandler)
	secureHandler := middleware.SecurityHeaders(a.cfg.DevMode)(csrfHandler)
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
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           h2c.NewHandler(promHandler, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return a.server.ListenAndServe()
}

func (a *AlephApp) Close(ctx context.Context) error {
	log.Println("[Aleph] Shutting down services...")

	var errs []error

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
	if a.authRlCleanup != nil {
		a.authRlCleanup()
	}
	if a.memStore != nil {
		if err := a.memStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("memstore close: %w", err))
		}
	}

	// Graceful server shutdown BEFORE cancel — allows in-flight requests to complete.
	// W3-03: a.cancel() was previously here, prematurely canceling request contexts.
	if a.server != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("server shutdown: %w", err))
		}
		shutdownCancel()
	}

	// Now that all servers are down, cancel the app context to stop background goroutines.
	if a.cancel != nil {
		a.cancel()
	}

	sentry.Flush(2 * time.Second)

	if a.nlpHandler != nil {
		a.nlpHandler.Close()
	}
	if err := a.eng.Close(); err != nil {
		errs = append(errs, fmt.Errorf("engine close: %w", err))
	}
	if err := a.pg.Close(); err != nil {
		errs = append(errs, fmt.Errorf("postgres close: %w", err))
	}
	if err := a.db.Close(); err != nil {
		errs = append(errs, fmt.Errorf("duckdb close: %w", err))
	}

	return errors.Join(errs...)
}

func (a *AlephApp) runSecurityScan() {
	scanner := sandbox.NewSecurityScanner()
	issues := scanner.Scan(a.cfg.DuckDBPath) // scan persisted tool definitions/code

	critical := 0
	high := 0
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			critical++
		case "high":
			high++
		}
		a.logger.Warn("security scan finding",
			"severity", issue.Severity,
			"rule", issue.RuleName,
			"line", issue.Line,
			"description", issue.Description,
		)
	}

	if critical > 0 || high > 0 {
		a.logger.Warn("security scan complete",
			"critical", critical,
			"high", high,
			"medium", len(issues)-critical-high,
		)
	} else {
		a.logger.Info("security scan passed", "total_issues", len(issues))
	}
}

func (a *AlephApp) makeSentimentHelper() func(ctx context.Context, text string) (string, error) {
	return func(ctx context.Context, text string) (string, error) {
		if a.nlpHandler == nil {
			slog.Warn("sentiment analysis unavailable — NLP sidecar not configured")
			return `{"score": 0, "label": "neutral", "error": "NLP sidecar unavailable"}`, nil
		}
		resp, err := a.nlpHandler.AnalyzeSentiment(ctx, connect.NewRequest(&nlpv1.AnalyzeSentimentRequest{Text: text}))
		if err != nil {
			slog.Warn("sentiment analysis failed", "err", err)
			return "", fmt.Errorf("Errore analisi sentiment: %w", err)
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
			return `{"error": "registry unavailable"}`, nil
		}
		comp, err := reg.GetComponentByID(ctx, entityID)
		if err != nil || comp == nil {
			return "", fmt.Errorf("entity %s not found", entityID)
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

func (a *AlephApp) makeComponentByIDHelper(reg *registry.DuckDBRegistry) func(ctx context.Context, id string) (*decision.ComponentMetadata, error) {
	return func(ctx context.Context, id string) (*decision.ComponentMetadata, error) {
		if reg == nil {
			return nil, fmt.Errorf("registry unavailable")
		}
		comp, err := reg.GetComponentByID(ctx, id)
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
				host, port, err := net.SplitHostPort(addr)
				if err == nil {
					if err := ssrf.ValidateHostname(host, port); err != nil {
						return nil, fmt.Errorf("SSRF validation of gRPC target %s: %w", addr, err)
					}
				}
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
}

func newTLSClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http2.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string, tlsCfg *tls.Config) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err == nil {
					if err := ssrf.ValidateHostname(host, port); err != nil {
						return nil, fmt.Errorf("SSRF validation of gRPC target %s: %w", addr, err)
					}
				}
				var d net.Dialer
				conn, err := d.DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
				if tlsCfg == nil {
					tlsCfg = &tls.Config{MinVersion: tls.VersionTLS13}
				}
				return tls.Client(conn, tlsCfg), nil
			},
		},
	}
}

func (a *AlephApp) watchSidecar(nlpHandler *handler.NLPHandler) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("watchSidecar panicked, restarting", "recover", r)
			go a.watchSidecar(nlpHandler)
		}
	}()

	addr := a.cfg.NLPAddr
	if strings.HasPrefix(addr, "http") {
		addr = strings.TrimPrefix(addr, "http://")
		addr = strings.TrimPrefix(addr, "https://")
	}

	slog.Info("starting neural monitoring", "addr", addr)
	var conn *grpc.ClientConn
	var err error
	if a.cfg.DevMode {
		conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13})))
	}
	if err != nil {
		slog.Error("connection to sidecar failed", "error", err)
		return
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)

	var (
		restartCount   int
		restartStart   time.Time
		consecutiveErr bool
	)

	ticker := time.NewTicker(sidecarCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			slog.Info("sidecar monitor stopped")
			return
		case <-ticker.C:
			if !a.checkSidecarOnce(client, nlpHandler, &consecutiveErr, &restartCount, &restartStart) {
				return
			}
		}
	}
}

// checkSidecarOnce performs a single health-check iteration.
// Returns true if the loop should continue, false to stop.
func (a *AlephApp) checkSidecarOnce(
	client grpc_health_v1.HealthClient,
	nlpHandler *handler.NLPHandler,
	consecutiveErr *bool,
	restartCount *int,
	restartStart *time.Time,
) bool {
	if nlpHandler == nil {
		if consecutiveErr != nil {
			*consecutiveErr = false
		}
		if restartCount != nil {
			*restartCount = 0
		}
		return true
	}

	ctx, cancel := context.WithTimeout(a.ctx, sidecarCheckTimeout)
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "aleph.nlp.v1.NLPService"})
	cancel()
	if err != nil {
		slog.Warn("sidecar non risponde", "error", err)
		nlpHandler.MarkUnhealthy()

		if !*consecutiveErr {
			*consecutiveErr = true
			*restartCount = 0
			*restartStart = time.Now()
		}

		*restartCount++
		if *restartCount > sidecarMaxRestarts && time.Since(*restartStart) < sidecarRestartWindow {
			slog.Error("sidecar watchdog: too many failures in window, giving up",
				"restarts", *restartCount, "window", sidecarRestartWindow)
			return false
		}

		step := *restartCount - 1
		if step >= len(sidecarBackoffSteps) {
			step = len(sidecarBackoffSteps) - 1
		}
		slog.Info("sidecar watchdog: will retry", "attempt", *restartCount, "backoff", sidecarBackoffSteps[step])
		time.Sleep(sidecarBackoffSteps[step])
	} else {
		*consecutiveErr = false
		*restartCount = 0
		if resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			nlpHandler.MarkHealthy()
			slog.Info("sidecar neurale operativo")
		} else {
			slog.Warn("sidecar non SERVING", "status", resp.GetStatus())
		}
	}

	return true
}

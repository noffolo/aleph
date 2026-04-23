package app

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"net"
	"crypto/tls"

	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
	"github.com/ff3300/aleph-v2/internal/config"
	"github.com/ff3300/aleph-v2/internal/ingestion"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/nlp_adapter"
	"github.com/ff3300/aleph-v2/internal/migrate"
	"github.com/ff3300/aleph-v2/internal/predict"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	"github.com/ff3300/aleph-v2/internal/service/notification"
	"github.com/ff3300/aleph-v2/internal/storage"
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
	wd, _ := os.Getwd()
	projectsRoot := filepath.Join(wd, "data", "projects")

	// Interceptors
	authInterceptor := middleware.NewAuthInterceptor(a.metaRepo)
	interceptors := connect.WithInterceptors(authInterceptor)

	// Handlers
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

	mux := http.NewServeMux()

	// Connect RPC Routes
	mux.Handle(v1connect.NewQueryServiceHandler(queryHandler, interceptors))
	mux.Handle(v1connect.NewProjectServiceHandler(projectHandler, interceptors))
	mux.Handle(v1connect.NewAgentServiceHandler(agentHandler, interceptors))
	mux.Handle(v1connect.NewSkillServiceHandler(skillHandler, interceptors))
	mux.Handle(v1connect.NewToolServiceHandler(toolHandler, interceptors))
	mux.Handle(v1connect.NewLibraryServiceHandler(libraryHandler, interceptors))
	mux.Handle(nlpconnect.NewNLPServiceHandler(a.nlpHandler, interceptors))
	mux.Handle(v1connect.NewNotificationServiceHandler(notificationHandler, interceptors))
	mux.Handle(v1connect.NewAuthServiceHandler(authHandler, interceptors))
	mux.Handle(v1connect.NewIngestionServiceHandler(ingestionHandler, interceptors))
	mux.Handle(v1connect.NewSandboxServiceHandler(sandboxHandler, interceptors))
	mux.Handle(v1connect.NewRegistryServiceHandler(registryHandler, interceptors))

	// API Documentation (OpenAPI/Swagger)
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "internal/api/proto/aleph_api.swagger.json")
	})

	// Frontend SPA Hosting with SPA Routing (fallback to index.html)
	subFS, _ := fs.Sub(a.frontend, "dist")
	fileServer := http.FileServer(http.FS(subFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If requesting a Connect RPC or Swagger, let the mux handle it via other registrations
		if strings.HasPrefix(r.URL.Path, "/aleph.v1.") || r.URL.Path == "/swagger.json" {
			mux.ServeHTTP(w, r)
			return
		}

		// Check if the file exists in the embedded FS
		f, err := subFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA routing
		index, err := subFS.Open("index.html")
		if err != nil {
			http.Error(w, "frontend not found", http.StatusNotFound)
			return
		}
		index.Close()
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	// CORS Setup
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173,http://localhost:3000"
	}
	originMap := map[string]bool{}
	for _, o := range strings.Split(allowedOrigins, ",") {
		trimmed := strings.TrimSpace(o)
		if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
			log.Printf("[Aleph] WARNING: skipping invalid CORS origin (must start with http:// or https://): %s", trimmed)
			continue
		}
		originMap[trimmed] = true
	}

	corsHandler := cors.New(cors.Options{
		AllowOriginFunc:  func(origin string) bool { return originMap[origin] },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Grpc-Status", "Grpc-Message"},
		AllowCredentials: true,
	}).Handler(mux)

	go a.watchSidecar(a.nlpHandler)

	log.Printf("[Aleph] Data OS starting on :%d", port)
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h2c.NewHandler(corsHandler, &http2.Server{}),
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
	a.eng.Close()
	a.pg.Close()
	if a.nlpHandler != nil {
		a.nlpHandler.Close()
	}
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

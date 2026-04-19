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

	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/ff3300/aleph-v2/internal/api/middleware"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1/v1connect"
	"github.com/ff3300/aleph-v2/internal/config"
	"github.com/ff3300/aleph-v2/internal/ingestion"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/service/notification"
	"github.com/ff3300/aleph-v2/internal/storage"
	"log/slog"
)

type AlephApp struct {
	db       *storage.DuckDB
	pg       *storage.Postgres
	cfg      *config.Config
	eng      *ingestion.Engine
	metaRepo *repository.MetadataRepository
	frontend embed.FS
	server   *http.Server
	logger   *slog.Logger
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
	eng := ingestion.NewEngine(projectsRoot, metaRepo, db, cfg.NLPAddr)

	return &AlephApp{
		db:       db,
		pg:       pg,
		cfg:      cfg,
		eng:      eng,
		metaRepo: metaRepo,
		frontend: frontend,
	}, nil
}


func (a *AlephApp) Serve(port int) error {
	wd, _ := os.Getwd()
	projectsRoot := filepath.Join(wd, "data", "projects")

	// Interceptors
	authInterceptor := middleware.NewAuthInterceptor(a.metaRepo)
	interceptors := connect.WithInterceptors(authInterceptor)

	// Handlers
	queryHandler := handler.NewQueryHandler(a.db, projectsRoot, a.metaRepo, a.cfg.NLPAddr)
	projectHandler := handler.NewProjectHandler(projectsRoot, a.db)
	agentHandler := handler.NewAgentHandler(projectsRoot, a.metaRepo)
	skillHandler := handler.NewSkillHandler(projectsRoot, a.metaRepo)
	toolHandler := handler.NewToolHandler(projectsRoot, a.metaRepo)
	libraryHandler := handler.NewLibraryHandler(projectsRoot)
	nlpClient := nlpconnect.NewNLPServiceClient(http.DefaultClient, a.cfg.NLPAddr)
	nlpHandler := handler.NewNLPHandler(a.logger, nlpClient)

	notificationSvc := notification.NewNotificationService()
	notificationHandler := handler.NewNotificationHandler(notificationSvc, a.metaRepo)
	
	authHandler := handler.NewAuthHandler(a.metaRepo)
	ingestionHandler := handler.NewIngestionHandler(projectsRoot, a.eng, a.metaRepo)

	mux := http.NewServeMux()

	// Connect RPC Routes
	mux.Handle(v1connect.NewQueryServiceHandler(queryHandler, interceptors))
	mux.Handle(v1connect.NewProjectServiceHandler(projectHandler, interceptors))
	mux.Handle(v1connect.NewAgentServiceHandler(agentHandler, interceptors))
	mux.Handle(v1connect.NewSkillServiceHandler(skillHandler, interceptors))
	mux.Handle(v1connect.NewToolServiceHandler(toolHandler, interceptors))
	mux.Handle(v1connect.NewLibraryServiceHandler(libraryHandler, interceptors))
	mux.Handle(nlpconnect.NewNLPServiceHandler(nlpHandler, interceptors))
	mux.Handle(v1connect.NewNotificationServiceHandler(notificationHandler, interceptors))
	mux.Handle(v1connect.NewAuthServiceHandler(authHandler, interceptors))
	mux.Handle(v1connect.NewIngestionServiceHandler(ingestionHandler, interceptors))

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
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Connect-Protocol-Version", "X-Aleph-Api-Key"},
	}).Handler(mux)

	go a.watchSidecar()

	log.Printf("[Aleph] Data OS starting on :%d", port)
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h2c.NewHandler(corsHandler, &http2.Server{}),
	}

	return a.server.ListenAndServe()
}

func (a *AlephApp) Close(ctx context.Context) error {
	log.Println("[Aleph] Shutting down services...")
	if a.server != nil {
		a.server.Shutdown(ctx)
	}
	a.eng.Close()
	a.pg.Close()
	return a.db.Close()
}

func (a *AlephApp) watchSidecar() {
	addr := a.cfg.NLPAddr
	if !strings.HasPrefix(addr, "http") {
		// gRPC connection uses the host:port format
	} else {
		addr = strings.TrimPrefix(addr, "http://")
		addr = strings.TrimPrefix(addr, "https://")
	}
	
	slog.Info("avvio monitoraggio neurale", "addr", addr)
	for {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			client := grpc_health_v1.NewHealthClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
			cancel()
			if err != nil {
				slog.Warn("sidecar non risponde", "error", err, "action", "attempting_recovery")
			}
			conn.Close()
		} else {
			slog.Error("connessione al sidecar fallita", "error", err)
		}
		time.Sleep(10 * time.Second) 
	}
}

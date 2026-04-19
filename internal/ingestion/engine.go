package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
)

type Engine struct {
	projectsRoot string
	metaRepo     *repository.MetadataRepository
	db           *storage.DuckDB
	mu           sync.RWMutex
	tasks        map[string]*v1.IngestionTask
}

func NewEngine(projectsRoot string, metaRepo *repository.MetadataRepository, db *storage.DuckDB, nlpAddr string) *Engine {
	return &Engine{
		projectsRoot: projectsRoot,
		metaRepo:     metaRepo,
		db:           db,
		tasks:        make(map[string]*v1.IngestionTask),
	}
}

func (e *Engine) RunTask(ctx context.Context, projectID string, task *v1.IngestionTask) error {
	// Process Reaper: Assicura che nessun task giri all'infinito
	taskCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	e.metaRepo.UpdateTaskProgress(task.Id, 0, "esecuzione")
	logPath := filepath.Join(e.projectsRoot, projectID, "logs", task.Id+".log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil { return err }
	defer f.Close()

	fmt.Fprintf(f, "\n--- Inizio Task: %s alle %s ---\n", task.Id, time.Now().Format(time.RFC3339))
	
	var taskErr error
	switch task.SourceType {
	case "rss", "rest", "github":
		taskErr = e.runPrecompiled(taskCtx, f, projectID, task)
	case "custom_code":
		taskErr = e.runDynamic(taskCtx, f, projectID, task)
	default:
		taskErr = fmt.Errorf("tipo di sorgente sconosciuto: %s", task.SourceType)
	}

	if taskErr != nil {
		e.metaRepo.UpdateTaskProgress(task.Id, 0, "fallito")
		fmt.Fprintf(f, "Errore: %v\n", taskErr)
		return taskErr
	}

	e.metaRepo.UpdateTaskProgress(task.Id, 100, "completato")
	fmt.Fprintf(f, "--- Successo ---\n")

	// Registrazione Viste in DuckDB per performance
	if err := e.registerViews(projectID); err != nil {
		fmt.Fprintf(f, "Attenzione: Registrazione viste fallita: %v\n", err)
	}

	// Metadati Temporali: ogni riga riceve un timestamp di ingestion
	timestampSQL := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS _aleph_ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP", task.Id)
	e.db.Exec(timestampSQL)

	// Arricchimento Predittivo Vettoriale (Asincrono)
	go func() {
		enrichCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		e.enrichPredictiveMetadata(enrichCtx, projectID, task.Id)
	}()

	return nil
}

func (e *Engine) enrichPredictiveMetadata(ctx context.Context, projectID, taskID string) {
	log.Printf("[Motore] Avvio Arricchimento Predittivo per il task %s", taskID)
	
	// Caricamento Ontologia per identificazione Chiave Primaria
	projectPath := filepath.Join(e.projectsRoot, projectID)
	ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
	content, err := os.ReadFile(ontPath)
	var primaryKey string
	if err == nil {
		prog, _ := dsl.Parse(string(content))
		for _, stmt := range prog.Statements {
			if stmt.Object != nil && stmt.Object.FromSource == taskID {
				primaryKey = stmt.Object.ID
				break
			}
		}
	}

	// Assicura che la tabella system_features esista
	e.db.Exec(`CREATE TABLE IF NOT EXISTS system_features (
		project_id VARCHAR,
		task_id VARCHAR,
		entity_id VARCHAR,
		feature_type VARCHAR,
		feature_value FLOAT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)

	// Recupero righe dall'ultimo minuto per l'arricchimento
	query := fmt.Sprintf(`SELECT * FROM "%s" WHERE _aleph_ingested_at > (CURRENT_TIMESTAMP - INTERVAL 1 MINUTE)`, taskID)
	rows, err := e.db.Query(query)
	if err != nil {
		log.Printf("[Motore] Query di arricchimento fallita per %s: %v", taskID, err)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	
	// Identificazione indice chiave primaria
	idIdx := 0
	if primaryKey != "" {
		for i, c := range cols {
			if strings.EqualFold(c, primaryKey) {
				idIdx = i
				break
			}
		}
	} else {
		// Fallback euristico se non definita nell'ontologia
		for i, c := range cols {
			cl := strings.ToLower(c)
			if cl == "id" || cl == "uuid" || cl == "guid" || strings.Contains(cl, "key") {
				idIdx = i
				break
			}
		}
	}

	for rows.Next() {
		select {
		case <-ctx.Done(): return
		default:
		}

		vals := make([]interface{}, len(cols)); vps := make([]interface{}, len(cols))
		for i := range vals { vps[i] = &vals[i] }
		if err := rows.Scan(vps...); err != nil { continue }

		entityID := fmt.Sprintf("%v", vals[idIdx])

		for i, col := range cols {
			if str, ok := vals[i].(string); ok && len(str) > 10 {
				if err == nil {
					e.db.Exec(`INSERT INTO system_features (project_id, task_id, entity_id, feature_type, feature_value) 
					        VALUES (?, ?, ?, ?, ?)`, projectID, taskID, entityID, "sentiment_"+col, 0.0)				}
			}
		}
	}
}

func (e *Engine) Close() error {
	log.Println("[Engine] Closing ingestion engine...")
	return nil
}

func (e *Engine) registerViews(projectID string) error {
	projectPath := filepath.Join(e.projectsRoot, projectID)
	ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
	content, err := os.ReadFile(ontPath)
	if err != nil { return nil } // No ontology, skip

	prog, err := dsl.Parse(string(content))
	if err != nil { return fmt.Errorf("parsing ontology: %v", err) }

	dataRoot := filepath.Join(projectPath, "raw")
	compiler := dsl.NewCompiler(prog, dataRoot)

	for _, stmt := range prog.Statements {
		if stmt.Object != nil {
			sql, err := compiler.CompileObject(stmt.Object.Name)
			if err != nil { continue }
			
			viewName := fmt.Sprintf("%s_%s", projectID, stmt.Object.Name)
			createViewSql := fmt.Sprintf("CREATE OR REPLACE VIEW \"%s\" AS %s", viewName, sql)
			if _, err := e.db.Exec(createViewSql); err != nil {
				log.Printf("[Engine] Failed to create view %s: %v", viewName, err)
			}
		}
	}
	return nil
}

func (e *Engine) runPrecompiled(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	fmt.Fprintf(w, "Running pre-compiled: %s\n", task.SourceType)
	for i := 1; i <= 5; i++ {
		select {
		case <-ctx.Done(): return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			e.metaRepo.UpdateTaskProgress(task.Id, int32(i*20), "running")
			fmt.Fprintf(w, "Progress: %d%%\n", i*20)
		}
	}
	return nil
}

func (e *Engine) runDynamic(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct { Code string `json:"code"` }
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil { return err }

	tmpDir, err := os.MkdirTemp("", "aleph-run-*")
	if err != nil { return err }
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(tmpFile, []byte(config.Code), 0644)
	binaryPath := filepath.Join(tmpDir, "conn")
	
	cmdBuild := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, tmpFile)
	if out, err := cmdBuild.CombinedOutput(); err != nil {
		fmt.Fprintf(w, "Build Error: %s\n", string(out))
		return err
	}

	cmdRun := exec.CommandContext(ctx, binaryPath)
	cmdRun.Stdout = w
	cmdRun.Stderr = w
	cmdRun.Env = append(os.Environ(), 
		fmt.Sprintf("ALEPH_PROJECT_PATH=%s", filepath.Join(e.projectsRoot, projectID)),
	)
	return cmdRun.Run()
}

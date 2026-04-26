package ingestion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

const SentimentUnavailable = -1.0

func computeChecksum(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func VerifyChecksum(data []byte, expected string) bool {
	return computeChecksum(data) == expected
}

func sanitizeIdentifier(id string) error {
	if !validIdentifier.MatchString(id) {
		return fmt.Errorf("identificativo non valido: %s", id)
	}
	return nil
}

func sanitizeFilePath(p string) error {
	cleaned := filepath.Clean(p)
	if cleaned != p {
		return fmt.Errorf("percorso non valido: %s", p)
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("percorso con traversale: %s", p)
	}
	return nil
}

type Engine struct {
	projectsRoot string
	metaRepo     *repository.MetadataRepository
	db           *storage.DuckDB
	nlpHandler   NLPAnalyzer
	mu           sync.RWMutex
	tasks        map[string]*v1.IngestionTask
	safeClient   *http.Client
}

// NLPAnalyzer abstracts sentiment analysis for ingestion enrichment.
// This allows the Engine to call real NLP during ingestion rather than
// inserting hardcoded 0.0 placeholder values.
type NLPAnalyzer interface {
	AnalyzeSentiment(ctx context.Context, text string) (score float32, label string, err error)
}

var safeHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("troppi redirect")
		}
		return validateHTTPRequest(req)
	},
}

func init() {
	safeHTTPClient.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil { host = addr }
			ip := net.ParseIP(host)
			if ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
				return nil, fmt.Errorf("accesso a rete interna non permesso: %s", addr)
			}
			var d net.Dialer
			return d.DialContext(ctx, network, addr)
		},
	}
}

func validateHTTPRequest(req *http.Request) error {
	host := req.URL.Hostname()
	ip := net.ParseIP(host)
	if ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
		return fmt.Errorf("accesso a rete interna non permesso: %s", host)
	}
	return nil
}

func NewEngine(projectsRoot string, metaRepo *repository.MetadataRepository, db *storage.DuckDB, nlp NLPAnalyzer) *Engine {
	return &Engine{
		projectsRoot: projectsRoot,
		metaRepo:     metaRepo,
		db:           db,
		nlpHandler:   nlp,
		tasks:        make(map[string]*v1.IngestionTask),
	}
}

func (e *Engine) updateProgress(id string, progress int32, status string) {
	if e.metaRepo != nil {
		e.metaRepo.UpdateTaskProgress(id, progress, status)
	}
}

func (e *Engine) RunTask(ctx context.Context, projectID string, task *v1.IngestionTask) error {
	e.mu.Lock()
	if running, ok := e.tasks[task.Id]; ok && running.Status == "running" {
		e.mu.Unlock()
		return fmt.Errorf("task %s già in esecuzione", task.Id)
	}
	e.tasks[task.Id] = task
	task.Status = "running"
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.tasks, task.Id)
		e.mu.Unlock()
	}()

	taskCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	e.updateProgress(task.Id, 0, "esecuzione")
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
	case "url":
		taskErr = e.runURLFetch(taskCtx, f, projectID, task)
	case "csv":
		taskErr = e.runCSVLoad(taskCtx, f, projectID, task)
	case "postgres":
		taskErr = e.runPostgresLoad(taskCtx, f, projectID, task)
	case "copy":
		taskErr = e.runCopy(taskCtx, f, projectID, task)
	case "email":
		taskErr = e.runEmailFetch(taskCtx, f, projectID, task)
	case "custom_code":
		taskErr = e.runDynamic(taskCtx, f, projectID, task)
	default:
		taskErr = fmt.Errorf("tipo di sorgente sconosciuto: %s", task.SourceType)
	}

	if taskErr != nil {
		e.updateProgress(task.Id, 0, "fallito")
		fmt.Fprintf(f, "Errore: %v\n", taskErr)
		return taskErr
	}

	e.updateProgress(task.Id, 100, "completato")
	fmt.Fprintf(f, "--- Successo ---\n")

	// Registrazione Viste in DuckDB per performance
	if err := e.registerViews(projectID); err != nil {
		fmt.Fprintf(f, "Attenzione: Registrazione viste fallita: %v\n", err)
	}

	// Metadati Temporali: ogni riga riceve un timestamp di ingestion
	if !validIdentifier.MatchString(task.Id) {
		fmt.Fprintf(f, "Attenzione: Identificativo task non valido: %s\n", task.Id)
		e.updateProgress(task.Id, 0, "fallito")
		return fmt.Errorf("identificativo task non valido (solo alfanumerici, _ e -): %s", task.Id)
	}
	timestampSQL := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS _aleph_ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP", task.Id)
	e.db.Exec(timestampSQL)

	// Arricchimento Predittivo Vettoriale (Asincrono)
	go func() {
		enrichCtx, enrichCancel := context.WithTimeout(ctx, 30*time.Minute)
		defer enrichCancel()
		resolvedTableName := resolveTableName(task)
		e.enrichPredictiveMetadata(enrichCtx, projectID, resolvedTableName)
		if err := enrichCtx.Err(); err != nil {
			log.Printf("[Motore] Arricchimento Predittivo interrotto o scaduto per la tabella %s: %v", resolvedTableName, err)
		}
	}()

	return nil
}

func resolveTableName(task *v1.IngestionTask) string {
	var config struct {
		TableName string `json:"tableName"`
	}
	if json.Unmarshal([]byte(task.ConfigJson), &config) == nil && config.TableName != "" {
		return strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(config.TableName, "_"))
	}
	if task.Name != "" {
		return strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(task.Name, "_"))
	}
	return task.Id
}

func (e *Engine) enrichPredictiveMetadata(ctx context.Context, projectID, tableName string) {
	log.Printf("[Motore] Avvio Arricchimento Predittivo per la tabella %s", tableName)
	
	projectPath := filepath.Join(e.projectsRoot, projectID)
	ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
	content, err := os.ReadFile(ontPath)
	var primaryKey string
	if err == nil {
		prog, parseErr := dsl.Parse(string(content))
		if parseErr == nil && prog != nil {
			for _, stmt := range prog.Statements {
				if stmt.Object != nil && stmt.Object.FromSource == tableName {
					primaryKey = stmt.Object.ID
					break
				}
			}
		}
	}

	// Table creation has been moved to migrations/000001_init_schema.up.sql
	// The following query is kept for reference but commented out:
	/*
	e.db.Exec(`CREATE TABLE IF NOT EXISTS system_features (
		project_id VARCHAR,
		task_id VARCHAR,
		entity_id VARCHAR,
		feature_type VARCHAR,
		feature_value FLOAT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	*/

	query := fmt.Sprintf(`SELECT * FROM "%s" WHERE _aleph_ingested_at > (CURRENT_TIMESTAMP - INTERVAL '1 MINUTE')`, tableName)
	rows, err := e.db.Query(query)
	if err != nil {
		log.Printf("[Motore] Query di arricchimento fallita per %s: %v", tableName, err)
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
				featureValue := SentimentUnavailable
				if e.nlpHandler != nil {
					score, _, sErr := e.nlpHandler.AnalyzeSentiment(ctx, str)
					if sErr == nil {
						featureValue = float64(score)
					} else {
						log.Printf("[Motore] Analisi del sentimento fallita per %s.%s: %v", tableName, col, sErr)
					}
				}
				e.db.Exec(`INSERT INTO system_features (project_id, task_id, entity_id, feature_type, feature_value) 
					VALUES (?, ?, ?, ?, ?)`, projectID, tableName, entityID, "sentiment_"+col, featureValue)
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
	var config struct {
		URL   string `json:"url"`
		Repo  string `json:"repo"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("config JSON non valido: %w", err)
	}

	urlToFetch := config.URL
	if urlToFetch == "" && config.Repo != "" {
		urlToFetch = "https://api.github.com/repos/" + config.Repo + "/contents"
	}
	if urlToFetch == "" {
		return fmt.Errorf("nessun URL o repository specificato nel config")
	}

	fmt.Fprintf(w, "Fetching: %s\n", urlToFetch)
	e.updateProgress(task.Id, 10, "running")

	req, err := http.NewRequestWithContext(ctx, "GET", urlToFetch, nil)
	if err != nil {
		return fmt.Errorf("creazione request fallita: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/csv, application/rss+xml, application/atom+xml, */*")
	if config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+config.Token)
	}

	resp, err := safeHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request fallita: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d: %s", resp.StatusCode, resp.Status)
	}

	e.updateProgress(task.Id, 40, "running")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("lettura body fallita: %w", err)
	}

	e.updateProgress(task.Id, 70, "running")

	tableName := task.Id
	if err := sanitizeIdentifier(tableName); err != nil { return err }
	contentType := resp.Header.Get("Content-Type")
	projectPath := filepath.Join(e.projectsRoot, projectID)
	os.MkdirAll(filepath.Join(projectPath, "raw"), 0755)

	if strings.Contains(contentType, "json") || strings.Contains(contentType, "javascript") {
		tmpFile := filepath.Join(projectPath, "raw", tableName+".json")
		if err := os.WriteFile(tmpFile, bodyBytes, 0644); err != nil {
			return fmt.Errorf("scrittura file temporaneo fallita: %w", err)
		}
		createSQL := fmt.Sprintf(`CREATE OR REPLACE VIEW "%s" AS SELECT * FROM read_json_auto('%s')`, tableName, strings.ReplaceAll(tmpFile, "'", "''"))
		if _, err := e.db.Exec(createSQL); err != nil {
			return fmt.Errorf("creazione vista JSON fallita: %w", err)
		}
	} else if strings.Contains(contentType, "csv") || strings.Contains(contentType, "text/plain") {
		tmpFile := filepath.Join(projectPath, "raw", tableName+".csv")
		os.WriteFile(tmpFile, bodyBytes, 0644)
		createSQL := fmt.Sprintf(`CREATE OR REPLACE VIEW "%s" AS SELECT * FROM read_csv_auto('%s')`, tableName, strings.ReplaceAll(tmpFile, "'", "''"))
		if _, err := e.db.Exec(createSQL); err != nil {
			return fmt.Errorf("creazione vista CSV fallita: %w", err)
		}
	} else {
		tmpFile := filepath.Join(projectPath, "raw", tableName+".dat")
		os.WriteFile(tmpFile, bodyBytes, 0644)
		fmt.Fprintf(w, "Dati salvati in %s (content-type: %s)\n", tmpFile, contentType)
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Completato: vista \"%s\" creata da %s\n", tableName, urlToFetch)
	return nil
}

func (e *Engine) runURLFetch(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("config JSON non valido: %w", err)
	}
	if config.URL == "" {
		return fmt.Errorf("URL vuoto nel config")
	}

	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return fmt.Errorf("URL non valido: %w", err)
	}
	if err := validateHTTPRequest(&http.Request{URL: parsedURL}); err != nil {
		return fmt.Errorf("URL non permesso: %w", err)
	}

	fmt.Fprintf(w, "Fetching URL: %s\n", config.URL)
	e.updateProgress(task.Id, 10, "running")

	req, err := http.NewRequestWithContext(ctx, "GET", config.URL, nil)
	if err != nil {
		return fmt.Errorf("creazione request fallita: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/csv, */*")

	resp, err := safeHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request fallita: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d: %s", resp.StatusCode, resp.Status)
	}

	fmt.Fprintf(w, "Risposta: %s (%d bytes)\n", resp.Header.Get("Content-Type"), resp.ContentLength)
	e.updateProgress(task.Id, 40, "running")

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("lettura body fallita: %w", err)
	}

	e.updateProgress(task.Id, 70, "running")

	contentType := resp.Header.Get("Content-Type")
	tableName := task.Id
	if err := sanitizeIdentifier(tableName); err != nil { return err }
	projectPath := filepath.Join(e.projectsRoot, projectID, "raw")
	os.MkdirAll(projectPath, 0755)

	isJSON := strings.Contains(contentType, "json") || strings.HasSuffix(config.URL, ".json")
	isCSV := strings.Contains(contentType, "csv") || strings.HasSuffix(config.URL, ".csv")

	if !isJSON && !isCSV {
		if json.Valid(bodyBytes) {
			isJSON = true
		} else {
			isCSV = true
		}
	}

	if isCSV {
		rawPath := filepath.Join(projectPath, tableName+".csv")
		if err := os.WriteFile(rawPath, bodyBytes, 0644); err != nil {
			return fmt.Errorf("scrittura file fallita: %w", err)
		}
		fmt.Fprintf(w, "Salvato CSV in %s (%d bytes)\n", rawPath, len(bodyBytes))
		createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" AS SELECT * FROM read_csv_auto('%s', ignore_errors=true)`, tableName, strings.ReplaceAll(rawPath, "'", "''"))
		if _, err := e.db.Exec(createSQL); err != nil {
			return fmt.Errorf("creazione tabella da CSV fallita: %w", err)
		}
	} else {
		rawPath := filepath.Join(projectPath, tableName+".json")
		if err := os.WriteFile(rawPath, bodyBytes, 0644); err != nil {
			return fmt.Errorf("scrittura file fallita: %w", err)
		}
		fmt.Fprintf(w, "Salvato JSON in %s (%d bytes)\n", rawPath, len(bodyBytes))
		createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" AS SELECT * FROM read_json_auto('%s', ignore_errors=true)`, tableName, strings.ReplaceAll(rawPath, "'", "''"))
		if _, err := e.db.Exec(createSQL); err != nil {
			return fmt.Errorf("creazione tabella da JSON fallita: %w", err)
		}
	}

	e.updateProgress(task.Id, 90, "running")
	fmt.Fprintf(w, "Caricamento completato nella tabella \"%s\"\n", tableName)
	return nil
}

func (e *Engine) insertJSONArray(tableName string, arr []interface{}, w *os.File) error {
	if len(arr) == 0 {
		fmt.Fprintf(w, "Array vuoto, nessun dato da inserire\n")
		return nil
	}

	first, ok := arr[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("il primo elemento non è un oggetto JSON")
	}

	columns := make([]string, 0, len(first))
	for k := range first {
		columns = append(columns, k)
	}
	sort.Strings(columns)

	fmt.Fprintf(w, "Colonne rilevate: %v (%d righe)\n", columns, len(arr))

	escapedCols := make([]string, len(columns))
	for i, c := range columns {
		escapedCols[i] = fmt.Sprintf(`"%s"`, c)
	}

	values := make([]string, 0, len(arr))
	params := make([]interface{}, 0)
	for _, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		placeholders := make([]string, len(columns))
		for i, c := range columns {
			placeholders[i] = "?"
			val := obj[c]
			if val == nil {
				params = append(params, nil)
			} else {
				params = append(params, fmt.Sprintf("%v", val))
			}
		}
		values = append(values, "("+strings.Join(placeholders, ",")+")")
	}

	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (%s)`, tableName,
		strings.Join(escapedCols, ", "))
	if _, err := e.db.Exec(createSQL); err != nil {
		return fmt.Errorf("creazione tabella fallita: %w", err)
	}

	batchSize := 500
	for i := 0; i < len(values); i += batchSize {
		end := i + batchSize
		if end > len(values) {
			end = len(values)
		}
		insertSQL := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES %s`, tableName,
			strings.Join(escapedCols, ", "),
			strings.Join(values[i:end], ", "))
		if _, err := e.db.Exec(insertSQL, params...); err != nil {
			fmt.Fprintf(w, "Warning: insert batch fallito: %v\n", err)
		}
	}

	return nil
}

func extractArray(m map[string]interface{}) ([]interface{}, bool) {
	for _, v := range m {
		if arr, ok := v.([]interface{}); ok {
			return arr, true
		}
	}
	return nil, false
}

func (e *Engine) runCSVLoad(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		Path      string `json:"path"`
		TableName string `json:"tableName"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("config JSON non valido: %w", err)
	}
	if config.Path == "" {
		return fmt.Errorf("percorso file vuoto nel config")
	}

	fmt.Fprintf(w, "Caricamento CSV/Parquet: %s\n", config.Path)
	e.updateProgress(task.Id, 20, "running")

	tableName := config.TableName
	if tableName == "" {
		tableName = task.Name
	}
	if tableName == "" {
		tableName = task.Id
	}
	tableName = strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(tableName, "_"))
	if err := sanitizeIdentifier(tableName); err != nil { return err }
	if err := sanitizeFilePath(config.Path); err != nil { return err }

	readerFunc := "read_csv_auto"
	if strings.HasSuffix(strings.ToLower(config.Path), ".parquet") {
		readerFunc = "read_parquet"
	}

	projectPath := filepath.Join(e.projectsRoot, projectID, "raw")
	os.MkdirAll(projectPath, 0755)
	localPath := filepath.Join(projectPath, tableName+filepath.Ext(config.Path))
	data, err := os.ReadFile(config.Path)
	if err != nil { return fmt.Errorf("lettura file fallita: %w", err) }
	if err := os.WriteFile(localPath, data, 0644); err != nil { return fmt.Errorf("scrittura file locale fallita: %w", err) }

	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" AS SELECT * FROM %s('%s')`, tableName, readerFunc, strings.ReplaceAll(localPath, "'", "''"))
	if _, err := e.db.Exec(createSQL); err != nil {
		return fmt.Errorf("caricamento file fallito: %w", err)
	}

	e.updateProgress(task.Id, 90, "running")
	fmt.Fprintf(w, "Tabella \"%s\" creata da %s\n", tableName, config.Path)
	return nil
}

func (e *Engine) runPostgresLoad(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		DSN   string `json:"dsn"`
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("config JSON non valido: %w", err)
	}
	if config.DSN == "" {
		return fmt.Errorf("DSN vuoto nel config")
	}

	fmt.Fprintf(w, "Loading from PostgreSQL: %s\n", config.DSN)
	e.updateProgress(task.Id, 10, "running")

	tableName := task.Id
	if err := sanitizeIdentifier(tableName); err != nil { return err }

	_, err := e.db.Exec("INSTALL postgres_scanner")
	if err != nil {
		fmt.Fprintf(w, "Warning: postgres_scanner extension not available: %v\n", err)
		return fmt.Errorf("estensione postgres_scanner non disponibile: %w", err)
	}
	_, err = e.db.Exec("LOAD postgres_scanner")
	if err != nil {
		return fmt.Errorf("caricamento postgres_scanner fallito: %w", err)
	}

	e.updateProgress(task.Id, 40, "running")

	safeDSN := strings.ReplaceAll(config.DSN, "'", "''")
	query := fmt.Sprintf("SELECT * FROM postgres_scan_pushdown('%s', 'public', '%s')", safeDSN, tableName)
	if config.Query != "" {
		return fmt.Errorf("query personalizzate non permesse per motivi di sicurezza — usa solo DSN + nome tabella")
	}

	createSQL := fmt.Sprintf(`CREATE OR REPLACE VIEW "%s" AS %s`, tableName, query)
	if _, err := e.db.Exec(createSQL); err != nil {
		return fmt.Errorf("creazione vista PostgreSQL fallita: %w", err)
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Completato: vista \"%s\" creata da PostgreSQL\n", tableName)
	return nil
}

func (e *Engine) runCopy(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("config JSON non valido: %w", err)
	}
	if config.Source == "" {
		return fmt.Errorf("progetto sorgente vuoto nel config")
	}

	fmt.Fprintf(w, "Copia da progetto: %s\n", config.Source)
	e.updateProgress(task.Id, 20, "running")

	sourcePath := filepath.Join(e.projectsRoot, config.Source, "raw")
	destPath := filepath.Join(e.projectsRoot, projectID, "raw")

	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("lettura directory sorgente fallita: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() { continue }
		srcFile := filepath.Join(sourcePath, entry.Name())
		dstFile := filepath.Join(destPath, entry.Name())
		data, err := os.ReadFile(srcFile)
		if err != nil { continue }
		if err := os.WriteFile(dstFile, data, 0644); err != nil {
			fmt.Fprintf(w, "Warning: copia %s fallita: %v\n", entry.Name(), err)
			continue
		}
		fmt.Fprintf(w, "Copiato: %s\n", entry.Name())
	}

	e.updateProgress(task.Id, 80, "running")

	for _, entry := range entries {
		if entry.IsDir() { continue }
		tableName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		filePath := filepath.Join(destPath, entry.Name())
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		var createSQL string
		if ext == ".csv" {
			createSQL = fmt.Sprintf(`CREATE OR REPLACE VIEW "%s" AS SELECT * FROM read_csv_auto('%s')`, tableName, strings.ReplaceAll(filePath, "'", "''"))
		} else if ext == ".json" {
			createSQL = fmt.Sprintf(`CREATE OR REPLACE VIEW "%s" AS SELECT * FROM read_json_auto('%s')`, tableName, strings.ReplaceAll(filePath, "'", "''"))
		} else if ext == ".parquet" {
			createSQL = fmt.Sprintf(`CREATE OR REPLACE VIEW "%s" AS SELECT * FROM read_parquet('%s')`, tableName, strings.ReplaceAll(filePath, "'", "''"))
		} else { continue }
		if _, err := e.db.Exec(createSQL); err != nil {
			fmt.Fprintf(w, "Warning: vista %s fallita: %v\n", tableName, err)
		}
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Completato: dati copiati da %s\n", config.Source)
	return nil
}

func (e *Engine) runDynamic(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct { Code string `json:"code"` }
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil { return err }
	if config.Code == "" {
		return fmt.Errorf("codice vuoto nel config")
	}

	if err := validateCode(config.Code); err != nil {
		return fmt.Errorf("codice non consentito: %w", err)
	}

	sandboxCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "aleph-run-*")
	if err != nil { return err }
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFile, []byte(config.Code), 0644); err != nil { return err }
	binaryPath := filepath.Join(tmpDir, "conn")

	cmdBuild := exec.CommandContext(sandboxCtx, "go", "build", "-o", binaryPath, tmpFile)
	cmdBuild.Dir = tmpDir
	if out, err := cmdBuild.CombinedOutput(); err != nil {
		fmt.Fprintf(w, "Build Error: %s\n", string(out))
		return err
	}

	cmdRun := exec.CommandContext(sandboxCtx, binaryPath)
	cmdRun.Stdout = w
	cmdRun.Stderr = w
	cmdRun.Dir = tmpDir
	cmdRun.Env = []string{
		fmt.Sprintf("ALEPH_PROJECT_PATH=%s", filepath.Join(e.projectsRoot, projectID)),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		"HOME=" + tmpDir,
	}
	return cmdRun.Run()
}

var blockedImports = map[string]bool{
	"os/signal": true, "syscall": true, "net": true, "os/exec": true,
}

func validateCode(code string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "dynamic.go", code, parser.ImportsOnly)
	if err != nil {
		return fmt.Errorf("codice Go non valido: %w", err)
	}
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			path = imp.Path.Value
		}
		if blockedImports[path] {
			return fmt.Errorf("import %s non consentito nel codice dinamico", path)
		}
		for blockedPath := range blockedImports {
			if strings.HasPrefix(path, blockedPath) && path != blockedPath {
				return fmt.Errorf("import %s non consentito (sotto-package di modulo bloccato)", path)
			}
		}
	}
	return nil
}

type emailConfig struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Pass     string `json:"pass"`
	Folder   string `json:"folder"`
}

func (e *Engine) runEmailFetch(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config emailConfig
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("invalid email config: %v", err)
	}
	if config.Host == "" || config.User == "" || config.Pass == "" {
		return fmt.Errorf("email config requires host, user, and pass")
	}
	if config.Folder == "" {
		config.Folder = "INBOX"
	}

	fmt.Fprintf(w, "Connecting to IMAP server: %s\n", config.Host)
	
	addr := config.Host
	if !containsColon(addr) {
		addr = addr + ":993"
	}

	tableName := task.Id
	projectPath := filepath.Join(e.projectsRoot, projectID)
	
	escapedPass := strconv.Quote(config.Pass)
	escapedUser := strconv.Quote(config.User)
	escapedFolder := strconv.Quote(config.Folder)

	escapedHost := strconv.Quote(config.Host)
	escapedAddr := strconv.Quote(addr)

	script := fmt.Sprintf(`
import imaplib, email, json, sys, csv, io
from email.header import decode_header

def decode_str(s):
    if s is None: return ""
    parts = decode_header(s)
    result = []
    for part, enc in parts:
        if isinstance(part, bytes):
            result.append(part.decode(enc or 'utf-8', errors='replace'))
        else:
            result.append(part)
    return ''.join(result)

host, port = %s.split(':') if ':' in %s else (%s, 993)
mail = imaplib.IMAP4_SSL(host, int(port))
mail.login(%s, %s)
mail.select(%s, True)
_, msg_ids = mail.search(None, 'ALL')
ids = msg_ids[0].split()
rows = []
for mid in ids[-200:]:
    _, data = mail.fetch(mid, '(RFC822)')
    if not data or not data[0]: continue
    msg = email.message_from_bytes(data[0][1])
    row = {'subject': decode_str(msg['Subject']), 'from': decode_str(msg['From']), 'date': decode_str(msg['Date']), 'message_id': decode_str(msg['Message-ID'])}
    if msg.is_multipart():
        for part in msg.walk():
            ct = part.get_content_type()
            if ct == 'text/plain':
                body = part.get_payload(decode=True)
                if body: row['body'] = body.decode('utf-8', errors='replace')[:2000]
                break
    else:
        body = msg.get_payload(decode=True)
        if body: row['body'] = body.decode('utf-8', errors='replace')[:2000]
    rows.append(row)
mail.logout()

if rows:
    fields = list(rows[0].keys())
    out = io.StringIO()
    writer = csv.DictWriter(out, fieldnames=fields)
    writer.writeheader()
    writer.writerows(rows)
    print(out.getvalue())
`, escapedAddr, escapedHost, escapedHost, escapedUser, escapedPass, escapedFolder)

	tmpDir, err := os.MkdirTemp("", "aleph-email-*")
	if err != nil { return err }
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "fetch_emails.py")
	os.WriteFile(scriptPath, []byte(script), 0600)

	cmd := exec.CommandContext(ctx, "python3", scriptPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	fmt.Fprintf(w, "Fetching emails from %s...\n", config.Folder)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(w, "Email fetch error: %s\n%s\n", err, stderr.String())
		return fmt.Errorf("email fetch failed: %v - %s", err, stderr.String())
	}

	csvData := stdout.String()
	if csvData == "" {
		fmt.Fprintf(w, "No emails found\n")
		return nil
	}

	csvPath := filepath.Join(projectPath, tableName+".csv")
	os.MkdirAll(projectPath, 0755)
	if err := os.WriteFile(csvPath, []byte(csvData), 0644); err != nil {
		return fmt.Errorf("failed to write CSV: %v", err)
	}

	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" AS SELECT * FROM read_csv_auto('%s', ignore_errors=true)`, tableName, csvPath)
	fmt.Fprintf(w, "Creating table '%s' in DuckDB...\n", tableName)
	if _, err := e.db.Exec(createSQL); err != nil {
		return fmt.Errorf("duckdb create table failed: %v", err)
	}

	fmt.Fprintf(w, "Email ingestion complete. Table '%s' created.\n", tableName)
	return nil
}

func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' { return true }
	}
	return false
}

func looksLikeNonDecimalIP(host string) bool {
	if host == "" {
		return false
	}
	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		for _, p := range parts {
			if looksLikeNonDecimalPart(p) {
				return true
			}
		}
		return false
	}
	if len(parts) == 1 {
		if _, err := strconv.Atoi(host); err == nil {
			return len(host) > 3
		}
	}
	return false
}

func looksLikeNonDecimalPart(part string) bool {
	if part == "" {
		return false
	}
	if part[0] == '0' && len(part) > 1 {
		if part[1] != 'x' && part[1] != 'X' {
			return true
		}
	}
	if len(part) > 2 && (part[0:2] == "0x" || part[0:2] == "0X") {
		return true
	}
	return false
}

var ssrfBlockedIPNets []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12",
		"192.168.0.0/16", "169.254.0.0/16",
	} {
		_, n, err := net.ParseCIDR(cidr)
		if err == nil {
			ssrfBlockedIPNets = append(ssrfBlockedIPNets, n)
		}
	}
}

func blockSSRF(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("url vuota")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("url non valida: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("host vuoto")
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".internal") || strings.HasSuffix(lower, ".arpa") {
		return fmt.Errorf("host locale non permesso: %s", host)
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return fmt.Errorf("indirizzo IP privato/loopback non permesso: %s", host)
		}
	}
	for _, n := range ssrfBlockedIPNets {
		if ip := net.ParseIP(host); ip != nil && n.Contains(ip) {
			return fmt.Errorf("indirizzo IP bloccato: %s", host)
		}
	}
	if looksLikeNonDecimalIP(host) {
		return fmt.Errorf("formato IP non decimale sospetto: %s", host)
	}
	return nil
}

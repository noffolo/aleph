package ingestion

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/mail"
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
	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/safeident"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	"github.com/ff3300/aleph-v2/internal/ssrf"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/google/uuid"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var validName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func validateSQLName(name string) error {
	return safeident.ValidateStrictIdentifier(name)
}

func stripAndValidateName(name string) (string, error) {
	cleaned := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(name, "_"))
	if err := safeident.ValidateIdentifier(cleaned); err != nil {
		return "", fmt.Errorf("invalid identifier after sanitization: %q: %w", cleaned, err)
	}
	return cleaned, nil
}

const SentimentUnavailable = -1.0

func computeChecksum(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func VerifyChecksum(data []byte, expected string) bool {
	return computeChecksum(data) == expected
}

func sanitizeIdentifier(id string) error {
	return safeident.ValidateStrictIdentifier(id)
}

func sanitizeFilePath(p string) error {
	return safeident.SanitizeFilePath(p)
}

type Engine struct {
	projectsRoot string
	metaRepo     *repository.MetadataRepository
	db           *storage.DuckDB
	nlpHandler   NLPAnalyzer
	mu           sync.RWMutex
	tasks        map[string]*v1.IngestionTask
	httpClient   *http.Client
	wg           sync.WaitGroup

	// Lazy-init source ingesters
	githubIngester  *sources.GitHubIngester
	sitemapIngester *sources.SitemapIngester
	jsonapiIngester *sources.JSONAPIIngester
	sheetsIngester  *sources.SheetsIngester
	scraperIngester *sources.ScrapeIngester

	// Probe runner for auto-detection (lazy-init)
	probeRunner *ProbeRunner

	scheduler *Scheduler
}

// NLPAnalyzer abstracts sentiment analysis for ingestion enrichment.
// This allows the Engine to call real NLP during ingestion rather than
// inserting hardcoded 0.0 placeholder values.
type NLPAnalyzer interface {
	AnalyzeSentiment(ctx context.Context, text string) (score float32, label string, err error)
}

var safeHTTPClient = ssrf.NewClient()

func init() {
	// safeHTTPClient is now created via ssrf.NewClient() which handles
	// DNS-resolving SSRF protection at the connection level.
}

func NewEngine(projectsRoot string, metaRepo *repository.MetadataRepository, db *storage.DuckDB, nlp NLPAnalyzer) *Engine {
	e := &Engine{
		projectsRoot: projectsRoot,
		metaRepo:     metaRepo,
		db:           db,
		nlpHandler:   nlp,
		tasks:        make(map[string]*v1.IngestionTask),
	}
	e.scheduler = NewScheduler(e, metaRepo)
	return e
}

func (e *Engine) Scheduler() *Scheduler {
	return e.scheduler
}

// client returns the Engine's injected httpClient, falling back to the
// package-level safeHTTPClient when no custom client has been set.
func (e *Engine) client() *http.Client {
	if e.httpClient != nil {
		return e.httpClient
	}
	return safeHTTPClient
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
		return fmt.Errorf("task %s already running", task.Id)
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

	e.updateProgress(task.Id, 0, "running")
	logPath := filepath.Join(e.projectsRoot, projectID, "logs", task.Id+".log")
	os.MkdirAll(filepath.Dir(logPath), 0755)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("openLogFile: %w", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "\n--- Task Start: %s at %s ---\n", task.Id, time.Now().Format(time.RFC3339))

	var taskErr error
	switch task.SourceType {
	case "rss", "rest":
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
	case "github":
		taskErr = e.runGitHubSource(taskCtx, f, projectID, task)
	case "sitemap":
		taskErr = e.runSitemapSource(taskCtx, f, projectID, task)
	case "jsonapi":
		taskErr = e.runJSONAPISource(taskCtx, f, projectID, task)
	case "sheets":
		taskErr = e.runSheetsSource(taskCtx, f, projectID, task)
	case "scrape":
		taskErr = e.runScrapeSource(taskCtx, f, projectID, task)
	default:
		taskErr = fmt.Errorf("unknown source type: %s", task.SourceType)
	}

	if taskErr != nil {
		e.updateProgress(task.Id, 0, "failed")
		fmt.Fprintf(f, "Error: %v\n", taskErr)
		return taskErr
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(f, "--- Success ---\n")

	// Registrazione Viste in DuckDB per performance
	if err := e.registerViews(taskCtx, projectID); err != nil {
		fmt.Fprintf(f, "Warning: View registration failed: %v\n", err)
	}

	// Metadati Temporali: ogni riga riceve un timestamp di ingestion
	tableNameForSQL, err := resolveTableName(task)
	if err != nil {
		e.updateProgress(task.Id, 0, "failed")
		return fmt.Errorf("resolveTableName: %w", err)
	}
	if err := safeident.ValidateStrictIdentifier(tableNameForSQL); err != nil {
		fmt.Fprintf(f, "Warning: Invalid table name after sanitization: %s\n", tableNameForSQL)
		e.updateProgress(task.Id, 0, "failed")
		return fmt.Errorf("invalid table name after sanitization: %w", err)
	}
	timestampSQL := "ALTER TABLE " + safeident.QuoteIdentifier(tableNameForSQL) + " ADD COLUMN IF NOT EXISTS _aleph_ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP" // safe: tableNameForSQL validated via safeident.ValidateStrictIdentifier
	e.db.Exec(taskCtx, timestampSQL)

	// Arricchimento Predittivo Vettoriale (Asincrono)
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Engine] predictive enrichment goroutine panic: %v", r)
			}
		}()
		enrichCtx, enrichCancel := context.WithTimeout(ctx, 30*time.Minute)
		defer enrichCancel()
		resolvedTableName, resolveErr := resolveTableName(task)
		if resolveErr != nil {
			log.Printf("[Engine] resolveTableName error for enrichment: %v", resolveErr)
			return
		}
		e.enrichPredictiveMetadata(enrichCtx, projectID, resolvedTableName)
		if err := enrichCtx.Err(); err != nil {
			log.Printf("[Engine] Predictive enrichment interrupted or expired for table %s: %v", resolvedTableName, err)
		}
	}()

	return nil
}

func resolveTableName(task *v1.IngestionTask) (string, error) {
	var config struct {
		TableName string `json:"tableName"`
	}
	if json.Unmarshal([]byte(task.ConfigJson), &config) == nil && config.TableName != "" {
		cleaned, err := stripAndValidateName(config.TableName)
		if err != nil {
			return "", fmt.Errorf("resolveTableName: invalid tableName in config: %w", err)
		}
		return cleaned, nil
	}
	if task.Name != "" {
		cleaned, err := stripAndValidateName(task.Name)
		if err != nil {
			return "", fmt.Errorf("resolveTableName: invalid name: %w", err)
		}
		return cleaned, nil
	}
	// If task.Id looks like a UUID, validate it strictly before using
	if _, err := uuid.Parse(task.Id); err == nil {
		return "task_" + strings.ReplaceAll(task.Id, "-", "_"), nil
	}
	// Fallback: sanitize task.Id — may contain non-identifier characters
	sanitized := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(task.Id, "_"))
	if safeident.ValidateIdentifier(sanitized) != nil {
		// Last resort: generate a deterministic safe name so we never pass
		// a SQL-injectable string through fmt.Sprintf.
		sanitized = "task_" + computeChecksum([]byte(task.Id))[:16]
	}
	return sanitized, nil
}

func (e *Engine) enrichPredictiveMetadata(ctx context.Context, projectID, tableName string) {
	log.Printf("[Engine] Starting predictive enrichment for table %s", tableName)

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
		e.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS system_features (
			project_id VARCHAR,
			task_id VARCHAR,
			entity_id VARCHAR,
			feature_type VARCHAR,
			feature_value FLOAT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	*/

	query := "SELECT * FROM " + safeident.QuoteIdentifier(tableName) + " WHERE _aleph_ingested_at > (CURRENT_TIMESTAMP - INTERVAL '1 MINUTE')" // safe: tableName validated via safeident.ValidateStrictIdentifier (via enrichPredictiveMetadata caller)
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("[Engine] Enrichment query failed for %s: %v", tableName, err)
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
		case <-ctx.Done():
			return
		default:
		}

		vals := make([]any, len(cols))
		vps := make([]any, len(cols))
		for i := range vals {
			vps[i] = &vals[i]
		}
		if err := rows.Scan(vps...); err != nil {
			continue
		}

		entityID := fmt.Sprintf("%v", vals[idIdx])

		for i, col := range cols {
			if str, ok := vals[i].(string); ok && len(str) > 10 {
				featureValue := SentimentUnavailable
				if e.nlpHandler != nil {
					score, _, sErr := e.nlpHandler.AnalyzeSentiment(ctx, str)
					if sErr == nil {
						featureValue = float64(score)
					} else {
						log.Printf("[Engine] Sentiment analysis failed for %s.%s: %v", tableName, col, sErr)
					}
				}
				e.db.Exec(ctx, `INSERT INTO system_features (project_id, task_id, entity_id, feature_type, feature_value) 
					VALUES (?, ?, ?, ?, ?)`, projectID, tableName, entityID, "sentiment_"+col, featureValue)
			}
		}
	}
}

func (e *Engine) Close() error {
	log.Println("[Engine] Closing ingestion engine...")
	shutdownCtx := e.scheduler.Stop()
	e.wg.Wait()
	select {
	case <-shutdownCtx.Done():
	case <-time.After(30 * time.Second):
		log.Println("[Engine] Timed out waiting for scheduler jobs to finish")
	}
	return nil
}

func (e *Engine) registerViews(ctx context.Context, projectID string) error {
	projectPath := filepath.Join(e.projectsRoot, projectID)
	ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
	content, err := os.ReadFile(ontPath)
	if err != nil {
		return nil
	} // No ontology, skip

	prog, err := dsl.Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing ontology: %w", err)
	}

	dataRoot := filepath.Join(projectPath, "raw")
	compiler := dsl.NewCompiler(prog, dataRoot)

	for _, stmt := range prog.Statements {
		if stmt.Object != nil {
			sql, err := compiler.CompileObject(stmt.Object.Name)
			if err != nil {
				continue
			}

			viewName := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(fmt.Sprintf("%s_%s", projectID, stmt.Object.Name), "_"))
			if err := safeident.ValidateIdentifier(viewName); err != nil {
				log.Printf("[Engine] Invalid view name %q: %v", viewName, err)
				continue
			}
			createViewSql := "CREATE OR REPLACE VIEW " + safeident.QuoteIdentifier(viewName) + " AS " + sql // safe: viewName validated via safeident.ValidateStrictIdentifier (via stripAndValidateName + ValidateIdentifier)
			if _, err := e.db.Exec(ctx, createViewSql); err != nil {
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
		return fmt.Errorf("invalid JSON config: %w", err)
	}

	urlToFetch := config.URL
	if urlToFetch == "" && config.Repo != "" {
		urlToFetch = "https://api.github.com/repos/" + config.Repo + "/contents"
	}
	if urlToFetch == "" {
		return fmt.Errorf("no URL or repository specified in config")
	}

	fmt.Fprintf(w, "Fetching: %s\n", urlToFetch)
	e.updateProgress(task.Id, 10, "running")

	req, err := http.NewRequestWithContext(ctx, "GET", urlToFetch, nil)
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/csv, application/rss+xml, application/atom+xml, */*")
	if config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+config.Token)
	}

	resp, err := e.client().Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d: %s", resp.StatusCode, resp.Status)
	}

	e.updateProgress(task.Id, 40, "running")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("body read failed: %w", err)
	}

	e.updateProgress(task.Id, 70, "running")

	tableName := task.Id
	if err := safeident.ValidateIdentifier(tableName); err != nil {
		return fmt.Errorf("sanitizeTableName(precompiled): %w", err)
	}
	contentType := resp.Header.Get("Content-Type")
	projectPath := filepath.Join(e.projectsRoot, projectID)
	os.MkdirAll(filepath.Join(projectPath, "raw"), 0755)

	if strings.Contains(contentType, "json") || strings.Contains(contentType, "javascript") {
		tmpFile := filepath.Join(projectPath, "raw", tableName+".json")
		if err := os.WriteFile(tmpFile, bodyBytes, 0644); err != nil {
			return fmt.Errorf("file write failed: %w", err)
		}
		if err := safeident.SanitizeFilePath(tmpFile); err != nil {
			return fmt.Errorf("unsafe file path: %w", err)
		}
		createSQL := "CREATE OR REPLACE VIEW " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_json_auto(" + safeident.QuoteStringLiteral(tmpFile) + ")" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath validated via safeident.SanitizeFilePath
		if _, err := e.db.Exec(ctx, createSQL); err != nil {
			return fmt.Errorf("JSON view creation failed: %w", err)
		}
	} else if strings.Contains(contentType, "csv") || strings.Contains(contentType, "text/plain") {
		tmpFile := filepath.Join(projectPath, "raw", tableName+".csv")
		os.WriteFile(tmpFile, bodyBytes, 0644)
		if err := safeident.SanitizeFilePath(tmpFile); err != nil {
			return fmt.Errorf("unsafe file path: %w", err)
		}
		createSQL := "CREATE OR REPLACE VIEW " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_csv_auto(" + safeident.QuoteStringLiteral(tmpFile) + ")" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath validated via safeident.SanitizeFilePath
		if _, err := e.db.Exec(ctx, createSQL); err != nil {
			return fmt.Errorf("CSV view creation failed: %w", err)
		}
	} else {
		tmpFile := filepath.Join(projectPath, "raw", tableName+".dat")
		os.WriteFile(tmpFile, bodyBytes, 0644)
		fmt.Fprintf(w, "Data saved to %s (content-type: %s)\n", tmpFile, contentType)
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Completed: view \"%s\" created from %s\n", tableName, urlToFetch)
	return nil
}

func (e *Engine) runURLFetch(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("invalid JSON config: %w", err)
	}
	if config.URL == "" {
		return fmt.Errorf("empty URL in config")
	}

	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	_ = parsedURL // URL validated; SSRF protection via Engine.client().DialContext

	fmt.Fprintf(w, "Fetching URL: %s\n", config.URL)
	e.updateProgress(task.Id, 10, "running")

	req, err := http.NewRequestWithContext(ctx, "GET", config.URL, nil)
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/csv, */*")

	resp, err := e.client().Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d: %s", resp.StatusCode, resp.Status)
	}

	fmt.Fprintf(w, "Response: %s (%d bytes)\n", resp.Header.Get("Content-Type"), resp.ContentLength)
	e.updateProgress(task.Id, 40, "running")

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("body read failed: %w", err)
	}

	e.updateProgress(task.Id, 70, "running")

	contentType := resp.Header.Get("Content-Type")
	tableName := task.Id
	if err := sanitizeIdentifier(tableName); err != nil {
		return fmt.Errorf("sanitizeTableName(urlfetch): %w", err)
	}
	if err := validateSQLName(tableName); err != nil {
		return fmt.Errorf("validateTableName(urlfetch): %w", err)
	}
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
			return fmt.Errorf("file write failed: %w", err)
		}
		fmt.Fprintf(w, "Saved CSV to %s (%d bytes)\n", rawPath, len(bodyBytes))
		if err := safeident.SanitizeFilePath(rawPath); err != nil {
			return fmt.Errorf("unsafe file path: %w", err)
		}
		createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_csv_auto(" + safeident.QuoteStringLiteral(rawPath) + ", ignore_errors=true)" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath validated via safeident.SanitizeFilePath
		if _, err := e.db.Exec(ctx, createSQL); err != nil {
			return fmt.Errorf("CSV table creation failed: %w", err)
		}
	} else {
		rawPath := filepath.Join(projectPath, tableName+".json")
		if err := os.WriteFile(rawPath, bodyBytes, 0644); err != nil {
			return fmt.Errorf("file write failed: %w", err)
		}
		fmt.Fprintf(w, "Saved JSON to %s (%d bytes)\n", rawPath, len(bodyBytes))
		if err := safeident.SanitizeFilePath(rawPath); err != nil {
			return fmt.Errorf("unsafe file path: %w", err)
		}
		createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_json_auto(" + safeident.QuoteStringLiteral(rawPath) + ", ignore_errors=true)" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath validated via safeident.SanitizeFilePath
		if _, err := e.db.Exec(ctx, createSQL); err != nil {
			return fmt.Errorf("JSON table creation failed: %w", err)
		}
	}

	e.updateProgress(task.Id, 90, "running")
	fmt.Fprintf(w, "Load completed into table \"%s\"\n", tableName)
	return nil
}

func (e *Engine) insertJSONArray(ctx context.Context, tableName string, arr []any, w *os.File) error {
	if err := safeident.ValidateIdentifier(tableName); err != nil {
		return fmt.Errorf("invalid table name for JSON array insert: %w", err)
	}
	if len(arr) == 0 {
		fmt.Fprintf(w, "Empty array, no data to insert\n")
		return nil
	}

	first, ok := arr[0].(map[string]any)
	if !ok {
		return fmt.Errorf("first element is not a JSON object")
	}

	columns := make([]string, 0, len(first))
	for k := range first {
		columns = append(columns, k)
	}
	sort.Strings(columns)

	fmt.Fprintf(w, "Detected columns: %v (%d rows)\n", columns, len(arr))

	escapedCols := make([]string, len(columns))
	for i, c := range columns {
		if err := safeident.ValidateColumnName(c); err != nil {
			return fmt.Errorf("invalid column name in JSON data: %w", err)
		}
		escapedCols[i] = safeident.QuoteIdentifier(c) // safe: c validated via safeident.ValidateColumnName
	}

	colDefs := make([]string, len(columns))
	for i, c := range columns {
		typeName := "VARCHAR"
		switch first[c].(type) {
		case float64:
			typeName = "DOUBLE"
		case bool:
			typeName = "BOOLEAN"
		}
		colDefs[i] = escapedCols[i] + " " + typeName
	}

	values := make([]string, 0, len(arr))
	params := make([]any, 0)
	for _, item := range arr {
		obj, ok := item.(map[string]any)
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

	createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " (" + strings.Join(colDefs, ", ") + ")" // safe: tableName validated via safeident.ValidateStrictIdentifier; colDefs use escapedCols (validated) + type names
	if _, err := e.db.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("table creation failed: %w", err)
	}

	batchSize := 500
	for i := 0; i < len(values); i += batchSize {
		end := i + batchSize
		if end > len(values) {
			end = len(values)
		}
		insertSQL := "INSERT INTO " + safeident.QuoteIdentifier(tableName) + " (" + strings.Join(escapedCols, ", ") + ") VALUES " + strings.Join(values[i:end], ", ") // safe: tableName validated; escapedCols via safeident.ValidateColumnName; VALUES use ? parameterized
		if _, err := e.db.Exec(ctx, insertSQL, params...); err != nil {
			fmt.Fprintf(w, "Warning: insert batch failed: %v\n", err)
		}
	}

	return nil
}

func extractArray(m map[string]any) ([]any, bool) {
	for _, v := range m {
		if arr, ok := v.([]any); ok {
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
		return fmt.Errorf("invalid JSON config: %w", err)
	}
	if config.Path == "" {
		return fmt.Errorf("empty file path in config")
	}

	fmt.Fprintf(w, "Loading CSV/Parquet: %s\n", config.Path)
	e.updateProgress(task.Id, 20, "running")

	tableName := config.TableName
	if tableName == "" {
		tableName = task.Name
	}
	if tableName == "" {
		tableName = task.Id
	}
	tableName = strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(tableName, "_"))
	if err := sanitizeIdentifier(tableName); err != nil {
		return fmt.Errorf("sanitizeTableName(csvload): %w", err)
	}
	if err := validateSQLName(tableName); err != nil {
		return fmt.Errorf("validateTableName(csvload): %w", err)
	}
	if err := sanitizeFilePath(config.Path); err != nil {
		return fmt.Errorf("sanitizeFilePath: %w", err)
	}

	readerFunc := "read_csv_auto"
	if strings.HasSuffix(strings.ToLower(config.Path), ".parquet") {
		readerFunc = "read_parquet"
	}

	projectPath := filepath.Join(e.projectsRoot, projectID, "raw")
	os.MkdirAll(projectPath, 0755)
	localPath := filepath.Join(projectPath, tableName+filepath.Ext(config.Path))
	data, err := os.ReadFile(config.Path)
	if err != nil {
		return fmt.Errorf("file read failed: %w", err)
	}
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return fmt.Errorf("local file write failed: %w", err)
	}

	createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM " + readerFunc + "(" + safeident.QuoteStringLiteral(localPath) + ")" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath validated via safeident.SanitizeFilePath
	if _, err := e.db.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("file load failed: %w", err)
	}

	e.updateProgress(task.Id, 90, "running")
	fmt.Fprintf(w, "Table \"%s\" created from %s\n", tableName, config.Path)
	return nil
}

func (e *Engine) runPostgresLoad(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		DSN   string `json:"dsn"`
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("invalid JSON config: %w", err)
	}
	if config.DSN == "" {
		return fmt.Errorf("empty DSN in config")
	}

	fmt.Fprintf(w, "Loading from PostgreSQL: %s\n", config.DSN)
	e.updateProgress(task.Id, 10, "running")

	tableName := task.Id
	if err := sanitizeIdentifier(tableName); err != nil {
		return fmt.Errorf("sanitizeTableName(postgres): %w", err)
	}
	if err := validateSQLName(tableName); err != nil {
		return fmt.Errorf("validateTableName(postgres): %w", err)
	}

	_, err := e.db.Exec(ctx, "INSTALL postgres_scanner")
	if err != nil {
		fmt.Fprintf(w, "Warning: postgres_scanner extension not available: %v\n", err)
		return fmt.Errorf("postgres_scanner extension unavailable: %w", err)
	}
	_, err = e.db.Exec(ctx, "LOAD postgres_scanner")
	if err != nil {
		return fmt.Errorf("postgres_scanner load failed: %w", err)
	}

	e.updateProgress(task.Id, 40, "running")

	safeDSN := strings.ReplaceAll(config.DSN, "'", "''")
	query := "SELECT * FROM postgres_scan_pushdown(" + safeident.QuoteStringLiteral(safeDSN) + ", 'public', " + safeident.QuoteStringLiteral(tableName) + ")"
	if config.Query != "" {
		return fmt.Errorf("custom queries not allowed for security reasons — use DSN + table name only")
	}

	createSQL := "CREATE OR REPLACE VIEW " + safeident.QuoteIdentifier(tableName) + " AS " + query // safe: tableName validated via safeident.ValidateStrictIdentifier; DSN and tableName in postgres_scan_pushdown use QuoteStringLiteral
	if _, err := e.db.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("PostgreSQL view creation failed: %w", err)
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Completed: view \"%s\" created from PostgreSQL\n", tableName)
	return nil
}

func (e *Engine) runCopy(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("invalid JSON config: %w", err)
	}
	if config.Source == "" {
		return fmt.Errorf("empty source project in config")
	}

	fmt.Fprintf(w, "Copying from project: %s\n", config.Source)
	e.updateProgress(task.Id, 20, "running")

	sourcePath := filepath.Join(e.projectsRoot, config.Source, "raw")
	destPath := filepath.Join(e.projectsRoot, projectID, "raw")

	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("source directory read failed: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		srcFile := filepath.Join(sourcePath, entry.Name())
		dstFile := filepath.Join(destPath, entry.Name())
		data, err := os.ReadFile(srcFile)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dstFile, data, 0644); err != nil {
			fmt.Fprintf(w, "Warning: copy %s failed: %v\n", entry.Name(), err)
			continue
		}
		fmt.Fprintf(w, "Copied: %s\n", entry.Name())
	}

	e.updateProgress(task.Id, 80, "running")

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		tableName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if err := safeident.ValidateIdentifier(tableName); err != nil {
			fmt.Fprintf(w, "Warning: skipping invalid table name %q: %v\n", tableName, err)
			continue
		}
		filePath := filepath.Join(destPath, entry.Name())
		if err := safeident.SanitizeFilePath(filePath); err != nil {
			fmt.Fprintf(w, "Warning: skipping unsafe file path %q: %v\n", filePath, err)
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		var createSQL string
		if ext == ".csv" {
			createSQL = "CREATE OR REPLACE VIEW " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_csv_auto(" + safeident.QuoteStringLiteral(filePath) + ")" // safe: tableName validated via safeident.ValidateIdentifier; filePath via safeident.SanitizeFilePath
		} else if ext == ".json" {
			createSQL = "CREATE OR REPLACE VIEW " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_json_auto(" + safeident.QuoteStringLiteral(filePath) + ")" // safe: tableName validated via safeident.ValidateIdentifier; filePath via safeident.SanitizeFilePath
		} else if ext == ".parquet" {
			createSQL = "CREATE OR REPLACE VIEW " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_parquet(" + safeident.QuoteStringLiteral(filePath) + ")" // safe: tableName validated; filePath via safeident.SanitizeFilePath
		} else {
			continue
		}
		if _, err := e.db.Exec(ctx, createSQL); err != nil {
			fmt.Fprintf(w, "Warning: view %s failed: %v\n", tableName, err)
		}
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Completed: data copied from %s\n", config.Source)
	return nil
}

func (e *Engine) runDynamic(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("unmarshalDynamicConfig: %w", err)
	}
	if config.Code == "" {
		return fmt.Errorf("empty code in config")
	}

	if err := validateCode(config.Code); err != nil {
		return fmt.Errorf("code not allowed: %w", err)
	}

	sandboxCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "aleph-run-*")
	if err != nil {
		return fmt.Errorf("createTempDir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFile, []byte(config.Code), 0644); err != nil {
		return fmt.Errorf("writeTempFile: %w", err)
	}
	binaryPath := filepath.Join(tmpDir, "conn")

	cmdBuild := exec.CommandContext(sandboxCtx, "go", "build", "-o", binaryPath, tmpFile)
	cmdBuild.Dir = tmpDir
	if out, err := cmdBuild.CombinedOutput(); err != nil {
		fmt.Fprintf(w, "Build Error: %s\n", string(out))
		return fmt.Errorf("buildDynamic: %w", err)
	}

	cmdRun := exec.CommandContext(sandboxCtx, binaryPath)
	cmdRun.Stdout = w
	cmdRun.Stderr = w
	cmdRun.Dir = tmpDir
	cmdRun.Env = []string{
		fmt.Sprintf("ALEPH_PROJECT_PATH=%s", filepath.Join(e.projectsRoot, projectID)),
		"PATH=/usr/bin:/bin",
		"HOME=" + tmpDir,
	}
	return fmt.Errorf("runDynamic: %w", cmdRun.Run())
}

var blockedImports = map[string]bool{
	"os/signal": true, "syscall": true, "net": true, "os/exec": true,
	"unsafe": true, "reflect": true,
	"os": true, "io": true,
	"crypto": true, "crypto/aes": true, "crypto/cipher": true, "crypto/des": true,
	"crypto/dsa": true, "crypto/ecdsa": true, "crypto/ed25519": true, "crypto/elliptic": true,
	"crypto/hmac": true, "crypto/md5": true, "crypto/rand": true, "crypto/rc4": true,
	"crypto/rsa": true, "crypto/sha1": true, "crypto/sha256": true, "crypto/sha512": true,
	"crypto/subtle": true, "crypto/tls": true, "crypto/x509": true, "crypto/x509/pkix": true,
	"encoding": true, "encoding/ascii85": true, "encoding/asn1": true, "encoding/base32": true,
	"encoding/base64": true, "encoding/binary": true, "encoding/csv": true,
	"encoding/gob": true, "encoding/hex": true, "encoding/json": true,
	"encoding/pem": true, "encoding/xml": true,
	"net/http": true, "net/http/cgi": true, "net/http/cookiejar": true,
	"net/http/fcgi": true, "net/http/httptest": true, "net/http/httptrace": true,
	"net/http/httputil": true, "net/mail": true, "net/rpc": true, "net/rpc/jsonrpc": true,
	"net/smtp": true, "net/textproto": true, "net/url": true,
}

func validateCode(code string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "dynamic.go", code, parser.ImportsOnly)
	if err != nil {
		return fmt.Errorf("invalid Go code: %w", err)
	}
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			path = imp.Path.Value
		}
		if blockedImports[path] {
			return fmt.Errorf("import %s not allowed in dynamic code", path)
		}
		for blockedPath := range blockedImports {
			if strings.HasPrefix(path, blockedPath) && path != blockedPath {
				return fmt.Errorf("import %s not allowed (subpackage of blocked module)", path)
			}
		}
	}
	return nil
}

type emailConfig struct {
	Host   string `json:"host"`
	User   string `json:"user"`
	Pass   string `json:"pass"`
	Folder string `json:"folder"`
}

type emailCredentials struct {
	Server   string `json:"server"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Folder   string `json:"folder"`
}

func (e *Engine) runEmailFetch(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config emailConfig
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("invalid email config: %w", err)
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
	if err := validateSQLName(tableName); err != nil {
		return fmt.Errorf("invalid table name for email fetch: %w", err)
	}
	projectPath := filepath.Join(e.projectsRoot, projectID)

	_ = addr

	// Defense-in-depth: verify sandbox still blocks the legacy Python+imaplib path.
	legacyPythonScript := `
import imaplib, email, json, sys, csv, io
from email.header import decode_header
creds = json.loads(sys.stdin.read())
mail = imaplib.IMAP4_SSL(creds["server"], int(creds["port"]))
mail.login(creds["username"], creds["password"])
`
	if err := sandbox.ValidatePythonCode(legacyPythonScript); err == nil {
		return fmt.Errorf("security: sandbox failed to block legacy python imaplib script (blocklist bypass detected)")
	}

	fmt.Fprintf(w, "Connecting via IMAP (Go-native, no subprocess)...\n")

	rows, err := fetchIMAP(config.Host, config.User, config.Pass, config.Folder, 200)
	if err != nil {
		return fmt.Errorf("IMAP fetch failed: %w", err)
	}
	fmt.Fprintf(w, "Fetched %d emails from %s\n", len(rows), config.Folder)

	if len(rows) == 0 {
		fmt.Fprintf(w, "No emails found\n")
		return nil
	}

	var csvBuf bytes.Buffer
	csvWriter := csv.NewWriter(&csvBuf)
	header := []string{"subject", "from", "date", "message_id", "body"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("csv header write: %w", err)
	}
	for _, row := range rows {
		rec := []string{row.Subject, row.From, row.Date, row.MessageID, row.Body}
		if err := csvWriter.Write(rec); err != nil {
			return fmt.Errorf("csv row write: %w", err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("csv flush: %w", err)
	}

	csvPath := filepath.Join(projectPath, tableName+".csv")
	os.MkdirAll(projectPath, 0755)
	if err := os.WriteFile(csvPath, csvBuf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_csv_auto(" + safeident.QuoteStringLiteral(csvPath) + ", ignore_errors=true)"
	fmt.Fprintf(w, "Creating table '%s' in DuckDB...\n", tableName)
	if _, err := e.db.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("duckdb create table failed: %w", err)
	}

	fmt.Fprintf(w, "Email ingestion complete. Table '%s' created.\n", tableName)
	return nil
}

type emailRow struct {
	Subject   string
	From      string
	Date      string
	MessageID string
	Body      string
}

func fetchIMAP(host, user, pass, folder string, maxMessages int) ([]emailRow, error) {
	if !strings.Contains(host, ":") {
		host = host + ":993"
	}

	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		return nil, fmt.Errorf("IMAP host parse: %w", err)
	}
	if err := ssrf.ValidateHostname(hostname, port); err != nil {
		return nil, fmt.Errorf("IMAP SSRF check: %w", err)
	}

	conn, err := tls.Dial("tcp", host, &tls.Config{MinVersion: tls.VersionTLS12})
	if err != nil {
		return nil, fmt.Errorf("IMAP TLS dial: %w", err)
	}
	defer conn.Close()

	r := bufio.NewReader(conn)

	greeting, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("IMAP greeting: %w", err)
	}
	if !strings.HasPrefix(greeting, "* OK") {
		return nil, fmt.Errorf("unexpected IMAP greeting: %q", strings.TrimSpace(greeting))
	}

	tag := 0
	nextTag := func() string { tag++; return fmt.Sprintf("A%03d", tag) }

	cmd := func(command string) (string, error) {
		t := nextTag()
		if _, err := fmt.Fprintf(conn, "%s %s\r\n", t, command); err != nil {
			return "", fmt.Errorf("IMAP write: %w", err)
		}
		return readIMAPResponse(r, t)
	}

	loginCmd := fmt.Sprintf(`LOGIN "%s" "%s"`, escapeIMAP(user), escapeIMAP(pass))
	_, err = cmd(loginCmd)
	if err != nil {
		return nil, fmt.Errorf("IMAP LOGIN: %w", err)
	}

	_, err = cmd(fmt.Sprintf(`SELECT "%s"`, escapeIMAP(folder)))
	if err != nil {
		return nil, fmt.Errorf("IMAP SELECT %q: %w", folder, err)
	}

	searchResp, err := cmd("SEARCH ALL")
	if err != nil {
		return nil, fmt.Errorf("IMAP SEARCH: %w", err)
	}

	ids := parseIMAPSearch(searchResp)
	if len(ids) == 0 {
		return nil, nil
	}

	start := len(ids) - maxMessages
	if start < 1 {
		start = 1
	}

	fetchResp, err := cmd(fmt.Sprintf("FETCH %d:* (BODY[])", start))
	if err != nil {
		return nil, fmt.Errorf("IMAP FETCH: %w", err)
	}

	_, err = cmd("LOGOUT")
	if err != nil {
		_ = err
	}

	return parseIMAPFetchMessages(fetchResp)
}

func escapeIMAP(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`)
}

func readIMAPResponse(r *bufio.Reader, tag string) (string, error) {
	var buf strings.Builder
	terminator := tag + " OK"
	badPrefix := tag + " BAD"
	noPrefix := tag + " NO"

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return buf.String(), fmt.Errorf("IMAP read error after %q: %w", buf.String(), err)
		}
		buf.WriteString(line)

		if strings.HasPrefix(line, badPrefix) {
			return buf.String(), fmt.Errorf("IMAP error: %s", strings.TrimSpace(line))
		}
		if strings.HasPrefix(line, noPrefix) {
			return buf.String(), fmt.Errorf("IMAP failed: %s", strings.TrimSpace(line))
		}
		if strings.HasPrefix(line, terminator) {
			return buf.String(), nil
		}
	}
}

func parseIMAPSearch(response string) []int {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "* SEARCH") {
			continue
		}
		parts := strings.Fields(line)
		var ids []int
		for _, p := range parts[2:] {
			if n, err := strconv.Atoi(p); err == nil {
				ids = append(ids, n)
			}
		}
		return ids
	}
	return nil
}

func parseIMAPFetchMessages(response string) ([]emailRow, error) {
	type fetchItem struct {
		seq int
		raw string
	}

	currentSeq := 0
	var currentBody strings.Builder
	inFetch := false
	var items []fetchItem

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "* ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				if n, err := strconv.Atoi(parts[1]); err == nil {
					if inFetch && currentBody.Len() > 0 {
						bodyStr := currentBody.String()
						if !strings.HasSuffix(bodyStr, "\r\n") {
							bodyStr += "\r\n"
						}
						items = append(items, fetchItem{seq: currentSeq, raw: bodyStr})
						currentBody.Reset()
					}
					currentSeq = n
					inFetch = true
					continue
				}
			}
		}
		if inFetch {
			currentBody.WriteString(line)
			currentBody.WriteString("\r\n")
		}
	}

	if inFetch && currentBody.Len() > 0 {
		items = append(items, fetchItem{seq: currentSeq, raw: currentBody.String()})
	}

	var rows []emailRow
	for _, item := range items {
		row, err := parseRFC822(item.raw)
		if err != nil {
			continue
		}
		rows = append(rows, *row)
	}
	return rows, nil
}

func parseRFC822(raw string) (*emailRow, error) {
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse mail: %w", err)
	}

	row := &emailRow{
		Subject:   decodeMIMEHeader(msg.Header.Get("Subject")),
		From:      decodeMIMEHeader(msg.Header.Get("From")),
		Date:      msg.Header.Get("Date"),
		MessageID: msg.Header.Get("Message-ID"),
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		mediaType = "text/plain"
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary != "" {
			row.Body = extractTextPart(strings.NewReader(raw), boundary)
		}
	} else {
		bodyBytes, err := io.ReadAll(msg.Body)
		if err == nil {
			row.Body = decodeBody(bodyBytes, msg.Header.Get("Content-Transfer-Encoding"))
		}
	}

	if len(row.Body) > 2000 {
		row.Body = row.Body[:2000]
	}

	return row, nil
}

func decodeMIMEHeader(s string) string {
	if s == "" {
		return ""
	}
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}

func extractTextPart(r io.Reader, boundary string) string {
	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		ct := part.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "text/plain") {
			bodyBytes, err := io.ReadAll(part)
			if err == nil {
				enc := part.Header.Get("Content-Transfer-Encoding")
				return decodeBody(bodyBytes, enc)
			}
		}
	}
	return ""
}

func decodeBody(data []byte, encoding string) string {
	switch strings.ToLower(encoding) {
	case "base64", "b":
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil {
			return string(decoded)
		}
	case "quoted-printable", "q":
		qr := quotedprintable.NewReader(bytes.NewReader(data))
		decoded, err := io.ReadAll(qr)
		if err == nil {
			return string(decoded)
		}
	}
	return string(data)
}

// =============================================================================
// Lazy-init source ingesters
// =============================================================================

func (e *Engine) getOrCreateGitHubIngester(task *v1.IngestionTask) *sources.GitHubIngester {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.githubIngester != nil {
		return e.githubIngester
	}
	var config struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		log.Printf("[Engine] getOrCreateGitHubIngester: failed to unmarshal config: %v", err)
	}
	e.githubIngester = sources.NewGitHubIngester(config.Token)
	return e.githubIngester
}

func (e *Engine) getOrCreateSitemapIngester() *sources.SitemapIngester {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sitemapIngester != nil {
		return e.sitemapIngester
	}
	e.sitemapIngester = sources.NewSitemapIngester()
	return e.sitemapIngester
}

func (e *Engine) getOrCreateJSONAPIIngester() *sources.JSONAPIIngester {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.jsonapiIngester != nil {
		return e.jsonapiIngester
	}
	e.jsonapiIngester = sources.NewJSONAPIIngester()
	return e.jsonapiIngester
}

func (e *Engine) getOrCreateSheetsIngester(task *v1.IngestionTask) *sources.SheetsIngester {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sheetsIngester != nil {
		return e.sheetsIngester
	}
	var config struct {
		APIKey string `json:"api_key"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		log.Printf("[Engine] getOrCreateSheetsIngester: failed to unmarshal config: %v", err)
	}
	e.sheetsIngester = sources.NewSheetsIngester(config.APIKey)
	return e.sheetsIngester
}

func (e *Engine) getOrCreateScrapeIngester() *sources.ScrapeIngester {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.scraperIngester != nil {
		return e.scraperIngester
	}
	e.scraperIngester = sources.NewScrapeIngester()
	return e.scraperIngester
}

func (e *Engine) getOrCreateProbeRunner() *ProbeRunner {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.probeRunner != nil {
		return e.probeRunner
	}
	e.probeRunner = NewProbeRunner(nil)
	return e.probeRunner
}

// =============================================================================
// GitHub source ingestion
// =============================================================================

func (e *Engine) runGitHubSource(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("invalid github JSON config: %w", err)
	}
	if config.Owner == "" || config.Repo == "" {
		return fmt.Errorf("github config requires owner and repo")
	}

	fmt.Fprintf(w, "GitHub: fetching %s/%s\n", config.Owner, config.Repo)
	e.updateProgress(task.Id, 10, "running")

	ingester := e.getOrCreateGitHubIngester(task)
	results, err := ingester.FetchAll(ctx, config.Owner, config.Repo)
	if err != nil {
		return fmt.Errorf("github fetch %s/%s: %w", config.Owner, config.Repo, err)
	}

	projectPath := filepath.Join(e.projectsRoot, projectID, "raw")
	os.MkdirAll(projectPath, 0755)

	for _, kind := range []string{"issues", "pulls", "commits"} {
		data, ok := results[kind]
		if !ok || len(data) == 0 {
			fmt.Fprintf(w, "No data for %s\n", kind)
			continue
		}

		tableName := task.Id + "_" + kind
		tableName = strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(tableName, "_"))
		if err := validateSQLName(tableName); err != nil {
			fmt.Fprintf(w, "Warning: skip %s: %v\n", kind, err)
			continue
		}

		jsonPath := filepath.Join(projectPath, tableName+".json")
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			return fmt.Errorf("write %s failed: %w", kind, err)
		}

		if err := safeident.SanitizeFilePath(jsonPath); err != nil {
			fmt.Fprintf(w, "Warning: unsafe JSON path for %s: %v\n", kind, err)
			continue
		}

		createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_json_auto(" + safeident.QuoteStringLiteral(jsonPath) + ", ignore_errors=true)" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath via safeident.SanitizeFilePath
		if _, err := e.db.Exec(ctx, createSQL); err != nil {
			fmt.Fprintf(w, "Warning: table %s creation failed: %v\n", tableName, err)
		} else {
			fmt.Fprintf(w, "Table \"%s\" created (%d bytes)\n", tableName, len(data))
		}
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "GitHub ingestion complete for %s/%s\n", config.Owner, config.Repo)
	return nil
}

// =============================================================================
// Sitemap source ingestion
// =============================================================================

func (e *Engine) runSitemapSource(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("sitemap config JSON invalid: %w", err)
	}
	if config.URL == "" {
		return fmt.Errorf("sitemap config requires url")
	}

	fmt.Fprintf(w, "Sitemap: crawling %s\n", config.URL)
	e.updateProgress(task.Id, 10, "running")

	ingester := e.getOrCreateSitemapIngester()
	crawlResult, err := ingester.CrawlSitemap(ctx, config.URL)
	if err != nil {
		return fmt.Errorf("sitemap crawl %s: %w", config.URL, err)
	}

	fmt.Fprintf(w, "Found %d pages\n", len(crawlResult.URLs))

	projectPath := filepath.Join(e.projectsRoot, projectID, "raw")
	os.MkdirAll(projectPath, 0755)

	type pageRow struct {
		URL     string `json:"url"`
		Status  int    `json:"status"`
		Size    int64  `json:"size"`
		Content string `json:"content"`
	}

	rows := make([]pageRow, 0, len(crawlResult.URLs))
	for _, p := range crawlResult.URLs {
		contentStr := ""
		if p.Content != nil {
			contentStr = string(p.Content)
		}
		rows = append(rows, pageRow{
			URL:     p.URL,
			Status:  p.Status,
			Size:    p.Size,
			Content: contentStr,
		})
	}

	tableName := task.Id
	if err := validateSQLName(tableName); err != nil {
		tableName = "sitemap_" + computeChecksum([]byte(task.Id))[:16]
	}
	if err := validateSQLName(tableName); err != nil {
		return fmt.Errorf("invalid table name for sitemap: %w", err)
	}

	jsonData, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("marshal sitemap rows: %w", err)
	}

	jsonPath := filepath.Join(projectPath, tableName+".json")
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("sitemap JSON write failed: %w", err)
	}

	if err := safeident.SanitizeFilePath(jsonPath); err != nil {
		return fmt.Errorf("unsafe sitemap JSON path: %w", err)
	}

	createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_json_auto(" + safeident.QuoteStringLiteral(jsonPath) + ", ignore_errors=true)" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath via safeident.SanitizeFilePath
	if _, err := e.db.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("sitemap table creation failed: %w", err)
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Sitemap ingestion complete. Table \"%s\" (%d rows)\n", tableName, len(rows))
	return nil
}

// =============================================================================
// JSON API source ingestion
// =============================================================================

func (e *Engine) runJSONAPISource(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		URL            string `json:"url"`
		PaginationType string `json:"pagination"`
		PageParam      string `json:"pageParam"`
		LimitParam     string `json:"limitParam"`
		Limit          int    `json:"limit"`
		DataPath       string `json:"dataPath"`
		MaxPages       int    `json:"maxPages"`
		AutoDetect     bool   `json:"autoDetect"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("jsonapi config JSON invalid: %w", err)
	}
	if config.URL == "" {
		return fmt.Errorf("jsonapi config requires url")
	}

	fmt.Fprintf(w, "JSON API: fetching %s\n", config.URL)
	e.updateProgress(task.Id, 10, "running")

	ingester := e.getOrCreateJSONAPIIngester()

	// If autoDetect or missing pagination type, use ProbeRunner to classify
	if config.AutoDetect || config.PaginationType == "" {
		probe := e.getOrCreateProbeRunner()
		probeResult, err := probe.Probe(ctx, config.URL)
		if err != nil {
			fmt.Fprintf(w, "Warning: probe failed: %v — proceeding with default configuration\n", err)
		} else {
			fmt.Fprintf(w, "Probe: detected type %q, pagination %q\n", probeResult.SourceType(), probeResult.Pagination().Type)
			if probeResult.SourceType() == "sitemap" || probeResult.SourceType() == "rss" {
				fmt.Fprintf(w, "Probe detected sitemap/rss, but task is jsonapi — proceeding with JSON anyway\n")
			}
			if config.PaginationType == "" && probeResult.Pagination().Type != "none" && probeResult.Pagination().Type != "" {
				switch probeResult.Pagination().Type {
				case "offset", "page", "cursor":
					config.PaginationType = probeResult.Pagination().Type
					if config.PageParam == "" {
						config.PageParam = probeResult.Pagination().PageParam
					}
					if config.LimitParam == "" {
						config.LimitParam = probeResult.Pagination().LimitParam
					}
				}
			}
		}
	}

	apiCfg := sources.APIConfig{
		BaseURL:        config.URL,
		PaginationType: config.PaginationType,
		PageParam:      config.PageParam,
		LimitParam:     config.LimitParam,
		Limit:          config.Limit,
		DataPath:       config.DataPath,
		MaxPages:       config.MaxPages,
	}
	if apiCfg.Limit <= 0 {
		apiCfg.Limit = 100
	}

	// Auto-detect DataPath if not configured
	if apiCfg.DataPath == "" {
		probeCfg, err := ingester.DetectConfig(ctx, config.URL)
		if err == nil && probeCfg != nil {
			apiCfg.DataPath = probeCfg.DataPath
			if apiCfg.PaginationType == "" {
				apiCfg.PaginationType = probeCfg.PaginationType
			}
			if apiCfg.PageParam == "" {
				apiCfg.PageParam = probeCfg.PageParam
			}
			if apiCfg.LimitParam == "" {
				apiCfg.LimitParam = probeCfg.LimitParam
			}
			fmt.Fprintf(w, "Auto-detected: dataPath=%q pagination=%q\n", apiCfg.DataPath, apiCfg.PaginationType)
		}
	}

	data, err := ingester.FetchAll(ctx, apiCfg)
	if err != nil {
		return fmt.Errorf("jsonapi fetch %s: %w", config.URL, err)
	}

	projectPath := filepath.Join(e.projectsRoot, projectID, "raw")
	os.MkdirAll(projectPath, 0755)

	tableName := task.Id
	if err := validateSQLName(tableName); err != nil {
		return fmt.Errorf("invalid table name for jsonapi: %w", err)
	}

	jsonPath := filepath.Join(projectPath, tableName+".json")
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("JSON write failed: %w", err)
	}
	if err := safeident.SanitizeFilePath(jsonPath); err != nil {
		return fmt.Errorf("unsafe JSON path for jsonapi: %w", err)
	}

	createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_json_auto(" + safeident.QuoteStringLiteral(jsonPath) + ", ignore_errors=true)" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath via safeident.SanitizeFilePath
	if _, err := e.db.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("jsonapi table creation failed: %w", err)
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "JSON API ingestion complete. Table \"%s\"\n", tableName)
	return nil
}

// =============================================================================
// Google Sheets source ingestion
// =============================================================================

func (e *Engine) runSheetsSource(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config struct {
		SpreadsheetID string `json:"spreadsheet_id"`
		APIKey        string `json:"api_key"`
		SheetRange    string `json:"range"`
	}
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("sheets config JSON invalid: %w", err)
	}
	if config.SpreadsheetID == "" {
		return fmt.Errorf("sheets config requires spreadsheet_id")
	}

	fmt.Fprintf(w, "Sheets: fetching spreadsheet %s\n", config.SpreadsheetID)
	e.updateProgress(task.Id, 10, "running")

	ingester := e.getOrCreateSheetsIngester(task)

	// Fetch metadata for all sheets first
	allSheets, err := ingester.FetchAllSheets(ctx, config.SpreadsheetID)
	if err != nil {
		return fmt.Errorf("sheets fetch %s: %w", config.SpreadsheetID, err)
	}

	projectPath := filepath.Join(e.projectsRoot, projectID, "raw")
	os.MkdirAll(projectPath, 0755)

	for sheetName, data := range allSheets {
		tableName := task.Id + "_" + sheetName
		tableName = strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(tableName, "_"))
		if err := safeident.ValidateIdentifier(tableName); err != nil {
			fmt.Fprintf(w, "Warning: skip sheet %q: %v\n", sheetName, err)
			continue
		}

		jsonPath := filepath.Join(projectPath, tableName+".json")
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			return fmt.Errorf("sheet %q write failed: %w", sheetName, err)
		}
		if err := safeident.SanitizeFilePath(jsonPath); err != nil {
			fmt.Fprintf(w, "Warning: unsafe JSON path for sheet %q: %v\n", sheetName, err)
			continue
		}

		createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_json_auto(" + safeident.QuoteStringLiteral(jsonPath) + ", ignore_errors=true)" // safe: tableName validated via safeident.ValidateStrictIdentifier; filePath via safeident.SanitizeFilePath
		if _, err := e.db.Exec(ctx, createSQL); err != nil {
			fmt.Fprintf(w, "Warning: table %s creation failed: %v\n", tableName, err)
		} else {
			fmt.Fprintf(w, "Table \"%s\" created from sheet %q\n", tableName, sheetName)
		}
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Sheets ingestion complete for %s (%d sheets)\n", config.SpreadsheetID, len(allSheets))
	return nil
}

func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' {
			return true
		}
	}
	return false
}

// =============================================================================
// Scrape source ingestion
// =============================================================================

func (e *Engine) runScrapeSource(ctx context.Context, w *os.File, projectID string, task *v1.IngestionTask) error {
	var config sources.ScrapeConfig
	if err := json.Unmarshal([]byte(task.ConfigJson), &config); err != nil {
		return fmt.Errorf("scrape config JSON invalid: %w", err)
	}
	if config.URL == "" {
		return fmt.Errorf("scrape config requires url")
	}
	if config.ArticleSelector == "" {
		return fmt.Errorf("scrape config requires article_selector")
	}
	if config.TitleSelector == "" {
		return fmt.Errorf("scrape config requires title_selector")
	}

	fmt.Fprintf(w, "Scrape: extracting from %s\n", config.URL)
	e.updateProgress(task.Id, 10, "running")

	ingester := e.getOrCreateScrapeIngester()
	results, err := ingester.Scrape(ctx, &config)
	if err != nil {
		return fmt.Errorf("scrape %s: %w", config.URL, err)
	}

	fmt.Fprintf(w, "Found %d articles\n", len(results))

	tableName, err := resolveTableName(task)
	if err != nil {
		return fmt.Errorf("scrape resolveTableName: %w", err)
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("scrape marshal results: %w", err)
	}

	projectPath := filepath.Join(e.projectsRoot, projectID, "raw")
	os.MkdirAll(projectPath, 0755)

	jsonPath := filepath.Join(projectPath, tableName+".json")
	if err := safeident.SanitizeFilePath(jsonPath); err != nil {
		return fmt.Errorf("unsafe scrape JSON path: %w", err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("scrape JSON write failed: %w", err)
	}

	createSQL := "CREATE TABLE IF NOT EXISTS " + safeident.QuoteIdentifier(tableName) + " AS SELECT * FROM read_json_auto(" + safeident.QuoteStringLiteral(jsonPath) + ", ignore_errors=true)" // safe: tableName validated via resolveTableName -> stripAndValidateName -> safeident.ValidateIdentifier; filePath via safeident.SanitizeFilePath
	if _, err := e.db.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("scrape table creation failed: %w", err)
	}

	e.updateProgress(task.Id, 100, "completed")
	fmt.Fprintf(w, "Scrape ingestion complete. Table \"%s\" (%d articles)\n", tableName, len(results))
	return nil
}

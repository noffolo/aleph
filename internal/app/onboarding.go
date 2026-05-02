package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
)

// setupDemoData creates a demo project with sample data and a starter agent
// when no projects exist. This runs once on first system startup.
//
// The function is designed to be idempotent and safe:
// - It only runs when the metaRepo reports zero projects
// - It gracefully degrades if DuckDB schema creation fails
// - It logs every step clearly so operators see what happened
// - It creates the agent only after the project exists
func (a *AlephApp) setupDemoData(projectsRoot string) {
	if a.metaRepo == nil {
		a.logger.Info("first-run setup skipped — metadata repository not available")
		return
	}

	count, err := a.metaRepo.CountProjects()
	if err != nil {
		a.logger.Warn("first-run check failed — cannot count projects", "error", err)
		return
	}
	if count > 0 {
		a.logger.Info("projects found, skipping first-run setup", "count", count)
		return
	}

	log.Println("[Onboarding] First run detected — setting up demo project with sample data")

	// ── 1. Demo Project ──────────────────────────────────────────────────
	projectID := "demo"
	projectName := "Demo Project"
	projectPath := filepath.Join(projectsRoot, projectID)

	dirs := []string{
		filepath.Join(projectPath, "raw"),
		filepath.Join(projectPath, "ontologies"),
		filepath.Join(projectPath, "agents"),
		filepath.Join(projectPath, "skills"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			a.logger.Warn("first-run: failed to create directory, aborting", "dir", d, "error", err)
			return
		}
	}

	// DuckDB schema
	si, err := storage.NewSchemaIdentity(projectID)
	if err != nil {
		a.logger.Warn("first-run: invalid projectID for schema — continuing without schema", "error", err)
	} else if err := storage.EnsureProjectSchema(a.db, si); err != nil {
		a.logger.Warn("first-run: failed to create DuckDB schema — continuing without schema", "error", err)
	}

	// Postgres record
	if err := a.metaRepo.CreateProjectRecord(projectID, projectName); err != nil {
		// Check if it's a duplicate (ON CONFLICT DO NOTHING)
		a.logger.Warn("first-run: project record may already exist", "error", err)
	}

	// ── 2. Sample CSV ─────────────────────────────────────────────────────
	// Copy the embedded sample CSV to the project raw directory
	sampleCSV := `date,product,category,revenue,units_sold,cost,region
2025-01-06,Widget Alpha,Hardware,12500.00,45,8750.00,North
2025-01-06,Gadget Beta,Electronics,23400.00,78,15210.00,North
2025-01-06,Widget Alpha,Hardware,9800.00,32,6860.00,South
2025-01-13,Gadget Beta,Electronics,31200.00,104,20280.00,East
2025-01-13,Component Gamma,Software,8500.00,170,2550.00,West
2025-01-13,Widget Alpha,Hardware,14100.00,51,9870.00,East
2025-01-20,Gadget Beta,Electronics,28900.00,96,18785.00,South
2025-01-20,Component Gamma,Software,10200.00,204,3060.00,North
2025-01-20,Widget Alpha,Hardware,11000.00,40,7700.00,West
2025-01-27,Gadget Beta,Electronics,35600.00,119,23140.00,West
2025-01-27,Component Gamma,Software,7600.00,152,2280.00,East
2025-02-03,Widget Alpha,Hardware,16800.00,61,11760.00,North
2025-02-03,Gadget Beta,Electronics,19800.00,66,12870.00,North
2025-02-03,Component Gamma,Software,13000.00,260,3900.00,South
2025-02-10,Widget Alpha,Hardware,9200.00,33,6440.00,East
2025-02-10,Gadget Beta,Electronics,27400.00,91,17810.00,South
2025-02-10,Component Gamma,Software,15500.00,310,4650.00,West
2025-02-17,Widget Alpha,Hardware,20500.00,74,14350.00,West
2025-02-17,Gadget Beta,Electronics,22100.00,74,14365.00,East
2025-02-24,Component Gamma,Software,18900.00,378,5670.00,North
2025-03-03,Widget Alpha,Hardware,14700.00,53,10290.00,South
2025-03-03,Gadget Beta,Electronics,33500.00,112,21775.00,West
2025-03-03,Component Gamma,Software,9800.00,196,2940.00,East
2025-03-10,Widget Alpha,Hardware,17800.00,65,12460.00,North
2025-03-10,Gadget Beta,Electronics,26200.00,87,17030.00,South
2025-03-17,Tool Delta,Services,4500.00,15,3150.00,West
2025-03-17,Widget Alpha,Hardware,13200.00,48,9240.00,East
2025-03-24,Component Gamma,Software,22100.00,442,6630.00,South
2025-03-24,Gadget Beta,Electronics,41000.00,137,26650.00,North
2025-03-31,Tool Delta,Services,6700.00,22,4690.00,East
`
	csvPath := filepath.Join(projectPath, "raw", "sample.csv")
	if err := os.WriteFile(csvPath, []byte(sampleCSV), 0644); err != nil {
		a.logger.Warn("first-run: failed to write sample CSV", "error", err)
	}

	// Create a sample ontology file
	ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
	ontContent := fmt.Sprintf(`// Aleph Ontology — Demo Project
// Auto-generated on first run

object Sales {
    dimension date: datetime
    dimension product: text
    dimension category: text
    dimension region: text
    measure revenue: number
    measure units_sold: number
    measure cost: number
}
`)
	if err := os.WriteFile(ontPath, []byte(ontContent), 0644); err != nil {
		a.logger.Warn("first-run: failed to write ontology", "error", err)
	}

	// ── 3. Demo Agent ─────────────────────────────────────────────────────
	agentID := fmt.Sprintf("agent-demo-%d", time.Now().UnixMilli())
	agentRec := &repository.AgentRecord{
		ID:        agentID,
		ProjectID: projectID,
		Name:      "Analista Demo",
		Provider:  "ollama",
		Model:     "llama3",
		BaseURL:   a.cfg.OllamaBaseURL,
		SystemPrompt: strings.TrimSpace(`
Sei un analista esperto. Il tuo compito è aiutare l'utente a esplorare i dati del progetto Demo.

Hai a disposizione i seguenti dati:
- sample.csv: vendite settimanali per prodotto, categoria, regione con revenue, unità vendute e costi

Puoi analizzare trend, confrontare performance tra prodotti e regioni, calcolare metriche.
Rispondi sempre in modo chiaro, strutturato e utile per prendere decisioni.
`),
	}
	if err := a.metaRepo.CreateAgent(agentRec); err != nil {
		a.logger.Warn("first-run: failed to create demo agent", "error", err)
		return
	}

	log.Printf("[Onboarding] Demo project created: id=%s agent=%s csv=30 rows", projectID, agentID)
	a.logger.Info("first-run completed successfully",
		"project", projectID,
		"project_name", projectName,
		"agent", agentID,
		"csv_rows", 30,
	)
}

package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Act struct {
	ID                  string `json:"id"`
	Legislature         int    `json:"legislature"`
	ActType             string `json:"act_type"`
	Title               string `json:"title"`
	PresentationDate    string `json:"presentation_date"`
	Status              string `json:"status"`
	StatusDate          string `json:"status_date"`
	FirstSigner         string `json:"first_signer"`
	PartyAtPresentation string `json:"party_at_presentation"`
	Chamber             string `json:"chamber"`
}

type Attendance struct {
	ParliamentarianName string  `json:"parliamentarian_name"`
	Legislature         int     `json:"legislature"`
	Year                int     `json:"year"`
	TotalSessions       int     `json:"total_sessions"`
	Attended            int     `json:"attended"`
	Absences            int     `json:"absences"`
	AttendancePct       float64 `json:"attendance_pct"`
	MissionAbsences     int     `json:"mission_absences"`
	GroupAtTime         string  `json:"group_at_time"`
	Chamber             string  `json:"chamber"`
}

type actSparqlBinding struct {
	Value string `json:"value"`
}

type actSparqlResult struct {
	Atto              actSparqlBinding `json:"atto"`
	Tipo              actSparqlBinding `json:"tipo"`
	Titolo            actSparqlBinding `json:"titolo"`
	DataPresentazione actSparqlBinding `json:"dataPresentazione"`
	PrimoFirmatario   actSparqlBinding `json:"primoFirmatario"`
	Stato             actSparqlBinding `json:"stato"`
	DataStato         actSparqlBinding `json:"dataStato"`
}

type actSparqlResults struct {
	Bindings []actSparqlResult `json:"bindings"`
}

type actSparqlResponse struct {
	Head    interface{}      `json:"head"`
	Results actSparqlResults `json:"results"`
}

type attSparqlBinding struct {
	Value string `json:"value"`
}

type attSparqlResult struct {
	Parlamentare attSparqlBinding `json:"parlamentare"`
	Presenze     attSparqlBinding `json:"presenze"`
	Assenze      attSparqlBinding `json:"assenze"`
	Missioni     attSparqlBinding `json:"missioni"`
	TotaleSedute attSparqlBinding `json:"totaleSedute"`
	Percentuale  attSparqlBinding `json:"percentuale"`
	Gruppo       attSparqlBinding `json:"gruppo"`
}

type attSparqlResults struct {
	Bindings []attSparqlResult `json:"bindings"`
}

type attSparqlResponse struct {
	Head    interface{}      `json:"head"`
	Results attSparqlResults `json:"results"`
}

func ensureParliamentActsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS parliamentary_acts (
		id VARCHAR PRIMARY KEY,
		legislature INTEGER NOT NULL,
		act_type VARCHAR NOT NULL,
		title TEXT,
		presentation_date VARCHAR,
		status VARCHAR,
		status_date VARCHAR,
		first_signer VARCHAR,
		party_at_presentation VARCHAR,
		chamber VARCHAR,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func ensureParliamentAttendanceTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS parliamentary_attendance (
		parliamentarian_name VARCHAR NOT NULL,
		legislature INTEGER NOT NULL,
		year INTEGER NOT NULL,
		total_sessions INTEGER,
		attended INTEGER,
		absences INTEGER,
		attendance_pct DOUBLE,
		mission_absences INTEGER,
		group_at_time VARCHAR,
		chamber VARCHAR,
		ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (parliamentarian_name, legislature, year)
	)`)
	return err
}

func BuildActsSPARQLQuery(legislatura int) string {
	return fmt.Sprintf(`PREFIX ocd: <http://dati.camera.it/ocd/>
PREFIX dc: <http://purl.org/dc/elements/1.1/>

SELECT ?atto ?tipo ?titolo ?dataPresentazione ?primoFirmatario ?stato ?dataStato
WHERE {
  ?atto a ocd:atto .
  ?atto dc:title ?titolo .
  ?atto ocd:rif_leg ?legislatura .
  ?atto dc:date ?dataPresentazione .
  OPTIONAL { ?atto dc:type ?tipo }
  OPTIONAL { ?atto ocd:primoFirmatario ?primoFirmatario }
  OPTIONAL { ?atto ocd:stato ?stato }
  OPTIONAL { ?atto ocd:dataStato ?dataStato }
  FILTER(?legislatura = <http://dati.camera.it/ocd/legislatura.rdf/repubblica_%d>)
} LIMIT 1000`, legislatura)
}

func BuildAttendanceSPARQLQuery(legislatura int, year int) string {
	return fmt.Sprintf(`PREFIX ocd: <http://dati.camera.it/ocd/>

SELECT ?parlamentare ?presenze ?assenze ?missioni ?totaleSedute ?percentuale ?gruppo
WHERE {
  ?presenza a ocd:presenza .
  ?presenza ocd:rif_leg ?legislatura .
  ?presenza ocd:anno "%d" .
  ?presenza ocd:parlamentare ?parlamentare .
  ?presenza ocd:presenze ?presenze .
  ?presenza ocd:assenze ?assenze .
  OPTIONAL { ?presenza ocd:missioni ?missioni }
  OPTIONAL { ?presenza ocd:totaleSedute ?totaleSedute }
  OPTIONAL { ?presenza ocd:percentuale ?percentuale }
  OPTIONAL { ?presenza ocd:gruppo ?gruppo }
  FILTER(?legislatura = <http://dati.camera.it/ocd/legislatura.rdf/repubblica_%d>)
} LIMIT 2000`, year, legislatura)
}

func executeSPARQL(ctx context.Context, endpoint string, query string) ([]byte, error) {
	formData := url.Values{}
	formData.Set("query", query)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create SPARQL request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SPARQL http post: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read SPARQL body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("SPARQL endpoint returned %d: %s", resp.StatusCode, string(body[:min(len(body), 500)]))
	}
	return body, nil
}

func saveParliamentRawJSON(rawDir string, subdir string, filename string, data []byte) error {
	rawPath := filepath.Join(rawDir, subdir)
	if err := os.MkdirAll(rawPath, 0755); err != nil {
		return err
	}
	rawFile := filepath.Join(rawPath, filename)
	return os.WriteFile(rawFile, data, 0644)
}

func ParseSPARQLActs(data []byte, legislature int, chamber string) ([]Act, error) {
	var resp actSparqlResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal acts SPARQL: %w", err)
	}
	acts := make([]Act, 0, len(resp.Results.Bindings))
	for _, b := range resp.Results.Bindings {
		acts = append(acts, Act{
			ID:                 extractActID(b.Atto.Value),
			Legislature:        legislature,
			ActType:            b.Tipo.Value,
			Title:              b.Titolo.Value,
			PresentationDate:   b.DataPresentazione.Value,
			Status:             b.Stato.Value,
			StatusDate:         b.DataStato.Value,
			FirstSigner:        b.PrimoFirmatario.Value,
			PartyAtPresentation: "",
			Chamber:            chamber,
		})
	}
	return acts, nil
}

func ParseSPARQLAttendance(data []byte, legislature int, year int, chamber string) ([]Attendance, error) {
	var resp attSparqlResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal attendance SPARQL: %w", err)
	}
	results := make([]Attendance, 0, len(resp.Results.Bindings))
	for _, b := range resp.Results.Bindings {
		results = append(results, Attendance{
			ParliamentarianName: b.Parlamentare.Value,
			Legislature:         legislature,
			Year:                year,
			TotalSessions:       parseIntSafe(b.TotaleSedute.Value),
			Attended:            parseIntSafe(b.Presenze.Value),
			Absences:            parseIntSafe(b.Assenze.Value),
			AttendancePct:       parseFloatSafe(b.Percentuale.Value),
			MissionAbsences:     parseIntSafe(b.Missioni.Value),
			GroupAtTime:         b.Gruppo.Value,
			Chamber:             chamber,
		})
	}
	return results, nil
}

func extractActID(uri string) string {
	if uri == "" {
		return ""
	}
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return uri
}

func parseIntSafe(s string) int {
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return 0
	}
	return v
}

func parseFloatSafe(s string) float64 {
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
		return 0.0
	}
	return v
}

func RunParliamentActs(ctx context.Context, db *sql.DB, sparqlEndpoint string, legislature int, chamber string) error {
	slog.Info("starting parliamentary acts ingestion", "endpoint", sparqlEndpoint, "legislature", legislature, "chamber", chamber)

	query := BuildActsSPARQLQuery(legislature)

	if err := ensureParliamentActsTable(db); err != nil {
		return fmt.Errorf("create acts table: %w", err)
	}

	body, err := executeSPARQL(ctx, sparqlEndpoint, query)
	if err != nil {
		return fmt.Errorf("SPARQL acts fetch: %w", err)
	}

	acts, err := ParseSPARQLActs(body, legislature, chamber)
	if err != nil {
		return fmt.Errorf("parse acts: %w", err)
	}

	slog.Info("parsed parliamentary acts", "count", len(acts), "legislature", legislature, "chamber", chamber)

	if len(acts) > 0 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		stmt, err := tx.PrepareContext(ctx,
			`INSERT OR REPLACE INTO parliamentary_acts
			 (id, legislature, act_type, title, presentation_date, status, status_date, first_signer, party_at_presentation, chamber)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("prepare acts stmt: %w", err)
		}

		for _, a := range acts {
			if _, err := stmt.ExecContext(ctx, a.ID, a.Legislature, a.ActType, a.Title,
				a.PresentationDate, a.Status, a.StatusDate, a.FirstSigner, a.PartyAtPresentation, a.Chamber); err != nil {
				stmt.Close()
				tx.Rollback()
				return fmt.Errorf("insert act: %w", err)
			}
		}
		stmt.Close()

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit acts: %w", err)
		}
	}

	return nil
}

func RunParliamentAttendance(ctx context.Context, db *sql.DB, sparqlEndpoint string, legislature int, year int, chamber string) error {
	slog.Info("starting parliamentary attendance ingestion", "endpoint", sparqlEndpoint, "legislature", legislature, "year", year, "chamber", chamber)

	query := BuildAttendanceSPARQLQuery(legislature, year)

	if err := ensureParliamentAttendanceTable(db); err != nil {
		return fmt.Errorf("create attendance table: %w", err)
	}

	body, err := executeSPARQL(ctx, sparqlEndpoint, query)
	if err != nil {
		return fmt.Errorf("SPARQL attendance fetch: %w", err)
	}

	records, err := ParseSPARQLAttendance(body, legislature, year, chamber)
	if err != nil {
		return fmt.Errorf("parse attendance: %w", err)
	}

	slog.Info("parsed parliament attendance", "count", len(records), "legislature", legislature, "year", year)

	if len(records) > 0 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		stmt, err := tx.PrepareContext(ctx,
			`INSERT OR REPLACE INTO parliamentary_attendance
			 (parliamentarian_name, legislature, year, total_sessions, attended, absences, attendance_pct, mission_absences, group_at_time, chamber)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("prepare attendance stmt: %w", err)
		}

		for _, r := range records {
			if _, err := stmt.ExecContext(ctx, r.ParliamentarianName, r.Legislature, r.Year,
				r.TotalSessions, r.Attended, r.Absences, r.AttendancePct, r.MissionAbsences, r.GroupAtTime, r.Chamber); err != nil {
				stmt.Close()
				tx.Rollback()
				return fmt.Errorf("insert attendance: %w", err)
			}
		}
		stmt.Close()

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit attendance: %w", err)
		}
	}

	return nil
}

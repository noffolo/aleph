package political

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// EntityMention represents a detected party mention in text.
type EntityMention struct {
	Party string
}

// EntityExtractor extracts party mentions from text using keyword matching.
type EntityExtractor struct {
	keywords map[string][]string
}

// NewEntityExtractor creates a new extractor with the given party keyword map.
func NewEntityExtractor(keywords map[string][]string) *EntityExtractor {
	return &EntityExtractor{keywords: keywords}
}

// SentimentResult holds a sentiment score for one party in one article.
type SentimentResult struct {
	ArticleID string
	Title     string
	Date      string
	Party     string
	Score     float64
	Coalition string
	Category  string
	Source    string
}

var positiveItalian = []string{
	"positivo", "buono", "crescita", "successo", "migliora", "vittoria",
	"forte", "stabile", "promettente", "innovazione", "sostegno", "riforma",
	"progresso", "sviluppo", "opportunità", "ottimo", "eccellente",
	"brillante", "efficace", "solido", "favorevole", "ottimismo",
	"fiducia", "importante",
}

var negativeItalian = []string{
	"negativo", "crisi", "crollo", "peggiora", "sconfitta", "debole",
	"instabile", "rischioso", "emergenza", "fallimento", "corruzione",
	"scandalo", "dimissioni", "calo", "perdita", "danno", "preoccupante",
	"allarmante", "pessimo", "grave", "critico", "vergogna",
	"insostenibile", "bloccato",
}

// DefaultPartyKeywords is the full party-to-keyword mapping.
var DefaultPartyKeywords = map[string][]string{
	"Fratelli d'Italia":       {"fratelli d'italia", "fdi", "meloni", "giorgia meloni"},
	"Partito Democratico":     {"partito democratico", "pd", "schlein", "elly schlein", "democratico"},
	"Movimento 5 Stelle":      {"movimento 5 stelle", "m5s", "cinque stelle", "grillo", "conte"},
	"Lega":                    {"lega", "salvini", "lega nord", "matteo salvini"},
	"Forza Italia":            {"forza italia", "berlusconi", "tajani", "antonio tajani"},
	"Azione":                  {"azione", "calenda", "carlo calenda"},
	"Italia Viva":             {"italia viva", "renzi", "matteo renzi"},
	"Alleanza Verdi Sinistra": {"verdi e sinistra", "avs", "fratoianni", "bonelli"},
	"Sinistra Italiana":       {"sinistra italiana", "fratoianni"},
	"Europa Verde":            {"europa verde", "bonelli"},
	"+Europa":                 {"piu europa", "+europa", "maggiori"},
	"Noi Moderati":            {"noi moderati", "lupi", "maurizio lupi"},
	"Unione di Centro":        {"unione di centro", "udc"},
	"Sud chiama Nord":         {"sud chiama nord", "cateno de luca"},
	"Italia al Centro":        {"italia al centro", "tot"},
}

// DefaultCoalition maps parties to their coalition.
var DefaultCoalition = map[string]string{
	"Partito Democratico":     "CSX",
	"Movimento 5 Stelle":      "CSX",
	"Alleanza Verdi Sinistra": "CSX",
	"Sinistra Italiana":       "CSX",
	"Europa Verde":            "CSX",
	"+Europa":                 "CSX",
	"Azione":                  "CSX",
	"Italia Viva":             "CSX",
	"Unione Popolare":         "CSX",
	"Fratelli d'Italia":       "CDX",
	"Lega":                    "CDX",
	"Forza Italia":            "CDX",
	"Noi Moderati":            "CDX",
	"Sud chiama Nord":         "CDX",
	"Italia al Centro":        "CDX",
}

// CoalitionFor returns the coalition (CSX/CDX/"") for a party.
func CoalitionFor(party string) string {
	if c, ok := DefaultCoalition[party]; ok {
		return c
	}
	return ""
}

// ExtractEntities finds party mentions in the given title and description.
func (e *EntityExtractor) ExtractEntities(title, description string) []EntityMention {
	text := strings.ToLower(title + " " + description)
	seen := make(map[string]bool)
	var mentions []EntityMention

	for party, patterns := range e.keywords {
		for _, pattern := range patterns {
			if strings.Contains(text, pattern) {
				if !seen[party] {
					seen[party] = true
					mentions = append(mentions, EntityMention{Party: party})
				}
				break
			}
		}
	}
	return mentions
}

// ScoreSentiment returns a heuristic sentiment score from -1.0 to +1.0.
func ScoreSentiment(text string) float64 {
	lower := strings.ToLower(text)
	posCount := 0
	negCount := 0

	for _, w := range positiveItalian {
		if strings.Contains(lower, w) {
			posCount++
		}
	}
	for _, w := range negativeItalian {
		if strings.Contains(lower, w) {
			negCount++
		}
	}

	total := posCount + negCount
	if total == 0 {
		return 0.0
	}
	return float64(posCount-negCount) / float64(total)
}

// StoreSentiment creates the political_sentiment table and inserts results.
func StoreSentiment(db *sql.DB, results []SentimentResult) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS political_sentiment (
		article_date DATE,
		party TEXT,
		coalition TEXT,
		score REAL,
		category TEXT,
		source TEXT,
		PRIMARY KEY (article_date, party)
	)`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	for _, r := range results {
		dateStr := r.Date
		if t, err := time.Parse("2006-01-02", r.Date); err == nil {
			dateStr = t.Format("2006-01-02")
		}
		_, err := db.Exec(
			`INSERT INTO political_sentiment
			 (article_date, party, coalition, score, category, source)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (article_date, party) DO UPDATE SET
			 coalition = excluded.coalition,
			 score = excluded.score,
			 category = excluded.category,
			 source = excluded.source`,
			dateStr, r.Party, r.Coalition, r.Score, r.Category, r.Source,
		)
		if err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	}
	return nil
}

// ExportWeeklySentiment exports weekly aggregated sentiment JSON files.
func ExportWeeklySentiment(db *sql.DB, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Weekly sentiment per party
	partyRows, err := db.Query(`SELECT
		DATE_TRUNC('week', article_date) AS week,
		party,
		AVG(score) AS avg_score,
		COUNT(*) AS article_count
	FROM political_sentiment
	GROUP BY week, party
	ORDER BY week, party`)
	if err != nil {
		return fmt.Errorf("query weekly: %w", err)
	}
	defer partyRows.Close()

	type WeeklyRow struct {
		Week         string  `json:"week"`
		Party        string  `json:"party"`
		AvgScore     float64 `json:"avg_score"`
		ArticleCount int     `json:"article_count"`
	}

	var weekly []WeeklyRow
	for partyRows.Next() {
		var weekRaw interface{}
		var party string
		var avgScore float64
		var count int
		if err := partyRows.Scan(&weekRaw, &party, &avgScore, &count); err != nil {
			return fmt.Errorf("scan weekly: %w", err)
		}
		weekly = append(weekly, WeeklyRow{
			Week:         fmt.Sprintf("%v", weekRaw),
			Party:        party,
			AvgScore:     avgScore,
			ArticleCount: count,
		})
	}
	if err := partyRows.Err(); err != nil {
		return err
	}

	weeklyJSON, err := json.MarshalIndent(weekly, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outDir+"/weekly_sentiment.json", weeklyJSON, 0644); err != nil {
		return err
	}

	// Coalition weekly sentiment
	coalRows, err := db.Query(`SELECT
		DATE_TRUNC('week', article_date) AS week,
		coalition,
		AVG(score) AS avg_score,
		COUNT(*) AS article_count
	FROM political_sentiment
	WHERE coalition != ''
	GROUP BY week, coalition
	ORDER BY week, coalition`)
	if err != nil {
		return fmt.Errorf("query coalition: %w", err)
	}
	defer coalRows.Close()

	type CoalitionRow struct {
		Week         string  `json:"week"`
		Coalition    string  `json:"coalition"`
		AvgScore     float64 `json:"avg_score"`
		ArticleCount int     `json:"article_count"`
	}

	var coalition []CoalitionRow
	for coalRows.Next() {
		var weekRaw interface{}
		var coal string
		var avgScore float64
		var count int
		if err := coalRows.Scan(&weekRaw, &coal, &avgScore, &count); err != nil {
			return fmt.Errorf("scan coalition: %w", err)
		}
		coalition = append(coalition, CoalitionRow{
			Week:         fmt.Sprintf("%v", weekRaw),
			Coalition:    coal,
			AvgScore:     avgScore,
			ArticleCount: count,
		})
	}
	if err := coalRows.Err(); err != nil {
		return err
	}

	coalJSON, err := json.MarshalIndent(coalition, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outDir+"/coalition_weekly.json", coalJSON, 0644)
}

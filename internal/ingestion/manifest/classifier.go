package manifest

import (
	"regexp"
	"strings"
)

var (
	datePattern   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	coordNames    = map[string]bool{"lat": true, "lon": true, "latitude": true, "longitude": true}
	numericTypes  = map[string]bool{"INTEGER": true, "BIGINT": true, "FLOAT": true, "DOUBLE": true, "DECIMAL": true, "NUMERIC": true, "REAL": true}
	temporalTypes = map[string]bool{"DATE": true, "TIMESTAMP": true, "DATETIME": true}
	ignoredTypes  = map[string]bool{"BLOB": true, "BINARY": true}
	varcharTypes  = map[string]bool{"VARCHAR": true, "TEXT": true, "CHAR": true, "STRING": true}
)

type Classifier struct {
	cfg DomainConfig
}

func NewClassifier(cfg DomainConfig) *Classifier {
	return &Classifier{cfg: cfg}
}

func (c *Classifier) Classify(tables []TableSchema) ([]TableSchema, error) {
	tablePK := buildTablePKLookup(tables)

	for ti := range tables {
		for ci := range tables[ti].Columns {
			col := &tables[ti].Columns[ci]
			c.classifyOne(col, tablePK)
		}
	}
	return tables, nil
}

func buildTablePKLookup(tables []TableSchema) map[string]string {
	m := make(map[string]string, len(tables))
	for _, table := range tables {
		for _, col := range table.Columns {
			if col.IsPK || col.Name == "id" {
				m[table.Name] = col.Name
				break
			}
		}
	}
	return m
}

// classifyOne applies classification rules in strict priority order matching Spec 3.
func (c *Classifier) classifyOne(col *ColumnSchema, tablePK map[string]string) {
	t := strings.ToUpper(col.Type)

	// ── a. PRIMARY KEY
	if col.IsPK || col.Name == "id" {
		col.Class = PrimaryKey
		return
	}

	// ── a/b. _id suffix: FK if matches another table's PK, else PK
	if strings.HasSuffix(col.Name, "_id") {
		base := strings.TrimSuffix(col.Name, "_id")
		if _, ok := tablePK[base]; ok {
			col.Class = ForeignKey
			col.FKTarget = base
			return
		}
		col.Class = PrimaryKey
		return
	}

	// ── b. FOREIGN KEY (explicit constraint)
	if col.IsFK {
		col.Class = ForeignKey
		return
	}

	// ── c. TEMPORAL
	if temporalTypes[t] {
		col.Class = Temporal
		return
	}
	// VARCHAR dates: YYYY-MM-DD
	if varcharTypes[t] && hasDatePattern(col.SampleValues) {
		col.Class = Temporal
		return
	}

	// ── d. BOOLEAN
	if t == "BOOLEAN" || (t == "INTEGER" && col.DistinctCount == 2) {
		col.Class = Boolean
		return
	}

	// ── e. COORDINATE
	if coordNames[strings.ToLower(col.Name)] {
		col.Class = Coordinate
		return
	}

	// ── f. IGNORED (BLOB/BINARY)
	if ignoredTypes[t] {
		col.Class = Ignored
		return
	}

	// ── g. MEASURE
	if numericTypes[t] {
		if isYearColumn(col) {
			col.Class = Ignored
			return
		}
		col.Class = Measure
		return
	}

	// ── h. CATEGORY
	if (varcharTypes[t] || t == "INTEGER") && col.DistinctCount > 0 &&
		col.DistinctCount <= int64(c.cfg.MaxCategoryDistinct) {
		col.Class = Category
		return
	}

	// ── i. LABEL
	if varcharTypes[t] && col.RowCount > 0 {
		ratio := float64(col.DistinctCount) / float64(col.RowCount)
		if ratio > c.cfg.MinDistinctRatio && avgSampleLen(col.SampleValues) < 100 {
			col.Class = Label
			return
		}
	}

	// ── j. DEFAULT
	col.Class = Ignored
}

// ── Helpers ────────────────────────────────────────────────────────────────

func hasDatePattern(samples []any) bool {
	for _, s := range samples {
		if s == nil {
			continue
		}
		str, ok := s.(string)
		if !ok {
			continue
		}
		if datePattern.MatchString(str) {
			return true
		}
	}
	return false
}

func isYearColumn(col *ColumnSchema) bool {
	if col.RowCount == 0 || col.DistinctCount == 0 {
		return false
	}
	if float64(col.DistinctCount)/float64(col.RowCount) <= 0.5 {
		return false
	}
	yearHits, total := 0, 0
	for _, s := range col.SampleValues {
		if s == nil {
			continue
		}
		total++
		var val float64
		switch v := s.(type) {
		case float64:
			val = v
		case int64:
			val = float64(v)
		case int:
			val = float64(v)
		case float32:
			val = float64(v)
		default:
			continue
		}
		if val >= 1900 && val <= 2100 {
			yearHits++
		}
	}
	if total == 0 {
		return false
	}
	return float64(yearHits)/float64(total) > 0.5
}

func avgSampleLen(samples []any) float64 {
	total, count := 0, 0
	for _, s := range samples {
		if s == nil {
			continue
		}
		str, ok := s.(string)
		if !ok {
			continue
		}
		total += len(str)
		count++
	}
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

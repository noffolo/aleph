package ingestion

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// RegisterCrossReferenceViews creates DuckDB views that join political data across
// source types for unified querying. Idempotent via CREATE OR REPLACE VIEW.
func RegisterCrossReferenceViews(db *sql.DB) error {
	views := []string{
		`CREATE OR REPLACE VIEW v_politician_full_profile AS
		 SELECT p.id, p.name, p.party, o.role, o.org_id, v.gruppo, v.esito
		 FROM pep_entities p
		 LEFT JOIN opdm_memberships o ON CAST(p.id AS VARCHAR) = CAST(o.person_id AS VARCHAR)
		 LEFT JOIN parliament_votes v ON p.name = v.deputato`,

		`CREATE OR REPLACE VIEW v_contract_party_link AS
		 SELECT c.cig, c.aggiudicatario, c.importo, p.name as pep_name, p.position
		 FROM public_contracts c
		 LEFT JOIN pep_entities p ON c.aggiudicatario LIKE '%' || p.name || '%'`,

		`CREATE OR REPLACE VIEW v_funding_timeline AS
		 SELECT recipient_party, donation_year, SUM(CAST(donation_amount AS DOUBLE)) as total_amount, COUNT(*) as donation_count
		 FROM party_funding
		 GROUP BY recipient_party, donation_year
		 ORDER BY donation_year DESC, total_amount DESC`,
	}
	for _, v := range views {
		if _, err := db.Exec(v); err != nil {
			return fmt.Errorf("create view: %w", err)
		}
	}
	slog.Info("cross-reference views registered")
	return nil
}

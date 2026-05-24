package ingestion

import (
	"database/sql"
	"fmt"
)

func RegisterPollViews(db *sql.DB) error {
	views := []string{
		`CREATE OR REPLACE VIEW v_poll_trends AS
		 SELECT
		   party_canonical,
		   DATE_TRUNC('month', date) AS month,
		   AVG(percentage) AS avg_percentage,
		   COUNT(*) AS poll_count,
		   STDDEV(percentage) AS stddev
		 FROM polls
		 GROUP BY party_canonical, DATE_TRUNC('month', date)`,

		`CREATE OR REPLACE VIEW v_poll_accuracy AS
		 SELECT
		   p.party_canonical,
		   p.month,
		   p.avg_percentage AS poll_avg,
		   e.percentuale AS election_result,
		   ABS(p.avg_percentage - e.percentuale) AS absolute_error,
		   p.poll_count
		 FROM v_poll_trends p
		 LEFT JOIN election_results e
		   ON p.party_canonical = e.party_canonical
		   AND EXTRACT(YEAR FROM p.month) = e.year`,
	}
	for _, v := range views {
		if _, err := db.Exec(v); err != nil {
			return fmt.Errorf("create view: %w", err)
		}
	}
	return nil
}

package ingestion

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func RegisterParliamentViews(db *sql.DB) error {
	views := []string{
		`CREATE OR REPLACE VIEW v_parliamentary_activity_profile AS
		 SELECT
		     COALESCE(a.first_signer, att.parliamentarian_name) as nome,
		     att.parliamentarian_name,
		     att.legislature,
		     att.attendance_pct,
		     att.total_sessions,
		     COUNT(DISTINCT a.id) as atti_presentati,
		     a.party_at_presentation as partito,
		     list(DISTINCT a.act_type) as tipi_atti
		 FROM parliamentary_attendance att
		 LEFT JOIN parliamentary_acts a
		     ON att.parliamentarian_name = a.first_signer
		     AND att.legislature = a.legislature
		 GROUP BY COALESCE(a.first_signer, att.parliamentarian_name),
		          att.parliamentarian_name, att.legislature,
		          att.attendance_pct, att.total_sessions, a.party_at_presentation`,

		`CREATE OR REPLACE VIEW v_group_discipline AS
		 SELECT
		     v.gruppo,
		     COUNT(DISTINCT v.deputato) as n_membri,
		     AVG(CASE WHEN v.esito IN ('APPROVATA', 'Favorevole') THEN 1.0 ELSE 0.0 END) as coesione_media,
		     COUNT(DISTINCT v.id) as votazioni_totali
		 FROM parliament_votes v
		 GROUP BY v.gruppo
		 ORDER BY coesione_media DESC`,
	}
	for _, v := range views {
		if _, err := db.Exec(v); err != nil {
			return fmt.Errorf("create parliament view: %w", err)
		}
	}
	slog.Info("parliament views registered")
	return nil
}

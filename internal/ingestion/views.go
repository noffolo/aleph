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

// RegisterISTATViews creates DuckDB views that join election results with ISTAT
// demographic and economic data (population, income, employment).
func RegisterISTATViews(db *sql.DB) error {
	views := []string{
		`CREATE OR REPLACE VIEW v_comune_electoral_demographics AS
		 SELECT e.election_type, e.year, e.comune_istat, e.comune,
		        e.party_canonical, SUM(e.voti) as voti_totali,
		        SUM(e.elettori) as elettori_totali, SUM(e.votanti) as votanti_totali,
		        p.popolazione_residente, p.eta_media, p.indice_vecchiaia,
		        i.reddito_medio, i.importo_totale,
		        em.tasso_occupazione, em.tasso_disoccupazione
		 FROM election_results e
		 LEFT JOIN istat_population p ON e.comune_istat = p.comune_istat AND e.year = p.year
		 LEFT JOIN istat_income i ON e.comune_istat = i.comune_istat AND e.year = i.year
		 LEFT JOIN istat_employment em ON e.comune_istat = em.comune_istat AND e.year = em.year
		 GROUP BY e.election_type, e.year, e.comune_istat, e.comune, e.party_canonical,
		          p.popolazione_residente, p.eta_media, p.indice_vecchiaia,
		          i.reddito_medio, i.importo_totale,
		          em.tasso_occupazione, em.tasso_disoccupazione`,

		`CREATE OR REPLACE VIEW v_party_demographic_profile AS
		 SELECT e.party_canonical, e.year, e.election_type,
		        AVG(p.eta_media) as eta_media_elettorato,
		        AVG(p.indice_vecchiaia) as indice_vecchiaia_medio,
		        AVG(i.reddito_medio) as reddito_medio_elettorato,
		        AVG(em.tasso_occupazione) as occupazione_media,
		        SUM(e.voti) as voti_totali,
		        COUNT(DISTINCT e.comune_istat) as comuni_con_dati
		 FROM election_results e
		 LEFT JOIN istat_population p ON e.comune_istat = p.comune_istat AND e.year = p.year
		 LEFT JOIN istat_income i ON e.comune_istat = i.comune_istat AND e.year = i.year
		 LEFT JOIN istat_employment em ON e.comune_istat = em.comune_istat AND e.year = em.year
		 GROUP BY e.party_canonical, e.year, e.election_type
		 ORDER BY e.year DESC, voti_totali DESC`,
	}
	for _, v := range views {
		if _, err := db.Exec(v); err != nil {
			return fmt.Errorf("create ISTAT view: %w", err)
		}
	}
	slog.Info("ISTAT cross-reference views registered")
	return nil
}

package ingestion

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func RegisterMicrotargetingViews(db *sql.DB) error {
	views := []string{
		microCommonSegments,
		microSegmentStrength,
		microSwingComuni,
		microIntelligenceDashboard,
	}
	for _, v := range views {
		if _, err := db.Exec(v); err != nil {
			return fmt.Errorf("create microtargeting view: %w", err)
		}
	}
	slog.Info("microtargeting views registered")
	return nil
}

const microCommonSegments = `CREATE OR REPLACE VIEW v_comune_party_segments AS
SELECT
    e.comune_istat, e.comune, e.party_canonical,
    e.year, e.election_type,
    e.voti, e.votanti,
    CAST(e.voti AS DOUBLE) / NULLIF(CAST(e.votanti AS DOUBLE), 0) * 100.0 AS pct_voti,
    p.popolazione_residente,
    p.eta_media,
    p.indice_vecchiaia,
    i.reddito_medio,
    em.tasso_occupazione,
    em.tasso_disoccupazione,
    CASE
        WHEN p.indice_vecchiaia > 200 THEN 'molto anziano'
        WHEN p.indice_vecchiaia > 150 THEN 'anziano'
        WHEN p.indice_vecchiaia > 100 THEN 'equilibrato'
        WHEN p.indice_vecchiaia IS NOT NULL THEN 'giovane'
    END AS profilo_demografico,
    CASE
        WHEN i.reddito_medio > 25000 THEN 'alto reddito'
        WHEN i.reddito_medio > 18000 THEN 'medio reddito'
        WHEN i.reddito_medio IS NOT NULL THEN 'basso reddito'
    END AS profilo_reddito
FROM election_results e
LEFT JOIN istat_population p ON e.comune_istat = p.comune_istat AND e.year = p.year
LEFT JOIN istat_income i ON e.comune_istat = i.comune_istat AND e.year = i.year
LEFT JOIN istat_employment em ON e.comune_istat = em.comune_istat AND e.year = em.year`

const microSegmentStrength = `CREATE OR REPLACE VIEW v_party_segment_strength AS
SELECT
    party_canonical,
    profilo_demografico,
    profilo_reddito,
    COUNT(DISTINCT comune_istat) AS n_comuni,
    CAST(SUM(voti) AS DOUBLE) AS voti_totali,
    ROUND(AVG(pct_voti), 1) AS pct_medio,
    ROUND(AVG(eta_media), 1) AS eta_media_segmento,
    ROUND(AVG(COALESCE(reddito_medio, 0)), 0) AS reddito_medio_segmento
FROM v_comune_party_segments
GROUP BY party_canonical, profilo_demografico, profilo_reddito
ORDER BY party_canonical, pct_medio DESC`

const microSwingComuni = `CREATE OR REPLACE VIEW v_swing_comuni AS
WITH ordered AS (
    SELECT party_canonical, comune_istat, year, pct_voti,
        profilo_demografico, profilo_reddito, reddito_medio, eta_media,
        LEAD(year) OVER (PARTITION BY party_canonical, comune_istat ORDER BY year) AS year_to,
        LEAD(pct_voti) OVER (PARTITION BY party_canonical, comune_istat ORDER BY year) AS pct_after
    FROM v_comune_party_segments
)
SELECT party_canonical, comune_istat,
    year AS year_from, year_to,
    pct_voti AS pct_before, pct_after,
    ROUND(pct_after - pct_voti, 1) AS swing,
    profilo_demografico, profilo_reddito,
    reddito_medio, eta_media
FROM ordered
WHERE year_to IS NOT NULL
ORDER BY ABS(swing) DESC NULLS LAST`

const microIntelligenceDashboard = `CREATE OR REPLACE VIEW v_party_intelligence_dashboard AS
SELECT
    e.party_canonical,
    e.year,
    CAST(SUM(e.voti) AS DOUBLE) AS voti_totali,
    COUNT(DISTINCT e.comune_istat) AS comuni_presenti,
    CASE WHEN SUM(e.voti) > 0
        THEN ROUND(SUM(CAST(e.voti AS DOUBLE) * COALESCE(p.eta_media, 45.0)) / SUM(CAST(e.voti AS DOUBLE)), 1)
    END AS eta_media_elettorato,
    CASE WHEN SUM(e.voti) > 0
        THEN ROUND(SUM(CAST(e.voti AS DOUBLE) * COALESCE(p.indice_vecchiaia, 150.0)) / SUM(CAST(e.voti AS DOUBLE)), 0)
    END AS indice_vecchiaia_elettorato,
    CASE WHEN SUM(e.voti) > 0
        THEN ROUND(SUM(CAST(e.voti AS DOUBLE) * COALESCE(i.reddito_medio, 20000.0)) / SUM(CAST(e.voti AS DOUBLE)), 0)
    END AS reddito_medio_elettorato,
    COALESCE(s.avg_sentiment, 0.0) AS sentiment_medio,
    COALESCE(s.total_mentions, 0) AS menzioni_social,
    COALESCE(po.avg_pct, 0.0) AS sondaggi_medi,
    COALESCE(po.n_polls, 0) AS n_sondaggi,
    COALESCE(pa.total_acts, 0) AS atti_presentati
FROM election_results e
LEFT JOIN istat_population p ON e.comune_istat = p.comune_istat AND e.year = p.year
LEFT JOIN istat_income i ON e.comune_istat = i.comune_istat AND e.year = i.year
LEFT JOIN (
    SELECT party, DATE_TRUNC('week', CAST(date AS DATE)) AS week, AVG(score) AS avg_sentiment, COUNT(*) AS total_mentions
    FROM sentiment_scores WHERE date IS NOT NULL AND date != ''
    GROUP BY party, DATE_TRUNC('week', CAST(date AS DATE))
) s ON e.party_canonical = s.party
LEFT JOIN (
    SELECT party_canonical, DATE_TRUNC('month', CAST(date AS DATE)) AS month, AVG(percentage) AS avg_pct, COUNT(*) AS n_polls
    FROM polls WHERE date IS NOT NULL AND date != ''
    GROUP BY party_canonical, DATE_TRUNC('month', CAST(date AS DATE))
) po ON e.party_canonical = po.party_canonical
LEFT JOIN (
    SELECT party_at_presentation AS party, COUNT(*) AS total_acts
    FROM parliamentary_acts
    GROUP BY party_at_presentation
) pa ON e.party_canonical = pa.party
WHERE e.level = 'comune'
GROUP BY e.party_canonical, e.year, s.avg_sentiment, s.total_mentions, po.avg_pct, po.n_polls, pa.total_acts
ORDER BY e.year DESC, voti_totali DESC`

package graphbuilder

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ff3300/aleph-v2/internal/gnn"
)

type PoliticalGraphBuilder struct {
	db        *sql.DB
	Graph     *gnn.Graph
	NodeIndex map[string]string
	Seed      int64
}

func NewPoliticalGraphBuilder(db *sql.DB) *PoliticalGraphBuilder {
	return &PoliticalGraphBuilder{
		db:        db,
		Graph:     gnn.NewGraph(),
		NodeIndex: make(map[string]string),
		Seed:      42,
	}
}

func normalizeID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "#", "-")
	return s
}

func (b *PoliticalGraphBuilder) getOrCreateID(name string, nodeType string) gnn.NodeID {
	norm := normalizeID(name)
	id := gnn.NodeID(nodeType + ":" + norm)
	if _, exists := b.NodeIndex[strings.ToLower(strings.TrimSpace(name))]; !exists {
		b.NodeIndex[strings.ToLower(strings.TrimSpace(name))] = norm
	}
	return id
}

func (b *PoliticalGraphBuilder) Build() error {
	if err := b.buildPartyNodes(); err != nil {
		return err
	}
	if err := b.buildPersonNodes(); err != nil {
		return err
	}
	if err := b.buildDonorNodes(); err != nil {
		return err
	}
	if err := b.buildElectionNodes(); err != nil {
		return err
	}
	if err := b.buildPartyElectionEdges(); err != nil {
		return err
	}
	if err := b.buildDonorPartyEdges(); err != nil {
		return err
	}
	if err := b.buildPersonPartyEdges(); err != nil {
		return err
	}
	return nil
}

func (b *PoliticalGraphBuilder) buildPartyNodes() error {
	rows, err := b.db.Query(`SELECT DISTINCT recipient_party FROM party_funding WHERE recipient_party != ''`)
	if err != nil {
		return fmt.Errorf("query party_funding for distinct parties: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan party_funding party name: %w", err)
		}
		id := b.getOrCreateID(name, "party")
		b.Graph.AddNode(&gnn.WorkflowNode{ID: id, Type: "party"})
	}

	rows2, err := b.db.Query(`SELECT DISTINCT desc_lis FROM election_results_2022_camera WHERE desc_lis != ''`)
	if err != nil {
		return fmt.Errorf("query election_results_2022_camera for distinct parties: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var name string
		if err := rows2.Scan(&name); err != nil {
			return fmt.Errorf("scan election_results_2022_camera party name: %w", err)
		}
		id := b.getOrCreateID(name, "party")
		b.Graph.AddNode(&gnn.WorkflowNode{ID: id, Type: "party"})
	}
	return nil
}

func (b *PoliticalGraphBuilder) buildPersonNodes() error {
	rows, err := b.db.Query(`SELECT DISTINCT name FROM pep_entities WHERE name != ''`)
	if err != nil {
		slog.Warn("skipping pep_entities (table not found)", "error", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return fmt.Errorf("scan pep_entities person name: %w", err)
			}
			id := b.getOrCreateID(name, "person")
			b.Graph.AddNode(&gnn.WorkflowNode{ID: id, Type: "person"})
		}
	}

	rows2, err := b.db.Query(`SELECT DISTINCT person_name FROM opdm_memberships WHERE person_name != ''`)
	if err != nil {
		slog.Warn("skipping opdm_memberships (table not found)", "error", err)
	} else {
		defer rows2.Close()
		for rows2.Next() {
			var name string
			if err := rows2.Scan(&name); err != nil {
				return fmt.Errorf("scan opdm_memberships person name: %w", err)
			}
			id := b.getOrCreateID(name, "person")
			b.Graph.AddNode(&gnn.WorkflowNode{ID: id, Type: "person"})
		}
	}
	return nil
}

func (b *PoliticalGraphBuilder) buildDonorNodes() error {
	rows, err := b.db.Query(`SELECT DISTINCT donor_name FROM party_funding WHERE donor_name != ''`)
	if err != nil {
		return fmt.Errorf("query party_funding for distinct donors: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan party_funding donor name: %w", err)
		}
		id := b.getOrCreateID(name, "donor")
		b.Graph.AddNode(&gnn.WorkflowNode{ID: id, Type: "donor"})
	}
	return nil
}

func (b *PoliticalGraphBuilder) buildElectionNodes() error {
	var count int
	if err := b.db.QueryRow(`SELECT COUNT(*) FROM election_results_2022_camera`).Scan(&count); err != nil {
		return fmt.Errorf("query election_results_2022_camera count: %w", err)
	}
	if count > 0 {
		id2022Camera := gnn.NodeID("election:camera:2022")
		b.Graph.AddNode(&gnn.WorkflowNode{ID: id2022Camera, Type: "election"})
	}
	return nil
}

func (b *PoliticalGraphBuilder) buildPartyElectionEdges() error {
	rows, err := b.db.Query(`SELECT desc_lis, TRY_CAST(perc AS DOUBLE) FROM election_results_2022_camera WHERE TRY_CAST(perc AS DOUBLE) > 0`)
	if err != nil {
		return fmt.Errorf("query election_results_2022_camera for edges: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var perc float64
		if err := rows.Scan(&name, &perc); err != nil {
			return fmt.Errorf("scan election_results_2022_camera edge: %w", err)
		}
		partyID := b.getOrCreateID(name, "party")
		electionID := gnn.NodeID("election:camera:2022")
		b.Graph.AddEdge(gnn.Edge{
			Source: partyID,
			Target: electionID,
			Weight: perc / 100.0,
		})
	}
	return nil
}

func (b *PoliticalGraphBuilder) buildDonorPartyEdges() error {
	rows, err := b.db.Query(`SELECT recipient_party, donor_name, COALESCE(SUM(donation_amount), 0) FROM party_funding WHERE recipient_party != '' AND donor_name != '' GROUP BY recipient_party, donor_name`)
	if err != nil {
		return fmt.Errorf("query party_funding for donor-party edges: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var party, donor string
		var amount float64
		if err := rows.Scan(&party, &donor, &amount); err != nil {
			return fmt.Errorf("scan party_funding donor-party edge: %w", err)
		}
		donorID := b.getOrCreateID(donor, "donor")
		partyID := b.getOrCreateID(party, "party")
		weight := 1.0
		if amount > 1 {
			weight = mathLog(amount)
		}
		b.Graph.AddEdge(gnn.Edge{Source: donorID, Target: partyID, Weight: weight})
	}
	return nil
}

func (b *PoliticalGraphBuilder) buildPersonPartyEdges() error {
	rows, err := b.db.Query(`SELECT DISTINCT name, party FROM pep_entities WHERE party != '' AND name != ''`)
	if err != nil {
		slog.Warn("skipping pep_entities person-party edges (table not found)", "error", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var person, party string
			if err := rows.Scan(&person, &party); err != nil {
				return fmt.Errorf("scan pep_entities person-party edge: %w", err)
			}
			personID := b.getOrCreateID(person, "person")
			partyID := b.getOrCreateID(party, "party")
			b.Graph.AddEdge(gnn.Edge{Source: personID, Target: partyID, Weight: 1.0})
		}
	}

	rows2, err := b.db.Query(`SELECT DISTINCT person_name, org_name FROM opdm_memberships WHERE person_name != '' AND org_name != ''`)
	if err != nil {
		slog.Warn("skipping opdm_memberships person-party edges (table not found)", "error", err)
	} else {
		defer rows2.Close()
		for rows2.Next() {
			var person, org string
			if err := rows2.Scan(&person, &org); err != nil {
				return fmt.Errorf("scan opdm_memberships person-party edge: %w", err)
			}
			personID := b.getOrCreateID(person, "person")
			partyID := b.getOrCreateID(org, "party")
			b.Graph.AddEdge(gnn.Edge{Source: personID, Target: partyID, Weight: 0.8})
		}
	}
	return nil
}

const logTaylorIterations = 50

func mathLog(x float64) float64 {
	n := x
	result := 0.0
	n = (n - 1) / (n + 1)
	term := n
	for i := 1; i < logTaylorIterations; i += 2 {
		result += term / float64(i)
		term *= n * n
	}
	return 2 * result
}

package graphbuilder

import (
	"sort"
	"strings"

	"github.com/ff3300/aleph-v2/internal/gnn"
)

type ExportGraph struct {
	Nodes []ExportNode `json:"nodes"`
	Edges []ExportEdge `json:"edges"`
}

type ExportNode struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type ExportEdge struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Weight float64 `json:"weight"`
	Type   string  `json:"type"`
}

type ExportPredictions struct {
	Predictions   []LinkPrediction `json:"predictions"`
	ModelMetadata struct {
		AUC           float64 `json:"auc"`
		MRR           float64 `json:"mrr"`
		EmbeddingDim  int     `json:"embedding_dim"`
	} `json:"model_metrics"`
}

type LinkPrediction struct {
	SourceNode     string  `json:"source_node"`
	TargetNode     string  `json:"target_node"`
	Score          float64 `json:"score"`
	Interpretation string  `json:"interpretation"`
}

func (b *PoliticalGraphBuilder) ExportGraph() *ExportGraph {
	export := &ExportGraph{
		Nodes: make([]ExportNode, 0, b.Graph.NumNodes()),
		Edges: make([]ExportEdge, 0, b.Graph.NumEdges()),
	}
	for _, node := range b.Graph.Nodes {
		export.Nodes = append(export.Nodes, ExportNode{
			ID:    string(node.ID),
			Type:  node.Type,
			Label: string(node.ID),
		})
	}
	for _, edge := range b.Graph.Edges {
		edgeType := classifyEdgeType(string(edge.Source), string(edge.Target))
		export.Edges = append(export.Edges, ExportEdge{
			Source: string(edge.Source),
			Target: string(edge.Target),
			Weight: edge.Weight,
			Type:   edgeType,
		})
	}
	return export
}

func classifyEdgeType(src, trg string) string {
	if strings.HasPrefix(src, "party:") && strings.HasPrefix(trg, "elect") {
		return "party_election"
	}
	if strings.HasPrefix(src, "donor:") && strings.HasPrefix(trg, "party:") {
		return "party_donor"
	}
	if strings.HasPrefix(src, "perso") && strings.HasPrefix(trg, "party:") {
		return "person_party"
	}
	if strings.HasPrefix(trg, "party:") && strings.HasPrefix(src, "party:") {
		return "other"
	}
	return "other"
}

const maxExportedPredictions = 50

func (b *PoliticalGraphBuilder) ExportPredictions(model *gnn.GNNModel, embeddings [][]float64, nodeIndex map[gnn.NodeID]int, auc, mrr float64) *ExportPredictions {
	if model == nil {
		return &ExportPredictions{}
	}

	reverseIndex := make(map[int]gnn.NodeID)
	for id, idx := range nodeIndex {
		reverseIndex[idx] = id
	}

	existingEdges := make(map[[2]int]bool)
	for _, edge := range b.Graph.Edges {
		u, okU := nodeIndex[edge.Source]
		v, okV := nodeIndex[edge.Target]
		if okU && okV {
			existingEdges[[2]int{u, v}] = true
			existingEdges[[2]int{v, u}] = true
		}
	}

	type scored struct {
		u     int
		v     int
		score float64
	}
	var all []scored
	for u := 0; u < len(reverseIndex); u++ {
		for v := u + 1; v < len(reverseIndex); v++ {
			if existingEdges[[2]int{u, v}] {
				continue
			}
			all = append(all, scored{u: u, v: v, score: gnn.PredictScore(embeddings, u, v)})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })

	limit := maxExportedPredictions
	if len(all) < limit {
		limit = len(all)
	}

	predictions := &ExportPredictions{}
	predictions.ModelMetadata.AUC = auc
	predictions.ModelMetadata.MRR = mrr
	predictions.ModelMetadata.EmbeddingDim = model.Dim

	for i := 0; i < limit; i++ {
		interpretation := "potential alliance"
		if classifyEdgeType(string(reverseIndex[all[i].u]), string(reverseIndex[all[i].v])) == "party_donor" {
			interpretation = "likely donation target"
		}
		predictions.Predictions = append(predictions.Predictions, LinkPrediction{
			SourceNode:     string(reverseIndex[all[i].u]),
			TargetNode:     string(reverseIndex[all[i].v]),
			Score:          all[i].score,
			Interpretation: interpretation,
		})
	}
	return predictions
}

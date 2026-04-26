// Package gnn implements a lightweight Graph Neural Network for workspace
// knowledge graph link prediction using positive-only training with
// negative sampling. The model uses a 2-layer LightGCN-style architecture
// with BPR (Bayesian Personalized Ranking) loss.
//
// Architecture overview:
//   - LightGCN convolution: H_{l+1} = A_norm @ H_l
//   - Final embedding: mean(H_0 + H_1 + H_2)
//   - Link prediction score: dot(emb_u, emb_v)
//   - BPR loss: -log(sigmoid(pos_score - neg_score))
//
// All operations use only Go standard library — no external ML frameworks.
package gnn

import "sort"

// NodeID is a unique identifier for a graph node.
type NodeID string

// WorkflowNode represents a node in the workspace knowledge graph.
type WorkflowNode struct {
	ID               NodeID
	Type             string
	InitialEmbedding []float64 // optional pre-trained features (may be nil)
}

// Edge represents a weighted relationship between two graph nodes.
// The graph is treated as undirected for convolution purposes.
type Edge struct {
	Source NodeID
	Target NodeID
	Weight float64
}

// Graph is a workspace knowledge graph with typed nodes and weighted edges.
type Graph struct {
	Nodes map[NodeID]*WorkflowNode
	Edges []Edge
}

// NewGraph creates an empty Graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[NodeID]*WorkflowNode),
	}
}

// AddNode inserts a node into the graph. If a node with the same ID
// already exists it is overwritten.
func (g *Graph) AddNode(n *WorkflowNode) {
	g.Nodes[n.ID] = n
}

// AddEdge appends an edge to the graph. The edge weight defaults to 1.0
// if zero.
func (g *Graph) AddEdge(e Edge) {
	if e.Weight == 0 {
		e.Weight = 1.0
	}
	g.Edges = append(g.Edges, e)
}

// NumNodes returns the number of nodes in the graph.
func (g *Graph) NumNodes() int { return len(g.Nodes) }

// NumEdges returns the number of edges in the graph.
func (g *Graph) NumEdges() int { return len(g.Edges) }

// NodeIDs returns all node IDs in insertion order (map iteration).
func (g *Graph) NodeIDs() []NodeID {
	ids := make([]NodeID, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	return ids
}

// BuildNodeIndex maps NodeID to a dense integer index (0..N-1).
// The ordering is deterministic (sorted by NodeID) so that repeated
// calls on the same graph always produce the same mapping.
func (g *Graph) BuildNodeIndex() map[NodeID]int {
	ids := make([]NodeID, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	idx := make(map[NodeID]int, len(g.Nodes))
	for i, id := range ids {
		idx[id] = i
	}
	return idx
}

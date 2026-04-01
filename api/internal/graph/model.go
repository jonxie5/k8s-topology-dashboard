package graph

import "time"

// Meta contains snapshot metadata returned with every graph response.
type Meta struct {
	SnapshotTime time.Time `json:"snapshotTime"`
	SnapshotID   string    `json:"snapshotId"`
	Truncated    bool      `json:"truncated"`
	Counts       Counts    `json:"counts"`
	Filters      Filters   `json:"filters"`
}

// Counts reports the number of nodes and edges in the returned graph.
type Counts struct {
	Nodes int `json:"nodes"`
	Edges int `json:"edges"`
}

// Filters echoes the effective filter parameters used to build the graph.
type Filters struct {
	Namespaces  []string `json:"namespaces"`
	Q           string   `json:"q,omitempty"`
	IncludePods bool     `json:"includePods"`
}

// NodeUI holds presentation hints consumed by the frontend.
type NodeUI struct {
	Group     string `json:"group"`
	Collapsed bool   `json:"collapsed"`
}

// Node represents a Kubernetes (or virtual) object in the topology graph.
type Node struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	APIVersion string            `json:"apiVersion"`
	Namespace  string            `json:"namespace"`
	Name       string            `json:"name"`
	Labels     map[string]string `json:"labels"`
	Status     map[string]any    `json:"status"`
	Tags       []string          `json:"tags"`
	UI         NodeUI            `json:"ui"`
}

// EdgeUI holds presentation hints for an edge.
type EdgeUI struct {
	Style string `json:"style"` // "solid" | "dashed"
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	ID     string         `json:"id"`
	From   string         `json:"from"`
	To     string         `json:"to"`
	Type   string         `json:"type"`   // owns | selects | routes | schedules
	Reason string         `json:"reason"` // ownerReference | labelSelector | backendRef | nodeName
	Meta   map[string]any `json:"meta,omitempty"`
	UI     EdgeUI         `json:"ui"`
}

// Graph is the top-level response for GET /api/graph.
type Graph struct {
	Meta  Meta   `json:"meta"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

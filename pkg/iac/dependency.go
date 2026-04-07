package iac

// DependencyNode represents a single IaC resource in the dependency graph.
type DependencyNode struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Type    string `json:"type"`    // "table", "function", "queue", "topic", "bucket", "module"
	Service string `json:"service"` // AWS service name
	Parent  string `json:"parent"`  // parent node ID (ComponentResource)
	URN     string `json:"urn,omitempty"`
}

// DependencyEdge represents a dependency between two IaC resources.
type DependencyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "parent", "dependsOn", "reference"
}

// DependencyGraph holds the full IaC resource graph.
type DependencyGraph struct {
	Nodes []DependencyNode `json:"nodes"`
	Edges []DependencyEdge `json:"dependencyEdges"`
}

// NewDependencyGraph creates an empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{}
}

// AddNode adds a resource node to the graph.
func (g *DependencyGraph) AddNode(n DependencyNode) {
	g.Nodes = append(g.Nodes, n)
}

// AddEdge adds a dependency edge to the graph.
func (g *DependencyGraph) AddEdge(e DependencyEdge) {
	g.Edges = append(g.Edges, e)
}

// Hierarchy returns a map of parent ID → child IDs, derived from node Parent fields.
func (g *DependencyGraph) Hierarchy() map[string][]string {
	h := make(map[string][]string)
	for _, n := range g.Nodes {
		if n.Parent != "" {
			h[n.Parent] = append(h[n.Parent], n.ID)
		}
	}
	return h
}

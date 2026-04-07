package iac

import "testing"

func TestDependencyGraph_AddNode(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode(DependencyNode{
		ID:      "table:membership-dev",
		Label:   "membership-dev",
		Type:    "table",
		Service: "dynamodb",
		Parent:  "module:dynamodb",
	})

	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
	if g.Nodes[0].ID != "table:membership-dev" {
		t.Errorf("expected ID table:membership-dev, got %s", g.Nodes[0].ID)
	}
}

func TestDependencyGraph_AddEdge(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge(DependencyEdge{
		Source: "fn:handler",
		Target: "table:membership-dev",
		Type:   "reference",
	})

	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].Type != "reference" {
		t.Errorf("expected type reference, got %s", g.Edges[0].Type)
	}
}

func TestDependencyGraph_Hierarchy(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode(DependencyNode{ID: "root", Label: "autotend", Type: "module", Parent: ""})
	g.AddNode(DependencyNode{ID: "module:ddb", Label: "DynamoDB", Type: "module", Parent: "root"})
	g.AddNode(DependencyNode{ID: "table:users", Label: "users", Type: "table", Service: "dynamodb", Parent: "module:ddb"})

	h := g.Hierarchy()
	if len(h["root"]) != 1 || h["root"][0] != "module:ddb" {
		t.Errorf("expected root -> [module:ddb], got %v", h["root"])
	}
	if len(h["module:ddb"]) != 1 || h["module:ddb"][0] != "table:users" {
		t.Errorf("expected module:ddb -> [table:users], got %v", h["module:ddb"])
	}
}

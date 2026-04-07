package iac

import "testing"

func TestParseStackState(t *testing.T) {
	stateJSON := `{
		"version": 3,
		"deployment": {
			"resources": [
				{
					"urn": "urn:pulumi:dev::autotend::pulumi:pulumi:Stack::autotend-dev",
					"type": "pulumi:pulumi:Stack",
					"parent": ""
				},
				{
					"urn": "urn:pulumi:dev::autotend::app:modules:Tables::tables",
					"type": "app:modules:Tables",
					"parent": "urn:pulumi:dev::autotend::pulumi:pulumi:Stack::autotend-dev"
				},
				{
					"urn": "urn:pulumi:dev::autotend::aws:dynamodb/table:Table::membership-dev",
					"type": "aws:dynamodb/table:Table",
					"parent": "urn:pulumi:dev::autotend::app:modules:Tables::tables",
					"outputs": {"name": "membership-dev-74a86de"}
				}
			]
		}
	}`

	graph, err := ParseStackState([]byte(stateJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(graph.Nodes))
	}
	h := graph.Hierarchy()
	if len(h) == 0 {
		t.Error("expected non-empty hierarchy")
	}
	// Find the DynamoDB table node
	var tableNode *DependencyNode
	for i, n := range graph.Nodes {
		if n.Service == "dynamodb" {
			tableNode = &graph.Nodes[i]
			break
		}
	}
	if tableNode == nil {
		t.Fatal("expected to find dynamodb node")
	}
	if tableNode.Label != "membership-dev-74a86de" {
		t.Errorf("expected label membership-dev-74a86de (from outputs.name), got %s", tableNode.Label)
	}
}

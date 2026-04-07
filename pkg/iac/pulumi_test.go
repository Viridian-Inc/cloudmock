package iac

import (
	"log/slog"
	"testing"
)

func TestParseDynamoTables(t *testing.T) {
	src := `
    this.membership = new aws.dynamodb.Table(` + "`membership${environmentSuffix}`" + `, {
      attributes: [
        { name: "pk", type: "S" },
        { name: "sk", type: "S" },
        { name: "cnSid", type: "S" },
        { name: "enSid", type: "S" },
      ],
      hashKey: "pk",
      rangeKey: "sk",
      billingMode: "PAY_PER_REQUEST",
      globalSecondaryIndexes: [
        {
          name: "userGsi",
          hashKey: "cnSid",
          rangeKey: "enSid",
          projectionType: "ALL",
        },
      ],
    }, { parent: this });

    this.userMetadata = new aws.dynamodb.Table(` + "`userMetadata${environmentSuffix}`" + `, {
      attributes: [
        { name: "pk", type: "S" },
      ],
      hashKey: "pk",
      billingMode: "PAY_PER_REQUEST",
    }, { parent: this });
  `

	tables := parseDynamoTables(src, "dev")
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}

	// Membership table
	m := tables[0]
	if m.Name != "membership-dev" {
		t.Errorf("name = %q, want membership-dev", m.Name)
	}
	if m.HashKey != "pk" || m.RangeKey != "sk" {
		t.Errorf("keys = %q/%q, want pk/sk", m.HashKey, m.RangeKey)
	}
	if len(m.Attributes) != 4 {
		t.Errorf("attributes = %d, want 4", len(m.Attributes))
	}
	if len(m.GSIs) != 1 {
		t.Fatalf("gsis = %d, want 1", len(m.GSIs))
	}
	if m.GSIs[0].Name != "userGsi" || m.GSIs[0].HashKey != "cnSid" {
		t.Errorf("gsi = %+v", m.GSIs[0])
	}

	// UserMetadata table (no range key, no GSIs)
	u := tables[1]
	if u.Name != "userMetadata-dev" {
		t.Errorf("name = %q, want userMetadata-dev", u.Name)
	}
	if u.HashKey != "pk" || u.RangeKey != "" {
		t.Errorf("keys = %q/%q, want pk/\"\"", u.HashKey, u.RangeKey)
	}
}

func TestParseDynamoTablesWithLSI(t *testing.T) {
	src := `
    this.session = new aws.dynamodb.Table(` + "`session${environmentSuffix}`" + `, {
      attributes: [
        { name: "pk", type: "S" },
        { name: "sk", type: "S" },
        { name: "eventDateLsiSk", type: "S" },
      ],
      hashKey: "pk",
      rangeKey: "sk",
      billingMode: "PAY_PER_REQUEST",
      streamEnabled: true,
      localSecondaryIndexes: [
        {
          name: "eventDateLsi",
          rangeKey: "eventDateLsiSk",
          projectionType: "ALL",
        },
      ],
    }, { parent: this });
  `

	tables := parseDynamoTables(src, "dev")
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	s := tables[0]
	if !s.StreamEnabled {
		t.Error("expected streamEnabled = true")
	}
	if len(s.LSIs) != 1 {
		t.Fatalf("lsis = %d, want 1", len(s.LSIs))
	}
	if s.LSIs[0].Name != "eventDateLsi" {
		t.Errorf("lsi name = %q", s.LSIs[0].Name)
	}
}

func TestExtractDependencyGraph(t *testing.T) {
	src := `
export class TablesModule extends pulumi.ComponentResource {
  constructor(name: string, args: any, opts: pulumi.ComponentResourceOptions) {
    super("app:modules:TablesModule", name, {}, opts);

    this.users = new aws.dynamodb.Table("users-dev", {
      attributes: [{ name: "pk", type: "S" }],
      hashKey: "pk",
      billingMode: "PAY_PER_REQUEST",
    }, { parent: this });

    this.orders = new aws.dynamodb.Table("orders-dev", {
      attributes: [{ name: "pk", type: "S" }],
      hashKey: "pk",
      billingMode: "PAY_PER_REQUEST",
    }, { parent: this, dependsOn: [this.users] });
  }
}
`
	graph := ExtractDependencyGraph(src, "dev")
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
	if len(graph.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes, got %d", len(graph.Nodes))
	}
	h := graph.Hierarchy()
	found := false
	for _, children := range h {
		if len(children) >= 2 {
			found = true
		}
	}
	if !found {
		t.Error("expected a parent with at least 2 children in hierarchy")
	}
	hasDep := false
	for _, e := range graph.Edges {
		if e.Type == "dependsOn" {
			hasDep = true
		}
	}
	if !hasDep {
		t.Error("expected at least one dependsOn edge")
	}
}

func TestImportPulumiDir(t *testing.T) {
	// Test against the real autotend-infra if available
	dir := "/Users/megan/work/neureaux/autotend-infra/pulumi/modules"
	result, err := ImportPulumiDir(dir, "dev", slog.Default())
	if err != nil {
		t.Skipf("skipping (infra dir not available): %v", err)
	}
	if len(result.Tables) == 0 {
		t.Fatal("expected tables from autotend-infra")
	}
	t.Logf("found %d tables", len(result.Tables))
	for _, table := range result.Tables {
		t.Logf("  %s (pk=%s sk=%s gsis=%d lsis=%d)", table.Name, table.HashKey, table.RangeKey, len(table.GSIs), len(table.LSIs))
	}
}

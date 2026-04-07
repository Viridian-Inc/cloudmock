# IaC Dependency View Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a layout toggle (Layered/Tree/Force) to the topology view so users can switch between runtime flow and IaC dependency tree.

**Architecture:** Extend the Pulumi parser to extract resource hierarchy, add a new `/api/topology/tree` endpoint, and add a layout dropdown to the frontend that switches ELK algorithms. Stack state reader is a secondary data source when `pulumi stack export` is available.

**Tech Stack:** Go (backend parser + API), TypeScript/Preact (frontend), ELK.js (graph layout)

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `pkg/iac/dependency.go` | Create | `DependencyGraph`, `DependencyNode`, `DependencyEdge` types |
| `pkg/iac/dependency_test.go` | Create | Unit tests for dependency types |
| `pkg/iac/pulumi.go` | Modify | Extend parser to extract hierarchy from `{ parent: this }` and `dependsOn` |
| `pkg/iac/pulumi_test.go` | Modify | Add tests for hierarchy extraction |
| `pkg/iac/pulumi_state.go` | Create | `pulumi stack export` JSON parser |
| `pkg/iac/pulumi_state_test.go` | Create | Unit tests for stack state parser |
| `pkg/admin/topology.go` | Modify | Add `TopologyTreeResponse` type, `SetDependencyGraph` method |
| `pkg/admin/api.go` | Modify | Add `GET /api/topology/tree` route and handler |
| `pkg/admin/api_test.go` | Modify | Add test for `/api/topology/tree` endpoint |
| `devtools/src/views/topology/layout-picker.tsx` | Create | Layout dropdown component |
| `devtools/src/views/topology/layout-picker.css` | Create | Dropdown styles |
| `devtools/src/views/topology/index.tsx` | Modify | Fetch tree data, pass layout mode to canvas |
| `devtools/src/views/topology/topology-canvas.tsx` | Modify | Accept layout mode, switch ELK algorithm |
| `cmd/gateway/main.go` | Modify | Wire dependency graph from IaC import to admin API |

---

### Task 1: Dependency Graph Types

**Files:**
- Create: `pkg/iac/dependency.go`
- Create: `pkg/iac/dependency_test.go`

- [ ] **Step 1: Write the test**

```go
// pkg/iac/dependency_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/iac/ -run TestDependencyGraph -v`
Expected: FAIL — types not defined

- [ ] **Step 3: Write the implementation**

```go
// pkg/iac/dependency.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/iac/ -run TestDependencyGraph -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/iac/dependency.go pkg/iac/dependency_test.go
git commit -m "feat(iac): add DependencyGraph types for IaC hierarchy"
```

---

### Task 2: Pulumi Source Parser — Hierarchy Extraction

**Files:**
- Modify: `pkg/iac/pulumi.go`
- Modify: `pkg/iac/pulumi_test.go`

- [ ] **Step 1: Write the test**

Add to `pkg/iac/pulumi_test.go`:

```go
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

	// Should have a module node + 2 table nodes
	if len(graph.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes, got %d", len(graph.Nodes))
	}

	// Check hierarchy
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

	// Check dependency edge
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/iac/ -run TestExtractDependencyGraph -v`
Expected: FAIL — `ExtractDependencyGraph` not defined

- [ ] **Step 3: Write the implementation**

Add to `pkg/iac/pulumi.go`:

```go
// componentClassRe matches "class Foo extends pulumi.ComponentResource"
var componentClassRe = regexp.MustCompile(`class\s+(\w+)\s+extends\s+pulumi\.ComponentResource`)

// parentThisRe matches "{ parent: this }" in resource options
var parentThisRe = regexp.MustCompile(`\},\s*\{\s*parent:\s*this`)

// dependsOnRe matches "dependsOn: [this.foo]" or "dependsOn: [this.foo, this.bar]"
var dependsOnRe = regexp.MustCompile(`dependsOn:\s*\[([^\]]+)\]`)

// resourceNewRe matches "new aws.SERVICE.RESOURCE("name", {" and captures service, resource type, and name
var resourceNewRe = regexp.MustCompile(`new\s+aws\.(\w+)\.(\w+)\(\s*["` + "`" + `]([^"` + "`" + `]+)["` + "`" + `]`)

// ExtractDependencyGraph parses Pulumi TypeScript source and builds a DependencyGraph
// capturing ComponentResource hierarchy, resource parent relationships, and dependsOn edges.
func ExtractDependencyGraph(src string, environment string) *DependencyGraph {
	graph := NewDependencyGraph()

	// Find ComponentResource classes (modules)
	classMatches := componentClassRe.FindAllStringSubmatchIndex(src, -1)
	for _, match := range classMatches {
		className := src[match[2]:match[3]]
		moduleID := "module:" + strings.ToLower(className)
		graph.AddNode(DependencyNode{
			ID:    moduleID,
			Label: className,
			Type:  "module",
		})

		// Find the class body (from match end to next class or EOF)
		classStart := match[0]
		classEnd := len(src)
		if nextClass := componentClassRe.FindStringIndex(src[match[1]:]); nextClass != nil {
			classEnd = match[1] + nextClass[0]
		}
		classBody := src[classStart:classEnd]

		// Find all resources within this class
		resMatches := resourceNewRe.FindAllStringSubmatch(classBody, -1)
		resIndexes := resourceNewRe.FindAllStringSubmatchIndex(classBody, -1)
		for i, res := range resMatches {
			svc := strings.ToLower(res[1])    // e.g., "dynamodb"
			resType := strings.ToLower(res[2]) // e.g., "table"
			resName := res[3]                  // e.g., "users-dev"

			nodeID := svc + ":" + resName
			nodeType := mapResourceType(svc, resType)

			node := DependencyNode{
				ID:      nodeID,
				Label:   resName,
				Type:    nodeType,
				Service: svc,
			}

			// Check if this resource has { parent: this }
			resEnd := classEnd - classStart
			if i+1 < len(resIndexes) {
				resEnd = resIndexes[i+1][0]
			}
			resBlock := classBody[resIndexes[i][0]:resEnd]
			if parentThisRe.MatchString(resBlock) {
				node.Parent = moduleID
			}

			graph.AddNode(node)

			// Check for dependsOn
			if depMatch := dependsOnRe.FindStringSubmatch(resBlock); depMatch != nil {
				deps := strings.Split(depMatch[1], ",")
				for _, dep := range deps {
					dep = strings.TrimSpace(dep)
					dep = strings.TrimPrefix(dep, "this.")
					// Try to resolve dep to a node ID by searching other resources
					for _, other := range resMatches {
						otherName := other[3]
						otherSvc := strings.ToLower(other[1])
						// Match by field name similarity
						if strings.Contains(strings.ToLower(dep), strings.ToLower(otherName)) ||
							strings.Contains(strings.ToLower(otherName), strings.ToLower(dep)) {
							graph.AddEdge(DependencyEdge{
								Source: nodeID,
								Target: otherSvc + ":" + otherName,
								Type:   "dependsOn",
							})
						}
					}
				}
			}
		}
	}

	// If no ComponentResource classes found, parse flat resources
	if len(classMatches) == 0 {
		resMatches := resourceNewRe.FindAllStringSubmatch(src, -1)
		for _, res := range resMatches {
			svc := strings.ToLower(res[1])
			resType := strings.ToLower(res[2])
			resName := res[3]
			graph.AddNode(DependencyNode{
				ID:      svc + ":" + resName,
				Label:   resName,
				Type:    mapResourceType(svc, resType),
				Service: svc,
			})
		}
	}

	if len(graph.Nodes) == 0 {
		return nil
	}
	return graph
}

// mapResourceType maps AWS service + resource class to a simplified type string.
func mapResourceType(svc, resClass string) string {
	switch svc {
	case "dynamodb":
		return "table"
	case "lambda":
		return "function"
	case "sqs":
		return "queue"
	case "sns":
		return "topic"
	case "s3":
		return "bucket"
	default:
		return strings.ToLower(resClass)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/iac/ -run TestExtractDependencyGraph -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/iac/pulumi.go pkg/iac/pulumi_test.go
git commit -m "feat(iac): extract dependency graph from Pulumi source"
```

---

### Task 3: Pulumi Stack State Reader

**Files:**
- Create: `pkg/iac/pulumi_state.go`
- Create: `pkg/iac/pulumi_state_test.go`

- [ ] **Step 1: Write the test**

```go
// pkg/iac/pulumi_state_test.go
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

	// The table should be a child of the tables module
	tableNode := findNode(graph, "membership-dev")
	if tableNode == nil {
		t.Fatal("expected to find membership-dev node")
	}
	if tableNode.Service != "dynamodb" {
		t.Errorf("expected service dynamodb, got %s", tableNode.Service)
	}
}

func findNode(g *DependencyGraph, labelContains string) *DependencyNode {
	for i, n := range g.Nodes {
		if n.Label == labelContains || n.ID == labelContains {
			return &g.Nodes[i]
		}
	}
	return nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/iac/ -run TestParseStackState -v`
Expected: FAIL — `ParseStackState` not defined

- [ ] **Step 3: Write the implementation**

```go
// pkg/iac/pulumi_state.go
package iac

import (
	"encoding/json"
	"strings"
)

// pulumiStackExport represents the top-level pulumi stack export JSON.
type pulumiStackExport struct {
	Version    int                    `json:"version"`
	Deployment pulumiDeployment       `json:"deployment"`
}

type pulumiDeployment struct {
	Resources []pulumiResource `json:"resources"`
}

type pulumiResource struct {
	URN     string                 `json:"urn"`
	Type    string                 `json:"type"`
	Parent  string                 `json:"parent"`
	Outputs map[string]interface{} `json:"outputs,omitempty"`
}

// ParseStackState parses the JSON output of `pulumi stack export` and returns a DependencyGraph.
func ParseStackState(data []byte) (*DependencyGraph, error) {
	var export pulumiStackExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, err
	}

	graph := NewDependencyGraph()
	urnToID := make(map[string]string)

	for _, res := range export.Deployment.Resources {
		id := urnToNodeID(res.URN)
		label := urnToLabel(res.URN)
		svc, nodeType := typeToServiceAndType(res.Type)

		// Use resolved name from outputs if available
		if name, ok := res.Outputs["name"]; ok {
			if s, ok := name.(string); ok {
				label = s
			}
		}

		parentID := ""
		if res.Parent != "" {
			parentID = urnToID[res.Parent]
		}

		node := DependencyNode{
			ID:      id,
			Label:   label,
			Type:    nodeType,
			Service: svc,
			Parent:  parentID,
			URN:     res.URN,
		}
		graph.AddNode(node)
		urnToID[res.URN] = id
	}

	return graph, nil
}

// urnToNodeID converts a Pulumi URN to a short node ID.
// "urn:pulumi:dev::autotend::aws:dynamodb/table:Table::membership-dev" → "dynamodb:membership-dev"
func urnToNodeID(urn string) string {
	parts := strings.Split(urn, "::")
	if len(parts) < 4 {
		return urn
	}
	typePart := parts[2]  // "aws:dynamodb/table:Table"
	namePart := parts[3]  // "membership-dev"

	svc := extractService(typePart)
	return svc + ":" + namePart
}

// urnToLabel extracts the resource name from the URN.
func urnToLabel(urn string) string {
	parts := strings.Split(urn, "::")
	if len(parts) < 4 {
		return urn
	}
	return parts[3]
}

// extractService extracts the AWS service name from a Pulumi type string.
// "aws:dynamodb/table:Table" → "dynamodb"
func extractService(typeStr string) string {
	if strings.HasPrefix(typeStr, "aws:") {
		rest := strings.TrimPrefix(typeStr, "aws:")
		if idx := strings.Index(rest, "/"); idx > 0 {
			return rest[:idx]
		}
		if idx := strings.Index(rest, ":"); idx > 0 {
			return rest[:idx]
		}
	}
	// For component resources like "app:modules:Tables"
	parts := strings.Split(typeStr, ":")
	if len(parts) >= 2 {
		return strings.ToLower(parts[len(parts)-1])
	}
	return typeStr
}

// typeToServiceAndType maps a Pulumi type to (service, nodeType).
func typeToServiceAndType(pulumiType string) (string, string) {
	if strings.HasPrefix(pulumiType, "aws:dynamodb") {
		return "dynamodb", "table"
	}
	if strings.HasPrefix(pulumiType, "aws:lambda") {
		return "lambda", "function"
	}
	if strings.HasPrefix(pulumiType, "aws:sqs") {
		return "sqs", "queue"
	}
	if strings.HasPrefix(pulumiType, "aws:sns") {
		return "sns", "topic"
	}
	if strings.HasPrefix(pulumiType, "aws:s3") {
		return "s3", "bucket"
	}
	if strings.HasPrefix(pulumiType, "pulumi:pulumi:Stack") {
		return "", "stack"
	}
	// Component resources
	return "", "module"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/iac/ -run TestParseStackState -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/iac/pulumi_state.go pkg/iac/pulumi_state_test.go
git commit -m "feat(iac): add Pulumi stack state parser for resolved dependencies"
```

---

### Task 4: API Endpoint — `/api/topology/tree`

**Files:**
- Modify: `pkg/admin/topology.go` (add response type)
- Modify: `pkg/admin/api.go` (add handler + route)

- [ ] **Step 1: Add the response type to `topology.go`**

Add after the existing `TopologyResponseV2` type (around line 44):

```go
// TopologyTreeResponse contains the IaC dependency hierarchy for tree layout.
type TopologyTreeResponse struct {
	Nodes           []DependencyNodeJSON  `json:"nodes"`
	Hierarchy       map[string][]string   `json:"hierarchy"`
	DependencyEdges []DependencyEdgeJSON  `json:"dependencyEdges"`
}

type DependencyNodeJSON struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Service string `json:"service"`
	Parent  string `json:"parent"`
}

type DependencyEdgeJSON struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}
```

- [ ] **Step 2: Add storage field and setter to admin API**

Add to `pkg/admin/api.go` in the `AdminAPI` struct fields (around line 120):

```go
depGraph *iac.DependencyGraph
```

Add setter method:

```go
// SetDependencyGraph stores the IaC dependency graph for the tree endpoint.
func (a *AdminAPI) SetDependencyGraph(g *iac.DependencyGraph) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.depGraph = g
}
```

- [ ] **Step 3: Add the handler**

Add to `pkg/admin/api.go`:

```go
func (a *AdminAPI) handleTopologyTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.RLock()
	g := a.depGraph
	a.mu.RUnlock()

	if g == nil || len(g.Nodes) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"no IaC project configured"}`))
		return
	}

	resp := TopologyTreeResponse{
		Hierarchy:       g.Hierarchy(),
		DependencyEdges: make([]DependencyEdgeJSON, len(g.Edges)),
	}

	for _, n := range g.Nodes {
		resp.Nodes = append(resp.Nodes, DependencyNodeJSON{
			ID:      n.ID,
			Label:   n.Label,
			Type:    n.Type,
			Service: n.Service,
			Parent:  n.Parent,
		})
	}

	for i, e := range g.Edges {
		resp.DependencyEdges[i] = DependencyEdgeJSON{
			Source: e.Source,
			Target: e.Target,
			Type:   e.Type,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
```

- [ ] **Step 4: Register the route**

Add to the route registration section in `api.go` (near the existing `/api/topology` route):

```go
mux.HandleFunc("/api/topology/tree", a.handleTopologyTree)
```

- [ ] **Step 5: Build and verify**

Run: `go build ./cmd/gateway/`
Expected: clean build

- [ ] **Step 6: Commit**

```bash
git add pkg/admin/topology.go pkg/admin/api.go
git commit -m "feat(api): add GET /api/topology/tree endpoint"
```

---

### Task 5: Wire IaC Import to Dependency Graph

**Files:**
- Modify: `cmd/gateway/main.go`

- [ ] **Step 1: Find the IaC import call and add dependency graph extraction**

In `cmd/gateway/main.go`, find where `iac.ImportPulumiDir` is called (search for `ImportPulumiDir`). After the existing IaC import, add:

```go
// Extract dependency graph from IaC source for tree view
if *iacDir != "" {
	depGraph := iac.ExtractDependencyGraphFromDir(*iacDir, *iacEnv)
	if depGraph != nil {
		adminAPI.SetDependencyGraph(depGraph)
		slog.Info("IaC dependency graph loaded", "nodes", len(depGraph.Nodes), "edges", len(depGraph.Edges))
	}
}
```

- [ ] **Step 2: Add `ExtractDependencyGraphFromDir` to `pulumi.go`**

Add to `pkg/iac/pulumi.go`:

```go
// ExtractDependencyGraphFromDir scans a Pulumi project directory and builds a DependencyGraph.
func ExtractDependencyGraphFromDir(dir string, environment string) *DependencyGraph {
	merged := NewDependencyGraph()

	files, err := filepath.Glob(filepath.Join(dir, "**", "*.ts"))
	if err != nil || len(files) == 0 {
		// Try modules subdirectory
		files, _ = filepath.Glob(filepath.Join(dir, "modules", "*.ts"))
	}
	// Also check root
	rootFiles, _ := filepath.Glob(filepath.Join(dir, "*.ts"))
	files = append(files, rootFiles...)

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		g := ExtractDependencyGraph(string(data), environment)
		if g != nil {
			merged.Nodes = append(merged.Nodes, g.Nodes...)
			merged.Edges = append(merged.Edges, g.Edges...)
		}
	}

	if len(merged.Nodes) == 0 {
		return nil
	}
	return merged
}
```

- [ ] **Step 3: Build and verify**

Run: `go build ./cmd/gateway/`
Expected: clean build

- [ ] **Step 4: Commit**

```bash
git add cmd/gateway/main.go pkg/iac/pulumi.go
git commit -m "feat: wire IaC dependency graph to admin API on startup"
```

---

### Task 6: Frontend — Layout Picker Component

**Files:**
- Create: `devtools/src/views/topology/layout-picker.tsx`
- Create: `devtools/src/views/topology/layout-picker.css`

- [ ] **Step 1: Create the CSS**

```css
/* devtools/src/views/topology/layout-picker.css */
.layout-picker {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--bg-surface, #1a1a2e);
  border: 1px solid var(--border-color, #333);
  border-radius: 6px;
  padding: 2px;
}

.layout-picker-btn {
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 500;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--text-secondary, #888);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.layout-picker-btn:hover {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-primary, #eee);
}

.layout-picker-btn.active {
  background: var(--accent, #52b788);
  color: #0a0e14;
  font-weight: 600;
}

.layout-picker-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
```

- [ ] **Step 2: Create the component**

```tsx
// devtools/src/views/topology/layout-picker.tsx
import './layout-picker.css';

export type LayoutMode = 'layered' | 'tree' | 'force';

interface LayoutPickerProps {
  value: LayoutMode;
  onChange: (mode: LayoutMode) => void;
  treeAvailable: boolean;
}

export function LayoutPicker({ value, onChange, treeAvailable }: LayoutPickerProps) {
  return (
    <div class="layout-picker">
      <button
        class={`layout-picker-btn ${value === 'layered' ? 'active' : ''}`}
        onClick={() => onChange('layered')}
      >
        Layered
      </button>
      <button
        class={`layout-picker-btn ${value === 'tree' ? 'active' : ''}`}
        onClick={() => onChange('tree')}
        disabled={!treeAvailable}
        title={treeAvailable ? 'IaC dependency tree' : 'No IaC project configured'}
      >
        Tree
      </button>
      <button
        class={`layout-picker-btn ${value === 'force' ? 'active' : ''}`}
        onClick={() => onChange('force')}
      >
        Force
      </button>
    </div>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add devtools/src/views/topology/layout-picker.tsx devtools/src/views/topology/layout-picker.css
git commit -m "feat(devtools): add layout picker component for topology view"
```

---

### Task 7: Frontend — Integrate Layout Picker into Topology View

**Files:**
- Modify: `devtools/src/views/topology/index.tsx`
- Modify: `devtools/src/views/topology/topology-canvas.tsx`

- [ ] **Step 1: Add tree data fetch and layout state to `index.tsx`**

Add imports at the top of `index.tsx`:

```tsx
import { LayoutPicker, type LayoutMode } from './layout-picker';
import { api } from '../../lib/api';
```

Add state variables (with the other `useState` calls):

```tsx
const [layoutMode, setLayoutMode] = useState<LayoutMode>(() => {
  return (localStorage.getItem('cloudmock:topo-layout') as LayoutMode) || 'layered';
});
const [treeData, setTreeData] = useState<any>(null);
const [treeAvailable, setTreeAvailable] = useState(false);
```

Add effect to fetch tree data:

```tsx
useEffect(() => {
  api<any>('/api/topology/tree')
    .then((data) => {
      setTreeData(data);
      setTreeAvailable(true);
    })
    .catch(() => {
      setTreeAvailable(false);
    });
}, []);
```

Add layout mode persistence:

```tsx
const handleLayoutChange = (mode: LayoutMode) => {
  setLayoutMode(mode);
  localStorage.setItem('cloudmock:topo-layout', mode);
};
```

- [ ] **Step 2: Render the LayoutPicker in the topology toolbar**

Find the topology header/toolbar area in `index.tsx` (look for the `<div>` that contains the toolbar buttons). Add the LayoutPicker next to existing controls:

```tsx
<LayoutPicker
  value={layoutMode}
  onChange={handleLayoutChange}
  treeAvailable={treeAvailable}
/>
```

- [ ] **Step 3: Pass layout mode to TopologyCanvas**

Update the `<TopologyCanvas>` render call to pass layout mode and tree data:

```tsx
<TopologyCanvas
  nodes={...}
  edges={...}
  layoutMode={layoutMode}
  treeHierarchy={treeData?.hierarchy}
  // ... existing props
/>
```

- [ ] **Step 4: Update `topology-canvas.tsx` to accept and use layout mode**

Add to the component props interface:

```tsx
layoutMode?: 'layered' | 'tree' | 'force';
treeHierarchy?: Record<string, string[]>;
```

Update the ELK layout options (around line 123-133) to switch based on mode:

```tsx
const getLayoutOptions = (mode: string) => {
  const base = {
    'elk.direction': 'DOWN',
    'elk.spacing.nodeNode': '50',
    'elk.spacing.edgeEdge': '20',
    'elk.padding': '[top=50,left=50,bottom=50,right=50]',
  };

  switch (mode) {
    case 'tree':
      return {
        ...base,
        'elk.algorithm': 'mrtree',
        'elk.mrtree.weighting': 'CONSTRAINT',
        'elk.spacing.nodeNode': '40',
      };
    case 'force':
      return {
        ...base,
        'elk.algorithm': 'force',
        'elk.force.iterations': '300',
        'elk.spacing.nodeNode': '80',
      };
    default:
      return {
        ...base,
        'elk.algorithm': 'layered',
        'elk.layered.spacing.nodeNodeBetweenLayers': '80',
        'elk.layered.nodePlacement.strategy': 'BRANDES_KOEPF',
        'elk.layered.crossingMinimization.strategy': 'LAYER_SWEEP',
        'elk.layered.considerModelOrder.strategy': 'NODES_AND_EDGES',
      };
  }
};
```

Replace the existing hardcoded `layoutOptions` with:

```tsx
const layoutOptions = getLayoutOptions(layoutMode || 'layered');
```

- [ ] **Step 5: Build and verify**

Run: `cd devtools && pnpm build`
Expected: clean build

- [ ] **Step 6: Commit**

```bash
git add devtools/src/views/topology/index.tsx devtools/src/views/topology/topology-canvas.tsx
git commit -m "feat(devtools): integrate layout picker with ELK algorithm switching"
```

---

### Task 8: Integration Test & Push

**Files:**
- None new — verifying everything works together

- [ ] **Step 1: Run all backend tests**

Run: `go test ./pkg/iac/... ./pkg/admin/... -timeout 60s -v`
Expected: all PASS

- [ ] **Step 2: Run the devtools build**

Run: `cd devtools && pnpm build`
Expected: clean build

- [ ] **Step 3: Manual smoke test**

```bash
# Start CloudMock with IaC pointing at autotend infra
go build -o /tmp/cm-tree ./cmd/gateway/
/tmp/cm-tree --iac /Users/megan/work/neureaux/autotend-infra/pulumi &
sleep 2

# Verify tree endpoint
curl -s http://localhost:4599/api/topology/tree | python3 -m json.tool | head -20

# Should show nodes with parent relationships and a hierarchy map
kill %1
```

- [ ] **Step 4: Push**

```bash
git push origin main
```

---

## Summary

| Task | Component | Est. Time |
|------|-----------|-----------|
| 1 | DependencyGraph types | 5 min |
| 2 | Pulumi source parser — hierarchy | 10 min |
| 3 | Stack state reader | 10 min |
| 4 | API endpoint `/api/topology/tree` | 5 min |
| 5 | Wire IaC import to dependency graph | 5 min |
| 6 | Layout picker component | 5 min |
| 7 | Integrate into topology view | 10 min |
| 8 | Integration test & push | 5 min |

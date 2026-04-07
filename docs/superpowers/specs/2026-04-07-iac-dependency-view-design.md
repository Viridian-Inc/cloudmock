# IaC Dependency View — Design Spec

**Date:** 2026-04-07
**Status:** Approved

## Summary

Add a layout toggle to the CloudMock devtools topology view that switches between runtime flow (current layered layout) and an IaC dependency tree. The dependency tree shows Pulumi resource hierarchy with parent-child relationships, enriched with runtime traffic data. No new view — a dropdown in the existing topology toolbar swaps the ELK layout algorithm.

## Architecture

### Data Sources (priority order)

1. **Pulumi stack state** (`pulumi stack export`) — resolved URN tree with actual resource names and computed dependencies. Used when `PULUMI_ACCESS_TOKEN` is set and a stack is available.
2. **Pulumi source parse** (`--iac` flag) — reads `.ts` files to extract resource definitions, `ComponentResource` nesting, explicit `dependsOn`, and implicit references (variable usage). Default when no stack is available.
3. **Runtime traffic** — existing request flow data enriches edges with `callCount`, `avgLatencyMs`, error rates regardless of layout mode.

### Backend

#### 1. IaC Dependency Extraction (`pkg/iac/pulumi.go`)

Extend the existing Pulumi parser to capture:
- Resource parent-child relationships (`ComponentResource` nesting via `{ parent: this }`)
- Explicit `dependsOn` references between resources
- Implicit dependencies (Lambda references a DynamoDB table name variable)

New types:
```go
type DependencyGraph struct {
    Nodes []DependencyNode
    Edges []DependencyEdge
}

type DependencyNode struct {
    ID       string // e.g., "dynamodb:membership-dev"
    Label    string // e.g., "membership-dev"
    Type     string // "table", "function", "queue", "topic", "bucket", "module"
    Service  string // AWS service name
    Parent   string // parent node ID (ComponentResource)
    URN      string // Pulumi URN if available
}

type DependencyEdge struct {
    Source string
    Target string
    Type   string // "parent", "dependsOn", "reference"
}
```

#### 2. Stack State Reader (new: `pkg/iac/pulumi_state.go`)

Parse `pulumi stack export` JSON output to extract:
- Resource URN tree (parent-child from URN hierarchy)
- Resolved resource names (not template expressions)
- Provider dependencies
- Cross-stack references

Falls back to source parse when `pulumi stack export` fails or is unavailable.

#### 3. API Endpoint (`pkg/admin/topology.go`)

New endpoint: `GET /api/topology/tree`

Response:
```json
{
    "nodes": [
        {"id": "module:dynamodb", "label": "DynamoDB Tables", "type": "module", "service": "dynamodb", "parent": "root"},
        {"id": "table:membership-dev", "label": "membership-dev", "type": "table", "service": "dynamodb", "parent": "module:dynamodb"}
    ],
    "hierarchy": {
        "root": ["module:dynamodb", "module:lambda", "module:sqs"],
        "module:dynamodb": ["table:membership-dev", "table:enterprise-dev", "table:attendance-dev"],
        "module:lambda": ["fn:attendance-handler", "fn:order-handler"]
    },
    "dependencyEdges": [
        {"source": "fn:attendance-handler", "target": "table:attendance-dev", "type": "reference"},
        {"source": "fn:attendance-handler", "target": "table:session-dev", "type": "reference"}
    ]
}
```

Returns `404` with `{"error": "no IaC project configured"}` when no `--iac` flag was passed.

### Frontend

#### 4. Layout Dropdown (`devtools/src/views/topology/`)

Add a layout selector to the topology toolbar. Three modes:

| Mode | ELK Algorithm | Data Source | Description |
|------|--------------|-------------|-------------|
| **Layered** | `elk.layered` | Runtime flow | Current behavior. Horizontal bands by call direction. |
| **Tree** | `elk.mrtree` | IaC hierarchy + runtime | Root at top, children below. Parent-child from IaC, traffic on edges. |
| **Force** | `elk.force` | Runtime flow | Organic clustering. Nodes repel, edges attract. |

Implementation:
- Fetch `/api/topology/tree` on mount. If 404, disable Tree option with tooltip.
- Switching modes re-runs ELK layout with different algorithm options.
- Same nodes, edges, inspector sidebar, minimap, health coloring, traffic animation.
- Tree mode adds hierarchy constraints: `elk.mrtree` with `parent` properties from the hierarchy map.
- Layout preference persisted in `localStorage`.

#### Visual Differences in Tree Mode

- Hierarchy edges (parent→child) rendered as thin gray lines
- Dependency edges (`dependsOn`, `reference`) rendered as dashed colored lines
- Runtime traffic edges remain as animated solid lines with thickness = call count
- Module/component nodes rendered as group boxes (similar to existing domain groups)

## Data Flow

```
Startup:
  --iac path/to/pulumi
    → pkg/iac/pulumi.go parses .ts files
    → extracts resources, hierarchy, dependencies
    → if PULUMI_ACCESS_TOKEN set:
        → tries pulumi stack export
        → merges resolved state (wins over source parse)
    → stores DependencyGraph in admin API

Runtime:
  GET /api/topology       → existing: nodes + edges + traffic metrics
  GET /api/topology/tree  → new: hierarchy + dependency edges

Frontend:
  On mount: fetch both endpoints
  Layout dropdown: switch ELK algorithm
  Tree mode: merge hierarchy into ELK parent constraints
  All modes: same traffic enrichment (callCount, latency, errors)
```

## Error Handling

| Condition | Behavior |
|-----------|----------|
| No `--iac` flag | Tree option disabled in dropdown, tooltip: "No IaC project configured" |
| Malformed Pulumi source | Log warning, Tree option disabled |
| `pulumi stack export` fails | Log warning, fall back to source parse |
| Circular dependencies | ELK `mrtree` breaks back-edges automatically |
| IaC has resources not in CloudMock | Show as gray "external" nodes |
| CloudMock has resources not in IaC | Show as normal (discovered at runtime) |

## Testing

- **Unit:** Pulumi dependency parser — mock `.ts` files with known `ComponentResource` nesting, `dependsOn`, variable references
- **Unit:** Stack state JSON parser — mock `pulumi stack export` output with URN tree
- **Unit:** `/api/topology/tree` endpoint — verify response shape, 404 when no IaC
- **Frontend:** Snapshot test for layout dropdown rendering
- **Integration:** Verify tree layout with autotend Pulumi project produces correct hierarchy

## Scope

**In scope:**
- Layout dropdown (Layered / Tree / Force)
- Pulumi source parse for hierarchy
- Pulumi stack export for resolved state
- Runtime traffic enrichment in all modes
- Tree mode visual styling

**Out of scope:**
- Terraform HCL dependency parsing (future)
- CDK/SAM support (future)
- Editing IaC from the devtools
- Diff view (comparing IaC vs runtime state)

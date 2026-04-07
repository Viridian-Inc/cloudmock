package iac

import (
	"encoding/json"
	"fmt"
	"strings"
)

// pulumiStateFile mirrors the JSON structure of `pulumi stack export` output.
type pulumiStateFile struct {
	Version    int `json:"version"`
	Deployment struct {
		Resources []pulumiResource `json:"resources"`
	} `json:"deployment"`
}

// pulumiResource mirrors a single resource entry in the Pulumi state.
type pulumiResource struct {
	URN     string                 `json:"urn"`
	Type    string                 `json:"type"`
	Parent  string                 `json:"parent"`
	Outputs map[string]interface{} `json:"outputs"`
}

// ParseStackState parses the JSON output of `pulumi stack export` into a DependencyGraph.
//
// URN format: urn:pulumi:STACK::PROJECT::TYPE::NAME
// TYPE examples:
//   - pulumi:pulumi:Stack         → type="stack", service=""
//   - aws:dynamodb/table:Table    → service="dynamodb", type="table"
//   - app:modules:Tables          → type="module", service=""  (no aws: prefix)
func ParseStackState(data []byte) (*DependencyGraph, error) {
	var state pulumiStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("pulumi_state: failed to parse JSON: %w", err)
	}

	// Build a URN → node-ID map so we can resolve parent references.
	urnToID := make(map[string]string, len(state.Deployment.Resources))
	for _, r := range state.Deployment.Resources {
		urnToID[r.URN] = r.URN // use URN as the stable node ID
	}

	graph := NewDependencyGraph()

	for _, r := range state.Deployment.Resources {
		service, nodeType := classifyType(r.Type)
		label := nameFromURN(r.URN)

		// If outputs contains a "name" key, prefer that as the label.
		if nameVal, ok := r.Outputs["name"]; ok {
			if nameStr, ok := nameVal.(string); ok && nameStr != "" {
				label = nameStr
			}
		}

		parentID := ""
		if r.Parent != "" {
			if pid, found := urnToID[r.Parent]; found {
				parentID = pid
			}
		}

		node := DependencyNode{
			ID:      r.URN,
			Label:   label,
			Type:    nodeType,
			Service: service,
			Parent:  parentID,
			URN:     r.URN,
		}
		graph.AddNode(node)

		if parentID != "" {
			graph.AddEdge(DependencyEdge{
				Source: parentID,
				Target: r.URN,
				Type:   "parent",
			})
		}
	}

	return graph, nil
}

// classifyType derives (service, nodeType) from a Pulumi resource type string.
//
// Mapping rules:
//   - "pulumi:pulumi:Stack"       → ("", "stack")
//   - "aws:<svc>/...:..."         → (svc, lowercase-resource-kind)
//   - anything else               → ("", "module")
func classifyType(pulumiType string) (service, nodeType string) {
	// Pulumi built-ins: pulumi:pulumi:Stack, pulumi:pulumi:StackReference, etc.
	if strings.HasPrefix(pulumiType, "pulumi:pulumi:") {
		kind := strings.ToLower(strings.TrimPrefix(pulumiType, "pulumi:pulumi:"))
		return "", kind
	}

	// AWS resources: "aws:<service>/<subpath>:<ResourceKind>"
	// e.g. "aws:dynamodb/table:Table" → service="dynamodb", type="table"
	if strings.HasPrefix(pulumiType, "aws:") {
		rest := strings.TrimPrefix(pulumiType, "aws:")
		// rest = "dynamodb/table:Table"
		slashIdx := strings.Index(rest, "/")
		if slashIdx != -1 {
			svc := rest[:slashIdx]                                         // "dynamodb"
			afterSlash := rest[slashIdx+1:]                                // "table:Table"
			colonIdx := strings.Index(afterSlash, ":")
			var kind string
			if colonIdx != -1 {
				kind = strings.ToLower(afterSlash[:colonIdx]) // "table"
			} else {
				kind = strings.ToLower(afterSlash)
			}
			return svc, kind
		}
		// Fallback for "aws:Service:Kind" style (no slash)
		colonIdx := strings.Index(rest, ":")
		if colonIdx != -1 {
			svc := strings.ToLower(rest[:colonIdx])
			kind := strings.ToLower(rest[colonIdx+1:])
			return svc, kind
		}
		return strings.ToLower(rest), "resource"
	}

	// Component / custom resources with no aws: prefix → treat as module.
	return "", "module"
}

// nameFromURN extracts the NAME segment (last segment) from a Pulumi URN.
// URN format: urn:pulumi:STACK::PROJECT::TYPE::NAME
func nameFromURN(urn string) string {
	parts := strings.Split(urn, "::")
	if len(parts) >= 4 {
		return parts[len(parts)-1]
	}
	return urn
}

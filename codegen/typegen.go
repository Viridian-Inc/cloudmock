package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// GoType converts a botocore shape reference to a Go type string.
func GoType(model *BotocoreModel, shapeName string) string {
	shape, ok := model.Shapes[shapeName]
	if !ok {
		return "any"
	}
	switch shape.Type {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "long":
		return "int64"
	case "double", "float":
		return "float64"
	case "boolean":
		return "bool"
	case "timestamp":
		return "time.Time"
	case "blob":
		return "[]byte"
	case "list":
		if shape.Member != nil {
			return "[]" + GoType(model, shape.Member.Shape)
		}
		return "[]any"
	case "map":
		valType := "any"
		if shape.Value != nil {
			valType = GoType(model, shape.Value.Shape)
		}
		return "map[string]" + valType
	case "structure":
		return exportedName(shapeName)
	default:
		return "any"
	}
}

// GoJSONTag returns the JSON tag for a struct field.
func GoJSONTag(memberName string, member ShapeMember) string {
	wireName := memberName
	if member.LocationName != "" {
		wireName = member.LocationName
	}
	return fmt.Sprintf("`json:\"%s,omitempty\"`", wireName)
}

// GenerateStructs generates Go struct definitions for all structure shapes
// referenced by the given operations.
func GenerateStructs(model *BotocoreModel, ops map[string]Operation) string {
	// Collect all shape names referenced by operation inputs/outputs
	needed := make(map[string]bool)
	for _, op := range ops {
		if op.Input != nil {
			collectShapes(model, op.Input.Shape, needed)
		}
		if op.Output != nil {
			collectShapes(model, op.Output.Shape, needed)
		}
	}

	// Sort for deterministic output
	names := make([]string, 0, len(needed))
	for name := range needed {
		if model.Shapes[name].Type == "structure" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var sb strings.Builder
	for _, name := range names {
		shape := model.Shapes[name]
		sb.WriteString(fmt.Sprintf("type %s struct {\n", exportedName(name)))

		// Sort members for deterministic output
		memberNames := make([]string, 0, len(shape.Members))
		for mn := range shape.Members {
			memberNames = append(memberNames, mn)
		}
		sort.Strings(memberNames)

		for _, mn := range memberNames {
			member := shape.Members[mn]
			goType := GoType(model, member.Shape)
			// Use pointer for optional non-slice/map types
			isRequired := false
			for _, req := range shape.Required {
				if req == mn {
					isRequired = true
					break
				}
			}
			typePrefix := ""
			if !isRequired && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map[") && goType != "bool" && goType != "int" && goType != "int64" && goType != "float64" {
				typePrefix = "*"
			}
			tag := GoJSONTag(mn, member)
			sb.WriteString(fmt.Sprintf("\t%s %s%s %s\n", exportedName(mn), typePrefix, goType, tag))
		}
		sb.WriteString("}\n\n")
	}
	return sb.String()
}

// collectShapes recursively collects all shape names referenced from a root shape.
func collectShapes(model *BotocoreModel, shapeName string, seen map[string]bool) {
	if seen[shapeName] {
		return
	}
	seen[shapeName] = true

	shape, ok := model.Shapes[shapeName]
	if !ok {
		return
	}

	switch shape.Type {
	case "structure":
		for _, member := range shape.Members {
			collectShapes(model, member.Shape, seen)
		}
	case "list":
		if shape.Member != nil {
			collectShapes(model, shape.Member.Shape, seen)
		}
	case "map":
		if shape.Key != nil {
			collectShapes(model, shape.Key.Shape, seen)
		}
		if shape.Value != nil {
			collectShapes(model, shape.Value.Shape, seen)
		}
	}
}

// reservedNames are Go type names that conflict with our service scaffold.
var reservedNames = map[string]bool{
	"Service": true, "Store": true, "New": true,
}

// exportedName converts a botocore name to a Go exported name.
// Avoids conflicts with reserved scaffold names by adding an underscore suffix.
func exportedName(name string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])
	result := string(runes)
	if reservedNames[result] {
		result = result + "Model"
	}
	return result
}

// goPackageName converts a service name to a valid Go package name.
// e.g., "guardduty" -> "guardduty", "security-hub" -> "securityhub"
func goPackageName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "-", ""))
}

// iamActionPrefix returns the IAM action prefix for a service.
// e.g., "elasticfilesystem" -> "elasticfilesystem"
func iamActionPrefix(model *BotocoreModel) string {
	return model.SigningName()
}

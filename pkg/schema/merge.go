package schema

import "sort"

// Registry holds all resource schemas, keyed by TerraformType.
type Registry struct {
	schemas map[string]*ResourceSchema
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		schemas: make(map[string]*ResourceSchema),
	}
}

// Add inserts one or more schemas into the registry.
// If a schema with the same TerraformType already exists, it is overwritten.
func (r *Registry) Add(schemas ...ResourceSchema) {
	for i := range schemas {
		s := schemas[i]
		r.schemas[s.TerraformType] = &s
	}
}

// Get returns a schema by TerraformType.
func (r *Registry) Get(terraformType string) (*ResourceSchema, bool) {
	s, ok := r.schemas[terraformType]
	return s, ok
}

// All returns every schema in the registry, sorted by TerraformType.
func (r *Registry) All() []ResourceSchema {
	result := make([]ResourceSchema, 0, len(r.schemas))
	for _, s := range r.schemas {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TerraformType < result[j].TerraformType
	})
	return result
}

// ByService returns all schemas for a given service name, sorted by TerraformType.
func (r *Registry) ByService(serviceName string) []ResourceSchema {
	var result []ResourceSchema
	for _, s := range r.schemas {
		if s.ServiceName == serviceName {
			result = append(result, *s)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TerraformType < result[j].TerraformType
	})
	return result
}

// Len returns the number of schemas in the registry.
func (r *Registry) Len() int {
	return len(r.schemas)
}

// BuildRegistry combines Tier 1 and Tier 2 schemas into a single Registry.
// If both tiers contain a schema for the same TerraformType, the Tier 1
// (hand-crafted) schema wins.
func BuildRegistry(tier1 []ResourceSchema, tier2 []ResourceSchema) *Registry {
	r := NewRegistry()
	// Add Tier 2 first so Tier 1 can overwrite.
	r.Add(tier2...)
	r.Add(tier1...)
	return r
}

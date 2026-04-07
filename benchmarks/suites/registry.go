package suites

import (
	"sort"

	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type Registry struct {
	suites map[string]harness.Suite
}

func NewRegistry() *Registry {
	return &Registry{suites: make(map[string]harness.Suite)}
}

func (r *Registry) Register(s harness.Suite) {
	r.suites[s.Name()] = s
}

func (r *Registry) Get(name string) (harness.Suite, bool) {
	s, ok := r.suites[name]
	return s, ok
}

func (r *Registry) List() []harness.Suite {
	var result []harness.Suite
	for _, s := range r.suites {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

func (r *Registry) FilterByTier(tier int) []harness.Suite {
	var result []harness.Suite
	for _, s := range r.suites {
		if s.Tier() == tier {
			result = append(result, s)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

package memory

import (
	"context"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/gateway"
)

// SLOStore wraps gateway.SLOEngine to satisfy the dataplane.SLOStore interface
// for local (in-memory) mode.
type SLOStore struct {
	engine *gateway.SLOEngine
}

// NewSLOStore creates an SLOStore wrapping the given gateway.SLOEngine.
func NewSLOStore(engine *gateway.SLOEngine) *SLOStore {
	return &SLOStore{engine: engine}
}

// Rules returns the configured SLO rules.
func (s *SLOStore) Rules(_ context.Context) ([]config.SLORule, error) {
	return s.engine.Rules(), nil
}

// SetRules updates the SLO rules.
func (s *SLOStore) SetRules(_ context.Context, rules []config.SLORule) error {
	s.engine.SetRules(rules)
	return nil
}

// Status returns the current SLO status.
func (s *SLOStore) Status(_ context.Context) (*dataplane.SLOStatus, error) {
	gs := s.engine.Status()

	windows := make([]dataplane.SLOWindowStatus, len(gs.Windows))
	for i, w := range gs.Windows {
		windows[i] = dataplane.SLOWindowStatus{
			Service:    w.Service,
			Action:     w.Action,
			Total:      w.Total,
			Violations: w.Violations,
			Errors:     w.Errors,
			ErrorRate:  w.ErrorRate,
			BudgetUsed: w.BudgetUsed,
			BurnRate:   w.BurnRate,
			P50Ms:      w.P50Ms,
			P95Ms:      w.P95Ms,
			P99Ms:      w.P99Ms,
			Breaching:  w.Breaching,
		}
	}

	alerts := make([]dataplane.SLOAlert, len(gs.Alerts))
	for i, a := range gs.Alerts {
		alerts[i] = dataplane.SLOAlert{
			Service: a.Service,
			Action:  a.Action,
			Message: a.Message,
		}
	}

	return &dataplane.SLOStatus{
		Windows: windows,
		Healthy: gs.Healthy,
		Alerts:  alerts,
	}, nil
}

// History returns SLO rule change history. In-memory mode has no history.
func (s *SLOStore) History(_ context.Context, _ int) ([]dataplane.SLORuleChange, error) {
	return nil, nil
}

// Compile-time interface check.
var _ dataplane.SLOStore = (*SLOStore)(nil)

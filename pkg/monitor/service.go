package monitor

import (
	"context"
	"time"
)

// Service owns the monitor and alert stores, the evaluator loop, and provides
// the high-level API used by the admin handlers.
type Service struct {
	monitors  MonitorStore
	alerts    AlertStore
	evaluator *Evaluator
}

// NewService creates a monitoring service. Pass nil for provider to skip
// automatic evaluation (useful for testing or when metrics are not available).
func NewService(
	monitors MonitorStore,
	alerts AlertStore,
	provider MetricsProvider,
	interval time.Duration,
) *Service {
	var eval *Evaluator
	if provider != nil {
		eval = NewEvaluator(monitors, alerts, provider, interval)
	}
	return &Service{
		monitors:  monitors,
		alerts:    alerts,
		evaluator: eval,
	}
}

// Monitors returns the underlying monitor store.
func (s *Service) Monitors() MonitorStore { return s.monitors }

// Alerts returns the underlying alert store.
func (s *Service) Alerts() AlertStore { return s.alerts }

// Evaluator returns the evaluator, which may be nil if no provider was given.
func (s *Service) Evaluator() *Evaluator { return s.evaluator }

// SetIncidentCreator wires the incident integration into the evaluator.
func (s *Service) SetIncidentCreator(ic IncidentCreator) {
	if s.evaluator != nil {
		s.evaluator.SetIncidentCreator(ic)
	}
}

// SetWebhookDispatcher wires the webhook integration into the evaluator.
func (s *Service) SetWebhookDispatcher(wd WebhookDispatcher) {
	if s.evaluator != nil {
		s.evaluator.SetWebhookDispatcher(wd)
	}
}

// Start begins the background evaluation loop. No-op if no evaluator is configured.
func (s *Service) Start(ctx context.Context) {
	if s.evaluator != nil {
		s.evaluator.Start(ctx)
	}
}

// Stop terminates the background evaluation loop.
func (s *Service) Stop() {
	if s.evaluator != nil {
		s.evaluator.Stop()
	}
}

package sdk

import "github.com/Viridian-Inc/cloudmock/pkg/gateway"

// ChaosOption configures a fault injection rule.
type ChaosOption func(*chaosOpts)

type chaosOpts struct {
	statusCode int
	message    string
	latencyMs  int
	percentage int
}

// WithStatusCode sets the HTTP status code returned for "error" faults.
func WithStatusCode(code int) ChaosOption { return func(o *chaosOpts) { o.statusCode = code } }

// WithMessage sets the error message returned with the fault.
func WithMessage(msg string) ChaosOption { return func(o *chaosOpts) { o.message = msg } }

// WithLatency sets the added latency in milliseconds for "latency" faults.
func WithLatency(ms int) ChaosOption { return func(o *chaosOpts) { o.latencyMs = ms } }

// WithPercentage sets the probability (0-100) that the fault fires on any given request.
func WithPercentage(pct int) ChaosOption { return func(o *chaosOpts) { o.percentage = pct } }

// InjectFault adds a fault injection rule to the in-process chaos engine.
// faultType is one of "error", "latency", "timeout", "blackhole", or "throttle".
func (cm *CloudMock) InjectFault(service, action, faultType string, opts ...ChaosOption) error {
	o := &chaosOpts{statusCode: 500, percentage: 100}
	for _, opt := range opts {
		opt(o)
	}
	cm.chaosEngine.AddRule(gateway.ChaosRule{
		Service:    service,
		Action:     action,
		Enabled:    true,
		Type:       faultType,
		ErrorCode:  o.statusCode,
		ErrorMsg:   o.message,
		LatencyMs:  o.latencyMs,
		Percentage: o.percentage,
	})
	return nil
}

// ClearFaults disables all chaos rules on the in-process engine.
func (cm *CloudMock) ClearFaults() error {
	cm.chaosEngine.DisableAll()
	return nil
}

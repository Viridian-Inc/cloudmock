package regression

import "context"

type MetricSource interface {
	WindowMetrics(ctx context.Context, service, action string, window TimeWindow) (*WindowMetrics, error)
	TenantWindowMetrics(ctx context.Context, service, tenantID string, window TimeWindow) (*WindowMetrics, error)
	FleetWindowMetrics(ctx context.Context, service string, window TimeWindow) (*WindowMetrics, error)
	ListServices(ctx context.Context) ([]string, error)
	ListTenants(ctx context.Context, service string) ([]string, error)
}

package target

import "context"

type Target interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Endpoint() string
	ResourceStats(ctx context.Context) (*Stats, error)
}

type Stats struct {
	MemoryMB float64
	CPUPct   float64
}

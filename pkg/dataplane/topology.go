package dataplane

import "context"

type ObservedEdge struct {
	Source       string
	Target       string
	EdgeType     string
	RequestCount int64
}

type TopologyNode struct {
	Name        string
	ServiceType string
	Group       string
}

type TopologyGraph struct {
	Nodes []TopologyNode
	Edges []ObservedEdge
}

type TopologyStore interface {
	GetTopology(ctx context.Context) (*TopologyGraph, error)
	RecordEdge(ctx context.Context, edge ObservedEdge) error
	Upstream(ctx context.Context, service string) ([]string, error)
	Downstream(ctx context.Context, service string) ([]string, error)
}

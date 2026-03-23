package dataplane

import "errors"

var ErrNotFound = errors.New("not found")

type DataPlane struct {
	Traces   TraceReader
	TraceW   TraceWriter
	Requests RequestReader
	RequestW RequestWriter
	Metrics  MetricReader
	MetricW  MetricWriter
	SLO      SLOStore
	Config   ConfigStore
	Topology TopologyStore
	Mode     string
}

package gateway

import (
	"github.com/Viridian-Inc/cloudmock/pkg/edge"
)

// CertPair lives in pkg/edge — this alias preserves the previous gateway.*
// name for any importer that still references it.
type CertPair = edge.CertPair

// EnsureCerts forwards to pkg/edge.
func EnsureCerts(domains ...string) (*CertPair, error) {
	return edge.EnsureCerts(domains...)
}

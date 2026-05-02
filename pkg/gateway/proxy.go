package gateway

import (
	"github.com/Viridian-Inc/cloudmock/pkg/edge"
)

// L7 reverse-proxy types and functions live in pkg/edge — these aliases /
// forwarders preserve the previous gateway.* API for cmd/gateway/main.go and
// any external importers (e.g. autotend tooling) that still reference them by
// their gateway-package names.
type (
	ProxyRoute   = edge.ProxyRoute
	ProxyServer  = edge.ProxyServer
	ProxyOpts    = edge.ProxyOpts
	ServicePorts = edge.ServicePorts
)

// DefaultServicePorts forwards to pkg/edge.
func DefaultServicePorts() ServicePorts {
	return edge.DefaultServicePorts()
}

// BuildRoutes forwards to pkg/edge.
func BuildRoutes(primaryDomain, cloudmockDomain string) []ProxyRoute {
	return edge.BuildRoutes(primaryDomain, cloudmockDomain)
}

// BuildRoutesWithPorts forwards to pkg/edge.
func BuildRoutesWithPorts(primaryDomain, cloudmockDomain string, p ServicePorts) []ProxyRoute {
	return edge.BuildRoutesWithPorts(primaryDomain, cloudmockDomain, p)
}

// NewProxyServer forwards to pkg/edge.
func NewProxyServer(routes []ProxyRoute) *ProxyServer {
	return edge.NewProxyServer(routes)
}

// NewProxyServerWithOpts forwards to pkg/edge.
func NewProxyServerWithOpts(routes []ProxyRoute, opts ProxyOpts) *ProxyServer {
	return edge.NewProxyServerWithOpts(routes, opts)
}

// StartProxy forwards to pkg/edge.
func StartProxy(routes []ProxyRoute, tlsCert *CertPair) {
	edge.StartProxy(routes, tlsCert)
}

// StartProxyWithOpts forwards to pkg/edge.
func StartProxyWithOpts(routes []ProxyRoute, tlsCert *CertPair, opts ProxyOpts) {
	edge.StartProxyWithOpts(routes, tlsCert, opts)
}

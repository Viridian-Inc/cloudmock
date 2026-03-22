package gateway

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// ProxyRoute defines a single routing rule for the reverse proxy.
type ProxyRoute struct {
	Host    string // e.g. "bff.local.autotend.io"
	Path    string // path prefix, e.g. "/bff/" — empty means match all
	Backend string // e.g. "http://localhost:3202"
}

// ProxyServer is a virtual-host reverse proxy that routes requests
// to backend services based on Host header and path prefix.
type ProxyServer struct {
	routes []ProxyRoute
	mux    http.Handler
}

// DefaultAutotendRoutes returns the standard routing table for autotend
// local development. Order matters — more specific paths must come first.
func DefaultAutotendRoutes() []ProxyRoute {
	return []ProxyRoute{
		// Subdomain routes (match entire host)
		{Host: "bff.local.autotend.io", Path: "", Backend: "http://localhost:3202"},
		{Host: "api.local.autotend.io", Path: "", Backend: "http://localhost:4566"},
		{Host: "auth.local.autotend.io", Path: "", Backend: "http://localhost:4566"},
		{Host: "dashboard.local.autotend.io", Path: "", Backend: "http://localhost:4500"},
		{Host: "admin.local.autotend.io", Path: "", Backend: "http://localhost:4599"},

		// Path-based routes on the main domain (order: most specific first)
		{Host: "local.autotend.io", Path: "/bff/", Backend: "http://localhost:3202"},
		{Host: "local.autotend.io", Path: "/v1/", Backend: "http://localhost:3202"},
		{Host: "local.autotend.io", Path: "/health", Backend: "http://localhost:3202"},
		{Host: "local.autotend.io", Path: "/graphql", Backend: "http://localhost:4000"},
		{Host: "local.autotend.io", Path: "/_cloudmock/", Backend: "http://localhost:4566"},

		// Default: dashboard
		{Host: "local.autotend.io", Path: "", Backend: "http://localhost:4500"},
	}
}

// NewProxyServer creates a new reverse proxy server with the given routes.
func NewProxyServer(routes []ProxyRoute) *ProxyServer {
	ps := &ProxyServer{routes: routes}
	ps.mux = ps.buildHandler()
	return ps
}

func (ps *ProxyServer) buildHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip port from host header for matching
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		for _, route := range ps.routes {
			if !strings.EqualFold(host, route.Host) {
				continue
			}
			if route.Path != "" && !strings.HasPrefix(r.URL.Path, route.Path) {
				continue
			}
			ps.proxyTo(route.Backend, w, r)
			return
		}

		http.Error(w, "no route matched", http.StatusNotFound)
	})
}

func (ps *ProxyServer) proxyTo(backend string, w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(backend)
	if err != nil {
		http.Error(w, "bad backend URL", http.StatusInternalServerError)
		return
	}

	// WebSocket upgrade detection
	if isWebSocketUpgrade(r) {
		proxyWebSocket(target, w, r)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
			// Preserve original path
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			addCORSHeaders(resp, r)
			return nil
		},
	}

	proxy.ServeHTTP(w, r)
}

// addCORSHeaders adds CORS headers to proxied responses.
func addCORSHeaders(resp *http.Response, req *http.Request) {
	origin := req.Header.Get("Origin")
	if origin != "" {
		resp.Header.Set("Access-Control-Allow-Origin", origin)
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH")
		resp.Header.Set("Access-Control-Allow-Headers",
			"Content-Type, Authorization, X-Amz-Target, X-Amz-Date, "+
				"X-Amz-Security-Token, X-Amz-Content-Sha256, X-Amz-User-Agent, "+
				"x-api-key, amz-sdk-invocation-id, amz-sdk-request")
		resp.Header.Set("Access-Control-Expose-Headers",
			"x-amzn-RequestId, x-amz-request-id, x-amz-id-2, ETag, x-amz-version-id")
		resp.Header.Set("Access-Control-Max-Age", "86400")
		resp.Header.Set("Access-Control-Allow-Credentials", "true")
	}
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Connection"), "upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func proxyWebSocket(target *url.URL, w http.ResponseWriter, r *http.Request) {
	// For WebSocket, we use a standard reverse proxy which handles upgrades
	// via the Hijacker interface. Go's httputil.ReverseProxy supports this
	// natively in Go 1.12+.
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
		},
	}
	proxy.ServeHTTP(w, r)
}

// ServeHTTP implements http.Handler.
func (ps *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers",
				"Content-Type, Authorization, X-Amz-Target, X-Amz-Date, "+
					"X-Amz-Security-Token, X-Amz-Content-Sha256, X-Amz-User-Agent, "+
					"x-api-key, amz-sdk-invocation-id, amz-sdk-request")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	ps.mux.ServeHTTP(w, r)
}

// StartProxy starts the reverse proxy on HTTP and optionally HTTPS.
// It tries port 80 first, falling back to 8080 if unavailable.
// If tlsCertFile and tlsKeyFile are provided, it also starts HTTPS on 443 (fallback 8443).
func StartProxy(routes []ProxyRoute, tlsCert *CertPair) {
	proxy := NewProxyServer(routes)

	// Start HTTP
	go func() {
		addr := ":80"
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("proxy: port 80 unavailable (%v), falling back to :8080", err)
			addr = ":8080"
			ln, err = net.Listen("tcp", addr)
			if err != nil {
				log.Printf("proxy: failed to listen on %s: %v", addr, err)
				return
			}
		}
		log.Printf("proxy: listening on http://local.autotend.io%s", addr)
		if err := http.Serve(ln, proxy); err != nil {
			log.Printf("proxy HTTP exited: %v", err)
		}
	}()

	// Start HTTPS if certs are available
	if tlsCert != nil {
		go func() {
			tlsConfig := tlsCert.TLSConfig()
			addr := ":443"
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				log.Printf("proxy: port 443 unavailable (%v), falling back to :8443", err)
				addr = ":8443"
				ln, err = net.Listen("tcp", addr)
				if err != nil {
					log.Printf("proxy: failed to listen on %s: %v", addr, err)
					return
				}
			}
			tlsLn := tls.NewListener(ln, tlsConfig)
			log.Printf("proxy: listening on https://local.autotend.io%s", addr)
			if err := http.Serve(tlsLn, proxy); err != nil {
				log.Printf("proxy HTTPS exited: %v", err)
			}
		}()
	}
}

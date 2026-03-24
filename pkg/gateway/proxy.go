package gateway

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ProxyRoute defines a single routing rule for the reverse proxy.
type ProxyRoute struct {
	Host         string // e.g. "bff.localhost" or "bff.localhost.example.com"
	Path         string // path prefix, e.g. "/bff/" — empty means match all
	Backend      string // e.g. "http://localhost:3202"
	PreserveHost bool   // if true, forward original Host header to backend
}

// ProxyServer is a virtual-host reverse proxy that routes requests
// to backend services based on Host header and path prefix.
type ProxyServer struct {
	routes      []ProxyRoute
	mux         http.Handler
	requestLog  *RequestLog
	stats       *RequestStats
	broadcaster RequestBroadcaster
}

// ServicePorts maps logical service names to their listen ports.
// These are read from cloudmock config and environment variables.
type ServicePorts struct {
	Gateway   int // cloudmock AWS API (default 4566)
	Dashboard int // cloudmock dashboard (default 4500)
	Admin     int // admin API (default 4599)
	App       int // Expo/Metro app (default 8081)
	BFF       int // BFF service (default 3202)
	GraphQL   int // GraphQL server (default 4000)
}

// DefaultServicePorts returns ports from environment or sensible defaults.
func DefaultServicePorts() ServicePorts {
	return ServicePorts{
		Gateway:   envInt("CLOUDMOCK_PORT", 4566),
		Dashboard: envInt("CLOUDMOCK_DASHBOARD_PORT", 4500),
		Admin:     envInt("CLOUDMOCK_ADMIN_PORT", 4599),
		App:       envInt("CLOUDMOCK_APP_PORT", 8081),
		BFF:       envInt("CLOUDMOCK_BFF_PORT", 3202),
		GraphQL:   envInt("CLOUDMOCK_GRAPHQL_PORT", 4000),
	}
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func backend(port int) string {
	return fmt.Sprintf("http://localhost:%d", port)
}

// BuildRoutes generates the routing table dynamically from domain names and port config.
// Order matters — more specific paths must come first.
func BuildRoutes(autotendDomain, cloudmockDomain string) []ProxyRoute {
	return BuildRoutesWithPorts(autotendDomain, cloudmockDomain, DefaultServicePorts())
}

// BuildRoutesWithPorts generates routes using explicit port configuration.
func BuildRoutesWithPorts(autotendDomain, cloudmockDomain string, p ServicePorts) []ProxyRoute {
	at := "localhost." + autotendDomain
	cm := "localhost." + cloudmockDomain

	return []ProxyRoute{
		// .localhost domains (RFC 6761, zero config)
		{Host: "autotend-app.localhost", Path: "/", Backend: backend(p.App), PreserveHost: true},
		{Host: "cloudmock.localhost", Path: "/_cloudmock/", Backend: backend(p.Gateway)},
		{Host: "cloudmock.localhost", Path: "/api/", Backend: backend(p.Admin)},
		{Host: "cloudmock.localhost", Path: "/", Backend: backend(p.Dashboard)},
		{Host: "bff.localhost", Path: "/", Backend: backend(p.BFF)},
		{Host: "api.localhost", Path: "/", Backend: backend(p.Gateway)},
		{Host: "auth.localhost", Path: "/", Backend: backend(p.Gateway)},
		{Host: "admin.localhost", Path: "/", Backend: backend(p.Admin)},
		{Host: "graphql.localhost", Path: "/", Backend: backend(p.GraphQL)},

		// custom domain: autotend app services
		{Host: "autotend-app." + at, Path: "/", Backend: backend(p.App), PreserveHost: true},
		{Host: "bff." + at, Path: "", Backend: backend(p.BFF)},
		{Host: "api." + at, Path: "", Backend: backend(p.Gateway)},
		{Host: "auth." + at, Path: "", Backend: backend(p.Gateway)},
		{Host: "admin." + at, Path: "", Backend: backend(p.Admin)},
		{Host: "graphql." + at, Path: "", Backend: backend(p.GraphQL)},
		{Host: at, Path: "/", Backend: backend(p.App), PreserveHost: true},

		// custom domain: cloudmock dashboard
		{Host: cm, Path: "/_cloudmock/", Backend: backend(p.Gateway)},
		{Host: cm, Path: "/api/", Backend: backend(p.Admin)},
		{Host: cm, Path: "/", Backend: backend(p.Dashboard)},
	}
}

// ProxyOpts configures the proxy server.
type ProxyOpts struct {
	RequestLog  *RequestLog
	Stats       *RequestStats
	Broadcaster RequestBroadcaster
}

// NewProxyServer creates a new reverse proxy server with the given routes.
func NewProxyServer(routes []ProxyRoute) *ProxyServer {
	return NewProxyServerWithOpts(routes, ProxyOpts{})
}

// NewProxyServerWithOpts creates a proxy server with logging and broadcasting.
func NewProxyServerWithOpts(routes []ProxyRoute, opts ProxyOpts) *ProxyServer {
	ps := &ProxyServer{
		routes:      routes,
		requestLog:  opts.RequestLog,
		stats:       opts.Stats,
		broadcaster: opts.Broadcaster,
	}
	ps.mux = ps.buildHandler()
	return ps
}

func (ps *ProxyServer) buildHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

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

			// Wrap response writer to capture status code
			rec := &statusRecorder{ResponseWriter: w, statusCode: 200}
			ps.proxyToWithOpts(route.Backend, rec, r, route.PreserveHost)

			// Log the proxied request
			ps.logProxyRequest(r, host, route, rec.statusCode, time.Since(start))
			return
		}

		http.Error(w, "no route matched", http.StatusNotFound)
	})
}

// logProxyRequest records a proxied request in the request log and broadcasts it.
// Skips cloudmock's own internal traffic (dashboard, admin API) to avoid log pollution.
func (ps *ProxyServer) logProxyRequest(r *http.Request, host string, route ProxyRoute, status int, latency time.Duration) {
	if ps.requestLog == nil {
		return
	}

	// Skip cloudmock's own traffic — dashboard, admin API, and gateway requests
	// are internal observability traffic, not application requests to debug.
	// Gateway requests are already logged by the gateway's own middleware.
	if strings.Contains(host, "cloudmock") || strings.Contains(host, "admin") {
		return
	}
	// Skip routes that proxy to cloudmock itself (dashboard, admin, gateway)
	ports := DefaultServicePorts()
	for _, skipPort := range []int{ports.Gateway, ports.Dashboard, ports.Admin} {
		if strings.Contains(route.Backend, fmt.Sprintf(":%d", skipPort)) {
			return
		}
	}
	// Skip noise: HEAD requests, bare "/" pings, health checks, favicon, HMR
	if r.Method == "HEAD" || r.URL.Path == "/" || r.URL.Path == "/favicon.ico" ||
		strings.HasPrefix(r.URL.Path, "/__") || strings.HasPrefix(r.URL.Path, "/hot") ||
		strings.HasPrefix(r.URL.Path, "/node_modules") || strings.HasPrefix(r.URL.Path, "/assets") {
		return
	}

	// Determine service name from the route host
	service := "proxy"
	if strings.Contains(host, "bff") {
		service = "bff"
	} else if strings.Contains(host, "graphql") {
		service = "graphql"
	} else if strings.Contains(host, "auth") {
		service = "cognito-idp"
	} else if strings.Contains(host, "api.") {
		service = "gateway"
	} else if strings.Contains(host, "app") || host == route.Host {
		service = "app"
	}

	latencyMs := float64(latency.Nanoseconds()) / 1e6

	entry := RequestEntry{
		ID:         GenerateTraceID(),
		Timestamp:  time.Now(),
		Service:    service,
		Action:     r.Method + " " + r.URL.Path,
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: status,
		Latency:    latency,
		LatencyMs:  latencyMs,
		Level:      "app", // user-facing requests through the proxy
		TraceID:    r.Header.Get("X-Cloudmock-Trace-Id"),
	}

	ps.requestLog.Add(entry)
	if ps.stats != nil {
		ps.stats.Increment(service)
	}
	if ps.broadcaster != nil {
		ps.broadcaster.Broadcast("request", entry)
	}
}

// statusRecorder wraps ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.written {
		r.statusCode = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (ps *ProxyServer) proxyTo(backend string, w http.ResponseWriter, r *http.Request) {
	ps.proxyToWithOpts(backend, w, r, false)
}

// proxyToPreserveHost proxies the request but preserves the original Host
// header. Use this for dev servers (like Metro/Expo) that embed the Host
// in asset URLs — the browser needs those URLs to point back through
// the proxy, not to the backend's internal address.
func (ps *ProxyServer) proxyToPreserveHost(backend string, w http.ResponseWriter, r *http.Request) {
	ps.proxyToWithOpts(backend, w, r, true)
}

func (ps *ProxyServer) proxyToWithOpts(backend string, w http.ResponseWriter, r *http.Request, preserveHost bool) {
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

	originalHost := r.Host
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			if preserveHost {
				req.Host = originalHost
			} else {
				req.Host = target.Host
			}
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			addCORSHeaders(resp, r)
			// Rewrite backend URLs in response bodies for PreserveHost routes.
			// Metro/Expo embeds http://localhost:8081 in JS bundles; the browser
			// on https://proxy.domain blocks these as mixed content.
			if preserveHost {
				rewriteResponseBody(resp, target, r)
			}
			return nil
		},
	}

	proxy.ServeHTTP(w, r)
}

// rewriteResponseBody replaces backend origin URLs with the proxy's origin
// in text responses. This fixes mixed-content issues where Metro embeds
// http://localhost:8081 in JS bundles but the browser is on https://proxy.domain.
func rewriteResponseBody(resp *http.Response, target *url.URL, req *http.Request) {
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "javascript") && !strings.Contains(ct, "json") && !strings.Contains(ct, "html") && !strings.Contains(ct, "text/") {
		return
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return
	}

	// Determine the proxy's public origin from the original request
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	proxyOrigin := scheme + "://" + req.Host
	backendOrigin := target.Scheme + "://" + target.Host

	if bytes.Contains(body, []byte(backendOrigin)) {
		body = bytes.ReplaceAll(body, []byte(backendOrigin), []byte(proxyOrigin))
		resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
		resp.ContentLength = int64(len(body))
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
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
	StartProxyWithOpts(routes, tlsCert, ProxyOpts{})
}

// StartProxyWithOpts starts the proxy with request logging.
func StartProxyWithOpts(routes []ProxyRoute, tlsCert *CertPair, opts ProxyOpts) {
	proxy := NewProxyServerWithOpts(routes, opts)

	// Start HTTP
	go func() {
		addr := ":80"
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			slog.Warn("proxy: port 80 unavailable, falling back to :8080", "error", err)
			addr = ":8080"
			ln, err = net.Listen("tcp", addr)
			if err != nil {
				slog.Error("proxy: failed to listen", "addr", addr, "error", err)
				return
			}
		}
		slog.Info("proxy HTTP listening", "addr", addr)
		if err := http.Serve(ln, proxy); err != nil {
			slog.Error("proxy HTTP exited", "error", err)
		}
	}()

	// Start HTTPS if certs are available
	if tlsCert != nil {
		go func() {
			tlsConfig := tlsCert.TLSConfig()
			addr := ":443"
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				slog.Warn("proxy: port 443 unavailable, falling back to :8443", "error", err)
				addr = ":8443"
				ln, err = net.Listen("tcp", addr)
				if err != nil {
					slog.Error("proxy: failed to listen", "addr", addr, "error", err)
					return
				}
			}
			tlsLn := tls.NewListener(ln, tlsConfig)
			slog.Info("proxy HTTPS listening", "addr", addr)
			if err := http.Serve(tlsLn, proxy); err != nil {
				slog.Error("proxy HTTPS exited", "error", err)
			}
		}()
	}
}

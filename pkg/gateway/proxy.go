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
	Host         string // e.g. "bff.localhost" or "bff.localhost.example.com"
	Path         string // path prefix, e.g. "/bff/" — empty means match all
	Backend      string // e.g. "http://localhost:3202"
	PreserveHost bool   // if true, forward original Host header to backend
}

// ProxyServer is a virtual-host reverse proxy that routes requests
// to backend services based on Host header and path prefix.
type ProxyServer struct {
	routes []ProxyRoute
	mux    http.Handler
}

// BuildRoutes generates the routing table dynamically from the provided domain names.
// Order matters — more specific paths must come first.
func BuildRoutes(autotendDomain, cloudmockDomain string) []ProxyRoute {
	at := "localhost." + autotendDomain
	cm := "localhost." + cloudmockDomain

	return []ProxyRoute{
		// .localhost domains (RFC 6761, zero config)
		// Expo app: PreserveHost so Metro serves assets with correct origin
		{Host: "autotend-app.localhost", Path: "/", Backend: "http://localhost:8081", PreserveHost: true},
		{Host: "cloudmock.localhost", Path: "/_cloudmock/", Backend: "http://localhost:4566"},
		{Host: "cloudmock.localhost", Path: "/api/", Backend: "http://localhost:4599"},
		{Host: "cloudmock.localhost", Path: "/", Backend: "http://localhost:4500"},
		{Host: "bff.localhost", Path: "/", Backend: "http://localhost:3202"},
		{Host: "api.localhost", Path: "/", Backend: "http://localhost:4566"},
		{Host: "auth.localhost", Path: "/", Backend: "http://localhost:4566"},
		{Host: "admin.localhost", Path: "/", Backend: "http://localhost:4599"},
		{Host: "graphql.localhost", Path: "/", Backend: "http://localhost:4000"},

		// custom domain: autotend app services
		{Host: "autotend-app." + at, Path: "/", Backend: "http://localhost:8081", PreserveHost: true},
		{Host: "bff." + at, Path: "", Backend: "http://localhost:3202"},
		{Host: "api." + at, Path: "", Backend: "http://localhost:4566"},
		{Host: "auth." + at, Path: "", Backend: "http://localhost:4566"},
		{Host: "admin." + at, Path: "", Backend: "http://localhost:4599"},
		{Host: "graphql." + at, Path: "", Backend: "http://localhost:4000"},
		{Host: at, Path: "/", Backend: "http://localhost:8081", PreserveHost: true},

		// custom domain: cloudmock dashboard
		{Host: cm, Path: "/_cloudmock/", Backend: "http://localhost:4566"},
		{Host: cm, Path: "/api/", Backend: "http://localhost:4599"},
		{Host: cm, Path: "/", Backend: "http://localhost:4500"},
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
			ps.proxyToWithOpts(route.Backend, w, r, route.PreserveHost)
			return
		}

		http.Error(w, "no route matched", http.StatusNotFound)
	})
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
		log.Printf("proxy HTTP%s: routing via .localhost and custom domains", addr)
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
			log.Printf("proxy HTTPS%s: same routes with TLS", addr)
			if err := http.Serve(tlsLn, proxy); err != nil {
				log.Printf("proxy HTTPS exited: %v", err)
			}
		}()
	}
}

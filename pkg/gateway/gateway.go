package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/config"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
)

// Gateway is the main HTTP handler that routes AWS API requests to service mocks.
type Gateway struct {
	cfg      *config.Config
	registry *routing.Registry
	mux      *http.ServeMux
	store    *iampkg.Store
	engine   *iampkg.Engine
}

// New creates a Gateway with routes pre-registered.
func New(cfg *config.Config, registry *routing.Registry) *Gateway {
	g := &Gateway{
		cfg:      cfg,
		registry: registry,
		mux:      http.NewServeMux(),
	}

	g.mux.HandleFunc("/_cloudmock/health", g.handleHealth)
	g.mux.HandleFunc("/_cloudmock/services", g.handleServices)
	g.mux.HandleFunc("/", g.handleAWSRequest)

	return g
}

// NewWithIAM creates a Gateway with IAM store and engine for authentication/authorization.
func NewWithIAM(cfg *config.Config, registry *routing.Registry, store *iampkg.Store, engine *iampkg.Engine) *Gateway {
	g := New(cfg, registry)
	g.store = store
	g.engine = engine
	return g
}

// ServeHTTP implements http.Handler.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (g *Gateway) handleServices(w http.ResponseWriter, r *http.Request) {
	svcs := g.registry.List()
	names := make([]string, 0, len(svcs))
	for _, svc := range svcs {
		names = append(names, svc.Name())
	}

	data, err := json.Marshal(names)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (g *Gateway) handleAWSRequest(w http.ResponseWriter, r *http.Request) {
	// 1. Detect the target service.
	svcName := routing.DetectService(r)
	if svcName == "" {
		awsErr := service.NewAWSError(
			"MissingAuthenticationToken",
			"No service could be detected from the request. Ensure a valid Authorization header is present.",
			http.StatusBadRequest,
		)
		_ = service.WriteErrorResponse(w, awsErr, service.FormatXML)
		return
	}

	// 2. Look up service in registry.
	svc, err := g.registry.Lookup(svcName)
	if err != nil {
		awsErr := service.NewAWSError(
			"ServiceUnavailable",
			fmt.Sprintf("Service %q is not registered in this cloudmock instance.", svcName),
			http.StatusServiceUnavailable,
		)
		_ = service.WriteErrorResponse(w, awsErr, service.FormatXML)
		return
	}

	// 3. Read request body.
	body, err := service.ParseRequestBody(r)
	if err != nil {
		awsErr := service.NewAWSError("InvalidRequest", "Failed to read request body.", http.StatusBadRequest)
		_ = service.WriteErrorResponse(w, awsErr, service.FormatXML)
		return
	}

	// 4. Build RequestContext.
	action := routing.DetectAction(r)
	params := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	ctx := &service.RequestContext{
		Action:     action,
		Region:     g.cfg.Region,
		AccountID:  g.cfg.AccountID,
		RawRequest: r,
		Body:       body,
		Params:     params,
		Service:    svcName,
	}

	// 5. Authenticate request.
	identity, authErr := g.authenticateRequest(r)
	if authErr != nil {
		_ = service.WriteErrorResponse(w, authErr, service.FormatJSON)
		return
	}
	ctx.Identity = identity

	// Authorize request.
	if g.engine != nil {
		iamAction := svcName + ":" + action
		if authzErr := g.authorizeRequest(identity, iamAction, "*"); authzErr != nil {
			_ = service.WriteErrorResponse(w, authzErr, service.FormatJSON)
			return
		}
	}

	// 6. Dispatch to service.
	resp, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		if awsErr, ok := svcErr.(*service.AWSError); ok {
			format := service.FormatXML
			if resp != nil {
				format = resp.Format
			}
			_ = service.WriteErrorResponse(w, awsErr, format)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// 7. Write success response.
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}

	// Raw body takes priority — write bytes directly without marshaling.
	if resp.RawBody != nil {
		if resp.RawContentType != "" {
			w.Header().Set("Content-Type", resp.RawContentType)
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(resp.RawBody)
		return
	}

	if resp.Body == nil {
		w.WriteHeader(resp.StatusCode)
		return
	}

	switch resp.Format {
	case service.FormatJSON:
		_ = service.WriteJSONResponse(w, resp.StatusCode, resp.Body)
	default:
		_ = service.WriteXMLResponse(w, resp.StatusCode, resp.Body)
	}
}

package route53

import (
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Route53Service is the cloudmock implementation of the AWS Route 53 DNS API.
type Route53Service struct {
	store *ZoneStore
}

// New returns a new Route53Service.
// Route 53 is global (no region), but accountID and region are accepted for
// compatibility with the standard constructor pattern.
func New(accountID, region string) *Route53Service {
	return &Route53Service{
		store: NewStore(),
	}
}

// Name returns the AWS service name used for routing.
func (s *Route53Service) Name() string { return "route53" }

// Actions returns the list of Route 53 API actions.
// Route 53 uses path-based routing rather than Action params or X-Amz-Target,
// so we describe actions without relying on those mechanisms.
func (s *Route53Service) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateHostedZone", Method: http.MethodPost, IAMAction: "route53:CreateHostedZone"},
		{Name: "ListHostedZones", Method: http.MethodGet, IAMAction: "route53:ListHostedZones"},
		{Name: "GetHostedZone", Method: http.MethodGet, IAMAction: "route53:GetHostedZone"},
		{Name: "DeleteHostedZone", Method: http.MethodDelete, IAMAction: "route53:DeleteHostedZone"},
		{Name: "ChangeResourceRecordSets", Method: http.MethodPost, IAMAction: "route53:ChangeResourceRecordSets"},
		{Name: "ListResourceRecordSets", Method: http.MethodGet, IAMAction: "route53:ListResourceRecordSets"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *Route53Service) HealthCheck() error { return nil }

// HandleRequest routes an incoming Route 53 request to the appropriate handler.
// Route 53 uses path-based REST routing; ctx.Action will be empty.
//
// Routes:
//
//	POST   /2013-04-01/hostedzone              → CreateHostedZone
//	GET    /2013-04-01/hostedzone              → ListHostedZones
//	GET    /2013-04-01/hostedzone/{id}         → GetHostedZone
//	DELETE /2013-04-01/hostedzone/{id}         → DeleteHostedZone
//	POST   /2013-04-01/hostedzone/{id}/rrset   → ChangeResourceRecordSets
//	GET    /2013-04-01/hostedzone/{id}/rrset   → ListResourceRecordSets
func (s *Route53Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := r.URL.Path

	// Normalise path: strip trailing slash.
	path = strings.TrimRight(path, "/")

	const basePrefix = "/2013-04-01/hostedzone"

	if !strings.HasPrefix(path, basePrefix) {
		return xmlErr(service.NewAWSError("NotImplemented",
			"Route not implemented by cloudmock.", http.StatusNotImplemented))
	}

	// Strip base prefix.
	rest := path[len(basePrefix):]

	if rest == "" {
		// /2013-04-01/hostedzone
		switch method {
		case http.MethodPost:
			return handleCreateHostedZone(ctx, s.store)
		case http.MethodGet:
			return handleListHostedZones(ctx, s.store)
		}
	} else if strings.HasSuffix(rest, "/rrset") {
		// /2013-04-01/hostedzone/{id}/rrset
		zoneID := zoneIDFromPath(path)
		switch method {
		case http.MethodPost:
			return handleChangeResourceRecordSets(ctx, s.store, zoneID)
		case http.MethodGet:
			return handleListResourceRecordSets(ctx, s.store, zoneID)
		}
	} else {
		// /2013-04-01/hostedzone/{id}
		zoneID := zoneIDFromPath(path)
		switch method {
		case http.MethodGet:
			return handleGetHostedZone(ctx, s.store, zoneID)
		case http.MethodDelete:
			return handleDeleteHostedZone(ctx, s.store, zoneID)
		}
	}

	return xmlErr(service.NewAWSError("NotImplemented",
		"This method and path combination is not implemented by cloudmock.", http.StatusNotImplemented))
}

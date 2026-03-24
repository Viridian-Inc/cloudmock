package plugin

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceAdapter wraps an existing service.Service to implement the Plugin interface.
// This enables gradual migration: existing Go services work as plugins without rewriting.
type ServiceAdapter struct {
	svc       service.Service
	region    string
	accountID string
}

// NewServiceAdapter wraps a legacy service.Service as a Plugin.
func NewServiceAdapter(svc service.Service, region, accountID string) *ServiceAdapter {
	return &ServiceAdapter{svc: svc, region: region, accountID: accountID}
}

func (a *ServiceAdapter) Init(_ context.Context, _ []byte, _ string, _ string) error {
	return nil
}

func (a *ServiceAdapter) Shutdown(_ context.Context) error {
	return nil
}

func (a *ServiceAdapter) HealthCheck(_ context.Context) (HealthStatus, string, error) {
	if err := a.svc.HealthCheck(); err != nil {
		return HealthUnhealthy, err.Error(), nil
	}
	return HealthHealthy, "", nil
}

func (a *ServiceAdapter) Describe(_ context.Context) (*Descriptor, error) {
	actions := a.svc.Actions()
	names := make([]string, len(actions))
	for i, act := range actions {
		names[i] = act.Name
	}
	return &Descriptor{
		Name:     a.svc.Name(),
		Version:  "1.0.0",
		Protocol: "aws",
		Actions:  names,
	}, nil
}

func (a *ServiceAdapter) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
	// Reconstruct a minimal *http.Request for the legacy service.
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.Path, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	q := httpReq.URL.Query()
	for k, v := range req.QueryParams {
		q.Set(k, v)
	}
	httpReq.URL.RawQuery = q.Encode()

	var identity *service.CallerIdentity
	if req.Auth != nil {
		identity = &service.CallerIdentity{
			AccountID:   req.Auth.AccountID,
			ARN:         req.Auth.ARN,
			UserID:      req.Auth.UserID,
			AccessKeyID: req.Auth.AccessKeyID,
			IsRoot:      req.Auth.IsRoot,
		}
	}

	svcCtx := &service.RequestContext{
		Action:     req.Action,
		Region:     a.region,
		AccountID:  a.accountID,
		Identity:   identity,
		RawRequest: httpReq,
		Body:       req.Body,
		Params:     req.QueryParams,
		Service:    a.svc.Name(),
	}

	resp, svcErr := a.svc.HandleRequest(svcCtx)
	if svcErr != nil {
		if awsErr, ok := svcErr.(*service.AWSError); ok {
			body, _ := json.Marshal(awsErr)
			return &Response{
				StatusCode: awsErr.StatusCode(),
				Body:       body,
				Headers:    map[string]string{"Content-Type": "application/json"},
			}, nil
		}
		return nil, svcErr
	}

	return serviceResponseToPluginResponse(resp)
}

// serviceResponseToPluginResponse converts a legacy service.Response to a plugin.Response.
func serviceResponseToPluginResponse(resp *service.Response) (*Response, error) {
	pr := &Response{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]string),
	}

	// Copy headers.
	for k, v := range resp.Headers {
		pr.Headers[k] = v
	}

	// Raw body takes priority.
	if resp.RawBody != nil {
		pr.Body = resp.RawBody
		if resp.RawContentType != "" {
			pr.Headers["Content-Type"] = resp.RawContentType
		}
		return pr, nil
	}

	if resp.Body == nil {
		return pr, nil
	}

	// Marshal body based on format.
	switch resp.Format {
	case service.FormatJSON:
		data, err := json.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
		pr.Body = data
		if _, ok := pr.Headers["Content-Type"]; !ok {
			pr.Headers["Content-Type"] = "application/x-amz-json-1.1"
		}
	default:
		// XML format.
		data, err := xml.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
		pr.Body = data
		if _, ok := pr.Headers["Content-Type"]; !ok {
			pr.Headers["Content-Type"] = "text/xml"
		}
	}

	return pr, nil
}

// Unwrap returns the underlying service.Service.
// This is useful for the registry to access the original service for backward compatibility.
func (a *ServiceAdapter) Unwrap() service.Service {
	return a.svc
}

// ensure ServiceAdapter satisfies Plugin at compile time.
var _ Plugin = (*ServiceAdapter)(nil)

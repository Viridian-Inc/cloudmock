package sdk

import (
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Observer wraps an http.RoundTripper to capture AWS API calls as OTel spans.
type Observer struct {
	inner       http.RoundTripper
	tracer      trace.Tracer
	environment string
	orgID       string
	appID       string
}

// NewObserver creates an observer that wraps the given transport.
// environment is "production", "staging", etc.
// orgID and appID are used for multi-tenant routing in the ingest service.
func NewObserver(inner http.RoundTripper, environment, orgID, appID string) *Observer {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &Observer{
		inner:       inner,
		tracer:      otel.Tracer("cloudmock-agent"),
		environment: environment,
		orgID:       orgID,
		appID:       appID,
	}
}

func (o *Observer) RoundTrip(req *http.Request) (*http.Response, error) {
	service, action := DetectServiceAction(req)

	ctx, span := o.tracer.Start(req.Context(), service+"."+action,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(
		attribute.String("aws.service", service),
		attribute.String("aws.action", action),
		attribute.String("aws.region", ExtractRegion(req)),
		attribute.String("cloudmock.environment", o.environment),
		attribute.String("cloudmock.source", "agent-sdk"),
		attribute.String("cloudmock.org_id", o.orgID),
		attribute.String("cloudmock.app_id", o.appID),
	)

	start := time.Now()
	resp, err := o.inner.RoundTrip(req.WithContext(ctx))
	duration := time.Since(start)

	span.SetAttributes(attribute.Float64("duration_ms", float64(duration.Milliseconds())))

	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return resp, err
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
	)

	if requestID := resp.Header.Get("X-Amzn-Requestid"); requestID != "" {
		span.SetAttributes(attribute.String("aws.request_id", requestID))
	}
	if errorCode := resp.Header.Get("X-Amzn-Errortype"); errorCode != "" {
		span.SetAttributes(attribute.String("aws.error_code", errorCode))
	}

	return resp, nil
}

// DetectServiceAction extracts the AWS service and action from the request.
// Uses X-Amz-Target header (e.g. "DynamoDB_20120810.GetItem") or
// the Host header (e.g. "s3.us-east-1.amazonaws.com").
func DetectServiceAction(req *http.Request) (string, string) {
	// Try X-Amz-Target first (DynamoDB, SQS, SNS, etc.)
	if target := req.Header.Get("X-Amz-Target"); target != "" {
		parts := strings.SplitN(target, ".", 2)
		if len(parts) == 2 {
			service := normalizeServiceName(parts[0])
			return service, parts[1]
		}
	}

	// Try Authorization header for service from credential scope
	if auth := req.Header.Get("Authorization"); auth != "" {
		// AWS4-HMAC-SHA256 Credential=.../20260403/us-east-1/s3/aws4_request
		if idx := strings.Index(auth, "Credential="); idx != -1 {
			parts := strings.Split(auth[idx:], "/")
			if len(parts) >= 4 {
				service := parts[3]
				action := detectActionFromPath(service, req.Method, req.URL.Path)
				return service, action
			}
		}
	}

	// Fallback: extract from host
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	service := extractServiceFromHost(host)
	action := detectActionFromPath(service, req.Method, req.URL.Path)
	return service, action
}

// ExtractRegion extracts the AWS region from the Authorization header.
func ExtractRegion(req *http.Request) string {
	if auth := req.Header.Get("Authorization"); auth != "" {
		if idx := strings.Index(auth, "Credential="); idx != -1 {
			parts := strings.Split(auth[idx:], "/")
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return ""
}

func normalizeServiceName(target string) string {
	target = strings.ToLower(target)
	// Strip version suffixes like "dynamodb_20120810"
	if idx := strings.Index(target, "_"); idx != -1 {
		target = target[:idx]
	}
	return target
}

func extractServiceFromHost(host string) string {
	// s3.us-east-1.amazonaws.com -> s3
	// dynamodb.us-east-1.amazonaws.com -> dynamodb
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

func detectActionFromPath(service, method, path string) string {
	// For S3: method + path structure
	// For most services: action is in X-Amz-Target (handled above)
	switch service {
	case "s3":
		switch method {
		case "PUT":
			return "PutObject"
		case "GET":
			return "GetObject"
		case "DELETE":
			return "DeleteObject"
		case "HEAD":
			return "HeadObject"
		default:
			return method
		}
	default:
		return method + " " + path
	}
}

// Package proxy provides a reverse proxy that captures real AWS traffic
// for later replay against CloudMock.
package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/traffic"
)

// serviceEndpoints maps AWS service names to their endpoint format strings.
// Use fmt.Sprintf(pattern, region) for regional services.
var serviceEndpoints = map[string]string{
	"s3":                      "s3.%s.amazonaws.com",
	"dynamodb":                "dynamodb.%s.amazonaws.com",
	"sqs":                     "sqs.%s.amazonaws.com",
	"sns":                     "sns.%s.amazonaws.com",
	"lambda":                  "lambda.%s.amazonaws.com",
	"iam":                     "iam.amazonaws.com",
	"sts":                     "sts.%s.amazonaws.com",
	"kms":                     "kms.%s.amazonaws.com",
	"kinesis":                 "kinesis.%s.amazonaws.com",
	"firehose":                "firehose.%s.amazonaws.com",
	"logs":                    "logs.%s.amazonaws.com",
	"events":                  "events.%s.amazonaws.com",
	"cloudwatch":              "monitoring.%s.amazonaws.com",
	"cloudformation":          "cloudformation.%s.amazonaws.com",
	"ec2":                     "ec2.%s.amazonaws.com",
	"ecs":                     "ecs.%s.amazonaws.com",
	"eks":                     "eks.%s.amazonaws.com",
	"rds":                     "rds.%s.amazonaws.com",
	"route53":                 "route53.amazonaws.com",
	"cloudfront":              "cloudfront.amazonaws.com",
	"ses":                     "email.%s.amazonaws.com",
	"cognito-idp":             "cognito-idp.%s.amazonaws.com",
	"apigateway":              "apigateway.%s.amazonaws.com",
	"secretsmanager":          "secretsmanager.%s.amazonaws.com",
	"ssm":                     "ssm.%s.amazonaws.com",
	"stepfunctions":           "states.%s.amazonaws.com",
	"codebuild":               "codebuild.%s.amazonaws.com",
	"codepipeline":            "codepipeline.%s.amazonaws.com",
	"configservice":           "config.%s.amazonaws.com",
	"cloudtrail":              "cloudtrail.%s.amazonaws.com",
}

// AWSProxy is a reverse proxy that forwards requests to real AWS endpoints
// and captures the request/response pairs as CapturedEntry values.
type AWSProxy struct {
	entries      []*traffic.CapturedEntry
	mu           sync.Mutex
	startTime    time.Time
	httpClient   *http.Client
	region       string
	entrySeq     int
	testEndpoint string // if set, all requests are forwarded here instead of real AWS
}

// New creates a new AWSProxy targeting the given AWS region.
func New(region string) *AWSProxy {
	return &AWSProxy{
		startTime: time.Now(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		region: region,
	}
}

// ServeHTTP handles an incoming AWS SDK request by detecting the target service,
// forwarding the request to the real AWS endpoint, and capturing the exchange.
func (p *AWSProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	svc := detectServiceFromAuth(r)
	if svc == "" {
		http.Error(w, "could not detect AWS service from request", http.StatusBadGateway)
		return
	}

	// Read request body.
	var reqBody []byte
	if r.Body != nil {
		reqBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
	}

	// Build the forwarding URL.
	var targetURL string
	var host string
	if p.testEndpoint != "" {
		// Test mode: forward everything to the test endpoint.
		targetURL = strings.TrimRight(p.testEndpoint, "/") + r.URL.RequestURI()
		host = ""
	} else {
		host = resolveEndpoint(svc, p.region)
		if host == "" {
			http.Error(w, fmt.Sprintf("unknown service: %s", svc), http.StatusBadGateway)
			return
		}

		// Use HTTPS for real AWS endpoints, HTTP otherwise (e.g., local test servers).
		scheme := "https"
		if !strings.HasSuffix(host, ".amazonaws.com") {
			scheme = "http"
		}
		targetURL = fmt.Sprintf("%s://%s%s", scheme, host, r.URL.RequestURI())
	}

	fwdReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, "failed to create forwarding request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy all headers from the original request.
	for k, vals := range r.Header {
		for _, v := range vals {
			fwdReq.Header.Add(k, v)
		}
	}
	// Override Host to the real AWS endpoint (skip in test mode).
	if host != "" {
		fwdReq.Host = host
	}

	resp, err := p.httpClient.Do(fwdReq)
	if err != nil {
		http.Error(w, "upstream request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	latencyMs := float64(time.Since(start).Nanoseconds()) / 1e6

	// Build captured entry.
	headers := make(map[string]string, len(r.Header))
	for k := range r.Header {
		headers[k] = r.Header.Get(k)
	}

	action := detectAction(r)

	p.mu.Lock()
	p.entrySeq++
	entryID := fmt.Sprintf("proxy-%d", p.entrySeq)
	entry := &traffic.CapturedEntry{
		ID:             entryID,
		Timestamp:      start,
		Service:        svc,
		Action:         action,
		Method:         r.Method,
		Path:           r.URL.Path,
		StatusCode:     resp.StatusCode,
		LatencyMs:      latencyMs,
		RequestHeaders: headers,
		RequestBody:    string(reqBody),
		ResponseBody:   string(respBody),
		OffsetMs:       float64(start.Sub(p.startTime).Milliseconds()),
	}
	p.entries = append(p.entries, entry)
	p.mu.Unlock()

	// Write the real response back to the caller.
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// Recording converts the captured entries into a traffic.Recording.
func (p *AWSProxy) Recording() *traffic.Recording {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	entries := make([]traffic.CapturedEntry, len(p.entries))
	for i, e := range p.entries {
		entries[i] = *e
	}

	return &traffic.Recording{
		ID:          fmt.Sprintf("proxy-%d", now.UnixMilli()),
		Name:        "aws-proxy-recording",
		Status:      traffic.RecordingCompleted,
		StartedAt:   p.startTime,
		StoppedAt:   &now,
		EntryCount:  len(entries),
		Entries:     entries,
		DurationSec: int(now.Sub(p.startTime).Seconds()),
	}
}

// SaveToFile marshals the recording to JSON and writes it to the given path.
func (p *AWSProxy) SaveToFile(path string) error {
	rec := p.Recording()
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal recording: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Entries returns a copy of the captured entries.
func (p *AWSProxy) Entries() []*traffic.CapturedEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]*traffic.CapturedEntry, len(p.entries))
	copy(out, p.entries)
	return out
}

// detectServiceFromAuth extracts the AWS service from the Authorization header
// credential scope: Credential=AKID/date/region/SERVICE/aws4_request.
func detectServiceFromAuth(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	const prefix = "Credential="
	idx := strings.Index(auth, prefix)
	if idx < 0 {
		return ""
	}
	rest := auth[idx+len(prefix):]

	end := strings.IndexAny(rest, ", ")
	if end >= 0 {
		rest = rest[:end]
	}

	parts := strings.Split(rest, "/")
	if len(parts) < 4 {
		return ""
	}
	return strings.ToLower(parts[3])
}

// detectAction extracts the AWS action from the request.
func detectAction(r *http.Request) string {
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		if dot := strings.LastIndex(target, "."); dot >= 0 {
			return target[dot+1:]
		}
	}
	return r.URL.Query().Get("Action")
}

// resolveEndpoint returns the real AWS hostname for a service and region.
func resolveEndpoint(service, region string) string {
	pattern, ok := serviceEndpoints[service]
	if !ok {
		return ""
	}
	if !strings.Contains(pattern, "%s") {
		return pattern // global service
	}
	return fmt.Sprintf(pattern, region)
}

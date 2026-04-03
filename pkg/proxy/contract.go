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

// ContractProxy is a dual-mode proxy that sends each incoming request to both
// real AWS and a local CloudMock instance, compares the responses, and returns
// the real AWS response to the caller. The comparison results are stored for
// later report generation.
type ContractProxy struct {
	awsProxy          *AWSProxy
	cloudmockEndpoint string
	results           []*ContractResult
	mu                sync.Mutex
	ignorePaths       []string
	httpClient        *http.Client
}

// ContractResult holds the comparison outcome for a single request pair.
type ContractResult struct {
	Timestamp       time.Time
	Service         string
	Action          string
	Method          string
	Path            string
	AWSStatus       int
	AWSBody         []byte
	CloudMockStatus int
	CloudMockBody   []byte
	Match           bool
	Diffs           []string
	Severity        string
	LatencyAWS      time.Duration
	LatencyCM       time.Duration
}

// ContractReport is the JSON-serialisable summary of a contract test run.
type ContractReport struct {
	StartedAt        string                    `json:"started_at"`
	DurationSec      float64                   `json:"duration_sec"`
	TotalRequests    int                       `json:"total_requests"`
	Matched          int                       `json:"matched"`
	Mismatched       int                       `json:"mismatched"`
	CompatibilityPct float64                   `json:"compatibility_pct"`
	ByService        map[string]*ServiceReport `json:"by_service"`
	Mismatches       []MismatchEntry           `json:"mismatches"`
}

// ServiceReport provides per-service match statistics.
type ServiceReport struct {
	Total   int     `json:"total"`
	Matched int     `json:"matched"`
	Pct     float64 `json:"pct"`
}

// MismatchEntry is a single mismatched request in the report.
type MismatchEntry struct {
	Service         string   `json:"service"`
	Action          string   `json:"action"`
	AWSStatus       int      `json:"aws_status"`
	CloudMockStatus int      `json:"cloudmock_status"`
	Diffs           []string `json:"diffs"`
	Severity        string   `json:"severity"`
}

// NewContractProxy creates a ContractProxy that forwards to real AWS via
// AWSProxy and simultaneously to the given CloudMock endpoint.
func NewContractProxy(region string, cloudmockEndpoint string, ignorePaths []string) *ContractProxy {
	return &ContractProxy{
		awsProxy:          New(region),
		cloudmockEndpoint: strings.TrimRight(cloudmockEndpoint, "/"),
		ignorePaths:       ignorePaths,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ServeHTTP handles an incoming request by sending it to both real AWS and
// CloudMock concurrently, comparing the responses, and returning the real
// AWS response to the caller.
func (cp *ContractProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Buffer the request body so we can send it to both endpoints.
	var reqBody []byte
	if r.Body != nil {
		reqBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
	}

	svc := detectServiceFromAuth(r)
	action := detectAction(r)

	type proxyResult struct {
		statusCode int
		headers    http.Header
		body       []byte
		err        error
		latency    time.Duration
	}

	var awsRes, cmRes proxyResult
	var wg sync.WaitGroup
	wg.Add(2)

	// Forward to real AWS.
	go func() {
		defer wg.Done()
		start := time.Now()

		// Build a new request with the buffered body for the AWSProxy.
		awsReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.RequestURI(), bytes.NewReader(reqBody))
		if err != nil {
			awsRes.err = err
			return
		}
		for k, vals := range r.Header {
			for _, v := range vals {
				awsReq.Header.Add(k, v)
			}
		}
		awsReq.Host = r.Host

		rec := &responseRecorder{headers: make(http.Header)}
		cp.awsProxy.ServeHTTP(rec, awsReq)

		awsRes.statusCode = rec.statusCode
		awsRes.headers = rec.headers
		awsRes.body = rec.body.Bytes()
		awsRes.latency = time.Since(start)
	}()

	// Forward to CloudMock.
	go func() {
		defer wg.Done()
		start := time.Now()

		targetURL := cp.cloudmockEndpoint + r.URL.RequestURI()
		cmReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(reqBody))
		if err != nil {
			cmRes.err = err
			return
		}
		for k, vals := range r.Header {
			for _, v := range vals {
				cmReq.Header.Add(k, v)
			}
		}

		resp, err := cp.httpClient.Do(cmReq)
		if err != nil {
			cmRes.err = err
			cmRes.latency = time.Since(start)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		cmRes.statusCode = resp.StatusCode
		cmRes.headers = resp.Header
		cmRes.body = body
		cmRes.latency = time.Since(start)
	}()

	wg.Wait()

	// Compare responses.
	result := &ContractResult{
		Timestamp:       time.Now(),
		Service:         svc,
		Action:          action,
		Method:          r.Method,
		Path:            r.URL.Path,
		AWSStatus:       awsRes.statusCode,
		AWSBody:         awsRes.body,
		CloudMockStatus: cmRes.statusCode,
		CloudMockBody:   cmRes.body,
		LatencyAWS:      awsRes.latency,
		LatencyCM:       cmRes.latency,
	}

	if awsRes.err != nil || cmRes.err != nil {
		result.Match = false
		if awsRes.err != nil {
			result.Diffs = append(result.Diffs, "AWS error: "+awsRes.err.Error())
		}
		if cmRes.err != nil {
			result.Diffs = append(result.Diffs, "CloudMock error: "+cmRes.err.Error())
		}
		result.Severity = "error"
	} else {
		result.Match = true

		// Compare status codes.
		if awsRes.statusCode != cmRes.statusCode {
			result.Match = false
			result.Diffs = append(result.Diffs, fmt.Sprintf("status: %d -> %d", awsRes.statusCode, cmRes.statusCode))
			result.Severity = "status"
		}

		// Compare response bodies.
		bodyDiffs := traffic.CompareJSON(awsRes.body, cmRes.body, cp.ignorePaths)
		if len(bodyDiffs) > 0 {
			result.Match = false
			result.Diffs = append(result.Diffs, bodyDiffs...)
			if result.Severity == "" {
				for _, d := range bodyDiffs {
					if strings.Contains(d, "missing key") || strings.Contains(d, "extra key") {
						result.Severity = "schema"
						break
					}
				}
				if result.Severity == "" {
					result.Severity = "data"
				}
			}
		}
	}

	cp.mu.Lock()
	cp.results = append(cp.results, result)
	cp.mu.Unlock()

	// Return the real AWS response to the caller.
	if awsRes.err != nil {
		http.Error(w, "upstream AWS request failed: "+awsRes.err.Error(), http.StatusBadGateway)
		return
	}
	for k, vals := range awsRes.headers {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(awsRes.statusCode)
	w.Write(awsRes.body)
}

// Results returns a copy of all contract comparison results.
func (cp *ContractProxy) Results() []*ContractResult {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	out := make([]*ContractResult, len(cp.results))
	copy(out, cp.results)
	return out
}

// Report generates a ContractReport summarising the contract test run.
func (cp *ContractProxy) Report() *ContractReport {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	report := &ContractReport{
		StartedAt:     cp.awsProxy.startTime.Format(time.RFC3339),
		DurationSec:   time.Since(cp.awsProxy.startTime).Seconds(),
		TotalRequests: len(cp.results),
		ByService:     make(map[string]*ServiceReport),
	}

	for _, r := range cp.results {
		svc := r.Service
		if svc == "" {
			svc = "unknown"
		}

		sr, ok := report.ByService[svc]
		if !ok {
			sr = &ServiceReport{}
			report.ByService[svc] = sr
		}
		sr.Total++

		if r.Match {
			report.Matched++
			sr.Matched++
		} else {
			report.Mismatched++
			report.Mismatches = append(report.Mismatches, MismatchEntry{
				Service:         r.Service,
				Action:          r.Action,
				AWSStatus:       r.AWSStatus,
				CloudMockStatus: r.CloudMockStatus,
				Diffs:           r.Diffs,
				Severity:        r.Severity,
			})
		}
	}

	if report.TotalRequests > 0 {
		report.CompatibilityPct = float64(report.Matched) / float64(report.TotalRequests) * 100
	}
	for _, sr := range report.ByService {
		if sr.Total > 0 {
			sr.Pct = float64(sr.Matched) / float64(sr.Total) * 100
		}
	}

	return report
}

// SaveReport writes the contract report as JSON to the given file path.
func (cp *ContractProxy) SaveReport(path string) error {
	report := cp.Report()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// responseRecorder captures an http.Handler's response without writing to a
// real http.ResponseWriter. Used to intercept the AWSProxy response.
type responseRecorder struct {
	statusCode int
	headers    http.Header
	body       bytes.Buffer
}

func (rr *responseRecorder) Header() http.Header {
	return rr.headers
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	return rr.body.Write(b)
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
}

// Package nlq provides natural language query parsing for CloudMock.
// It parses plain-English questions into structured Query objects that
// can be executed against the CloudMock data stores.
package nlq

import (
	"fmt"
	"strings"
)

// Query represents a structured query parsed from natural language.
type Query struct {
	Type      string            // "errors", "requests", "metrics", "deploys", "health"
	Service   string            // specific service filter (empty = all)
	TimeRange string            // "1h", "24h", "7d" (empty = default)
	Filters   map[string]string // additional key-value filters
	Sort      string            // "desc", "asc", or field name
	Limit     int               // max results (0 = default)
}

// Parse converts a natural language question into a structured Query.
// It uses pattern matching (no LLM) to detect intent.
func Parse(question string) (*Query, error) {
	q := strings.ToLower(strings.TrimSpace(question))
	if q == "" {
		return nil, fmt.Errorf("empty question")
	}

	// Try each pattern in priority order.
	if query := matchHealth(q); query != nil {
		return query, nil
	}
	if query := matchDeployRegression(q); query != nil {
		return query, nil
	}
	if query := matchMostErrors(q); query != nil {
		return query, nil
	}
	if query := matchSlowRequests(q); query != nil {
		return query, nil
	}
	if query := matchErrorRate(q); query != nil {
		return query, nil
	}
	if query := matchRequests(q); query != nil {
		return query, nil
	}
	if query := matchDeploys(q); query != nil {
		return query, nil
	}
	if query := matchMetrics(q); query != nil {
		return query, nil
	}

	// Fallback: if we detect a service name, query errors for it.
	if svc := extractService(q); svc != "" {
		return &Query{
			Type:    "errors",
			Service: svc,
			Filters: make(map[string]string),
		}, nil
	}

	return nil, fmt.Errorf("could not understand question: %q", question)
}

// Describe returns a human-readable description of what the query does.
func (q *Query) Describe() string {
	var parts []string

	switch q.Type {
	case "errors":
		parts = append(parts, "Query error records")
	case "requests":
		parts = append(parts, "Query request logs")
	case "metrics":
		parts = append(parts, "Query metrics")
	case "deploys":
		parts = append(parts, "Query deployment history")
	case "health":
		parts = append(parts, "Check system health")
	default:
		parts = append(parts, "Query "+q.Type)
	}

	if q.Service != "" {
		parts = append(parts, fmt.Sprintf("for service %q", q.Service))
	}

	if q.TimeRange != "" {
		parts = append(parts, fmt.Sprintf("over the last %s", q.TimeRange))
	}

	for k, v := range q.Filters {
		parts = append(parts, fmt.Sprintf("where %s=%s", k, v))
	}

	if q.Sort != "" {
		parts = append(parts, fmt.Sprintf("sorted %s", q.Sort))
	}

	if q.Limit > 0 {
		parts = append(parts, fmt.Sprintf("limit %d", q.Limit))
	}

	return strings.Join(parts, ", ")
}

// matchHealth detects system health questions.
func matchHealth(q string) *Query {
	healthPhrases := []string{
		"how is the system",
		"system health",
		"is everything ok",
		"is everything okay",
		"overall status",
		"how are things",
		"status overview",
		"dashboard",
	}
	for _, p := range healthPhrases {
		if strings.Contains(q, p) {
			return &Query{
				Type:    "health",
				Filters: make(map[string]string),
			}
		}
	}
	return nil
}

// matchDeployRegression detects post-deploy queries.
func matchDeployRegression(q string) *Query {
	deployPhrases := []string{
		"after the last deploy",
		"since the last deploy",
		"after deploy",
		"since deploy",
		"post-deploy",
		"post deploy",
		"after the latest deploy",
	}
	for _, p := range deployPhrases {
		if strings.Contains(q, p) {
			query := &Query{
				Type:    "errors",
				Filters: map[string]string{"since": "last_deploy"},
				Sort:    "desc",
			}
			if svc := extractService(q); svc != "" {
				query.Service = svc
			}
			return query
		}
	}
	return nil
}

// matchMostErrors detects "which service has the most errors" questions.
func matchMostErrors(q string) *Query {
	if (strings.Contains(q, "which service") || strings.Contains(q, "what service")) &&
		strings.Contains(q, "most error") {
		return &Query{
			Type:    "errors",
			Filters: map[string]string{"aggregate": "service"},
			Sort:    "desc",
			Limit:   10,
		}
	}
	if strings.Contains(q, "top error") || strings.Contains(q, "error ranking") {
		return &Query{
			Type:    "errors",
			Filters: map[string]string{"aggregate": "service"},
			Sort:    "desc",
			Limit:   10,
		}
	}
	return nil
}

// matchSlowRequests detects slow request queries.
func matchSlowRequests(q string) *Query {
	slowPhrases := []string{
		"slow request",
		"slow response",
		"high latency",
		"slowest",
		"taking long",
		"latency spike",
	}
	for _, p := range slowPhrases {
		if strings.Contains(q, p) {
			query := &Query{
				Type:    "requests",
				Filters: map[string]string{"latency": "p95"},
				Sort:    "desc",
				Limit:   20,
			}
			if svc := extractService(q); svc != "" {
				query.Service = svc
			}
			if tr := extractTimeRange(q); tr != "" {
				query.TimeRange = tr
			}
			return query
		}
	}
	return nil
}

// matchErrorRate detects error rate queries.
func matchErrorRate(q string) *Query {
	if strings.Contains(q, "error rate") || strings.Contains(q, "error count") ||
		(strings.Contains(q, "error") && strings.Contains(q, "for")) ||
		(strings.Contains(q, "errors") && strings.Contains(q, "in")) {
		query := &Query{
			Type:    "errors",
			Filters: make(map[string]string),
		}
		if svc := extractService(q); svc != "" {
			query.Service = svc
		}
		if tr := extractTimeRange(q); tr != "" {
			query.TimeRange = tr
		}
		return query
	}
	return nil
}

// matchRequests detects request log queries.
func matchRequests(q string) *Query {
	if strings.Contains(q, "request") || strings.Contains(q, "traffic") ||
		strings.Contains(q, "throughput") || strings.Contains(q, "rps") {
		query := &Query{
			Type:    "requests",
			Filters: make(map[string]string),
		}
		if svc := extractService(q); svc != "" {
			query.Service = svc
		}
		if tr := extractTimeRange(q); tr != "" {
			query.TimeRange = tr
		}
		return query
	}
	return nil
}

// matchDeploys detects deploy-related queries.
func matchDeploys(q string) *Query {
	if strings.Contains(q, "deploy") || strings.Contains(q, "deployment") ||
		strings.Contains(q, "release") || strings.Contains(q, "ship") {
		query := &Query{
			Type:    "deploys",
			Filters: make(map[string]string),
			Sort:    "desc",
		}
		if svc := extractService(q); svc != "" {
			query.Service = svc
		}
		if tr := extractTimeRange(q); tr != "" {
			query.TimeRange = tr
		}
		return query
	}
	return nil
}

// matchMetrics detects metric queries.
func matchMetrics(q string) *Query {
	if strings.Contains(q, "metric") || strings.Contains(q, "cpu") ||
		strings.Contains(q, "memory") || strings.Contains(q, "disk") ||
		strings.Contains(q, "p99") || strings.Contains(q, "p95") || strings.Contains(q, "p50") {
		query := &Query{
			Type:    "metrics",
			Filters: make(map[string]string),
		}
		if svc := extractService(q); svc != "" {
			query.Service = svc
		}
		if tr := extractTimeRange(q); tr != "" {
			query.TimeRange = tr
		}
		return query
	}
	return nil
}

// extractService attempts to find a service name in the question.
// Looks for common patterns like "for <service>", "in <service>", "the <service> service".
func extractService(q string) string {
	// Pattern: "for <service>"
	for _, prep := range []string{"for ", "in ", "from "} {
		idx := strings.Index(q, prep)
		if idx >= 0 {
			rest := q[idx+len(prep):]
			// Take the next word (service name).
			word := firstWord(rest)
			if word != "" && !isStopWord(word) {
				return word
			}
		}
	}
	// Pattern: "<service> service"
	if idx := strings.Index(q, " service"); idx > 0 {
		before := q[:idx]
		word := lastWord(before)
		if word != "" && !isStopWord(word) {
			return word
		}
	}
	return ""
}

// extractTimeRange extracts a time range like "1h", "24h", "7d" from the question.
func extractTimeRange(q string) string {
	timePatterns := map[string]string{
		"last hour":     "1h",
		"past hour":     "1h",
		"last 1h":       "1h",
		"last 24h":      "24h",
		"last 24 hours": "24h",
		"last day":      "24h",
		"past day":      "24h",
		"today":         "24h",
		"last week":     "7d",
		"past week":     "7d",
		"last 7 days":   "7d",
		"last 7d":       "7d",
		"last 30d":      "30d",
		"last month":    "30d",
		"past month":    "30d",
		"last 30 days":  "30d",
	}
	for pattern, tr := range timePatterns {
		if strings.Contains(q, pattern) {
			return tr
		}
	}
	return ""
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	// Strip trailing punctuation.
	w := fields[0]
	w = strings.TrimRight(w, "?.!,;:")
	return w
}

func lastWord(s string) string {
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func isStopWord(w string) bool {
	stops := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "been": true, "be": true, "have": true,
		"has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "shall": true, "should": true,
		"may": true, "might": true, "must": true, "can": true, "could": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "at": true, "by": true, "from": true, "it": true,
		"this": true, "that": true, "my": true, "all": true, "me": true,
		"what": true, "which": true, "who": true, "how": true, "there": true,
		"last": true, "most": true, "each": true, "every": true,
	}
	return stops[w]
}

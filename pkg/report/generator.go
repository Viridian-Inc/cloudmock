package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	"github.com/Viridian-Inc/cloudmock/pkg/incident"
	"github.com/Viridian-Inc/cloudmock/pkg/regression"
)

// Report is the assembled data for an incident report.
type Report struct {
	Incident    incident.Incident        `json:"incident"`
	Regressions []regression.Regression  `json:"regressions"`
	Timeline    []TimelineEntry          `json:"timeline"`
	GeneratedAt time.Time                `json:"generated_at"`
}

// TimelineEntry is a single event on the incident timeline.
type TimelineEntry struct {
	Time    time.Time `json:"time"`
	Event   string    `json:"event"`
	Details string    `json:"details"`
}

// Generator assembles and renders incident reports.
type Generator struct {
	incidents   incident.IncidentStore
	regressions regression.RegressionStore
	traces      dataplane.TraceReader
}

// New creates a new Generator. traces may be nil if trace data is unavailable.
func New(incidents incident.IncidentStore, regressions regression.RegressionStore, traces dataplane.TraceReader) *Generator {
	return &Generator{
		incidents:   incidents,
		regressions: regressions,
		traces:      traces,
	}
}

// Generate assembles data for the incident identified by id and renders it in
// the requested format ("json", "csv", or "html").  It returns the content
// bytes, a content-type string, and any error.
func (g *Generator) Generate(ctx context.Context, id string, format string) ([]byte, string, error) {
	if format == "" {
		format = "json"
	}

	inc, err := g.incidents.Get(ctx, id)
	if err != nil {
		if errors.Is(err, incident.ErrNotFound) {
			return nil, "", fmt.Errorf("incident %q not found", id)
		}
		return nil, "", fmt.Errorf("fetching incident: %w", err)
	}

	// Gather related regressions (filter by any affected service).
	var regs []regression.Regression
	if g.regressions != nil {
		seen := map[string]bool{}
		for _, svc := range inc.AffectedServices {
			results, listErr := g.regressions.List(ctx, regression.RegressionFilter{
				Service: svc,
				Limit:   50,
			})
			if listErr != nil {
				continue
			}
			for _, r := range results {
				if seen[r.ID] {
					continue
				}
				// Keep regressions detected within a ±1h window around the incident.
				window := time.Hour
				if !r.DetectedAt.Before(inc.FirstSeen.Add(-window)) &&
					!r.DetectedAt.After(inc.LastSeen.Add(window)) {
					regs = append(regs, r)
					seen[r.ID] = true
				}
			}
		}
	}

	// Build timeline.
	timeline := buildTimeline(*inc, regs)

	rpt := Report{
		Incident:    *inc,
		Regressions: regs,
		Timeline:    timeline,
		GeneratedAt: time.Now().UTC(),
	}

	switch format {
	case "json":
		data, err := renderJSON(rpt)
		return data, "application/json", err
	case "csv":
		data, err := renderCSV(rpt)
		return data, "text/csv", err
	case "html":
		data, err := renderHTML(rpt)
		return data, "text/html; charset=utf-8", err
	default:
		return nil, "", fmt.Errorf("unsupported format %q: must be json, csv, or html", format)
	}
}

// buildTimeline produces a sorted timeline of events from the incident and its
// related regressions.
func buildTimeline(inc incident.Incident, regs []regression.Regression) []TimelineEntry {
	var entries []TimelineEntry

	entries = append(entries, TimelineEntry{
		Time:    inc.FirstSeen,
		Event:   "Incident detected",
		Details: inc.Title,
	})

	for _, r := range regs {
		entries = append(entries, TimelineEntry{
			Time:    r.DetectedAt,
			Event:   fmt.Sprintf("Regression detected (%s)", r.Algorithm),
			Details: r.Title,
		})
	}

	if inc.ResolvedAt != nil {
		entries = append(entries, TimelineEntry{
			Time:    *inc.ResolvedAt,
			Event:   "Incident resolved",
			Details: inc.Title,
		})
	}

	// Sort by time ascending.
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Time.Before(entries[j-1].Time); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}

	return entries
}

// renderJSON marshals the report to JSON.
func renderJSON(rpt Report) ([]byte, error) {
	data, err := json.MarshalIndent(rpt, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling JSON report: %w", err)
	}
	return data, nil
}

// renderCSV flattens the report into a two-dimensional CSV.
func renderCSV(rpt Report) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	rows := [][]string{
		{"Section", "Field", "Value"},
		{"Incident", "ID", rpt.Incident.ID},
		{"Incident", "Title", rpt.Incident.Title},
		{"Incident", "Severity", rpt.Incident.Severity},
		{"Incident", "Status", rpt.Incident.Status},
		{"Incident", "First Seen", rpt.Incident.FirstSeen.Format(time.RFC3339)},
		{"Incident", "Last Seen", rpt.Incident.LastSeen.Format(time.RFC3339)},
		{"Incident", "Affected Services", strings.Join(rpt.Incident.AffectedServices, ",")},
	}

	for _, r := range rpt.Regressions {
		rows = append(rows, []string{
			"Regression",
			r.Title,
			fmt.Sprintf("%.2f%% change", r.ChangePercent),
		})
	}

	if err := w.WriteAll(rows); err != nil {
		return nil, fmt.Errorf("writing CSV: %w", err)
	}
	return buf.Bytes(), nil
}

// htmlTmpl is the self-contained HTML report template.
var htmlTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"join": strings.Join,
}).Parse(`<!DOCTYPE html>
<html>
<head><title>Incident Report: {{.Incident.Title}}</title>
<style>
body { font-family: -apple-system, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; }
h1 { color: #e53e3e; }
.meta { color: #666; }
table { border-collapse: collapse; width: 100%; margin: 20px 0; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
th { background: #f5f5f5; }
.severity-critical { color: #e53e3e; font-weight: bold; }
.severity-warning { color: #dd6b20; }
.severity-info { color: #3182ce; }
</style>
</head>
<body>
<h1>Incident Report: {{.Incident.Title}}</h1>
<p class="meta">Generated {{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}}</p>

<h2>Details</h2>
<table>
<tr><th>ID</th><td>{{.Incident.ID}}</td></tr>
<tr><th>Severity</th><td class="severity-{{.Incident.Severity}}">{{.Incident.Severity}}</td></tr>
<tr><th>Status</th><td>{{.Incident.Status}}</td></tr>
<tr><th>First Seen</th><td>{{.Incident.FirstSeen.Format "2006-01-02 15:04:05"}}</td></tr>
<tr><th>Affected Services</th><td>{{join .Incident.AffectedServices ", "}}</td></tr>
{{if .Incident.RootCause}}<tr><th>Root Cause</th><td>{{.Incident.RootCause}}</td></tr>{{end}}
{{if .Incident.Owner}}<tr><th>Owner</th><td>{{.Incident.Owner}}</td></tr>{{end}}
</table>

{{if .Regressions}}
<h2>Related Regressions</h2>
<table>
<tr><th>Algorithm</th><th>Service</th><th>Change</th><th>Severity</th><th>Confidence</th></tr>
{{range .Regressions}}
<tr><td>{{.Algorithm}}</td><td>{{.Service}}</td><td>{{printf "%.1f" .ChangePercent}}%</td><td>{{.Severity}}</td><td>{{.Confidence}}%</td></tr>
{{end}}
</table>
{{end}}

{{if .Timeline}}
<h2>Timeline</h2>
<table>
<tr><th>Time</th><th>Event</th><th>Details</th></tr>
{{range .Timeline}}
<tr><td>{{.Time.Format "15:04:05"}}</td><td>{{.Event}}</td><td>{{.Details}}</td></tr>
{{end}}
</table>
{{end}}

</body>
</html>
`))

// renderHTML executes the HTML template against the report.
func renderHTML(rpt Report) ([]byte, error) {
	var buf bytes.Buffer
	if err := htmlTmpl.Execute(&buf, rpt); err != nil {
		return nil, fmt.Errorf("rendering HTML report: %w", err)
	}
	return buf.Bytes(), nil
}

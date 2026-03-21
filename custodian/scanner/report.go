package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// Report is the top-level compliance scan report.
type Report struct {
	Timestamp string         `json:"timestamp"`
	Summary   ReportSummary  `json:"summary"`
	Findings  []Finding      `json:"findings"`
	Rules     []RuleSummary  `json:"rules"`
}

// ReportSummary holds aggregate scan statistics.
type ReportSummary struct {
	TotalRules int `json:"total_rules"`
	Passed     int `json:"passed"`
	Failed     int `json:"failed"`
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
}

// RuleSummary records per-rule pass/fail status.
type RuleSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Status   string `json:"status"` // passed or failed
	Findings int    `json:"findings"`
}

// GenerateReport builds a Report from findings and the rule definitions.
func GenerateReport(findings []Finding, rules []Rule) Report {
	// Build a map from rule ID to finding count.
	findingsByRule := make(map[string]int)
	for _, f := range findings {
		findingsByRule[f.RuleID]++
	}

	var ruleSummaries []RuleSummary
	passed := 0
	failed := 0
	for _, r := range rules {
		count := findingsByRule[r.ID]
		status := "passed"
		if count > 0 {
			status = "failed"
			failed++
		} else {
			passed++
		}
		ruleSummaries = append(ruleSummaries, RuleSummary{
			ID:       r.ID,
			Name:     r.Name,
			Severity: r.Severity,
			Status:   status,
			Findings: count,
		})
	}

	// Count severities.
	sevCounts := map[string]int{}
	for _, f := range findings {
		sevCounts[strings.ToLower(f.Severity)]++
	}

	return Report{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Summary: ReportSummary{
			TotalRules: len(rules),
			Passed:     passed,
			Failed:     failed,
			Critical:   sevCounts["critical"],
			High:       sevCounts["high"],
			Medium:     sevCounts["medium"],
			Low:        sevCounts["low"],
		},
		Findings: findings,
		Rules:    ruleSummaries,
	}
}

// RenderJSON renders the report as indented JSON.
func RenderJSON(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// RenderJUnit renders the report as JUnit XML, with each rule as a test case.
func RenderJUnit(report Report) ([]byte, error) {
	type junitFailure struct {
		XMLName xml.Name `xml:"failure"`
		Message string   `xml:"message,attr"`
		Type    string   `xml:"type,attr"`
		Content string   `xml:",chardata"`
	}

	type junitTestCase struct {
		XMLName   xml.Name      `xml:"testcase"`
		Name      string        `xml:"name,attr"`
		ClassName string        `xml:"classname,attr"`
		Time      string        `xml:"time,attr"`
		Failure   *junitFailure `xml:"failure,omitempty"`
	}

	type junitTestSuite struct {
		XMLName  xml.Name        `xml:"testsuite"`
		Name     string          `xml:"name,attr"`
		Tests    int             `xml:"tests,attr"`
		Failures int             `xml:"failures,attr"`
		Time     string          `xml:"time,attr"`
		Cases    []junitTestCase `xml:"testcase"`
	}

	type junitTestSuites struct {
		XMLName xml.Name         `xml:"testsuites"`
		Suites  []junitTestSuite `xml:"testsuite"`
	}

	// Build finding details indexed by rule ID.
	findingsByRule := make(map[string][]Finding)
	for _, f := range report.Findings {
		findingsByRule[f.RuleID] = append(findingsByRule[f.RuleID], f)
	}

	var cases []junitTestCase
	for _, rs := range report.Rules {
		tc := junitTestCase{
			Name:      rs.ID + " — " + rs.Name,
			ClassName: "cloudmock-compliance." + rs.Severity,
			Time:      "0",
		}
		if rs.Status == "failed" {
			var details []string
			for _, f := range findingsByRule[rs.ID] {
				details = append(details, fmt.Sprintf("[%s] %s: %s — %s", f.Severity, f.ResourceID, f.Message, f.Remediation))
			}
			tc.Failure = &junitFailure{
				Message: fmt.Sprintf("%d finding(s)", rs.Findings),
				Type:    rs.Severity,
				Content: strings.Join(details, "\n"),
			}
		}
		cases = append(cases, tc)
	}

	suites := junitTestSuites{
		Suites: []junitTestSuite{
			{
				Name:     "cloudmock-compliance",
				Tests:    report.Summary.TotalRules,
				Failures: report.Summary.Failed,
				Time:     "0",
				Cases:    cases,
			},
		},
	}

	out, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

// RenderHTML renders the report as a self-contained HTML page.
func RenderHTML(report Report) ([]byte, error) {
	const tmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>cloudmock Compliance Report</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 2rem; background: #f5f5f5; color: #333; }
  h1 { color: #1a1a2e; }
  .summary { display: flex; gap: 1rem; margin-bottom: 2rem; flex-wrap: wrap; }
  .card { background: white; border-radius: 8px; padding: 1rem 1.5rem; box-shadow: 0 1px 3px rgba(0,0,0,0.1); min-width: 120px; }
  .card .label { font-size: 0.85rem; color: #666; text-transform: uppercase; }
  .card .value { font-size: 2rem; font-weight: bold; }
  .card.critical .value { color: #d32f2f; }
  .card.high .value { color: #f57c00; }
  .card.medium .value { color: #fbc02d; }
  .card.low .value { color: #388e3c; }
  .card.passed .value { color: #388e3c; }
  .card.failed .value { color: #d32f2f; }
  table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.1); margin-bottom: 2rem; }
  th { background: #1a1a2e; color: white; text-align: left; padding: 0.75rem 1rem; }
  td { padding: 0.75rem 1rem; border-bottom: 1px solid #eee; }
  tr:last-child td { border-bottom: none; }
  .badge { display: inline-block; padding: 0.2rem 0.6rem; border-radius: 4px; font-size: 0.8rem; font-weight: bold; color: white; }
  .badge.critical { background: #d32f2f; }
  .badge.high { background: #f57c00; }
  .badge.medium { background: #fbc02d; color: #333; }
  .badge.low { background: #388e3c; }
  .badge.passed { background: #388e3c; }
  .badge.failed { background: #d32f2f; }
  .timestamp { color: #999; font-size: 0.85rem; margin-bottom: 1rem; }
</style>
</head>
<body>
<h1>cloudmock Compliance Report</h1>
<p class="timestamp">Generated: {{.Timestamp}}</p>

<div class="summary">
  <div class="card"><div class="label">Total Rules</div><div class="value">{{.Summary.TotalRules}}</div></div>
  <div class="card passed"><div class="label">Passed</div><div class="value">{{.Summary.Passed}}</div></div>
  <div class="card failed"><div class="label">Failed</div><div class="value">{{.Summary.Failed}}</div></div>
  <div class="card critical"><div class="label">Critical</div><div class="value">{{.Summary.Critical}}</div></div>
  <div class="card high"><div class="label">High</div><div class="value">{{.Summary.High}}</div></div>
  <div class="card medium"><div class="label">Medium</div><div class="value">{{.Summary.Medium}}</div></div>
  <div class="card low"><div class="label">Low</div><div class="value">{{.Summary.Low}}</div></div>
</div>

<h2>Rules</h2>
<table>
<thead><tr><th>Rule</th><th>Name</th><th>Severity</th><th>Status</th><th>Findings</th></tr></thead>
<tbody>
{{range .Rules}}
<tr>
  <td>{{.ID}}</td>
  <td>{{.Name}}</td>
  <td><span class="badge {{.Severity}}">{{.Severity}}</span></td>
  <td><span class="badge {{.Status}}">{{.Status}}</span></td>
  <td>{{.Findings}}</td>
</tr>
{{end}}
</tbody>
</table>

{{if .Findings}}
<h2>Findings</h2>
<table>
<thead><tr><th>Rule</th><th>Resource</th><th>Type</th><th>Severity</th><th>Message</th><th>Remediation</th></tr></thead>
<tbody>
{{range .Findings}}
<tr>
  <td>{{.RuleID}}</td>
  <td>{{.ResourceID}}</td>
  <td>{{.ResourceType}}</td>
  <td><span class="badge {{.Severity}}">{{.Severity}}</span></td>
  <td>{{.Message}}</td>
  <td>{{.Remediation}}</td>
</tr>
{{end}}
</tbody>
</table>
{{end}}

</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML template: %w", err)
	}

	var buf strings.Builder
	if err := t.Execute(&buf, report); err != nil {
		return nil, fmt.Errorf("executing HTML template: %w", err)
	}
	return []byte(buf.String()), nil
}

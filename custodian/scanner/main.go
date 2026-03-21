// Package main implements the cloudmock-compliance CLI tool.
// It scans a running cloudmock instance for security and compliance issues.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		endpoint string
		format   string
		output   string
		severity string
		services string
	)

	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	scanCmd.StringVar(&endpoint, "endpoint", "http://localhost:4566", "cloudmock gateway endpoint URL")
	scanCmd.StringVar(&format, "format", "json", "output format: json, html, junit")
	scanCmd.StringVar(&output, "output", "", "output file path (default: stdout)")
	scanCmd.StringVar(&severity, "severity", "", "minimum severity to report: critical, high, medium, low")
	scanCmd.StringVar(&services, "services", "", "comma-separated list of services to scan (default: all)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		scanCmd.Parse(os.Args[2:])
	case "list-rules":
		listRules()
		return
	case "version":
		fmt.Println("cloudmock-compliance v0.1.0")
		return
	case "help", "--help", "-h":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// Filter rules by service if specified.
	rules := AllRules()
	if services != "" {
		svcList := strings.Split(services, ",")
		svcSet := make(map[string]bool)
		for _, s := range svcList {
			svcSet[strings.TrimSpace(s)] = true
		}
		var filtered []Rule
		for _, r := range rules {
			if svcSet[r.Service] {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	}

	// Run scan.
	results := RunScan(client, endpoint, rules)

	// Filter by severity if specified.
	if severity != "" {
		results = filterBySeverity(results, severity)
	}

	// Generate report.
	report := GenerateReport(results, rules)

	var reportBytes []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		reportBytes, err = RenderJSON(report)
	case "html":
		reportBytes, err = RenderHTML(report)
	case "junit":
		reportBytes, err = RenderJUnit(report)
	default:
		fmt.Fprintf(os.Stderr, "unsupported format: %s (use json, html, or junit)\n", format)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating report: %v\n", err)
		os.Exit(1)
	}

	if output != "" {
		if err := os.WriteFile(output, reportBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "report written to %s\n", output)
	} else {
		os.Stdout.Write(reportBytes)
	}

	// Exit with non-zero status if there are critical or high findings.
	if report.Summary.Critical > 0 || report.Summary.High > 0 {
		os.Exit(2)
	}
}

func filterBySeverity(results []Finding, minSeverity string) []Finding {
	levels := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
	minLevel := levels[strings.ToLower(minSeverity)]
	var filtered []Finding
	for _, f := range results {
		if levels[strings.ToLower(f.Severity)] >= minLevel {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func listRules() {
	rules := AllRules()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	type ruleInfo struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Severity    string `json:"severity"`
		Service     string `json:"service"`
	}
	var infos []ruleInfo
	for _, r := range rules {
		infos = append(infos, ruleInfo{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Severity:    r.Severity,
			Service:     r.Service,
		})
	}
	enc.Encode(infos)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `cloudmock-compliance — Compliance scanner for cloudmock

Usage:
  cloudmock-compliance scan [flags]     Run compliance scan
  cloudmock-compliance list-rules       List all available rules
  cloudmock-compliance version          Print version
  cloudmock-compliance help             Show this help

Scan flags:
  --endpoint URL    cloudmock gateway endpoint (default: http://localhost:4566)
  --format FORMAT   Output format: json, html, junit (default: json)
  --output FILE     Write report to file instead of stdout
  --severity LEVEL  Minimum severity to report: critical, high, medium, low
  --services LIST   Comma-separated services to scan (default: all)
`)
}

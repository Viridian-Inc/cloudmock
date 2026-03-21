package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"strings"
)

func runReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	format := fs.String("format", "markdown", "output format: markdown|json|junit")
	input := fs.String("input", resultsFile, "path to results file")
	output := fs.String("output", "", "output file (default: stdout)")
	fs.Parse(args)

	data, err := os.ReadFile(*input)
	if err != nil {
		return fmt.Errorf("failed to read results: %w", err)
	}

	var results []RunResult
	if err := json.Unmarshal(data, &results); err != nil {
		return fmt.Errorf("failed to parse results: %w", err)
	}

	var report string
	switch *format {
	case "markdown":
		report = generateMarkdown(results)
	case "json":
		report = string(data) // already JSON
	case "junit":
		report, err = generateJUnit(results)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format: %s", *format)
	}

	if *output != "" {
		return os.WriteFile(*output, []byte(report), 0o644)
	}

	fmt.Print(report)
	return nil
}

func generateMarkdown(results []RunResult) string {
	var b strings.Builder
	b.WriteString("# cloudmock CI Report\n\n")
	b.WriteString("| Tool | Args | Exit Code | Duration | Status |\n")
	b.WriteString("|------|------|-----------|----------|--------|\n")

	passed := 0
	failed := 0
	for _, r := range results {
		status := "PASS"
		if r.ExitCode != 0 {
			status = "FAIL"
			failed++
		} else {
			passed++
		}
		args := strings.Join(r.Args, " ")
		b.WriteString(fmt.Sprintf("| %s | %s | %d | %s | %s |\n",
			r.Tool, args, r.ExitCode, r.Duration, status))
	}

	b.WriteString(fmt.Sprintf("\n**Total: %d** | **Passed: %d** | **Failed: %d**\n",
		len(results), passed, failed))

	return b.String()
}

// JUnit XML types

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Time     string          `xml:"time,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
}

func generateJUnit(results []RunResult) (string, error) {
	failures := 0
	var cases []junitTestCase
	for _, r := range results {
		tc := junitTestCase{
			Name:      fmt.Sprintf("%s %s", r.Tool, strings.Join(r.Args, " ")),
			ClassName: "cloudmock." + r.Tool,
			Time:      r.Duration,
		}
		if r.ExitCode != 0 {
			failures++
			tc.Failure = &junitFailure{
				Message: fmt.Sprintf("exit code %d", r.ExitCode),
				Type:    "ExitCodeFailure",
			}
		}
		cases = append(cases, tc)
	}

	suites := junitTestSuites{
		Suites: []junitTestSuite{
			{
				Name:     "cloudmock",
				Tests:    len(results),
				Failures: failures,
				Time:     "0",
				Cases:    cases,
			},
		},
	}

	out, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(out) + "\n", nil
}

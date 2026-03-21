package main

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitGeneratesCorrectFiles(t *testing.T) {
	systems := []struct {
		ci       string
		contains string
	}{
		{"github", "runs-on:"},
		{"gitlab", "stages:"},
		{"circleci", "version: 2.1"},
		{"bitbucket", "pipelines:"},
		{"buildkite", "steps:"},
		{"codebuild", "version: 0.2"},
		{"travis", "language: minimal"},
	}

	for _, tc := range systems {
		t.Run(tc.ci, func(t *testing.T) {
			mapping, ok := ciFileMapping[tc.ci]
			if !ok {
				t.Fatalf("no mapping for CI system: %s", tc.ci)
			}

			data, err := templateFS.ReadFile(mapping.template)
			if err != nil {
				t.Fatalf("failed to read template for %s: %v", tc.ci, err)
			}

			content := string(data)
			if !strings.Contains(content, tc.contains) {
				t.Errorf("template for %s should contain %q", tc.ci, tc.contains)
			}

			if !strings.Contains(content, "cloudmock") {
				t.Errorf("template for %s should mention cloudmock", tc.ci)
			}
		})
	}
}

func TestInitWritesFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runInit([]string{"--ci", "github"})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	outputPath := filepath.Join(tmpDir, ".github", "workflows", "cloudmock.yml")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("expected file %s to exist", outputPath)
	}
}

func TestReportMarkdown(t *testing.T) {
	results := []RunResult{
		{Tool: "terraform", Args: []string{"plan"}, ExitCode: 0, Duration: "1.5s"},
		{Tool: "terraform", Args: []string{"apply"}, ExitCode: 1, Duration: "3.2s"},
	}

	tmpDir := t.TempDir()
	resultsPath := filepath.Join(tmpDir, "results.json")
	data, _ := json.Marshal(results)
	os.WriteFile(resultsPath, data, 0o644)

	outputPath := filepath.Join(tmpDir, "report.md")
	err := runReport([]string{"--format", "markdown", "--input", resultsPath, "--output", outputPath})
	if err != nil {
		t.Fatalf("runReport failed: %v", err)
	}

	content, _ := os.ReadFile(outputPath)
	md := string(content)

	if !strings.Contains(md, "cloudmock CI Report") {
		t.Error("markdown should contain title")
	}
	if !strings.Contains(md, "terraform") {
		t.Error("markdown should contain tool name")
	}
	if !strings.Contains(md, "PASS") {
		t.Error("markdown should contain PASS")
	}
	if !strings.Contains(md, "FAIL") {
		t.Error("markdown should contain FAIL")
	}
	if !strings.Contains(md, "Passed: 1") {
		t.Error("markdown should show 1 passed")
	}
	if !strings.Contains(md, "Failed: 1") {
		t.Error("markdown should show 1 failed")
	}
}

func TestReportJUnit(t *testing.T) {
	results := []RunResult{
		{Tool: "terraform", Args: []string{"plan"}, ExitCode: 0, Duration: "1.5s"},
		{Tool: "terraform", Args: []string{"apply"}, ExitCode: 1, Duration: "3.2s"},
	}

	tmpDir := t.TempDir()
	resultsPath := filepath.Join(tmpDir, "results.json")
	data, _ := json.Marshal(results)
	os.WriteFile(resultsPath, data, 0o644)

	outputPath := filepath.Join(tmpDir, "report.xml")
	err := runReport([]string{"--format", "junit", "--input", resultsPath, "--output", outputPath})
	if err != nil {
		t.Fatalf("runReport failed: %v", err)
	}

	content, _ := os.ReadFile(outputPath)

	// Validate it's valid XML
	var suites junitTestSuites
	if err := xml.Unmarshal(content, &suites); err != nil {
		t.Fatalf("invalid JUnit XML: %v", err)
	}

	if len(suites.Suites) != 1 {
		t.Fatalf("expected 1 test suite, got %d", len(suites.Suites))
	}

	suite := suites.Suites[0]
	if suite.Tests != 2 {
		t.Errorf("expected 2 tests, got %d", suite.Tests)
	}
	if suite.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", suite.Failures)
	}
	if len(suite.Cases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(suite.Cases))
	}

	// First case should pass (no failure element)
	if suite.Cases[0].Failure != nil {
		t.Error("first test case should not have failure")
	}

	// Second case should fail
	if suite.Cases[1].Failure == nil {
		t.Error("second test case should have failure")
	}

	xmlStr := string(content)
	if !strings.Contains(xmlStr, "<?xml") {
		t.Error("JUnit output should contain XML declaration")
	}
}

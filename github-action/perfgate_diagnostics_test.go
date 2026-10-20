package action_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	perfEnvironmentBefore = "Record performance environment before"
	perfEnvironmentAfter  = "Record performance environment after"
	perfEnvironmentUpload = "Upload performance environment"
)

func readPerfEnvironmentSteps(t *testing.T) []perfGateStep {
	t.Helper()
	data, err := os.ReadFile("../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Steps []perfGateStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	job, ok := workflow.Jobs["perfgate"]
	if !ok {
		t.Fatal("performance gate job missing")
	}
	return job.Steps
}

func findPerfEnvironmentStep(t *testing.T, steps []perfGateStep, name string) (perfGateStep, int) {
	t.Helper()
	var found perfGateStep
	index := -1
	for i, step := range steps {
		if step.Name != name {
			continue
		}
		if index != -1 {
			t.Fatalf("duplicate workflow step %q", name)
		}
		found, index = step, i
	}
	if index == -1 {
		t.Fatalf("workflow step %q missing", name)
	}
	return found, index
}

func TestPerformanceGateDiagnosticWorkflow(t *testing.T) {
	steps := readPerfEnvironmentSteps(t)
	_, startIndex := findPerfEnvironmentStep(t, steps, "Start CloudMock")
	before, beforeIndex := findPerfEnvironmentStep(t, steps, perfEnvironmentBefore)
	_, gateIndex := findPerfEnvironmentStep(t, steps, "Run Performance Gate")
	after, afterIndex := findPerfEnvironmentStep(t, steps, perfEnvironmentAfter)
	upload, uploadIndex := findPerfEnvironmentStep(t, steps, perfEnvironmentUpload)
	if !(startIndex < beforeIndex && beforeIndex < gateIndex && gateIndex < afterIndex && afterIndex < uploadIndex) {
		t.Fatalf("diagnostics must bracket the gate and upload afterward: start=%d before=%d gate=%d after=%d upload=%d",
			startIndex, beforeIndex, gateIndex, afterIndex, uploadIndex)
	}
	for _, step := range []perfGateStep{before, after} {
		if step.Run == "" {
			t.Errorf("%q has no collector command", step.Name)
		}
	}
	for _, step := range []perfGateStep{after, upload} {
		condition := strings.TrimSpace(step.If)
		condition = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(condition, "${{"), "}}"))
		if condition != "always()" {
			t.Errorf("%q must run after a failed gate; if=%q", step.Name, step.If)
		}
	}
	if !strings.HasPrefix(upload.Uses, "actions/upload-artifact@") ||
		upload.With["name"] != "perfgate-environment" ||
		upload.With["path"] != "perfgate-environment.txt" {
		t.Errorf("unexpected environment artifact configuration: uses=%q with=%v", upload.Uses, upload.With)
	}
}

func TestPerformanceGateDiagnostics(t *testing.T) {
	steps := readPerfEnvironmentSteps(t)
	before, _ := findPerfEnvironmentStep(t, steps, perfEnvironmentBefore)
	after, _ := findPerfEnvironmentStep(t, steps, perfEnvironmentAfter)
	collector, err := os.ReadFile("../.github/scripts/perfgate-environment.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		lscpu  string
		marker string
		status string
	}{
		{
			name:   "append-snapshots",
			lscpu:  "#!/bin/sh\nprintf 'diagnostics-fixture-cpu\\n'\n",
			marker: "diagnostics-fixture-cpu",
			status: "[exit=0]",
		},
		{
			name:   "unavailable-optional-tool",
			lscpu:  "#!/bin/sh\nprintf 'diagnostics-fixture-lscpu-unavailable\\n' >&2\nexit 127\n",
			marker: "diagnostics-fixture-lscpu-unavailable",
			status: "[unavailable exit=127]",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspace := t.TempDir()
			scriptDir := filepath.Join(workspace, ".github", "scripts")
			binDir := filepath.Join(workspace, "bin")
			for _, dir := range []string{scriptDir, binDir} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(scriptDir, "perfgate-environment.sh"), collector, 0o644); err != nil {
				t.Fatal(err)
			}
			for name, script := range map[string]string{
				"go":    "#!/bin/sh\nprintf 'go version diagnostics-fixture linux/amd64\\n'\n",
				"lscpu": tc.lscpu,
			} {
				if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			environment := make([]string, 0, len(os.Environ())+1)
			for _, entry := range os.Environ() {
				name, _, _ := strings.Cut(entry, "=")
				if name != "PATH" && name != "BASH_ENV" {
					environment = append(environment, entry)
				}
			}
			environment = append(environment, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			run := func(step perfGateStep) string {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				// Execute the checked-in step, including its output redirection,
				// rather than appending collector output from the test itself.
				cmd := exec.CommandContext(ctx, "bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", step.Run)
				cmd.Dir, cmd.Env = workspace, environment
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("%s failed: %v\n%s", step.Name, err, output)
				}
				data, err := os.ReadFile(filepath.Join(workspace, "perfgate-environment.txt"))
				if err != nil {
					t.Fatal(err)
				}
				return string(data)
			}
			assertSnapshot := func(snapshot, phase string) {
				t.Helper()
				header, _, _ := strings.Cut(strings.TrimSpace(snapshot), "\n")
				prefix := "=== phase=" + phase + " utc="
				if !strings.HasPrefix(header, prefix) || !strings.HasSuffix(header, " ===") {
					t.Fatalf("%s snapshot has no labeled UTC header: %q", phase, header)
				}
				timestamp := strings.TrimSuffix(strings.TrimPrefix(header, prefix), " ===")
				if _, err := time.Parse("2006-01-02T15:04:05Z", timestamp); err != nil {
					t.Fatalf("%s snapshot has invalid UTC timestamp %q: %v", phase, timestamp, err)
				}
				section := "--- CPU model and topology ---\n" + tc.marker + "\n" + tc.status
				if !strings.Contains(snapshot, section) || !strings.Contains(snapshot, "go version diagnostics-fixture") {
					t.Fatalf("%s snapshot lost labeled tool output or status: %s", phase, snapshot)
				}
			}
			first := run(before)
			assertSnapshot(first, "before")
			both := run(after)
			if !strings.HasPrefix(both, first) || len(both) == len(first) {
				t.Fatalf("after step did not append a snapshot while retaining before")
			}
			second := strings.TrimPrefix(both, first)
			assertSnapshot(second, "after")
		})
	}
}

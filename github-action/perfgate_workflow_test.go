package action_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type perfGateStep struct {
	Name  string            `yaml:"name"`
	Run   string            `yaml:"run"`
	Shell string            `yaml:"shell"`
	Uses  string            `yaml:"uses"`
	If    string            `yaml:"if"`
	Env   map[string]string `yaml:"env"`
	With  map[string]string `yaml:"with"`
}

type perfGateDefaults struct {
	Run struct {
		Shell string `yaml:"shell"`
	} `yaml:"run"`
}

func TestPerformanceGateWorkflowPropagatesStatus(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Defaults perfGateDefaults `yaml:"defaults"`
		Jobs     map[string]struct {
			Defaults perfGateDefaults `yaml:"defaults"`
			Steps    []perfGateStep   `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	job, ok := workflow.Jobs["perfgate"]
	if !ok {
		t.Fatal("performance gate job missing")
	}
	var gate perfGateStep
	var uploads int
	for _, step := range job.Steps {
		if step.Name == "Run Performance Gate" {
			gate = step
		}
		if strings.HasPrefix(step.Uses, "actions/upload-artifact@") && step.With["path"] == "perfgate-results.txt" {
			uploads++
			condition := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(step.If, "${{"), "}}"))
			if condition != "always()" {
				t.Errorf("performance results must upload after failure; if=%q", step.If)
			}
		}
	}
	if gate.Run == "" {
		t.Fatal("Run Performance Gate step missing")
	}
	if uploads != 1 {
		t.Errorf("want one unconditional performance output artifact, got %d", uploads)
	}
	shell := gate.Shell
	if shell == "" {
		shell = job.Defaults.Run.Shell
	}
	if shell == "" {
		shell = workflow.Defaults.Run.Shell
	}
	// Match the Linux runner: an unspecified shell is bash -e, whereas an
	// explicitly selected Bash shell also disables profiles and sets pipefail.
	var shellArgs []string
	switch shell {
	case "":
		shellArgs = []string{"-e"}
	case "bash":
		shellArgs = []string{"--noprofile", "--norc", "-e", "-o", "pipefail"}
	default:
		t.Fatalf("unsupported performance gate shell %q", shell)
	}
	for _, tc := range []struct {
		name      string
		stdout    string
		stderr    string
		status    string
		code      int
		teeStatus string
	}{
		{"pass", "s3 p50=2ms\n--- PASS: TestPerfGate (0.01s)\nPASS\n", "", "0", 0, ""},
		{"test-failure", "s3 p50=900ms\n--- FAIL: TestPerfGate (0.01s)\nFAIL\n", "", "1", 1, ""},
		{"compilation-error-without-summary", "", "# example/benchmarks\n./perfgate_test.go:9: undefined: missingSymbol\n", "2", 2, ""},
		{"timeout", "=== RUN   TestPerfGate\n", "panic: test timed out after 2m0s\n", "124", 124, ""},
		{"success-without-summary", "ok example/benchmarks 0.01s\n", "", "0", 0, ""},
		{"tee-failure", "PASS\n", "", "0", 74, "74"},
		{"test-and-tee-failure", "FAIL\n", "", "1", 74, "74"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspace := t.TempDir()
			binDir := filepath.Join(workspace, "bin")
			if err := os.Mkdir(binDir, 0o755); err != nil {
				t.Fatal(err)
			}
			// Replace Go while keeping shell and summary tools real. The tee
			// failure cases below additionally inject an output-write status.
			fakeGo := `#!/bin/bash
printf '%s\0' "$@" > "$CM_PERFGATE_ARGS"
printf '%s' "$CM_PERFGATE_STDOUT"
printf '%s' "$CM_PERFGATE_STDERR" >&2
exit "$CM_PERFGATE_STATUS"
`
			if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(fakeGo), 0o755); err != nil {
				t.Fatal(err)
			}
			script := filepath.Join(workspace, "gate step.sh")
			if err := os.WriteFile(script, []byte(gate.Run), 0o600); err != nil {
				t.Fatal(err)
			}
			argsFile := filepath.Join(workspace, "go-args")
			env := map[string]string{
				"PATH":             binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
				"CM_PERFGATE_ARGS": argsFile, "CM_PERFGATE_STDOUT": tc.stdout,
				"CM_PERFGATE_STDERR": tc.stderr, "CM_PERFGATE_STATUS": tc.status,
			}
			for name, value := range gate.Env {
				env[name] = value
			}
			if tc.teeStatus != "" {
				// Preserve the output while simulating tee reporting an I/O error.
				// pipefail must retain the rightmost failing command's status.
				realTee, err := exec.LookPath("tee")
				if err != nil {
					t.Fatal(err)
				}
				env["CM_PERFGATE_REAL_TEE"], env["CM_PERFGATE_TEE_STATUS"] = realTee, tc.teeStatus
				fakeTee := "#!/bin/bash\n\"$CM_PERFGATE_REAL_TEE\" \"$@\"\nexit \"$CM_PERFGATE_TEE_STATUS\"\n"
				if err := os.WriteFile(filepath.Join(binDir, "tee"), []byte(fakeTee), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			var processEnv []string
			for _, entry := range os.Environ() {
				name, _, _ := strings.Cut(entry, "=")
				if _, replaced := env[name]; !replaced && name != "BASH_ENV" && !strings.HasPrefix(name, "CM_PERFGATE_") {
					processEnv = append(processEnv, entry)
				}
			}
			for name, value := range env {
				processEnv = append(processEnv, name+"="+value)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			args := append(append([]string(nil), shellArgs...), script)
			cmd := exec.CommandContext(ctx, "bash", args...)
			cmd.Dir, cmd.Env = workspace, processEnv
			output, runErr := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("workflow step did not finish: %v", ctx.Err())
			}
			code := 0
			if runErr != nil {
				if exit, ok := runErr.(*exec.ExitError); ok {
					code = exit.ExitCode()
				} else {
					t.Fatalf("execute workflow step: %v", runErr)
				}
			}
			if code != tc.code {
				t.Errorf("workflow exit=%d, want pipeline status %d; output:\n%s", code, tc.code, output)
			}
			if !strings.Contains(string(output), "=== Performance Gate Summary ===") {
				t.Errorf("workflow skipped the diagnostic summary: %s", output)
			}
			artifact, err := os.ReadFile(filepath.Join(workspace, "perfgate-results.txt"))
			if err != nil {
				t.Fatalf("performance output artifact missing: %v", err)
			}
			if string(artifact) != tc.stdout+tc.stderr {
				t.Errorf("performance artifact lost diagnostics: got %q want %q", artifact, tc.stdout+tc.stderr)
			}
			captured, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			gotArgs := strings.Split(strings.TrimSuffix(string(captured), "\x00"), "\x00")
			// The external gateway and runner load are not Go test cache inputs.
			// Force execution so a previous successful run cannot satisfy this gate.
			wantArgs := []string{"test", "./benchmarks/", "-tags", "perfgate", "-count=1", "-timeout", "120s", "-v"}
			if !reflect.DeepEqual(gotArgs, wantArgs) {
				t.Errorf("benchmark arguments changed: got %q want %q", gotArgs, wantArgs)
			}
		})
	}
}

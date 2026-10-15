package action_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

const sourceInstallSHA = "0123456789abcdef0123456789abcdef01234567"

type installToolCall struct {
	Tool string   `json:"tool"`
	Dir  string   `json:"dir"`
	Args []string `json:"args"`
}

// Every installer subprocess reaches these shims, so a regression cannot
// install a published package or compile the gateway during this unit test.
func TestSourceInstallTool(t *testing.T) {
	tool := os.Getenv("CM_INSTALL_MOCK_TOOL")
	if tool == "" {
		return
	}
	var args []string
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	call, err := json.Marshal(installToolCall{Tool: tool, Dir: dir, Args: args})
	if err != nil {
		t.Fatal(err)
	}
	capture, err := os.OpenFile(os.Getenv("CM_INSTALL_CAPTURE"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := capture.Write(append(call, '\n')); err != nil {
		t.Fatal(err)
	}
	if err := capture.Close(); err != nil {
		t.Fatal(err)
	}
	switch tool {
	case "git":
		if len(args) < 2 || !reflect.DeepEqual(args[len(args)-2:], []string{"rev-parse", "HEAD"}) {
			t.Fatalf("unexpected git operation: %q", args)
		}
		fmt.Println(sourceInstallSHA)
	case "go":
		if len(args) == 0 || args[0] != "build" {
			t.Fatalf("unexpected go operation: %q", args)
		}
		if os.Getenv("CM_INSTALL_BUILD_FAIL") == "1" {
			os.Exit(23)
		}
		var output string
		for i, arg := range args {
			if arg == "-o" && i+1 < len(args) {
				output = args[i+1]
			}
		}
		if output == "" {
			t.Fatalf("go build did not set output: %q", args)
		}
		binary := "#!/bin/bash\nprintf '%s\\n' 'source-" + sourceInstallSHA + "'\n"
		if err := os.WriteFile(output, []byte(binary), 0o755); err != nil {
			t.Fatal(err)
		}
	case "npm":
		if len(args) == 0 {
			t.Fatal("npm invoked without arguments")
		}
		if args[0] == "list" {
			fmt.Println("cloudmock@9.8.7")
		} else if args[0] != "install" {
			t.Fatalf("unexpected npm operation: %q", args)
		}
	default:
		t.Fatalf("unexpected mock tool %q", tool)
	}
	os.Exit(0)
}

type sourceInstallResult struct {
	Calls      []installToolCall
	Outputs    map[string]string
	Path       string
	Workspace  string
	Binary     string
	RunOutput  string
	RunError   error
	ProcessEnv []string
}

func runSourceInstaller(t *testing.T, overrides map[string]string, failBuild bool) sourceInstallResult {
	t.Helper()
	action := readAction(t, "../.github/actions/cloudmock/action.yml")
	inputs := make(map[string]string)
	for name, input := range action.Inputs {
		inputs[name] = input.Default
	}
	for name, value := range overrides {
		inputs[name] = value
	}
	root := t.TempDir()
	workspace := filepath.Join(root, "checkout with spaces")
	runnerTemp := filepath.Join(root, "runner temp")
	binDir := filepath.Join(root, "tools")
	for _, dir := range []string{workspace, runnerTemp, binDir, filepath.Join(workspace, "cmd", "gateway")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.invalid/source-fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"go", "git", "npm"} {
		shim := "#!/bin/bash\nexport CM_INSTALL_MOCK_TOOL=" + tool + "\nexec \"$CM_INSTALL_TEST_EXECUTABLE\" -test.run=^TestSourceInstallTool$ -- \"$@\"\n"
		if err := os.WriteFile(filepath.Join(binDir, tool), []byte(shim), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	capture := filepath.Join(root, "tool-calls.jsonl")
	outputFile := filepath.Join(root, "github-output")
	pathFile := filepath.Join(root, "github-path")
	env := map[string]string{
		"PATH":        binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"RUNNER_TEMP": runnerTemp, "GITHUB_WORKSPACE": workspace,
		"GITHUB_OUTPUT": outputFile, "GITHUB_PATH": pathFile,
		"CM_INSTALL_TEST_EXECUTABLE": executable, "CM_INSTALL_CAPTURE": capture,
	}
	if failBuild {
		env["CM_INSTALL_BUILD_FAIL"] = "1"
	}
	var install *actionStep
	for i := range action.Runs.Steps {
		if action.Runs.Steps[i].ID == "install" {
			install = &action.Runs.Steps[i]
			break
		}
	}
	if install == nil {
		t.Fatal("local action has no install step")
	}
	expand := func(value string) string {
		return expression.ReplaceAllStringFunc(value, func(match string) string {
			name := strings.TrimSpace(expression.FindStringSubmatch(match)[1])
			switch {
			case strings.HasPrefix(name, "inputs."):
				return inputs[strings.TrimPrefix(name, "inputs.")]
			case name == "github.workspace":
				return workspace
			case name == "runner.temp":
				return runnerTemp
			default:
				t.Fatalf("unsupported installer expression %q", name)
				return ""
			}
		})
	}
	for name, value := range install.Env {
		env[name] = expand(value)
	}
	var processEnv []string
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		if _, replaced := env[name]; !replaced && !strings.HasPrefix(name, "CM_") && !strings.HasPrefix(name, "GITHUB_") {
			processEnv = append(processEnv, value)
		}
	}
	for name, value := range env {
		processEnv = append(processEnv, name+"="+value)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", expand(install.Run))
	// Deliberately start elsewhere: the installer must select the checkout.
	cmd.Dir, cmd.Env = root, processEnv
	output, runErr := cmd.CombinedOutput()
	result := sourceInstallResult{Workspace: workspace, Binary: filepath.Join(runnerTemp, "cloudmock-source", "cloudmock"), RunOutput: string(output), RunError: runErr, ProcessEnv: processEnv, Outputs: make(map[string]string)}
	if data, err := os.ReadFile(capture); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var call installToolCall
			if err := json.Unmarshal([]byte(line), &call); err != nil {
				t.Fatal(err)
			}
			result.Calls = append(result.Calls, call)
		}
	}
	if _, err := os.Stat(outputFile); err == nil {
		result.Outputs = readRunnerFile(t, outputFile)
	}
	if data, err := os.ReadFile(pathFile); err == nil {
		result.Path = strings.TrimSpace(string(data))
	}
	return result
}

func TestActionSourceInstaller(t *testing.T) {
	action := readAction(t, "../.github/actions/cloudmock/action.yml")
	if source, ok := action.Inputs["source"]; !ok || source.Default != "false" {
		t.Errorf("source input must explicitly default to false; got %+v, exists=%t", source, ok)
	}
	data, err := os.ReadFile("../.github/actions/cloudmock/action.yml")
	if err != nil {
		t.Fatal(err)
	}
	var declaration struct {
		Outputs map[string]struct {
			Value string `yaml:"value"`
		} `yaml:"outputs"`
	}
	if err := yaml.Unmarshal(data, &declaration); err != nil {
		t.Fatal(err)
	}
	if value := declaration.Outputs["source-sha"].Value; !strings.Contains(value, "steps.install.outputs.source-sha") {
		t.Errorf("source-sha action output does not expose installer provenance: %q", value)
	}
	t.Run("checkout-build", func(t *testing.T) {
		result := runSourceInstaller(t, map[string]string{"source": "true"}, false)
		if result.RunError != nil {
			t.Fatalf("installer failed: %v\n%s", result.RunError, result.RunOutput)
		}
		var builds, revisions int
		for _, call := range result.Calls {
			switch call.Tool {
			case "npm":
				t.Error("source install invoked npm")
			case "go":
				builds++
				if call.Dir != result.Workspace {
					t.Errorf("build cwd=%q, want checkout %q", call.Dir, result.Workspace)
				}
				if len(call.Args) == 0 || call.Args[len(call.Args)-1] != "./cmd/gateway" {
					t.Errorf("build must target checked-out gateway: %q", call.Args)
				}
				foundOutput := false
				for i, arg := range call.Args {
					if arg == "-o" && i+1 < len(call.Args) && call.Args[i+1] == result.Binary {
						foundOutput = true
					}
				}
				if !foundOutput {
					t.Errorf("build did not target isolated source binary: %q", call.Args)
				}
			case "git":
				revisions++
				if call.Dir != result.Workspace && !reflect.DeepEqual(call.Args, []string{"-C", result.Workspace, "rev-parse", "HEAD"}) {
					t.Errorf("revision must come from checkout: cwd=%q args=%q", call.Dir, call.Args)
				}
			}
		}
		if builds != 1 || revisions != 1 {
			t.Errorf("want one source build and revision lookup; got builds=%d revisions=%d", builds, revisions)
		}
		if result.Outputs["source-sha"] != sourceInstallSHA || !strings.Contains(result.Outputs["version"], sourceInstallSHA) {
			t.Errorf("source provenance not emitted: %v", result.Outputs)
		}
		if result.Path != filepath.Dir(result.Binary) {
			t.Errorf("GITHUB_PATH=%q, want isolated source binary directory %q", result.Path, filepath.Dir(result.Binary))
		}
		if _, err := os.Stat(result.Binary); err != nil {
			t.Errorf("source binary missing: %v", err)
		} else if result.Path == filepath.Dir(result.Binary) {
			// Model the runner applying GITHUB_PATH for the next action step.
			// The unchanged start step must resolve the source-built executable.
			processEnv := append([]string(nil), result.ProcessEnv...)
			for i, entry := range processEnv {
				if value, ok := strings.CutPrefix(entry, "PATH="); ok {
					processEnv[i] = "PATH=" + result.Path + string(os.PathListSeparator) + value
				}
			}
			cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", "command -v cloudmock; cloudmock --version")
			cmd.Env = processEnv
			output, err := cmd.CombinedOutput()
			want := result.Binary + "\nsource-" + sourceInstallSHA + "\n"
			if err != nil || string(output) != want {
				t.Errorf("source binary is not selected by the runner PATH: err=%v output=%q want=%q", err, output, want)
			}
		}
	})
	t.Run("build-failure-propagates", func(t *testing.T) {
		result := runSourceInstaller(t, map[string]string{"source": "true"}, true)
		if result.RunError == nil {
			t.Error("failed source build was reported as successful")
		}
		if result.Path != "" || result.Outputs["version"] != "" {
			t.Errorf("failed build advertised an installed binary: path=%q outputs=%v", result.Path, result.Outputs)
		}
		for _, call := range result.Calls {
			if call.Tool == "npm" {
				t.Error("failed source build fell back to npm")
			}
		}
	})
	t.Run("released-default", func(t *testing.T) {
		assertReleasedInstall(t, nil, "cloudmock")
	})
	t.Run("released-version-is-data", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "must-not-run")
		version := "1.2.3-$(touch " + marker + ")"
		assertReleasedInstall(t, map[string]string{"source": "false", "version": version}, "cloudmock@"+version)
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Errorf("version input executed as shell code: %v", err)
		}
	})
}

func assertReleasedInstall(t *testing.T, inputs map[string]string, packageSpec string) {
	t.Helper()
	result := runSourceInstaller(t, inputs, false)
	if result.RunError != nil {
		t.Fatalf("released installer failed: %v\n%s", result.RunError, result.RunOutput)
	}
	var installs int
	for _, call := range result.Calls {
		if call.Tool != "npm" {
			t.Errorf("released install invoked source tool %q", call.Tool)
		}
		if len(call.Args) > 0 && call.Args[0] == "install" {
			installs++
			if !reflect.DeepEqual(call.Args, []string{"install", "-g", packageSpec}) {
				t.Errorf("npm install arguments=%q", call.Args)
			}
		}
	}
	if installs != 1 || result.Outputs["version"] != "9.8.7" {
		t.Errorf("released install was not preserved: installs=%d outputs=%v", installs, result.Outputs)
	}
	if result.Outputs["source-sha"] != "" || result.Path != "" {
		t.Errorf("released install advertised source provenance: outputs=%v path=%q", result.Outputs, result.Path)
	}
}

type compatibilityStep struct {
	Name            string            `yaml:"name"`
	Uses            string            `yaml:"uses"`
	ID              string            `yaml:"id"`
	If              string            `yaml:"if"`
	Run             string            `yaml:"run"`
	Shell           string            `yaml:"shell"`
	With            map[string]string `yaml:"with"`
	Env             map[string]string `yaml:"env"`
	ContinueOnError bool              `yaml:"continue-on-error"`
}

type compatibilityDefaults struct {
	Run struct {
		Shell string `yaml:"shell"`
	} `yaml:"run"`
}

func TestCompatibilityWorkflowUsesCheckout(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/compat.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		On       map[string]any        `yaml:"on"`
		Defaults compatibilityDefaults `yaml:"defaults"`
		Jobs     map[string]struct {
			Steps    []compatibilityStep   `yaml:"steps"`
			Defaults compatibilityDefaults `yaml:"defaults"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	if _, ok := workflow.On["pull_request"]; !ok {
		t.Error("compatibility must run on pull requests")
	}
	for _, jobName := range []string{"sdk-compat", "terraform-compat", "open-source-compat"} {
		t.Run(jobName, func(t *testing.T) {
			job, ok := workflow.Jobs[jobName]
			if !ok {
				t.Fatal("compatibility job missing")
			}
			var sources int
			var sourceID string
			var recordsRevision bool
			var baselineReports int
			for _, step := range job.Steps {
				if step.Uses == "./.github/actions/cloudmock" {
					sources++
					sourceID = step.ID
					if step.With["source"] != "true" {
						t.Errorf("compatibility action must opt into checked-out source: with=%v", step.With)
					}
					if step.With["version"] != "" {
						t.Errorf("compatibility source job selects a published version: %v", step.With)
					}
				}
				if step.Run != "" && (step.ContinueOnError || strings.Contains(step.Run, "|| true")) {
					t.Errorf("step %q masks command failures", step.Name)
				}
				if strings.Contains(step.Run, " | ") && !strings.Contains(step.Run, "pipefail") {
					shell := step.Shell
					if shell == "" {
						shell = job.Defaults.Run.Shell
					}
					if shell == "" {
						shell = workflow.Defaults.Run.Shell
					}
					if shell != "bash" {
						t.Errorf("step %q has a pipeline without Bash pipefail", step.Name)
					}
				}
				if strings.Contains(step.Run, "git push") && (!strings.Contains(step.If, "github.event_name") || !strings.Contains(step.If, "schedule") || !strings.Contains(step.If, "github.ref")) {
					t.Errorf("report publication must be limited to trusted scheduled/default-branch runs: %q", step.If)
				}
			}
			if sources != 1 || sourceID == "" {
				t.Errorf("want one identifiable source action; got count=%d id=%q", sources, sourceID)
			}
			for _, step := range job.Steps {
				if jobName == "sdk-compat" && strings.Contains(step.Run, "./benchmarks/compat/cmd/") && strings.Contains(step.Run, "--baseline ") {
					baselineReports++
					if !strings.Contains(step.Run, `--source-sha "$CM_SOURCE_SHA"`) || !strings.Contains(step.Env["CM_SOURCE_SHA"], "steps."+sourceID+".outputs.source-sha") {
						t.Errorf("baseline comparison report must receive the installed source SHA: step=%q", step.Name)
					}
				}
				if strings.Contains(step.Run, "steps."+sourceID+".outputs.source-sha") {
					recordsRevision = true
				}
				for _, value := range step.Env {
					if strings.Contains(value, "steps."+sourceID+".outputs.source-sha") && step.Run != "" {
						recordsRevision = true
					}
				}
			}
			if !recordsRevision {
				t.Error("workflow does not record the installed source SHA in a run step")
			}
			if jobName == "sdk-compat" && baselineReports != 1 {
				t.Errorf("SDK report must invoke the compatibility CLI with --baseline exactly once; got %d", baselineReports)
			}
		})
	}
}

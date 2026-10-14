package action_test

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type actionStep struct {
	ID    string            `yaml:"id"`
	Shell string            `yaml:"shell"`
	Env   map[string]string `yaml:"env"`
	Run   string            `yaml:"run"`
}

type actionDefinition struct {
	Inputs map[string]struct {
		Default string `yaml:"default"`
	} `yaml:"inputs"`
	Runs struct {
		Steps []actionStep `yaml:"steps"`
	} `yaml:"runs"`
}

type actionInvocation struct {
	Environment map[string]string `json:"environment"`
	Args        []string          `json:"args"`
}

type actionResult struct {
	Outputs     map[string]string
	Environment map[string]string
	Invocation  actionInvocation
}

var actionPaths = []string{"action.yml", "../.github/actions/cloudmock/action.yml"}
var expression = regexp.MustCompile(`\$\{\{\s*([^}]+?)\s*\}\}`)
var forwardedEnvironment = []string{
	"CLOUDMOCK_GATEWAY_PORT", "CLOUDMOCK_PROFILE", "CLOUDMOCK_IAM_MODE",
	"CLOUDMOCK_SERVICES", "CLOUDMOCK_STATE", "CLOUDMOCK_TEST_MODE", "CLOUDMOCK_IAC_DIR",
}

func readAction(t *testing.T, path string) actionDefinition {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var action actionDefinition
	if err := yaml.Unmarshal(data, &action); err != nil {
		t.Fatal(err)
	}
	return action
}

// TestActionMockCloudMock is invoked only by the cloudmock executable shim.
// Its capture is also the readiness signal for the mock HTTP health endpoint.
func TestActionMockCloudMock(t *testing.T) {
	if os.Getenv("CLOUDMOCK_ACTION_MOCK") != "1" {
		return
	}
	invocation := actionInvocation{Environment: make(map[string]string), Args: []string{}}
	for _, name := range forwardedEnvironment {
		invocation.Environment[name] = os.Getenv(name)
	}
	for i, arg := range os.Args {
		if arg == "--" {
			invocation.Args = os.Args[i+1:]
			break
		}
	}
	data, err := json.Marshal(invocation)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("CLOUDMOCK_ACTION_CAPTURE"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	os.Exit(0)
}

func runAction(t *testing.T, path string, overrides map[string]string, binary string, runtimeEnvironment ...map[string]string) actionResult {
	t.Helper()
	action := readAction(t, path)
	inputs := make(map[string]string)
	for name, input := range action.Inputs {
		inputs[name] = input.Default
	}
	for name, value := range overrides {
		inputs[name] = value
	}
	runnerTemp := t.TempDir()
	binDir := filepath.Join(runnerTemp, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(runnerTemp, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	capturePath := filepath.Join(runnerTemp, "invocation.json")
	pidPath := filepath.Join(runnerTemp, "launched.pid")
	outputPath := filepath.Join(runnerTemp, "github-output")
	envPath := filepath.Join(runnerTemp, "github-env")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var projectDir string
	if binary != "" {
		// Use the supported project namespace to keep startup persistence away
		// from the developer's shared metadata, without changing HOME.
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		projects := filepath.Join(home, ".cloudmock", "projects")
		if err := os.MkdirAll(projects, 0o755); err != nil {
			t.Fatal(err)
		}
		projectDir, err = os.MkdirTemp(projects, "action-smoke-")
		if err != nil {
			t.Fatal(err)
		}
		// A local empty topology seed prevents the fallback to a home seed.
		if err := os.WriteFile(filepath.Join(runnerTemp, "topology-seed.json"), []byte(`{"nodes":[],"edges":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	wrapper := "#!/bin/bash\nprintf '%s\\n' \"$$\" > \"$CLOUDMOCK_ACTION_PID_FILE\"\n"
	if binary == "" {
		wrapper += "exec \"$CLOUDMOCK_ACTION_EXECUTABLE\" -test.run=^TestActionMockCloudMock$ -- \"$@\"\n"
	} else {
		wrapper += "exec \"$CLOUDMOCK_ACTION_TEST_BINARY\" --plugin-dir \"$RUNNER_TEMP/plugins\" --project \"$CLOUDMOCK_ACTION_PROJECT\" \"$@\"\n"
	}
	if err := os.WriteFile(filepath.Join(binDir, "cloudmock"), []byte(wrapper), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			if log, err := os.ReadFile(filepath.Join(runnerTemp, "cloudmock.log")); err == nil {
				t.Logf("%s gateway log:\n%s", path, log)
			}
		}
		stopped := true
		data, err := os.ReadFile(pidPath)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				t.Errorf("invalid owned process PID: %v", err)
				stopped = false
			} else if process, err := os.FindProcess(pid); err == nil {
				_ = process.Kill()
				deadline := time.Now().Add(5 * time.Second)
				for process.Signal(syscall.Signal(0)) == nil && time.Now().Before(deadline) {
					time.Sleep(10 * time.Millisecond)
				}
				stopped = process.Signal(syscall.Signal(0)) != nil
				if !stopped {
					t.Errorf("owned gateway process %d did not stop; preserving project %s", pid, projectDir)
				}
			}
		}
		if projectDir != "" && stopped {
			if err := os.RemoveAll(projectDir); err != nil {
				t.Errorf("remove isolated gateway project: %v", err)
			}
		}
	})

	baseEnv := make(map[string]string)
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "CLOUDMOCK_") || strings.HasPrefix(name, "AWS_") || strings.HasPrefix(name, "GITHUB_") || strings.HasPrefix(name, "INPUT_") {
			continue
		}
		baseEnv[name] = value
	}
	baseEnv["PATH"] = binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	baseEnv["RUNNER_TEMP"] = runnerTemp
	baseEnv["GITHUB_OUTPUT"] = outputPath
	baseEnv["GITHUB_ENV"] = envPath
	baseEnv["CLOUDMOCK_ACTION_PID_FILE"] = pidPath
	baseEnv["CLOUDMOCK_ACTION_CAPTURE"] = capturePath
	baseEnv["CLOUDMOCK_ACTION_EXECUTABLE"] = executable
	baseEnv["CLOUDMOCK_ACTION_TEST_BINARY"] = binary
	baseEnv["CLOUDMOCK_ACTION_PROJECT"] = filepath.Base(projectDir)
	for _, additions := range runtimeEnvironment {
		for name, value := range additions {
			baseEnv[name] = value
		}
	}
	if binary == "" {
		baseEnv["CLOUDMOCK_ACTION_MOCK"] = "1"
		health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if _, err := os.Stat(capturePath); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(health.Close)
		healthURL, err := url.Parse(health.URL)
		if err != nil {
			t.Fatal(err)
		}
		curl, err := exec.LookPath("curl")
		if err != nil {
			t.Fatal(err)
		}
		baseEnv["CLOUDMOCK_ACTION_CURL"] = curl
		baseEnv["CLOUDMOCK_ACTION_HEALTH_ADDRESS"] = healthURL.Host
		baseEnv["CLOUDMOCK_ACTION_EXPECTED_PORT"] = inputs["port"]
		curlShim := "#!/bin/bash\nexec \"$CLOUDMOCK_ACTION_CURL\" --noproxy '*' --connect-to \"localhost:$CLOUDMOCK_ACTION_EXPECTED_PORT:$CLOUDMOCK_ACTION_HEALTH_ADDRESS\" --connect-to \"localhost:4599:$CLOUDMOCK_ACTION_HEALTH_ADDRESS\" \"$@\"\n"
		if err := os.WriteFile(filepath.Join(binDir, "curl"), []byte(curlShim), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	outputs := make(map[string]string)
	expand := func(value string) string {
		return expression.ReplaceAllStringFunc(value, func(match string) string {
			name := strings.TrimSpace(expression.FindStringSubmatch(match)[1])
			switch {
			case strings.HasPrefix(name, "inputs."):
				return inputs[strings.TrimPrefix(name, "inputs.")]
			case name == "runner.temp":
				return runnerTemp
			case strings.HasPrefix(name, "steps.start.outputs."):
				return outputs[strings.TrimPrefix(name, "steps.start.outputs.")]
			default:
				t.Fatalf("unsupported action expression %q", name)
				return ""
			}
		})
	}
	started, exported := false, false
	for _, step := range action.Runs.Steps {
		if step.ID != "start" && !strings.Contains(step.Run, "GITHUB_ENV") {
			continue
		}
		if step.Shell != "bash" {
			t.Fatalf("unsupported shell %q", step.Shell)
		}
		env := make(map[string]string)
		for name, value := range baseEnv {
			env[name] = value
		}
		for name, value := range step.Env {
			env[name] = expand(value)
		}
		var processEnv []string
		for name, value := range env {
			processEnv = append(processEnv, name+"="+value)
		}
		script := expand(step.Run)
		// Isolate legacy action bookkeeping without modifying its commands or
		// input handling; newer scripts use RUNNER_TEMP directly.
		for _, suffix := range []string{"log", "pid"} {
			script = strings.ReplaceAll(script, "/tmp/cloudmock."+suffix, filepath.Join(runnerTemp, "cloudmock."+suffix))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		cmd := exec.CommandContext(ctx, "bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
		cmd.Dir, cmd.Env = runnerTemp, processEnv
		output, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			log, _ := os.ReadFile(filepath.Join(runnerTemp, "cloudmock.log"))
			t.Fatalf("%s step %q failed: %v\n%s\ncloudmock log:\n%s", path, step.ID, err, output, log)
		}
		if step.ID == "start" {
			started = true
			outputs = readRunnerFile(t, outputPath)
		}
		if strings.Contains(step.Run, "GITHUB_ENV") {
			exported = true
		}
	}
	if !started || !exported {
		t.Fatalf("action did not execute both start and environment export steps")
	}
	result := actionResult{Outputs: outputs, Environment: readRunnerFile(t, envPath)}
	if binary == "" {
		data, err := os.ReadFile(capturePath)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &result.Invocation); err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func readRunnerFile(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			t.Fatalf("invalid runner file entry %q", scanner.Text())
		}
		values[name] = value
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return values
}

func assertActionOutputs(t *testing.T, result actionResult, port string) {
	t.Helper()
	endpoint := "http://localhost:" + port
	if result.Outputs["endpoint"] != endpoint {
		t.Errorf("endpoint=%q, want %q", result.Outputs["endpoint"], endpoint)
	}
	if result.Outputs["admin-url"] != "http://localhost:4599" {
		t.Errorf("unexpected admin-url=%q", result.Outputs["admin-url"])
	}
	wantEnv := map[string]string{"AWS_ENDPOINT_URL": endpoint, "AWS_ACCESS_KEY_ID": "test", "AWS_SECRET_ACCESS_KEY": "test", "AWS_DEFAULT_REGION": "us-east-1"}
	for name, value := range wantEnv {
		if result.Environment[name] != value {
			t.Errorf("exported %s=%q, want %q", name, result.Environment[name], value)
		}
	}
}

func TestActionInputForwarding(t *testing.T) {
	for _, path := range actionPaths {
		for _, custom := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/custom-%t", path, custom), func(t *testing.T) {
				action := readAction(t, path)
				inputs := make(map[string]string)
				for name, input := range action.Inputs {
					inputs[name] = input.Default
				}
				marker := filepath.Join(t.TempDir(), "must-not-execute")
				if custom {
					inputs["port"], inputs["profile"], inputs["iam-mode"] = "14567", "standard", "log"
					inputs["services"] = "s3, sts"
					inputs["state"] = "state space ; $(touch " + marker + ") & 'quoted'.json"
					if _, ok := action.Inputs["test-mode"]; ok {
						inputs["test-mode"] = "false"
					}
					if _, ok := action.Inputs["iac"]; ok {
						inputs["iac"] = "iac space ; $(touch " + marker + ") & 'quoted'"
					}
				}
				result := runAction(t, path, inputs, "")
				assertActionOutputs(t, result, inputs["port"])
				want := map[string]string{
					"CLOUDMOCK_GATEWAY_PORT": inputs["port"], "CLOUDMOCK_PROFILE": inputs["profile"],
					"CLOUDMOCK_IAM_MODE": inputs["iam-mode"], "CLOUDMOCK_SERVICES": inputs["services"], "CLOUDMOCK_STATE": inputs["state"],
				}
				if _, ok := action.Inputs["test-mode"]; ok {
					want["CLOUDMOCK_TEST_MODE"] = inputs["test-mode"]
				}
				for name, value := range want {
					if got := result.Invocation.Environment[name]; got != value {
						t.Errorf("forwarded %s=%q, want %q", name, got, value)
					}
				}
				if iac := inputs["iac"]; iac != "" {
					viaEnv := result.Invocation.Environment["CLOUDMOCK_IAC_DIR"] == iac
					viaArgs := reflect.DeepEqual(result.Invocation.Args, []string{"--iac", iac})
					if !viaEnv && !viaArgs {
						t.Errorf("IaC path was not preserved: env=%q args=%q", result.Invocation.Environment["CLOUDMOCK_IAC_DIR"], result.Invocation.Args)
					}
					if len(result.Invocation.Args) > 0 && !viaArgs {
						t.Errorf("IaC arguments were split or altered: %q", result.Invocation.Args)
					}
				} else if len(result.Invocation.Args) != 0 {
					t.Errorf("unexpected cloudmock arguments: %q", result.Invocation.Args)
				}
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Errorf("input was executed by the shell: marker check=%v", err)
				}
			})
		}
	}
}

func TestActionLiveGateway(t *testing.T) {
	binary := os.Getenv("CLOUDMOCK_ACTION_TEST_BINARY")
	if binary == "" {
		t.Skip("set CLOUDMOCK_ACTION_TEST_BINARY to exercise the checked-out gateway binary")
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range actionPaths {
		modes := []string{"false"}
		if _, ok := readAction(t, path).Inputs["test-mode"]; ok {
			modes = append(modes, "true")
		}
		for _, testMode := range modes {
			for _, explicit := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/test-mode-%s/explicit-%t", path, testMode, explicit), func(t *testing.T) {
					// Keep gateway and observability listeners separate from any
					// existing local instance. Profile/services/state remain action
					// defaults unless this is the explicit-input scenario.
					ports := make(map[string]string)
					var listeners []net.Listener
					for _, name := range []string{"gateway", "CLOUDMOCK_ADMIN_PORT", "CLOUDMOCK_DASHBOARD_PORT", "CLOUDMOCK_OTLP_PORT", "CLOUDMOCK_OTLP_GRPC_PORT", "CLOUDMOCK_DAX_PORT"} {
						listener, err := net.Listen("tcp", "127.0.0.1:0")
						if err != nil {
							t.Fatal(err)
						}
						listeners = append(listeners, listener)
						ports[name] = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
					}
					for _, listener := range listeners {
						if err := listener.Close(); err != nil {
							t.Fatal(err)
						}
					}
					port := ports["gateway"]
					delete(ports, "gateway")
					ports["CLOUDMOCK_DATAPLANE_MODE"] = "local"
					ports["CLOUDMOCK_STATE_INTERVAL"] = "0"
					inputs := map[string]string{"port": port, "test-mode": testMode}
					const restoredBody = "snapshot restored through the action"
					if explicit {
						statePath := filepath.Join(t.TempDir(), "snapshot with spaces.json")
						state := map[string]any{"version": 1, "services": map[string]any{"s3": map[string]any{"buckets": []any{map[string]any{"name": "action-restored", "objects": []any{map[string]any{"key": "fixture.txt", "body_base64": base64.StdEncoding.EncodeToString([]byte(restoredBody)), "content_type": "text/plain"}}}}}}}
						data, err := json.Marshal(state)
						if err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(statePath, data, 0o600); err != nil {
							t.Fatal(err)
						}
						inputs["services"], inputs["state"] = "s3, secretsmanager, stepfunctions", statePath
					}
					result := runAction(t, path, inputs, binary, ports)
					assertActionOutputs(t, result, port)
					endpoint := result.Outputs["endpoint"]
					if explicit {
						status, response := actionAWSRequest(t, endpoint, "s3", http.MethodGet, "/action-restored/fixture.txt", "")
						if status != http.StatusOK || response != restoredBody {
							t.Fatalf("snapshot restore: status=%d body=%q", status, response)
						}
					}
					status, response := actionAWSRequest(t, endpoint, "s3", http.MethodPut, "/action-created", "")
					if status != http.StatusOK {
						t.Fatalf("create bucket: status=%d body=%s", status, response)
					}
					status, response = actionAWSRequest(t, endpoint, "s3", http.MethodGet, "/", "")
					if status != http.StatusOK || !strings.Contains(response, "action-created") {
						t.Fatalf("list buckets: status=%d body=%s", status, response)
					}
					if explicit {
						status, response = actionAWSRequest(t, endpoint, "secretsmanager", http.MethodPost, "/", "{}")
						if status != http.StatusOK || !strings.Contains(response, "SecretList") {
							t.Fatalf("explicit service outside minimal profile: status=%d body=%s", status, response)
						}
						status, response = actionAWSRequest(t, endpoint, "states", http.MethodPost, "/", "{}")
						if status != http.StatusOK || !strings.Contains(response, "stateMachines") {
							t.Fatalf("explicit Step Functions service alias: status=%d body=%s", status, response)
						}
					}
					status, response = actionAWSRequest(t, endpoint, "sqs", http.MethodPost, "/", "Action=ListQueues&Version=2012-11-05")
					if explicit {
						if status != http.StatusServiceUnavailable || !strings.Contains(response, "ServiceUnavailable") {
							t.Fatalf("unselected SQS must be unavailable: status=%d body=%s", status, response)
						}
					} else if status != http.StatusOK || !strings.Contains(response, "ListQueuesResponse") {
						t.Fatalf("default minimal profile must retain SQS: status=%d body=%s", status, response)
					}
				})
			}
		}
	}
}

func actionAWSRequest(t *testing.T, endpoint, service, method, path, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, endpoint+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/"+service+"/aws4_request, SignedHeaders=host, Signature=test")
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if service == "secretsmanager" {
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
		req.Header.Set("X-Amz-Target", "secretsmanager.ListSecrets")
	}
	if service == "states" {
		req.Header.Set("Content-Type", "application/x-amz-json-1.0")
		req.Header.Set("X-Amz-Target", "AWSStepFunctions.ListStateMachines")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(data)
}

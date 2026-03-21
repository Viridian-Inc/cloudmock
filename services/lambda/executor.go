package lambda

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Executor manages function code extraction and execution.
type Executor struct {
	mu       sync.Mutex
	tempDirs map[string]string // functionName -> temp dir with extracted code
}

// NewExecutor returns a new Executor.
func NewExecutor() *Executor {
	return &Executor{
		tempDirs: make(map[string]string),
	}
}

// Cleanup removes all temporary directories.
func (e *Executor) Cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, dir := range e.tempDirs {
		os.RemoveAll(dir)
	}
	e.tempDirs = make(map[string]string)
}

// InvalidateCache removes the cached temp directory for a function,
// forcing re-extraction on next invocation.
func (e *Executor) InvalidateCache(functionName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if dir, ok := e.tempDirs[functionName]; ok {
		os.RemoveAll(dir)
		delete(e.tempDirs, functionName)
	}
}

// extractCode extracts the function's zip code to a temp directory.
// Returns the temp directory path.
func (e *Executor) extractCode(fn *Function) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if dir, ok := e.tempDirs[fn.FunctionName]; ok {
		// Already extracted.
		return dir, nil
	}

	dir, err := os.MkdirTemp("", "cloudmock-lambda-"+fn.FunctionName+"-")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	if err := extractZip(fn.Code, dir); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("extract zip: %w", err)
	}

	e.tempDirs[fn.FunctionName] = dir
	return dir, nil
}

// extractZip extracts zip bytes to the target directory.
func extractZip(data []byte, targetDir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	for _, f := range r.File {
		targetPath := filepath.Join(targetDir, f.Name)

		// Prevent zip slip.
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		outFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// Invoke executes the function with the given event and returns the result.
func (e *Executor) Invoke(fn *Function, event []byte) ([]byte, error) {
	if fn.Code == nil || len(fn.Code) == 0 {
		return json.Marshal(map[string]interface{}{
			"statusCode": 200,
			"body":       "cloudmock: no function code, returning mock response",
		})
	}

	codeDir, err := e.extractCode(fn)
	if err != nil {
		return nil, fmt.Errorf("extract code: %w", err)
	}

	runtime := fn.Runtime
	handler := fn.Handler

	timeout := fn.Timeout
	if timeout <= 0 {
		timeout = 3
	}

	switch {
	case strings.HasPrefix(runtime, "nodejs"):
		return e.invokeNode(fn, codeDir, handler, event, timeout)
	case strings.HasPrefix(runtime, "python"):
		return e.invokePython(fn, codeDir, handler, event, timeout)
	default:
		return json.Marshal(map[string]interface{}{
			"statusCode": 200,
			"body":       fmt.Sprintf("cloudmock: runtime %q not available, returning mock response", runtime),
		})
	}
}

// invokeNode executes a Node.js Lambda handler.
func (e *Executor) invokeNode(fn *Function, codeDir, handler string, event []byte, timeoutSec int) ([]byte, error) {
	// Check if node is available.
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return json.Marshal(map[string]interface{}{
			"statusCode": 200,
			"body":       "cloudmock: runtime not available, returning mock response",
		})
	}

	// handler format: "index.handler" -> module="index", method="handler"
	parts := strings.SplitN(handler, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid handler format %q: expected module.method", handler)
	}
	module := parts[0]
	method := parts[1]

	// Write wrapper script.
	wrapper := fmt.Sprintf(`
const mod = require('./%s');
const handler = mod['%s'] || mod.default?.['%s'] || mod.default;
const event = JSON.parse(process.env._EVENT);
const context = {
  functionName: process.env.AWS_LAMBDA_FUNCTION_NAME || '',
  memoryLimitInMB: process.env.AWS_LAMBDA_FUNCTION_MEMORY_SIZE || '128',
  invokedFunctionArn: process.env._FUNCTION_ARN || '',
  getRemainingTimeInMillis: () => %d * 1000
};
Promise.resolve(handler(event, context))
  .then(r => process.stdout.write(JSON.stringify(r)))
  .catch(e => process.stdout.write(JSON.stringify({errorMessage: e.message, errorType: e.constructor.name})));
`, module, method, method, timeoutSec)

	wrapperPath := filepath.Join(codeDir, "__cloudmock_wrapper.js")
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o644); err != nil {
		return nil, fmt.Errorf("write wrapper: %w", err)
	}

	return e.execProcess(nodePath, wrapperPath, codeDir, fn, event, timeoutSec)
}

// invokePython executes a Python Lambda handler.
func (e *Executor) invokePython(fn *Function, codeDir, handler string, event []byte, timeoutSec int) ([]byte, error) {
	// Check if python3 is available.
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		return json.Marshal(map[string]interface{}{
			"statusCode": 200,
			"body":       "cloudmock: runtime not available, returning mock response",
		})
	}

	// handler format: "lambda_function.handler" -> module="lambda_function", func="handler"
	parts := strings.SplitN(handler, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid handler format %q: expected module.function", handler)
	}
	module := parts[0]
	function := parts[1]

	wrapper := fmt.Sprintf(`
import json, importlib, os, sys
sys.path.insert(0, '.')
mod = importlib.import_module('%s')
handler = getattr(mod, '%s')
event = json.loads(os.environ['_EVENT'])
context = type('Context', (), {
    'function_name': os.environ.get('AWS_LAMBDA_FUNCTION_NAME', ''),
    'memory_limit_in_mb': os.environ.get('AWS_LAMBDA_FUNCTION_MEMORY_SIZE', '128'),
    'invoked_function_arn': os.environ.get('_FUNCTION_ARN', ''),
    'get_remaining_time_in_millis': lambda: %d * 1000
})()
result = handler(event, context)
print(json.dumps(result))
`, module, function, timeoutSec)

	wrapperPath := filepath.Join(codeDir, "__cloudmock_wrapper.py")
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o644); err != nil {
		return nil, fmt.Errorf("write wrapper: %w", err)
	}

	return e.execProcess(pythonPath, wrapperPath, codeDir, fn, event, timeoutSec)
}

// execProcess runs the wrapper script and captures stdout.
func (e *Executor) execProcess(interpreterPath, wrapperPath, codeDir string, fn *Function, event []byte, timeoutSec int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, interpreterPath, wrapperPath)
	cmd.Dir = codeDir

	// Build environment.
	env := os.Environ()
	env = append(env,
		"_EVENT="+string(event),
		"_HANDLER="+fn.Handler,
		"_HANDLER_METHOD="+handlerMethod(fn.Handler),
		"_FUNCTION_ARN="+fn.FunctionArn,
		"AWS_LAMBDA_FUNCTION_NAME="+fn.FunctionName,
		"AWS_LAMBDA_FUNCTION_MEMORY_SIZE="+fmt.Sprintf("%d", fn.MemorySize),
		"AWS_LAMBDA_FUNCTION_VERSION="+fn.Version,
		"AWS_REGION=us-east-1",
	)
	if fn.Environment != nil {
		for k, v := range fn.Environment.Variables {
			env = append(env, k+"="+v)
		}
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return json.Marshal(map[string]interface{}{
				"errorMessage": fmt.Sprintf("Task timed out after %d.00 seconds", timeoutSec),
				"errorType":    "TimeoutError",
			})
		}
		// Return the error as a Lambda error response.
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return json.Marshal(map[string]interface{}{
			"errorMessage": errMsg,
			"errorType":    "RuntimeError",
		})
	}

	result := bytes.TrimSpace(stdout.Bytes())
	if len(result) == 0 {
		return []byte("null"), nil
	}
	return result, nil
}

// handlerMethod extracts the method/function name from a handler string.
func handlerMethod(handler string) string {
	parts := strings.SplitN(handler, ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return handler
}

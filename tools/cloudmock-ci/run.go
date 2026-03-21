package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/neureaux/cloudmock/tools/common"
)

// RunResult captures the outcome of a tool execution.
type RunResult struct {
	Tool     string    `json:"tool"`
	Args     []string  `json:"args"`
	ExitCode int       `json:"exit_code"`
	Duration string    `json:"duration"`
	Time     time.Time `json:"time"`
}

const resultsFile = ".cloudmock-results.json"

func runRun(args []string) (int, error) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	tool := fs.String("tool", "", "tool to run: terraform|pulumi|cdk|sam|custodian")
	endpoint := fs.String("endpoint", "", "cloudmock endpoint URL")
	fs.Parse(args)

	if *tool == "" {
		return 1, fmt.Errorf("--tool flag is required")
	}

	ep := *endpoint
	if ep == "" {
		ep = common.DetectEndpoint()
	}

	if err := common.WaitForHealth(ep, 30*time.Second); err != nil {
		return 1, err
	}

	toolArgs := fs.Args()

	// Set up environment
	envVars := common.AWSEnvVars(ep, common.DefaultRegion, common.DefaultAccessKey, common.DefaultSecretKey)
	env := os.Environ()
	env = append(env, envVars...)

	// Run the tool and capture exit code
	start := time.Now()
	cmd := exec.Command(*tool, toolArgs...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 1, fmt.Errorf("failed to run %s: %w", *tool, err)
		}
	}
	duration := time.Since(start)

	// Record result
	result := RunResult{
		Tool:     *tool,
		Args:     toolArgs,
		ExitCode: exitCode,
		Duration: duration.String(),
		Time:     time.Now(),
	}

	if err := appendResult(result); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save result: %v\n", err)
	}

	return exitCode, nil
}

func appendResult(result RunResult) error {
	var results []RunResult

	data, err := os.ReadFile(resultsFile)
	if err == nil {
		json.Unmarshal(data, &results)
	}

	results = append(results, result)

	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(resultsFile, out, 0o644)
}

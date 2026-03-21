package common

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// AWSEnvVars returns a slice of environment variable strings that configure
// AWS CLI tools to point at a cloudmock endpoint.
func AWSEnvVars(endpoint, region, accessKey, secretKey string) []string {
	return []string{
		"AWS_ENDPOINT_URL=" + endpoint,
		"AWS_DEFAULT_REGION=" + region,
		"AWS_REGION=" + region,
		"AWS_ACCESS_KEY_ID=" + accessKey,
		"AWS_SECRET_ACCESS_KEY=" + secretKey,
		"CLOUDMOCK_ENDPOINT=" + endpoint,
	}
}

// ExecTool replaces the current process with the given tool, passing along
// the provided args and injecting cloudmock-aware environment variables.
// On Unix systems this uses syscall.Exec for a true exec; on error it falls
// back to os/exec.
func ExecTool(tool string, args []string, endpoint string) error {
	envVars := AWSEnvVars(endpoint, DefaultRegion, DefaultAccessKey, DefaultSecretKey)
	return execWithEnv(tool, args, envVars)
}

// ExecToolPassthrough execs the tool with the current environment unchanged.
// Used for --real-aws mode.
func ExecToolPassthrough(tool string, args []string) error {
	return execWithEnv(tool, args, nil)
}

func execWithEnv(tool string, args []string, extraEnv []string) error {
	binary, err := exec.LookPath(tool)
	if err != nil {
		return fmt.Errorf("could not find %q in PATH: %w", tool, err)
	}

	env := os.Environ()
	env = append(env, extraEnv...)

	// Build argv: tool name + args
	argv := append([]string{binary}, args...)

	// Try syscall.Exec for a true process replacement
	err = syscall.Exec(binary, argv, env)
	// If syscall.Exec returns, it failed — fall back to os/exec
	cmd := exec.Command(binary, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

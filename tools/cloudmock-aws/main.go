package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/tools/common"
)

func main() {
	endpoint := flag.String("endpoint", "", "cloudmock endpoint URL")
	realAWS := flag.Bool("real-aws", false, "use real AWS (bypass cloudmock)")
	yes := flag.Bool("yes", false, "skip confirmation prompts")
	flag.Parse()

	args := flag.Args()

	if *realAWS {
		if !*yes {
			if !confirm("You are about to run against REAL AWS. Continue?") {
				fmt.Fprintln(os.Stderr, "Aborted.")
				os.Exit(1)
			}
		}
		if err := common.ExecToolPassthrough("aws", args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ep := *endpoint
	if ep == "" {
		ep = common.DetectEndpoint()
	}

	// Handle extra subcommands
	if len(args) > 0 {
		switch args[0] {
		case "configure":
			if err := configureMockProfile(ep); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			return
		case "reset":
			if err := postReset(ep); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("cloudmock state reset successfully.")
			return
		case "status":
			if err := getStatus(ep); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	if err := common.WaitForHealth(ep, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := common.ExecTool("aws", args, ep); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		resp := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return resp == "y" || resp == "yes"
	}
	return false
}

func configureMockProfile(endpoint string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find home directory: %w", err)
	}

	awsDir := filepath.Join(home, ".aws")
	if err := os.MkdirAll(awsDir, 0o755); err != nil {
		return fmt.Errorf("cannot create ~/.aws: %w", err)
	}

	configPath := filepath.Join(awsDir, "config")
	profile := fmt.Sprintf(`
[profile cloudmock]
region = %s
endpoint_url = %s
`, common.DefaultRegion, endpoint)

	credPath := filepath.Join(awsDir, "credentials")
	creds := fmt.Sprintf(`
[cloudmock]
aws_access_key_id = %s
aws_secret_access_key = %s
`, common.DefaultAccessKey, common.DefaultSecretKey)

	if err := appendIfMissing(configPath, "[profile cloudmock]", profile); err != nil {
		return err
	}
	if err := appendIfMissing(credPath, "[cloudmock]", creds); err != nil {
		return err
	}

	fmt.Println("Configured AWS profile 'cloudmock'.")
	fmt.Println("Use with: aws --profile cloudmock <command>")
	return nil
}

func appendIfMissing(path, marker, content string) error {
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), marker) {
		return nil // already configured
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

func postReset(endpoint string) error {
	resp, err := http.Post(endpoint+"/api/reset", "application/json", nil)
	if err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reset returned status %d", resp.StatusCode)
	}
	return nil
}

func getStatus(endpoint string) error {
	resp, err := http.Get(endpoint + "/api/health")
	if err != nil {
		return fmt.Errorf("status check failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

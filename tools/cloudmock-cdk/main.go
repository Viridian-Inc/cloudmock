package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
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
		if err := common.ExecToolPassthrough("cdk", args); err != nil {
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
	if len(args) > 0 && args[0] == "reset" {
		resp, err := http.Post(ep+"/api/reset", "application/json", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: reset failed: %v\n", err)
			os.Exit(1)
		}
		resp.Body.Close()
		fmt.Println("cloudmock state reset successfully.")
		return
	}

	if err := common.WaitForHealth(ep, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Set CDK-specific env vars
	os.Setenv("CDK_DEFAULT_ACCOUNT", common.DefaultAccountID)
	os.Setenv("CDK_DEFAULT_REGION", common.DefaultRegion)
	os.Setenv("AWS_ENDPOINT_URL", ep)

	if err := common.ExecTool("cdk", args, ep); err != nil {
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

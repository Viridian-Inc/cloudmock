package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/tools/common"
)

func main() {
	// Parse --endpoint, --real-aws flags manually to stay consistent with other wrappers.
	endpoint := ""
	realAWS := false
	var args []string

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "--real-aws":
			realAWS = true
		case strings.HasPrefix(arg, "--endpoint="):
			endpoint = strings.TrimPrefix(arg, "--endpoint=")
		case arg == "--endpoint" && i+1 < len(os.Args):
			i++
			endpoint = os.Args[i]
		default:
			args = append(args, arg)
		}
	}

	if realAWS {
		if !confirm("You are about to run against REAL AWS. Continue?") {
			fmt.Fprintln(os.Stderr, "Aborted.")
			os.Exit(1)
		}
		if err := common.ExecToolPassthrough("pulumi", args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ep := endpoint
	if ep == "" {
		ep = common.DetectEndpoint()
	}

	if err := common.WaitForHealth(ep, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Pulumi's @pulumi/aws provider respects AWS_ENDPOINT_URL as a global override.
	os.Setenv("AWS_ENDPOINT_URL", ep)
	// Avoid backend passphrase prompts for local state.
	os.Setenv("PULUMI_CONFIG_PASSPHRASE", "test")
	// Use local backend by default (no Pulumi Cloud login needed).
	if os.Getenv("PULUMI_BACKEND_URL") == "" {
		os.Setenv("PULUMI_BACKEND_URL", "file://~/.pulumi")
	}

	if err := common.ExecTool("pulumi", args, ep); err != nil {
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

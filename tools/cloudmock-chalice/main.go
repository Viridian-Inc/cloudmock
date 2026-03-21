package main

import (
	"bufio"
	"flag"
	"fmt"
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
		if err := common.ExecToolPassthrough("chalice", args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ep := *endpoint
	if ep == "" {
		ep = common.DetectEndpoint()
	}

	if err := common.WaitForHealth(ep, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := common.ExecTool("chalice", args, ep); err != nil {
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

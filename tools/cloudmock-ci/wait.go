package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/neureaux/cloudmock/tools/common"
)

func runWait(args []string) error {
	fs := flag.NewFlagSet("wait", flag.ExitOnError)
	timeout := fs.Duration("timeout", 30*time.Second, "timeout for health check")
	endpoint := fs.String("endpoint", "", "cloudmock endpoint URL")
	fs.Parse(args)

	ep := *endpoint
	if ep == "" {
		ep = common.DetectEndpoint()
	}

	fmt.Printf("Waiting for cloudmock at %s (timeout: %s)...\n", ep, *timeout)

	if err := common.WaitForHealth(ep, *timeout); err != nil {
		return err
	}

	fmt.Println("cloudmock is ready.")
	return nil
}

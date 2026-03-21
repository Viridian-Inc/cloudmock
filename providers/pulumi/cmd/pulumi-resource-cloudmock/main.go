// Package main is the entrypoint for the Pulumi cloudmock resource provider.
//
// This provider bridges terraform-provider-cloudmock to Pulumi using tfbridge,
// allowing Pulumi programs to manage cloudmock resources natively.
//
// Build:
//
//	go build -o pulumi-resource-cloudmock .
//
// Install:
//
//	cp pulumi-resource-cloudmock ~/.pulumi/bin/
package main

import (
	"fmt"
	"os"

	// Uncomment when pulumi SDK dependencies are installed:
	// "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	// provider "github.com/neureaux/cloudmock/providers/pulumi"
)

func main() {
	// When the Pulumi SDK and tfbridge dependencies are installed, replace this
	// with the standard tfbridge.Main invocation:
	//
	//   tfbridge.Main(context.Background(), "cloudmock", provider.ProviderInfo(), nil)
	//
	// For now, print a helpful message since this is a scaffold.
	fmt.Fprintln(os.Stderr, "pulumi-resource-cloudmock: provider scaffold")
	fmt.Fprintln(os.Stderr, "Install pulumi-terraform-bridge to build the full provider.")
	fmt.Fprintln(os.Stderr, "See providers/pulumi/README.md for instructions.")
	os.Exit(1)
}

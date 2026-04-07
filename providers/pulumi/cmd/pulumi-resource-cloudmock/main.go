// Package main is the entrypoint for the Pulumi cloudmock resource provider.
//
// This is a native Pulumi provider that implements the gRPC ResourceProvider
// protocol directly, without requiring pulumi-terraform-bridge.
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
	provider "github.com/Viridian-Inc/cloudmock/providers/pulumi/internal"
)

func main() {
	registry := provider.DefaultRegistry()
	provider.ServeMain(registry)
}

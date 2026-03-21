// Package main is a placeholder for future Terraform provider code generation.
//
// The current provider uses a dynamic approach — it reads schemas at runtime
// from the schema registry and builds Terraform resources on the fly.
// This codegen tool will eventually emit static Go files for better IDE support,
// compile-time type checking, and more detailed resource implementations.
package main

import (
	"fmt"
	"os"

	"github.com/neureaux/cloudmock/pkg/schema"
)

func main() {
	// Build a sample registry to demonstrate schema counts.
	reg := schema.NewRegistry()

	// In the future this will:
	// 1. Import all services and stubs
	// 2. Build the full schema registry
	// 3. Generate providers/terraform/generated/*.go files

	fmt.Printf("cloudmock codegen: registry has %d schemas\n", reg.Len())
	fmt.Println("Code generation is not yet implemented. The dynamic provider reads schemas at runtime.")
	os.Exit(0)
}

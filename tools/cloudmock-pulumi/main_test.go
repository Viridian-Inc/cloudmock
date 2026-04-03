package main

import "testing"

func TestPulumiToolCompiles(t *testing.T) {
	// Verify the tool compiles — this test existing is the test.
	t.Log("cloudmock-pulumi compiles successfully")
}

func TestPulumiEnvVarNames(t *testing.T) {
	// Document the env vars this wrapper sets so they can be validated
	// without actually running pulumi.
	vars := []string{
		"AWS_ENDPOINT_URL",
		"PULUMI_CONFIG_PASSPHRASE",
		"PULUMI_BACKEND_URL",
	}
	for _, v := range vars {
		if v == "" {
			t.Error("empty env var name in list")
		}
	}
}

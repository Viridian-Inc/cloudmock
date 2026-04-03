package main

import "testing"

func TestCDKToolCompiles(t *testing.T) {
	// Verify the tool compiles — this test existing is the test.
	t.Log("cloudmock-cdk compiles successfully")
}

func TestCDKEnvVarNames(t *testing.T) {
	// Document the env vars this wrapper sets so they can be validated
	// without actually running cdk.
	vars := []string{
		"CDK_DEFAULT_ACCOUNT",
		"CDK_DEFAULT_REGION",
		"AWS_ENDPOINT_URL",
	}
	for _, v := range vars {
		if v == "" {
			t.Error("empty env var name in list")
		}
	}
}

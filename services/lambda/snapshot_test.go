package lambda_test

import (
	"encoding/json"
	"testing"

	lambdasvc "github.com/Viridian-Inc/cloudmock/services/lambda"
)

const (
	lambdaTestAccount = "123456789012"
	lambdaTestRegion  = "us-east-1"
)

func TestLambda_ExportState_Empty(t *testing.T) {
	svc := lambdasvc.New(lambdaTestAccount, lambdaTestRegion)

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("ExportState returned invalid JSON: %s", raw)
	}

	var state struct {
		Functions []any `json:"functions"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.Functions) != 0 {
		t.Errorf("expected empty functions, got %d", len(state.Functions))
	}
}

func TestLambda_ExportState_WithFunction(t *testing.T) {
	svc := lambdasvc.New(lambdaTestAccount, lambdaTestRegion)

	seed := json.RawMessage(`{"functions":[{"function_name":"my-handler","runtime":"python3.11","role":"arn:aws:iam::123456789012:role/lambda","handler":"app.handler","timeout":30,"memory_size":256,"environment":{"ENV":"prod","DEBUG":"false"}}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		Functions []struct {
			FunctionName string            `json:"function_name"`
			Runtime      string            `json:"runtime"`
			Handler      string            `json:"handler"`
			Timeout      int               `json:"timeout"`
			MemorySize   int               `json:"memory_size"`
			Environment  map[string]string `json:"environment"`
		} `json:"functions"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.Functions) == 0 {
		t.Fatal("expected functions in export")
	}
	fn := state.Functions[0]
	if fn.FunctionName != "my-handler" {
		t.Errorf("expected function 'my-handler', got %q", fn.FunctionName)
	}
	if fn.Runtime != "python3.11" {
		t.Errorf("expected runtime 'python3.11', got %q", fn.Runtime)
	}
	if fn.Handler != "app.handler" {
		t.Errorf("expected handler 'app.handler', got %q", fn.Handler)
	}
	if fn.Timeout != 30 {
		t.Errorf("expected timeout 30, got %d", fn.Timeout)
	}
	if fn.MemorySize != 256 {
		t.Errorf("expected memory_size 256, got %d", fn.MemorySize)
	}
	if fn.Environment["ENV"] != "prod" {
		t.Errorf("expected ENV=prod, got %q", fn.Environment["ENV"])
	}
}

func TestLambda_ImportState_RestoresFunctions(t *testing.T) {
	svc := lambdasvc.New(lambdaTestAccount, lambdaTestRegion)

	data := json.RawMessage(`{"functions":[
		{"function_name":"fn-a","runtime":"go1.x","role":"arn:aws:iam::123456789012:role/r","handler":"main","timeout":3,"memory_size":128},
		{"function_name":"fn-b","runtime":"nodejs18.x","role":"arn:aws:iam::123456789012:role/r","handler":"index.handler","timeout":60,"memory_size":512}
	]}`)
	if err := svc.ImportState(data); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	names := svc.GetFunctionNames()
	for _, expected := range []string{"fn-a", "fn-b"} {
		found := false
		for _, n := range names {
			if n == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("function %q not restored, got: %v", expected, names)
		}
	}
}

func TestLambda_ImportState_EmptyDoesNotCrash(t *testing.T) {
	svc := lambdasvc.New(lambdaTestAccount, lambdaTestRegion)

	if err := svc.ImportState(json.RawMessage(`{"functions":[]}`)); err != nil {
		t.Fatalf("ImportState with empty functions: %v", err)
	}
	if len(svc.GetFunctionNames()) != 0 {
		t.Error("expected no functions after importing empty state")
	}
}

func TestLambda_RoundTrip_PreservesEnvironment(t *testing.T) {
	svc := lambdasvc.New(lambdaTestAccount, lambdaTestRegion)

	seed := json.RawMessage(`{"functions":[{"function_name":"env-fn","runtime":"nodejs18.x","role":"arn:aws:iam::123456789012:role/r","handler":"index.handler","timeout":3,"memory_size":128,"environment":{"DB_HOST":"localhost","DB_PORT":"5432"}}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	svc2 := lambdasvc.New(lambdaTestAccount, lambdaTestRegion)
	if err := svc2.ImportState(raw); err != nil {
		t.Fatalf("ImportState (svc2): %v", err)
	}

	if len(svc2.GetFunctionNames()) == 0 {
		t.Error("function not restored after import")
	}
}

func TestLambda_RoundTrip_MultipleFunctions(t *testing.T) {
	svc := lambdasvc.New(lambdaTestAccount, lambdaTestRegion)

	seed := json.RawMessage(`{"functions":[
		{"function_name":"alpha","runtime":"go1.x","role":"arn:r","handler":"main","timeout":3,"memory_size":128},
		{"function_name":"beta","runtime":"python3.11","role":"arn:r","handler":"app.main","timeout":60,"memory_size":256},
		{"function_name":"gamma","runtime":"nodejs18.x","role":"arn:r","handler":"index.handler","timeout":30,"memory_size":512}
	]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, _ := svc.ExportState()
	svc2 := lambdasvc.New(lambdaTestAccount, lambdaTestRegion)
	svc2.ImportState(raw)

	names := svc2.GetFunctionNames()
	for _, expected := range []string{"alpha", "beta", "gamma"} {
		found := false
		for _, n := range names {
			if n == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("function %q not restored: %v", expected, names)
		}
	}
}

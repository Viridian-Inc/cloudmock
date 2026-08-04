package azurearm_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
)

func TestJSONResponseUsesAzureContentType(t *testing.T) {
	resp, err := azurearm.JSONResponse(http.StatusCreated, map[string]string{"name": "rg-a"})
	if err != nil {
		t.Fatalf("JSONResponse returned error: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
	if resp.RawContentType != "application/json" {
		t.Errorf("expected application/json content type, got %q", resp.RawContentType)
	}

	var body map[string]string
	if err := json.Unmarshal(resp.RawBody, &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if body["name"] != "rg-a" {
		t.Errorf("expected response body name rg-a, got %q", body["name"])
	}
}

func TestErrorResponseUsesAzureErrorEnvelope(t *testing.T) {
	resp, err := azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The Resource 'rg-a' was not found.")
	if err != nil {
		t.Fatalf("ErrorResponse returned error: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(resp.RawBody, &body); err != nil {
		t.Fatalf("failed to unmarshal error body: %v", err)
	}
	if body.Error.Code != "ResourceNotFound" {
		t.Errorf("expected ResourceNotFound code, got %q", body.Error.Code)
	}
	if body.Error.Message != "The Resource 'rg-a' was not found." {
		t.Errorf("unexpected error message: %q", body.Error.Message)
	}
}

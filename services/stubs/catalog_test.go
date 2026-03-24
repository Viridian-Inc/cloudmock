package stubs_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/neureaux/cloudmock/pkg/stub"
	"github.com/neureaux/cloudmock/services/stubs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllModels_Returns73(t *testing.T) {
	models := stubs.AllModels()
	assert.Equal(t, 73, len(models), "expected exactly 73 Tier 2 service models")
}

func TestAllModels_EachHasAtLeastOneAction(t *testing.T) {
	for _, m := range stubs.AllModels() {
		assert.NotEmpty(t, m.Actions, "service %s has no actions", m.ServiceName)
	}
}

func TestAllModels_EachHasAtLeastOneResourceType(t *testing.T) {
	for _, m := range stubs.AllModels() {
		assert.NotEmpty(t, m.ResourceTypes, "service %s has no resource types", m.ServiceName)
	}
}

func TestAllModels_NoDuplicateServiceNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, m := range stubs.AllModels() {
		require.False(t, seen[m.ServiceName], "duplicate service name: %s", m.ServiceName)
		seen[m.ServiceName] = true
	}
}

func TestAllModels_ValidProtocols(t *testing.T) {
	valid := map[string]bool{"json": true, "query": true, "rest-json": true, "rest-xml": true}
	for _, m := range stubs.AllModels() {
		assert.True(t, valid[m.Protocol], "service %s has invalid protocol %q", m.ServiceName, m.Protocol)
	}
}

func TestAllModels_JSONServicesHaveTargetPrefix(t *testing.T) {
	for _, m := range stubs.AllModels() {
		if m.Protocol == "json" {
			assert.NotEmpty(t, m.TargetPrefix, "JSON service %s missing TargetPrefix", m.ServiceName)
		}
	}
}

// TestEC2_HandleRequest and TestEC2_CreateVpc were removed because EC2 is now
// a Tier 1 service and no longer present in the stubs catalog.

func TestGlue_JSONProtocol(t *testing.T) {
	var glueModel *stub.ServiceModel
	for _, m := range stubs.AllModels() {
		if m.ServiceName == "glue" {
			glueModel = m
			break
		}
	}
	require.NotNil(t, glueModel)

	svc := stub.NewStubService(glueModel, "123456789012", "us-west-2")

	// Create database via JSON body
	createResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "CreateDatabase",
		Body:   []byte(`{"DatabaseName":"testdb"}`),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, createResp.StatusCode)
	assert.Equal(t, service.FormatJSON, createResp.Format)
}

func TestRegisterAll(t *testing.T) {
	// Verify RegisterAll doesn't panic and populates the routing registry.
	registry := routing.NewRegistry()
	stubs.RegisterAll(registry, "123456789012", "us-east-1")

	services := registry.List()
	assert.Equal(t, 73, len(services), "expected 73 services registered")
}

func decodeBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

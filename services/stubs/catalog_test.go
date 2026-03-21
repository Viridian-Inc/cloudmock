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

func TestAllModels_Returns74(t *testing.T) {
	models := stubs.AllModels()
	assert.Equal(t, 74, len(models), "expected exactly 74 Tier 2 service models")
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

func TestEC2_HandleRequest(t *testing.T) {
	var ec2Model *stub.ServiceModel
	for _, m := range stubs.AllModels() {
		if m.ServiceName == "ec2" {
			ec2Model = m
			break
		}
	}
	require.NotNil(t, ec2Model, "ec2 model not found")

	svc := stub.NewStubService(ec2Model, "123456789012", "us-east-1")
	assert.Equal(t, "ec2", svc.Name())

	// RunInstances (create)
	createResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "RunInstances",
		Body:   []byte("ImageId=ami-12345678&InstanceType=t2.micro"),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, createResp.StatusCode)

	created := decodeBody(t, createResp)
	instanceID, ok := created["InstanceId"].(string)
	require.True(t, ok, "expected InstanceId in response")
	assert.NotEmpty(t, instanceID)

	// DescribeInstances (list)
	listResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "DescribeInstances",
		Body:   nil,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	listed := decodeBody(t, listResp)
	instances, ok := listed["Instances"].([]interface{})
	require.True(t, ok, "expected Instances list")
	assert.Len(t, instances, 1)

	// TerminateInstances (delete)
	delResp, err := svc.HandleRequest(&service.RequestContext{
		Action: "TerminateInstances",
		Body:   []byte("InstanceId=" + instanceID),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)

	// After delete, list should be empty.
	listResp2, err := svc.HandleRequest(&service.RequestContext{
		Action: "DescribeInstances",
		Body:   nil,
	})
	require.NoError(t, err)
	listed2 := decodeBody(t, listResp2)
	instances2, ok := listed2["Instances"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, instances2)
}

func TestEC2_CreateVpc(t *testing.T) {
	var ec2Model *stub.ServiceModel
	for _, m := range stubs.AllModels() {
		if m.ServiceName == "ec2" {
			ec2Model = m
			break
		}
	}
	require.NotNil(t, ec2Model)

	svc := stub.NewStubService(ec2Model, "123456789012", "us-east-1")

	resp, err := svc.HandleRequest(&service.RequestContext{
		Action: "CreateVpc",
		Body:   []byte("CidrBlock=10.0.0.0/16"),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := decodeBody(t, resp)
	vpcID, ok := body["VpcId"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, vpcID)
	assert.Contains(t, body["Arn"].(string), "arn:aws:ec2:us-east-1:123456789012:vpc/")
}

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
	assert.Equal(t, 74, len(services), "expected 74 services registered")
}

func decodeBody(t *testing.T, resp *service.Response) map[string]interface{} {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

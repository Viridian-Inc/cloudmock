package servicediscovery_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/servicediscovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ServiceDiscoveryService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func decode(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// helper to create an HTTP namespace and return its ID
func mustCreateNS(t *testing.T, s *svc.ServiceDiscoveryService, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateHttpNamespace", map[string]any{"Name": name}))
	require.NoError(t, err)
	opID := decode(t, resp)["OperationId"].(string)
	require.NotEmpty(t, opID)
	// Find the namespace by listing
	listResp, err := s.HandleRequest(jsonCtx("ListNamespaces", map[string]any{}))
	require.NoError(t, err)
	namespaces := decode(t, listResp)["Namespaces"].([]any)
	for _, ns := range namespaces {
		nsMap := ns.(map[string]any)
		if nsMap["Name"] == name {
			return nsMap["Id"].(string)
		}
	}
	t.Fatalf("namespace %s not found", name)
	return ""
}

// ---- Namespace tests ----

func TestCreateHttpNamespace(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateHttpNamespace", map[string]any{"Name": "my-namespace"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := decode(t, resp)
	assert.NotEmpty(t, m["OperationId"])
}

func TestCreatePrivateDnsNamespace(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreatePrivateDnsNamespace", map[string]any{
		"Name": "private.local", "Vpc": "vpc-12345",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["OperationId"])
}

func TestCreatePublicDnsNamespace(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreatePublicDnsNamespace", map[string]any{"Name": "public.example.com"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["OperationId"])
}

func TestCreateNamespaceMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateHttpNamespace", map[string]any{}))
	require.Error(t, err)
}

func TestCreateNamespaceDuplicate(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateHttpNamespace", map[string]any{"Name": "my-ns"}))
	_, err := s.HandleRequest(jsonCtx("CreateHttpNamespace", map[string]any{"Name": "my-ns"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NamespaceAlreadyExists", awsErr.Code)
}

func TestGetNamespace(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")

	resp, err := s.HandleRequest(jsonCtx("GetNamespace", map[string]any{"Id": nsID}))
	require.NoError(t, err)
	m := decode(t, resp)
	ns := m["Namespace"].(map[string]any)
	assert.Equal(t, "my-ns", ns["Name"])
	assert.Equal(t, "HTTP", ns["Type"])
}

func TestGetNamespaceNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetNamespace", map[string]any{"Id": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NamespaceNotFound", awsErr.Code)
}

func TestListNamespaces(t *testing.T) {
	s := newService()
	mustCreateNS(t, s, "ns-1")
	mustCreateNS(t, s, "ns-2")

	resp, err := s.HandleRequest(jsonCtx("ListNamespaces", map[string]any{}))
	require.NoError(t, err)
	m := decode(t, resp)
	namespaces := m["Namespaces"].([]any)
	assert.Len(t, namespaces, 2)
}

func TestDeleteNamespace(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")

	resp, err := s.HandleRequest(jsonCtx("DeleteNamespace", map[string]any{"Id": nsID}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["OperationId"])

	_, err = s.HandleRequest(jsonCtx("GetNamespace", map[string]any{"Id": nsID}))
	require.Error(t, err)
}

// ---- Service tests ----

func TestCreateService(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")

	resp, err := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-service", "NamespaceId": nsID, "Description": "test svc",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	svcData := m["Service"].(map[string]any)
	assert.Equal(t, "my-service", svcData["Name"])
	assert.NotEmpty(t, svcData["Id"])
	assert.NotEmpty(t, svcData["Arn"])
}

func TestGetService(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")

	createResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-service", "NamespaceId": nsID,
	}))
	svcID := decode(t, createResp)["Service"].(map[string]any)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetService", map[string]any{"Id": svcID}))
	require.NoError(t, err)
	m := decode(t, resp)
	svcData := m["Service"].(map[string]any)
	assert.Equal(t, "my-service", svcData["Name"])
}

func TestListServices(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")

	s.HandleRequest(jsonCtx("CreateService", map[string]any{"Name": "svc-1", "NamespaceId": nsID}))
	s.HandleRequest(jsonCtx("CreateService", map[string]any{"Name": "svc-2", "NamespaceId": nsID}))

	resp, err := s.HandleRequest(jsonCtx("ListServices", map[string]any{}))
	require.NoError(t, err)
	m := decode(t, resp)
	services := m["Services"].([]any)
	assert.Len(t, services, 2)
}

func TestUpdateService(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")

	createResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-service", "NamespaceId": nsID,
	}))
	svcID := decode(t, createResp)["Service"].(map[string]any)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("UpdateService", map[string]any{
		"Id": svcID, "Service": map[string]any{"Description": "updated"},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["OperationId"])
}

func TestDeleteService(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")

	createResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-service", "NamespaceId": nsID,
	}))
	svcID := decode(t, createResp)["Service"].(map[string]any)["Id"].(string)

	_, err := s.HandleRequest(jsonCtx("DeleteService", map[string]any{"Id": svcID}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetService", map[string]any{"Id": svcID}))
	require.Error(t, err)
}

// ---- Instance tests ----

func TestRegisterAndGetInstance(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")
	createResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-service", "NamespaceId": nsID,
	}))
	svcID := decode(t, createResp)["Service"].(map[string]any)["Id"].(string)

	regResp, err := s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1",
		"Attributes": map[string]any{"AWS_INSTANCE_IPV4": "10.0.0.1"},
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, decode(t, regResp)["OperationId"])

	getResp, err := s.HandleRequest(jsonCtx("GetInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1",
	}))
	require.NoError(t, err)
	inst := decode(t, getResp)["Instance"].(map[string]any)
	assert.Equal(t, "inst-1", inst["Id"])
	attrs := inst["Attributes"].(map[string]any)
	assert.Equal(t, "10.0.0.1", attrs["AWS_INSTANCE_IPV4"])
}

func TestListInstances(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")
	createResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-service", "NamespaceId": nsID,
	}))
	svcID := decode(t, createResp)["Service"].(map[string]any)["Id"].(string)

	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1", "Attributes": map[string]any{"ip": "10.0.0.1"},
	}))
	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-2", "Attributes": map[string]any{"ip": "10.0.0.2"},
	}))

	resp, err := s.HandleRequest(jsonCtx("ListInstances", map[string]any{"ServiceId": svcID}))
	require.NoError(t, err)
	m := decode(t, resp)
	instances := m["Instances"].([]any)
	assert.Len(t, instances, 2)
}

func TestDeregisterInstance(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")
	createResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-service", "NamespaceId": nsID,
	}))
	svcID := decode(t, createResp)["Service"].(map[string]any)["Id"].(string)

	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1",
	}))

	deregResp, err := s.HandleRequest(jsonCtx("DeregisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1",
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, decode(t, deregResp)["OperationId"])

	_, err = s.HandleRequest(jsonCtx("GetInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1",
	}))
	require.Error(t, err)
}

// ---- DiscoverInstances ----

func TestDiscoverInstances(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")
	createResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-service", "NamespaceId": nsID,
	}))
	svcID := decode(t, createResp)["Service"].(map[string]any)["Id"].(string)

	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1",
		"Attributes": map[string]any{"stage": "prod"},
	}))
	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-2",
		"Attributes": map[string]any{"stage": "dev"},
	}))

	resp, err := s.HandleRequest(jsonCtx("DiscoverInstances", map[string]any{
		"NamespaceName":   "my-ns",
		"ServiceName":     "my-service",
		"QueryParameters": map[string]any{"stage": "prod"},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	instances := m["Instances"].([]any)
	assert.Len(t, instances, 1)
	assert.Equal(t, "inst-1", instances[0].(map[string]any)["InstanceId"])
}

// ---- Tagging ----

func TestTagNamespace(t *testing.T) {
	s := newService()
	nsID := mustCreateNS(t, s, "my-ns")
	// Get the ARN
	getResp, _ := s.HandleRequest(jsonCtx("GetNamespace", map[string]any{"Id": nsID}))
	arn := decode(t, getResp)["Namespace"].(map[string]any)["Arn"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": arn,
		"Tags":        []map[string]any{{"Key": "env", "Value": "prod"}},
	}))
	require.NoError(t, err)

	tagsResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": arn}))
	require.NoError(t, err)
	m := decode(t, tagsResp)
	tags := m["Tags"].([]any)
	assert.Len(t, tags, 1)
}

// ---- Invalid action ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Behavioral: Health status filtering ----

func getFirstNamespaceID(t *testing.T, s *svc.ServiceDiscoveryService) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("ListNamespaces", map[string]any{}))
	require.NoError(t, err)
	m := decode(t, resp)
	nsList := m["Namespaces"].([]any)
	require.NotEmpty(t, nsList)
	return nsList[0].(map[string]any)["Id"].(string)
}

func TestDiscoverInstancesHealthFilter(t *testing.T) {
	s := newService()

	// Create namespace + service + instances
	_, err := s.HandleRequest(jsonCtx("CreateHttpNamespace", map[string]any{"Name": "my-ns"}))
	require.NoError(t, err)
	nsID := getFirstNamespaceID(t, s)

	svcResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-svc", "NamespaceId": nsID,
	}))
	svcID := decode(t, svcResp)["Service"].(map[string]any)["Id"].(string)

	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1",
		"Attributes": map[string]any{"AWS_INSTANCE_IPV4": "10.0.0.1"},
	}))
	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-2",
		"Attributes": map[string]any{"AWS_INSTANCE_IPV4": "10.0.0.2"},
	}))

	// Mark inst-2 as unhealthy
	_, err2 := s.HandleRequest(jsonCtx("UpdateInstanceCustomHealthStatus", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-2", "Status": "UNHEALTHY",
	}))
	require.NoError(t, err2)

	// DiscoverInstances should only return healthy instances by default
	discResp, discErr := s.HandleRequest(jsonCtx("DiscoverInstances", map[string]any{
		"NamespaceName": "my-ns", "ServiceName": "my-svc",
	}))
	require.NoError(t, discErr)
	discData := decode(t, discResp)
	instances := discData["Instances"].([]any)
	assert.Len(t, instances, 1)
	assert.Equal(t, "inst-1", instances[0].(map[string]any)["InstanceId"])
	assert.Equal(t, "HEALTHY", instances[0].(map[string]any)["HealthStatus"])
}

func TestDiscoverInstancesAllHealth(t *testing.T) {
	s := newService()

	s.HandleRequest(jsonCtx("CreateHttpNamespace", map[string]any{"Name": "my-ns"}))
	nsID := getFirstNamespaceID(t, s)

	svcResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"Name": "my-svc", "NamespaceId": nsID,
	}))
	svcID := decode(t, svcResp)["Service"].(map[string]any)["Id"].(string)

	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-1",
		"Attributes": map[string]any{"AWS_INSTANCE_IPV4": "10.0.0.1"},
	}))
	s.HandleRequest(jsonCtx("RegisterInstance", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-2",
		"Attributes": map[string]any{"AWS_INSTANCE_IPV4": "10.0.0.2"},
	}))

	s.HandleRequest(jsonCtx("UpdateInstanceCustomHealthStatus", map[string]any{
		"ServiceId": svcID, "InstanceId": "inst-2", "Status": "UNHEALTHY",
	}))

	// Discover with ALL health filter
	discResp, err := s.HandleRequest(jsonCtx("DiscoverInstances", map[string]any{
		"NamespaceName": "my-ns", "ServiceName": "my-svc", "HealthStatus": "ALL",
	}))
	require.NoError(t, err)
	discData := decode(t, discResp)
	instances := discData["Instances"].([]any)
	assert.Len(t, instances, 2)
}

func TestUpdateInstanceCustomHealthStatusNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("UpdateInstanceCustomHealthStatus", map[string]any{
		"ServiceId": "nonexistent", "InstanceId": "inst-1", "Status": "UNHEALTHY",
	}))
	require.Error(t, err)
}

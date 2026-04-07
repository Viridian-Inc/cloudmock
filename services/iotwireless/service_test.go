package iotwireless_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/iotwireless"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.IoTWirelessService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		Params:     make(map[string]string),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// ---- Test 1: CreateWirelessDevice ----

func TestCreateWirelessDevice(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateWirelessDevice", map[string]any{
		"Name":            "device-1",
		"Type":            "LoRaWAN",
		"DestinationName": "my-dest",
		"Description":     "Test device",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["Id"])
	assert.NotEmpty(t, body["Arn"])
}

// ---- Test 2: GetWirelessDevice ----

func TestGetWirelessDevice(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateWirelessDevice", map[string]any{
		"Name": "device-get",
		"Type": "LoRaWAN",
	}))
	require.NoError(t, err)
	deviceId := respBody(t, createResp)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetWirelessDevice", map[string]any{"Identifier": deviceId}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "device-get", body["Name"])
	assert.Equal(t, "LoRaWAN", body["Type"])
}

// ---- Test 3: ListWirelessDevices ----

func TestListWirelessDevices(t *testing.T) {
	s := newService()
	for _, name := range []string{"wd-1", "wd-2", "wd-3"} {
		_, err := s.HandleRequest(jsonCtx("CreateWirelessDevice", map[string]any{"Name": name}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListWirelessDevices", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	devices := body["WirelessDeviceList"].([]any)
	assert.Len(t, devices, 3)
}

// ---- Test 4: DeleteWirelessDevice ----

func TestDeleteWirelessDevice(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateWirelessDevice", map[string]any{"Name": "wd-del"}))
	require.NoError(t, err)
	deviceId := respBody(t, createResp)["Id"].(string)

	_, err = s.HandleRequest(jsonCtx("DeleteWirelessDevice", map[string]any{"Id": deviceId}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetWirelessDevice", map[string]any{"Identifier": deviceId}))
	require.Error(t, err)
}

// ---- Test 5: UpdateWirelessDevice ----

func TestUpdateWirelessDevice(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateWirelessDevice", map[string]any{
		"Name": "wd-upd",
	}))
	require.NoError(t, err)
	deviceId := respBody(t, createResp)["Id"].(string)

	_, err = s.HandleRequest(jsonCtx("UpdateWirelessDevice", map[string]any{
		"Id":          deviceId,
		"Name":        "wd-updated",
		"Description": "Updated description",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetWirelessDevice", map[string]any{"Identifier": deviceId}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "wd-updated", body["Name"])
	assert.Equal(t, "Updated description", body["Description"])
}

// ---- Test 6: WirelessGateway CRUD ----

func TestWirelessGatewayCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateWirelessGateway", map[string]any{
		"Name":        "gw-1",
		"Description": "Test gateway",
		"LoRaWAN":     map[string]any{"GatewayEui": "aabbccddee112233"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)
	gwId := respBody(t, createResp)["Id"].(string)

	// Get
	getResp, err := s.HandleRequest(jsonCtx("GetWirelessGateway", map[string]any{"Identifier": gwId}))
	require.NoError(t, err)
	body := respBody(t, getResp)
	assert.Equal(t, "gw-1", body["Name"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListWirelessGateways", map[string]any{}))
	require.NoError(t, err)
	gateways := respBody(t, listResp)["WirelessGatewayList"].([]any)
	assert.Len(t, gateways, 1)

	// Update
	_, err = s.HandleRequest(jsonCtx("UpdateWirelessGateway", map[string]any{
		"Id":          gwId,
		"Name":        "gw-updated",
		"Description": "Updated gw",
	}))
	require.NoError(t, err)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteWirelessGateway", map[string]any{"Id": gwId}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetWirelessGateway", map[string]any{"Identifier": gwId}))
	require.Error(t, err)
}

// ---- Test 7: DeviceProfile CRUD ----

func TestDeviceProfileCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateDeviceProfile", map[string]any{
		"Name":    "dp-1",
		"LoRaWAN": map[string]any{"MacVersion": "1.0.3", "RegParamsRevision": "RP002-1.0.1"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)
	dpId := respBody(t, createResp)["Id"].(string)

	// Get
	getResp, err := s.HandleRequest(jsonCtx("GetDeviceProfile", map[string]any{"Id": dpId}))
	require.NoError(t, err)
	body := respBody(t, getResp)
	assert.Equal(t, "dp-1", body["Name"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListDeviceProfiles", map[string]any{}))
	require.NoError(t, err)
	profiles := respBody(t, listResp)["DeviceProfileList"].([]any)
	assert.Len(t, profiles, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteDeviceProfile", map[string]any{"Id": dpId}))
	require.NoError(t, err)
}

// ---- Test 8: ServiceProfile CRUD ----

func TestServiceProfileCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateServiceProfile", map[string]any{
		"Name":    "sp-1",
		"LoRaWAN": map[string]any{"AddGwMetadata": true},
	}))
	require.NoError(t, err)
	spId := respBody(t, createResp)["Id"].(string)

	// Get
	getResp, err := s.HandleRequest(jsonCtx("GetServiceProfile", map[string]any{"Id": spId}))
	require.NoError(t, err)
	assert.Equal(t, "sp-1", respBody(t, getResp)["Name"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListServiceProfiles", map[string]any{}))
	require.NoError(t, err)
	profiles := respBody(t, listResp)["ServiceProfileList"].([]any)
	assert.Len(t, profiles, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteServiceProfile", map[string]any{"Id": spId}))
	require.NoError(t, err)
}

// ---- Test 9: Destination CRUD ----

func TestDestinationCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateDestination", map[string]any{
		"Name":           "dest-1",
		"Expression":     "my-rule",
		"ExpressionType": "RuleName",
		"Description":    "Test destination",
		"RoleArn":        "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)
	body := respBody(t, createResp)
	assert.Equal(t, "dest-1", body["Name"])

	// Get
	getResp, err := s.HandleRequest(jsonCtx("GetDestination", map[string]any{"Name": "dest-1"}))
	require.NoError(t, err)
	getBody := respBody(t, getResp)
	assert.Equal(t, "dest-1", getBody["Name"])
	assert.Equal(t, "my-rule", getBody["Expression"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListDestinations", map[string]any{}))
	require.NoError(t, err)
	dests := respBody(t, listResp)["DestinationList"].([]any)
	assert.Len(t, dests, 1)

	// Update
	_, err = s.HandleRequest(jsonCtx("UpdateDestination", map[string]any{
		"Name":        "dest-1",
		"Description": "Updated",
	}))
	require.NoError(t, err)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteDestination", map[string]any{"Name": "dest-1"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetDestination", map[string]any{"Name": "dest-1"}))
	require.Error(t, err)
}

// ---- Test 10: Destination duplicate ----

func TestDuplicateDestination(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateDestination", map[string]any{
		"Name":       "dest-dup",
		"Expression": "rule",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("CreateDestination", map[string]any{
		"Name":       "dest-dup",
		"Expression": "rule",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ConflictException", awsErr.Code)
}

// ---- Test 11: Device NotFound ----

func TestDeviceNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetWirelessDevice", map[string]any{"Identifier": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

// ---- Test 12: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 13: Tagging ----

func TestTagging(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateWirelessDevice", map[string]any{
		"Name": "tag-device",
		"Tags": []any{map[string]any{"Key": "env", "Value": "test"}},
	}))
	require.NoError(t, err)
	arn := respBody(t, createResp)["Arn"].(string)

	// Add tags
	_, err = s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags":        []any{map[string]any{"Key": "team", "Value": "iot"}},
	}))
	require.NoError(t, err)

	// List tags
	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	tags := respBody(t, listResp)["Tags"].([]any)
	assert.Len(t, tags, 2)

	// Untag
	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	tags2 := respBody(t, listResp2)["Tags"].([]any)
	assert.Len(t, tags2, 1)
}

// ---- Test 14: Invalid device type ----

func TestInvalidDeviceType(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateWirelessDevice", map[string]any{
		"Name": "bad-device",
		"Type": "InvalidType",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationException", awsErr.Code)
}

// ---- Test 15: Invalid frequency band ----

func TestInvalidFrequencyBand(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateDeviceProfile", map[string]any{
		"Name":    "bad-dp",
		"LoRaWAN": map[string]any{"RfRegion": "INVALID_BAND"},
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationException", awsErr.Code)
}

// ---- Test 16: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "iot-wireless", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Test 15: Multiple device profiles ----

func TestMultipleDeviceProfiles(t *testing.T) {
	s := newService()
	for _, name := range []string{"dp-a", "dp-b", "dp-c"} {
		_, err := s.HandleRequest(jsonCtx("CreateDeviceProfile", map[string]any{"Name": name}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListDeviceProfiles", map[string]any{}))
	require.NoError(t, err)
	profiles := respBody(t, resp)["DeviceProfileList"].([]any)
	assert.Len(t, profiles, 3)
}

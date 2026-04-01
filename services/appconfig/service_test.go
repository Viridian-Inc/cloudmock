package appconfig_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/appconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.AppConfigService {
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

// ---- Application tests ----

func TestCreateApplication(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app", "Description": "test app"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	m := decode(t, resp)
	assert.Equal(t, "my-app", m["Name"])
	assert.NotEmpty(t, m["Id"])
}

func TestCreateApplicationMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

func TestGetApplication(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	require.NoError(t, err)
	appID := decode(t, createResp)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetApplication", map[string]any{"ApplicationId": appID}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := decode(t, resp)
	assert.Equal(t, "my-app", m["Name"])
	assert.Equal(t, appID, m["Id"])
}

func TestGetApplicationNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetApplication", map[string]any{"ApplicationId": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

func TestListApplications(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "app-1"}))
	s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "app-2"}))

	resp, err := s.HandleRequest(jsonCtx("ListApplications", map[string]any{}))
	require.NoError(t, err)
	m := decode(t, resp)
	items := m["Items"].([]any)
	assert.Len(t, items, 2)
}

func TestUpdateApplication(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, createResp)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("UpdateApplication", map[string]any{
		"ApplicationId": appID,
		"Name":          "updated-app",
		"Description":   "updated desc",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := decode(t, resp)
	assert.Equal(t, "updated-app", m["Name"])
}

func TestDeleteApplication(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, createResp)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteApplication", map[string]any{"ApplicationId": appID}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify it's gone
	_, err = s.HandleRequest(jsonCtx("GetApplication", map[string]any{"ApplicationId": appID}))
	require.Error(t, err)
}

func TestDeleteApplicationNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteApplication", map[string]any{"ApplicationId": "nonexistent"}))
	require.Error(t, err)
}

// ---- Environment tests ----

func TestCreateEnvironment(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, createResp)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"ApplicationId": appID,
		"Name":          "production",
		"Description":   "prod env",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	m := decode(t, resp)
	assert.Equal(t, "production", m["Name"])
	assert.Equal(t, "READY_FOR_DEPLOYMENT", m["State"])
}

func TestListEnvironments(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, createResp)["Id"].(string)

	s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "dev"}))
	s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "prod"}))

	resp, err := s.HandleRequest(jsonCtx("ListEnvironments", map[string]any{"ApplicationId": appID}))
	require.NoError(t, err)
	m := decode(t, resp)
	items := m["Items"].([]any)
	assert.Len(t, items, 2)
}

func TestDeleteEnvironment(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, createResp)["Id"].(string)

	envResp, _ := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "dev"}))
	envID := decode(t, envResp)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteEnvironment", map[string]any{"ApplicationId": appID, "EnvironmentId": envID}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// ---- Configuration Profile tests ----

func TestCreateConfigurationProfile(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, createResp)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("CreateConfigurationProfile", map[string]any{
		"ApplicationId": appID,
		"Name":          "my-config",
		"LocationUri":   "hosted",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	m := decode(t, resp)
	assert.Equal(t, "my-config", m["Name"])
	assert.Equal(t, "AWS.Freeform", m["Type"])
}

func TestListConfigurationProfiles(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, createResp)["Id"].(string)

	s.HandleRequest(jsonCtx("CreateConfigurationProfile", map[string]any{"ApplicationId": appID, "Name": "config-1", "LocationUri": "hosted"}))
	s.HandleRequest(jsonCtx("CreateConfigurationProfile", map[string]any{"ApplicationId": appID, "Name": "config-2", "LocationUri": "hosted"}))

	resp, err := s.HandleRequest(jsonCtx("ListConfigurationProfiles", map[string]any{"ApplicationId": appID}))
	require.NoError(t, err)
	m := decode(t, resp)
	items := m["Items"].([]any)
	assert.Len(t, items, 2)
}

// ---- Deployment Strategy tests ----

func TestCreateDeploymentStrategy(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateDeploymentStrategy", map[string]any{
		"Name":                        "my-strategy",
		"DeploymentDurationInMinutes": 10,
		"GrowthFactor":                10.0,
		"FinalBakeTimeInMinutes":      5,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	m := decode(t, resp)
	assert.Equal(t, "my-strategy", m["Name"])
	assert.Equal(t, "LINEAR", m["GrowthType"])
	assert.Equal(t, "NONE", m["ReplicateTo"])
}

func TestListDeploymentStrategies(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateDeploymentStrategy", map[string]any{"Name": "strat-1"}))
	s.HandleRequest(jsonCtx("CreateDeploymentStrategy", map[string]any{"Name": "strat-2"}))

	resp, err := s.HandleRequest(jsonCtx("ListDeploymentStrategies", map[string]any{}))
	require.NoError(t, err)
	m := decode(t, resp)
	items := m["Items"].([]any)
	assert.Len(t, items, 2)
}

// ---- Deployment lifecycle tests ----

func TestStartDeploymentAndGet(t *testing.T) {
	s := newService()
	// Create prerequisites
	appResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, appResp)["Id"].(string)

	envResp, _ := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "prod"}))
	envID := decode(t, envResp)["Id"].(string)

	stratResp, _ := s.HandleRequest(jsonCtx("CreateDeploymentStrategy", map[string]any{"Name": "my-strat"}))
	stratID := decode(t, stratResp)["Id"].(string)

	profileResp, _ := s.HandleRequest(jsonCtx("CreateConfigurationProfile", map[string]any{
		"ApplicationId": appID, "Name": "my-config", "LocationUri": "hosted",
	}))
	profileID := decode(t, profileResp)["Id"].(string)

	// Start deployment
	depResp, err := s.HandleRequest(jsonCtx("StartDeployment", map[string]any{
		"ApplicationId":          appID,
		"EnvironmentId":          envID,
		"DeploymentStrategyId":   stratID,
		"ConfigurationProfileId": profileID,
		"ConfigurationVersion":   "1",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, depResp.StatusCode)
	depData := decode(t, depResp)
	assert.Equal(t, float64(1), depData["DeploymentNumber"])

	// Lifecycle config has delays disabled, so transitions are instant.
	// Get deployment - should be COMPLETE already.
	getResp, err := s.HandleRequest(jsonCtx("GetDeployment", map[string]any{
		"ApplicationId":    appID,
		"EnvironmentId":    envID,
		"DeploymentNumber": 1,
	}))
	require.NoError(t, err)
	getDepData := decode(t, getResp)
	assert.Equal(t, "COMPLETE", getDepData["State"])
}

func TestStopDeploymentNotFound(t *testing.T) {
	s := newService()
	appResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, appResp)["Id"].(string)

	s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "prod"}))

	_, err := s.HandleRequest(jsonCtx("StopDeployment", map[string]any{
		"ApplicationId": appID, "EnvironmentId": "nonexistent", "DeploymentNumber": 99,
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

func TestListDeployments(t *testing.T) {
	s := newService()
	appResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, appResp)["Id"].(string)

	envResp, _ := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "prod"}))
	envID := decode(t, envResp)["Id"].(string)

	stratResp, _ := s.HandleRequest(jsonCtx("CreateDeploymentStrategy", map[string]any{"Name": "my-strat"}))
	stratID := decode(t, stratResp)["Id"].(string)

	profileResp, _ := s.HandleRequest(jsonCtx("CreateConfigurationProfile", map[string]any{
		"ApplicationId": appID, "Name": "my-config", "LocationUri": "hosted",
	}))
	profileID := decode(t, profileResp)["Id"].(string)

	s.HandleRequest(jsonCtx("StartDeployment", map[string]any{
		"ApplicationId": appID, "EnvironmentId": envID,
		"DeploymentStrategyId": stratID, "ConfigurationProfileId": profileID,
		"ConfigurationVersion": "1",
	}))
	s.HandleRequest(jsonCtx("StartDeployment", map[string]any{
		"ApplicationId": appID, "EnvironmentId": envID,
		"DeploymentStrategyId": stratID, "ConfigurationProfileId": profileID,
		"ConfigurationVersion": "2",
	}))

	resp, err := s.HandleRequest(jsonCtx("ListDeployments", map[string]any{
		"ApplicationId": appID, "EnvironmentId": envID,
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	items := m["Items"].([]any)
	assert.Len(t, items, 2)
}

// ---- Tagging tests ----

func TestTagResource(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, createResp)["Id"].(string)
	arn := "arn:aws:appconfig:us-east-1:123456789012:application/" + appID

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags":        map[string]any{"env": "prod", "team": "alpha"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	m := decode(t, resp)
	tags := m["Tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
	assert.Equal(t, "alpha", tags["team"])
}

func TestUntagResource(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{
		"Name": "my-app",
		"Tags": map[string]any{"env": "prod", "team": "alpha"},
	}))
	appID := decode(t, createResp)["Id"].(string)
	arn := "arn:aws:appconfig:us-east-1:123456789012:application/" + appID

	_, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []string{"team"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	m := decode(t, resp)
	tags := m["Tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
	assert.Nil(t, tags["team"])
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

// ---- Behavioral: Deployment strategy execution ----

func TestDeploymentLinearStrategy(t *testing.T) {
	s := newService()

	// Create app, env, profile, strategy
	appResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, appResp)["Id"].(string)

	envResp, _ := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "prod"}))
	envID := decode(t, envResp)["Id"].(string)

	profileResp, _ := s.HandleRequest(jsonCtx("CreateConfigurationProfile", map[string]any{
		"ApplicationId": appID, "Name": "my-profile", "LocationUri": "hosted",
	}))
	profileID := decode(t, profileResp)["Id"].(string)

	stratResp, _ := s.HandleRequest(jsonCtx("CreateDeploymentStrategy", map[string]any{
		"Name": "linear-10", "DeploymentDurationInMinutes": 10,
		"GrowthFactor": 10.0, "GrowthType": "LINEAR", "FinalBakeTimeInMinutes": 1,
	}))
	stratID := decode(t, stratResp)["Id"].(string)

	// Start deployment
	depResp, err := s.HandleRequest(jsonCtx("StartDeployment", map[string]any{
		"ApplicationId":          appID,
		"EnvironmentId":          envID,
		"DeploymentStrategyId":   stratID,
		"ConfigurationProfileId": profileID,
		"ConfigurationVersion":   "1",
	}))
	require.NoError(t, err)
	depData := decode(t, depResp)

	// In instant mode, deployment should complete immediately.
	assert.Equal(t, "COMPLETE", depData["State"])
	assert.Equal(t, 100.0, depData["PercentageComplete"])
	assert.NotEmpty(t, depData["CompletedAt"])
}

func TestDeploymentExponentialStrategy(t *testing.T) {
	s := newService()

	appResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, appResp)["Id"].(string)

	envResp, _ := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "prod"}))
	envID := decode(t, envResp)["Id"].(string)

	profileResp, _ := s.HandleRequest(jsonCtx("CreateConfigurationProfile", map[string]any{
		"ApplicationId": appID, "Name": "my-profile", "LocationUri": "hosted",
	}))
	profileID := decode(t, profileResp)["Id"].(string)

	stratResp, _ := s.HandleRequest(jsonCtx("CreateDeploymentStrategy", map[string]any{
		"Name": "exponential", "DeploymentDurationInMinutes": 10,
		"GrowthFactor": 10.0, "GrowthType": "EXPONENTIAL", "FinalBakeTimeInMinutes": 1,
	}))
	stratID := decode(t, stratResp)["Id"].(string)

	depResp, err := s.HandleRequest(jsonCtx("StartDeployment", map[string]any{
		"ApplicationId":          appID,
		"EnvironmentId":          envID,
		"DeploymentStrategyId":   stratID,
		"ConfigurationProfileId": profileID,
		"ConfigurationVersion":   "1",
	}))
	require.NoError(t, err)
	depData := decode(t, depResp)
	assert.Equal(t, "COMPLETE", depData["State"])
	assert.Equal(t, 100.0, depData["PercentageComplete"])
}

func TestStopDeployment(t *testing.T) {
	s := newService()

	appResp, _ := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"Name": "my-app"}))
	appID := decode(t, appResp)["Id"].(string)

	envResp, _ := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{"ApplicationId": appID, "Name": "prod"}))
	envID := decode(t, envResp)["Id"].(string)

	profileResp, _ := s.HandleRequest(jsonCtx("CreateConfigurationProfile", map[string]any{
		"ApplicationId": appID, "Name": "my-profile", "LocationUri": "hosted",
	}))
	profileID := decode(t, profileResp)["Id"].(string)

	stratResp, _ := s.HandleRequest(jsonCtx("CreateDeploymentStrategy", map[string]any{
		"Name": "quick", "DeploymentDurationInMinutes": 1,
		"GrowthFactor": 50.0, "GrowthType": "LINEAR",
	}))
	stratID := decode(t, stratResp)["Id"].(string)

	s.HandleRequest(jsonCtx("StartDeployment", map[string]any{
		"ApplicationId": appID, "EnvironmentId": envID,
		"DeploymentStrategyId": stratID, "ConfigurationProfileId": profileID,
		"ConfigurationVersion": "1",
	}))

	// Stop the deployment
	stopResp, err := s.HandleRequest(jsonCtx("StopDeployment", map[string]any{
		"ApplicationId": appID, "EnvironmentId": envID, "DeploymentNumber": 1,
	}))
	require.NoError(t, err)
	stopData := decode(t, stopResp)
	assert.Equal(t, "ROLLED_BACK", stopData["State"])
}

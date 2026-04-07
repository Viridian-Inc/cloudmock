package codedeploy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/codedeploy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.CodeDeployService {
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

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func createApp(t *testing.T, s *svc.CodeDeployService, name string) string {
	t.Helper()
	ctx := jsonCtx("CreateApplication", map[string]any{
		"applicationName": name,
		"computePlatform": "Server",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	return body["applicationId"].(string)
}

func createGroup(t *testing.T, s *svc.CodeDeployService, appName, groupName string) string {
	t.Helper()
	ctx := jsonCtx("CreateDeploymentGroup", map[string]any{
		"applicationName":     appName,
		"deploymentGroupName": groupName,
		"serviceRoleArn":      "arn:aws:iam::123456789012:role/deploy-role",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	return body["deploymentGroupId"].(string)
}

// --- Application Tests ---

func TestCreateApplication(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "my-app")
	assert.NotEmpty(t, appID)
}

func TestCreateApplicationDuplicate(t *testing.T) {
	s := newService()
	createApp(t, s, "dup-app")
	ctx := jsonCtx("CreateApplication", map[string]any{"applicationName": "dup-app"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ApplicationAlreadyExistsException")
}

func TestCreateApplicationMissingName(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateApplication", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestGetApplication(t *testing.T) {
	s := newService()
	createApp(t, s, "get-app")

	ctx := jsonCtx("GetApplication", map[string]any{"applicationName": "get-app"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	app := body["application"].(map[string]any)
	assert.Equal(t, "get-app", app["applicationName"])
	assert.Equal(t, "Server", app["computePlatform"])
}

func TestGetApplicationNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("GetApplication", map[string]any{"applicationName": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ApplicationDoesNotExistException")
}

func TestListApplications(t *testing.T) {
	s := newService()
	createApp(t, s, "app-1")
	createApp(t, s, "app-2")

	resp, err := s.HandleRequest(jsonCtx("ListApplications", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	apps := body["applications"].([]any)
	assert.Len(t, apps, 2)
}

func TestDeleteApplication(t *testing.T) {
	s := newService()
	createApp(t, s, "del-app")

	ctx := jsonCtx("DeleteApplication", map[string]any{"applicationName": "del-app"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify removed
	resp2, _ := s.HandleRequest(jsonCtx("ListApplications", map[string]any{}))
	body := respBody(t, resp2)
	assert.Len(t, body["applications"].([]any), 0)
}

func TestDeleteApplicationNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("DeleteApplication", map[string]any{"applicationName": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ApplicationDoesNotExistException")
}

// --- Deployment Group Tests ---

func TestCreateDeploymentGroup(t *testing.T) {
	s := newService()
	createApp(t, s, "dg-app")
	groupID := createGroup(t, s, "dg-app", "my-group")
	assert.NotEmpty(t, groupID)
}

func TestCreateDeploymentGroupDuplicate(t *testing.T) {
	s := newService()
	createApp(t, s, "dgdup-app")
	createGroup(t, s, "dgdup-app", "dup-group")
	ctx := jsonCtx("CreateDeploymentGroup", map[string]any{
		"applicationName":     "dgdup-app",
		"deploymentGroupName": "dup-group",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeploymentGroupAlreadyExistsException")
}

func TestCreateDeploymentGroupAppNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateDeploymentGroup", map[string]any{
		"applicationName":     "nope",
		"deploymentGroupName": "grp",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ApplicationDoesNotExistException")
}

func TestGetDeploymentGroup(t *testing.T) {
	s := newService()
	createApp(t, s, "getdg-app")
	createGroup(t, s, "getdg-app", "get-group")

	ctx := jsonCtx("GetDeploymentGroup", map[string]any{
		"applicationName":     "getdg-app",
		"deploymentGroupName": "get-group",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	info := body["deploymentGroupInfo"].(map[string]any)
	assert.Equal(t, "get-group", info["deploymentGroupName"])
	assert.Equal(t, "getdg-app", info["applicationName"])
}

func TestListDeploymentGroups(t *testing.T) {
	s := newService()
	createApp(t, s, "ldg-app")
	createGroup(t, s, "ldg-app", "grp-1")
	createGroup(t, s, "ldg-app", "grp-2")

	ctx := jsonCtx("ListDeploymentGroups", map[string]any{"applicationName": "ldg-app"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	groups := body["deploymentGroups"].([]any)
	assert.Len(t, groups, 2)
}

func TestDeleteDeploymentGroup(t *testing.T) {
	s := newService()
	createApp(t, s, "deldg-app")
	createGroup(t, s, "deldg-app", "del-grp")

	ctx := jsonCtx("DeleteDeploymentGroup", map[string]any{
		"applicationName":     "deldg-app",
		"deploymentGroupName": "del-grp",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateDeploymentGroup(t *testing.T) {
	s := newService()
	createApp(t, s, "upddg-app")
	createGroup(t, s, "upddg-app", "upd-grp")

	ctx := jsonCtx("UpdateDeploymentGroup", map[string]any{
		"applicationName":            "upddg-app",
		"currentDeploymentGroupName": "upd-grp",
		"newDeploymentGroupName":     "renamed-grp",
		"serviceRoleArn":             "arn:aws:iam::123456789012:role/new-role",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["deploymentGroupId"])

	// Verify renamed
	ctx2 := jsonCtx("GetDeploymentGroup", map[string]any{
		"applicationName":     "upddg-app",
		"deploymentGroupName": "renamed-grp",
	})
	resp2, err2 := s.HandleRequest(ctx2)
	require.NoError(t, err2)
	body2 := respBody(t, resp2)
	info := body2["deploymentGroupInfo"].(map[string]any)
	assert.Equal(t, "renamed-grp", info["deploymentGroupName"])
}

// --- Deployment Tests ---

func TestCreateDeployment(t *testing.T) {
	s := newService()
	createApp(t, s, "dep-app")
	createGroup(t, s, "dep-app", "dep-grp")

	ctx := jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "dep-app",
		"deploymentGroupName": "dep-grp",
		"description":         "test deploy",
		"revision": map[string]any{
			"revisionType": "S3",
			"s3Location": map[string]any{
				"bucket":     "my-bucket",
				"key":        "app.zip",
				"bundleType": "zip",
			},
		},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["deploymentId"])
}

func TestCreateDeploymentAppNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateDeployment", map[string]any{
		"applicationName": "nope",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ApplicationDoesNotExistException")
}

func TestGetDeployment(t *testing.T) {
	s := newService()
	createApp(t, s, "getdep-app")
	createGroup(t, s, "getdep-app", "getdep-grp")

	resp, _ := s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "getdep-app",
		"deploymentGroupName": "getdep-grp",
	}))
	body := respBody(t, resp)
	depID := body["deploymentId"].(string)

	ctx := jsonCtx("GetDeployment", map[string]any{"deploymentId": depID})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	info := body2["deploymentInfo"].(map[string]any)
	assert.Equal(t, depID, info["deploymentId"])
	// With default lifecycle config (disabled), transitions are instant
	assert.Contains(t, []string{"Created", "InProgress", "Succeeded"}, info["status"])
}

func TestGetDeploymentNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("GetDeployment", map[string]any{"deploymentId": "nonexistent"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeploymentDoesNotExistException")
}

func TestListDeployments(t *testing.T) {
	s := newService()
	createApp(t, s, "listdep-app")
	createGroup(t, s, "listdep-app", "listdep-grp")
	s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "listdep-app",
		"deploymentGroupName": "listdep-grp",
	}))
	s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "listdep-app",
		"deploymentGroupName": "listdep-grp",
	}))

	ctx := jsonCtx("ListDeployments", map[string]any{"applicationName": "listdep-app"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	deps := body["deployments"].([]any)
	assert.Len(t, deps, 2)
}

func TestBatchGetDeployments(t *testing.T) {
	s := newService()
	createApp(t, s, "batchdep-app")
	createGroup(t, s, "batchdep-app", "batchdep-grp")

	resp, _ := s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "batchdep-app",
		"deploymentGroupName": "batchdep-grp",
	}))
	body := respBody(t, resp)
	depID := body["deploymentId"].(string)

	ctx := jsonCtx("BatchGetDeployments", map[string]any{"deploymentIds": []any{depID}})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	infos := body2["deploymentsInfo"].([]any)
	assert.Len(t, infos, 1)
}

func TestStopDeployment(t *testing.T) {
	s := newService()
	createApp(t, s, "stopdep-app")
	createGroup(t, s, "stopdep-app", "stopdep-grp")

	resp, _ := s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "stopdep-app",
		"deploymentGroupName": "stopdep-grp",
	}))
	body := respBody(t, resp)
	depID := body["deploymentId"].(string)

	ctx := jsonCtx("StopDeployment", map[string]any{"deploymentId": depID})
	_, err := s.HandleRequest(ctx)
	// With instant lifecycle, deployment may already be Succeeded and thus not stoppable.
	if err != nil {
		assert.Contains(t, err.Error(), "DeploymentAlreadyCompletedException")
	}
}

func TestStopDeploymentNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("StopDeployment", map[string]any{"deploymentId": "nonexistent"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeploymentDoesNotExistException")
}

func TestDeploymentLifecycleTransition(t *testing.T) {
	s := newService()
	createApp(t, s, "lcdep-app")
	createGroup(t, s, "lcdep-app", "lcdep-grp")

	resp, _ := s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "lcdep-app",
		"deploymentGroupName": "lcdep-grp",
	}))
	body := respBody(t, resp)
	depID := body["deploymentId"].(string)

	// With default lifecycle config (disabled), transitions are instant.
	// Give goroutine callbacks a moment to complete.
	time.Sleep(50 * time.Millisecond)

	ctx := jsonCtx("GetDeployment", map[string]any{"deploymentId": depID})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	status := body2["deploymentInfo"].(map[string]any)["status"].(string)
	assert.Equal(t, "Succeeded", status)
}

func TestBatchGetDeploymentTargets(t *testing.T) {
	s := newService()
	createApp(t, s, "target-app")
	createGroup(t, s, "target-app", "target-grp")

	resp, _ := s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "target-app",
		"deploymentGroupName": "target-grp",
	}))
	body := respBody(t, resp)
	depID := body["deploymentId"].(string)

	ctx := jsonCtx("BatchGetDeploymentTargets", map[string]any{
		"deploymentId": depID,
		"targetIds":    []any{"i-12345", "i-67890"},
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	targets := body2["deploymentTargets"].([]any)
	assert.Len(t, targets, 2)
}

// --- Behavioral: Per-Instance Deployment Tracking ---

func TestDeploymentTargets_LifecycleEvents(t *testing.T) {
	s := newService()
	createApp(t, s, "lcevent-app")
	createGroup(t, s, "lcevent-app", "lcevent-grp")

	resp, _ := s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "lcevent-app",
		"deploymentGroupName": "lcevent-grp",
	}))
	body := respBody(t, resp)
	depID := body["deploymentId"].(string)

	// Wait for lifecycle completion
	time.Sleep(100 * time.Millisecond)

	// BatchGetDeploymentTargets with synthetic targets
	ctx := jsonCtx("BatchGetDeploymentTargets", map[string]any{
		"deploymentId": depID,
		"targetIds":    []any{"i-target1", "i-target2"},
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	targets := body2["deploymentTargets"].([]any)
	assert.Len(t, targets, 2)

	// Check that targets have the deployment's status
	for _, target := range targets {
		tm := target.(map[string]any)
		assert.Equal(t, "InstanceTarget", tm["deploymentTargetType"])
		it := tm["instanceTarget"].(map[string]any)
		assert.NotEmpty(t, it["status"])
		assert.Equal(t, depID, it["deploymentId"])
	}
}

func TestDeploymentLifecycle_CompletesWithTargetStatus(t *testing.T) {
	s := newService()
	createApp(t, s, "targetlc-app")
	createGroup(t, s, "targetlc-app", "targetlc-grp")

	resp, _ := s.HandleRequest(jsonCtx("CreateDeployment", map[string]any{
		"applicationName":     "targetlc-app",
		"deploymentGroupName": "targetlc-grp",
	}))
	body := respBody(t, resp)
	depID := body["deploymentId"].(string)

	// With instant lifecycle, deployment should complete
	time.Sleep(100 * time.Millisecond)

	resp2, _ := s.HandleRequest(jsonCtx("GetDeployment", map[string]any{"deploymentId": depID}))
	body2 := respBody(t, resp2)
	info := body2["deploymentInfo"].(map[string]any)
	assert.Equal(t, "Succeeded", info["status"])
}

// --- On-Premises Instance Tags ---

func TestAddAndRemoveOnPremisesTags(t *testing.T) {
	s := newService()
	ctx := jsonCtx("AddTagsToOnPremisesInstances", map[string]any{
		"instanceNames": []any{"instance-1"},
		"tags":          []any{map[string]any{"Key": "env", "Value": "prod"}},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ctx2 := jsonCtx("RemoveTagsFromOnPremisesInstances", map[string]any{
		"instanceNames": []any{"instance-1"},
		"tags":          []any{map[string]any{"Key": "env"}},
	})
	resp2, err2 := s.HandleRequest(ctx2)
	require.NoError(t, err2)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}

// --- UpdateApplication ---

func TestUpdateApplication(t *testing.T) {
	s := newService()
	createApp(t, s, "rename-app")

	ctx := jsonCtx("UpdateApplication", map[string]any{
		"applicationName":    "rename-app",
		"newApplicationName": "renamed-app",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify new name exists
	resp2, err2 := s.HandleRequest(jsonCtx("GetApplication", map[string]any{"applicationName": "renamed-app"}))
	require.NoError(t, err2)
	body := respBody(t, resp2)
	assert.Equal(t, "renamed-app", body["application"].(map[string]any)["applicationName"])

	// Old name gone
	_, err3 := s.HandleRequest(jsonCtx("GetApplication", map[string]any{"applicationName": "rename-app"}))
	require.Error(t, err3)
}

func TestUpdateApplicationNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("UpdateApplication", map[string]any{
		"applicationName":    "nope",
		"newApplicationName": "new-name",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ApplicationDoesNotExistException")
}

func TestUpdateApplicationSameName(t *testing.T) {
	s := newService()
	createApp(t, s, "same-app")
	ctx := jsonCtx("UpdateApplication", map[string]any{
		"applicationName":    "same-app",
		"newApplicationName": "same-app",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- Deployment Config Tests ---

func TestCreateDeploymentConfig(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateDeploymentConfig", map[string]any{
		"deploymentConfigName": "my-config",
		"computePlatform":      "Server",
		"minimumHealthyHosts": map[string]any{
			"type":  "HOST_COUNT",
			"value": 2,
		},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["deploymentConfigId"])
}

func TestCreateDeploymentConfigDuplicate(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateDeploymentConfig", map[string]any{"deploymentConfigName": "dup-cfg"}))
	_, err := s.HandleRequest(jsonCtx("CreateDeploymentConfig", map[string]any{"deploymentConfigName": "dup-cfg"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeploymentConfigAlreadyExistsException")
}

func TestCreateDeploymentConfigMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateDeploymentConfig", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestGetDeploymentConfig(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateDeploymentConfig", map[string]any{
		"deploymentConfigName": "get-cfg",
		"computePlatform":      "Server",
	}))

	resp, err := s.HandleRequest(jsonCtx("GetDeploymentConfig", map[string]any{"deploymentConfigName": "get-cfg"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	info := body["deploymentConfigInfo"].(map[string]any)
	assert.Equal(t, "get-cfg", info["deploymentConfigName"])
	assert.Equal(t, "Server", info["computePlatform"])
}

func TestGetDeploymentConfigBuiltIn(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetDeploymentConfig", map[string]any{
		"deploymentConfigName": "CodeDeployDefault.OneAtATime",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	info := body["deploymentConfigInfo"].(map[string]any)
	assert.Equal(t, "CodeDeployDefault.OneAtATime", info["deploymentConfigName"])
}

func TestGetDeploymentConfigNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetDeploymentConfig", map[string]any{"deploymentConfigName": "nope"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeploymentConfigDoesNotExistException")
}

func TestListDeploymentConfigs(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateDeploymentConfig", map[string]any{"deploymentConfigName": "custom-cfg"}))

	resp, err := s.HandleRequest(jsonCtx("ListDeploymentConfigs", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	configs := body["deploymentConfigsList"].([]any)
	// Includes 7 built-ins + 1 custom
	assert.GreaterOrEqual(t, len(configs), 8)

	names := make(map[string]bool)
	for _, c := range configs {
		names[c.(string)] = true
	}
	assert.True(t, names["CodeDeployDefault.OneAtATime"])
	assert.True(t, names["custom-cfg"])
}

func TestDeleteDeploymentConfig(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateDeploymentConfig", map[string]any{"deploymentConfigName": "del-cfg"}))

	resp, err := s.HandleRequest(jsonCtx("DeleteDeploymentConfig", map[string]any{"deploymentConfigName": "del-cfg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err2 := s.HandleRequest(jsonCtx("GetDeploymentConfig", map[string]any{"deploymentConfigName": "del-cfg"}))
	require.Error(t, err2)
}

func TestDeleteDeploymentConfigBuiltIn(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteDeploymentConfig", map[string]any{
		"deploymentConfigName": "CodeDeployDefault.OneAtATime",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidDeploymentConfigNameException")
}

func TestCreateDeploymentConfigWithTrafficRouting(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateDeploymentConfig", map[string]any{
		"deploymentConfigName": "canary-cfg",
		"computePlatform":      "Lambda",
		"trafficRoutingConfig": map[string]any{
			"type": "TimeBasedCanary",
			"timeBasedCanary": map[string]any{
				"canaryPercentage": 10,
				"canaryInterval":   5,
			},
		},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, respBody(t, resp)["deploymentConfigId"])

	resp2, _ := s.HandleRequest(jsonCtx("GetDeploymentConfig", map[string]any{"deploymentConfigName": "canary-cfg"}))
	info := respBody(t, resp2)["deploymentConfigInfo"].(map[string]any)
	tr := info["trafficRoutingConfig"].(map[string]any)
	assert.Equal(t, "TimeBasedCanary", tr["type"])
	canary := tr["timeBasedCanary"].(map[string]any)
	assert.Equal(t, float64(10), canary["canaryPercentage"])
}

// --- Invalid Action ---

func TestInvalidAction(t *testing.T) {
	s := newService()
	ctx := jsonCtx("BogusAction", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

// --- Service Metadata ---

func TestServiceName(t *testing.T) {
	s := newService()
	assert.Equal(t, "codedeploy", s.Name())
}

func TestHealthCheck(t *testing.T) {
	s := newService()
	assert.NoError(t, s.HealthCheck())
}

func TestDefaultComputePlatform(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateApplication", map[string]any{"applicationName": "default-plat"})
	s.HandleRequest(ctx)

	resp, _ := s.HandleRequest(jsonCtx("GetApplication", map[string]any{"applicationName": "default-plat"}))
	body := respBody(t, resp)
	assert.Equal(t, "Server", body["application"].(map[string]any)["computePlatform"])
}

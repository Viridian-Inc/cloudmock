package codepipeline

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocatorForExec is the interface used by the executor to call other services.
type ServiceLocatorForExec interface {
	Lookup(name string) (service.Service, error)
}

// executeStages progresses through pipeline stages sequentially.
// Source actions succeed immediately, Build actions trigger CodeBuild,
// Deploy actions trigger CodeDeploy, and Approval actions pause.
// This is called asynchronously after StartPipelineExecution.
func executeStages(store *Store, exec *PipelineExecution, pipeline *Pipeline, locator ServiceLocatorForExec) {
	for i, stage := range pipeline.Stages {
		store.mu.Lock()
		if exec.Status != ExecStatusInProgress {
			store.mu.Unlock()
			return // execution was stopped
		}
		exec.StageStates[i].Status = ActionStatusInProgress
		store.mu.Unlock()

		allSucceeded := true
		for j, action := range stage.Actions {
			store.mu.Lock()
			if exec.Status != ExecStatusInProgress {
				store.mu.Unlock()
				return
			}
			exec.StageStates[i].ActionStates[j].Status = ActionStatusInProgress
			exec.StageStates[i].ActionStates[j].LastUpdate = time.Now().UTC()
			store.mu.Unlock()

			status := executeAction(action, locator)

			store.mu.Lock()
			exec.StageStates[i].ActionStates[j].Status = status
			exec.StageStates[i].ActionStates[j].LastUpdate = time.Now().UTC()
			if status != ActionStatusSucceeded {
				allSucceeded = false
			}
			store.mu.Unlock()

			if status == ActionStatusFailed {
				break
			}
		}

		store.mu.Lock()
		if allSucceeded {
			exec.StageStates[i].Status = ActionStatusSucceeded
		} else {
			exec.StageStates[i].Status = ActionStatusFailed
			exec.Status = ExecStatusFailed
			now := time.Now().UTC()
			exec.EndTime = &now
			store.mu.Unlock()
			return
		}
		store.mu.Unlock()
	}

	// All stages completed
	store.mu.Lock()
	exec.Status = ExecStatusSucceeded
	now := time.Now().UTC()
	exec.EndTime = &now
	store.mu.Unlock()
}

// executeAction runs a single action based on its category.
func executeAction(action ActionDeclaration, locator ServiceLocatorForExec) string {
	category := action.ActionTypeID.Category

	switch category {
	case "Source":
		// Source actions succeed immediately
		return ActionStatusSucceeded

	case "Build":
		return executeBuildAction(action, locator)

	case "Deploy":
		return executeDeployAction(action, locator)

	case "Approval":
		// Approval actions are handled externally via PutApprovalResult.
		// In automated mode, we auto-approve.
		return ActionStatusSucceeded

	default:
		// Unknown category: succeed by default
		return ActionStatusSucceeded
	}
}

// executeBuildAction triggers a CodeBuild StartBuild and returns success/failure.
func executeBuildAction(action ActionDeclaration, locator ServiceLocatorForExec) string {
	if locator == nil {
		return ActionStatusSucceeded // no locator, degrade gracefully
	}

	codebuildSvc, err := locator.Lookup("codebuild")
	if err != nil {
		return ActionStatusSucceeded // CodeBuild not available, degrade
	}

	projectName := action.Configuration["ProjectName"]
	if projectName == "" {
		return ActionStatusSucceeded
	}

	body, _ := json.Marshal(map[string]any{
		"projectName": projectName,
	})

	resp, err := codebuildSvc.HandleRequest(&service.RequestContext{
		Action:     "StartBuild",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	})
	if err != nil {
		return ActionStatusFailed
	}

	// Extract build ID and poll for completion
	buildID := extractBuildID(resp)
	if buildID == "" {
		return ActionStatusSucceeded
	}

	// Poll for build completion (simple loop with short sleeps)
	for attempts := 0; attempts < 100; attempts++ {
		time.Sleep(50 * time.Millisecond)
		status := getBuildStatus(codebuildSvc, buildID)
		switch status {
		case "SUCCEEDED":
			return ActionStatusSucceeded
		case "FAILED":
			return ActionStatusFailed
		case "STOPPED":
			return ActionStatusFailed
		}
	}

	return ActionStatusSucceeded // timed out polling, assume success
}

// executeDeployAction triggers a CodeDeploy CreateDeployment.
func executeDeployAction(action ActionDeclaration, locator ServiceLocatorForExec) string {
	if locator == nil {
		return ActionStatusSucceeded
	}

	codedeploySvc, err := locator.Lookup("codedeploy")
	if err != nil {
		return ActionStatusSucceeded
	}

	appName := action.Configuration["ApplicationName"]
	groupName := action.Configuration["DeploymentGroupName"]
	if appName == "" {
		return ActionStatusSucceeded
	}

	body, _ := json.Marshal(map[string]any{
		"applicationName":     appName,
		"deploymentGroupName": groupName,
	})

	_, err = codedeploySvc.HandleRequest(&service.RequestContext{
		Action:     "CreateDeployment",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	})
	if err != nil {
		return ActionStatusFailed
	}

	return ActionStatusSucceeded
}

// extractBuildID pulls the build ID from a CodeBuild StartBuild response.
func extractBuildID(resp *service.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}

	data, err := json.Marshal(resp.Body)
	if err != nil {
		return ""
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return ""
	}

	if build, ok := result["build"].(map[string]any); ok {
		if id, ok := build["id"].(string); ok {
			return id
		}
	}
	return ""
}

// getBuildStatus retrieves the current status of a CodeBuild build.
func getBuildStatus(codebuildSvc service.Service, buildID string) string {
	body, _ := json.Marshal(map[string]any{
		"ids": []string{buildID},
	})

	resp, err := codebuildSvc.HandleRequest(&service.RequestContext{
		Action:     "BatchGetBuilds",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	})
	if err != nil || resp == nil {
		return ""
	}

	data, _ := json.Marshal(resp.Body)
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return ""
	}

	if builds, ok := result["builds"].([]any); ok && len(builds) > 0 {
		if build, ok := builds[0].(map[string]any); ok {
			if status, ok := build["buildStatus"].(string); ok {
				return status
			}
		}
	}
	return ""
}

package sagemaker_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/sagemaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.SageMakerService {
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

// ---- Test 1: CreateNotebookInstance ----

func TestCreateNotebookInstance(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateNotebookInstance", map[string]any{
		"NotebookInstanceName": "test-nb",
		"InstanceType":         "ml.t2.medium",
		"RoleArn":              "arn:aws:iam::123456789012:role/SageMakerRole",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	assert.Contains(t, body["NotebookInstanceArn"], "test-nb")
}

// ---- Test 2: DescribeNotebookInstance ----

func TestDescribeNotebookInstance(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-desc",
		"RoleArn":              "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("DescribeNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-desc",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	assert.Equal(t, "nb-desc", body["NotebookInstanceName"])
	assert.NotEmpty(t, body["NotebookInstanceStatus"])
}

// ---- Test 3: ListNotebookInstances ----

func TestListNotebookInstances(t *testing.T) {
	s := newService()
	for _, name := range []string{"nb-1", "nb-2", "nb-3"} {
		_, err := s.HandleRequest(jsonCtx("CreateNotebookInstance", map[string]any{
			"NotebookInstanceName": name,
			"RoleArn":              "arn:aws:iam::123456789012:role/Role",
		}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListNotebookInstances", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	instances := body["NotebookInstances"].([]any)
	assert.Len(t, instances, 3)
}

// ---- Test 4: DeleteNotebookInstance ----

func TestDeleteNotebookInstance(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-del",
		"RoleArn":              "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	// Stop first (must be stopped to delete)
	_, err = s.HandleRequest(jsonCtx("StopNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-del",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("DeleteNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-del",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify removed
	_, err = s.HandleRequest(jsonCtx("DescribeNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-del",
	}))
	require.Error(t, err)
}

// ---- Test 5: UpdateNotebookInstance ----

func TestUpdateNotebookInstance(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-upd",
		"InstanceType":         "ml.t2.medium",
		"RoleArn":              "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("UpdateNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-upd",
		"InstanceType":         "ml.m5.large",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	descResp, err := s.HandleRequest(jsonCtx("DescribeNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-upd",
	}))
	require.NoError(t, err)
	body := respBody(t, descResp)
	assert.Equal(t, "ml.m5.large", body["InstanceType"])
}

// ---- Test 6: NotebookInstance NotFound ----

func TestNotebookInstanceNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeNotebookInstance", map[string]any{
		"NotebookInstanceName": "nonexistent",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFound", awsErr.Code)
}

// ---- Test 7: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 8: CreateTrainingJob + Describe + Lifecycle ----

func TestTrainingJobLifecycle(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateTrainingJob", map[string]any{
		"TrainingJobName": "train-1",
		"AlgorithmSpecification": map[string]any{
			"TrainingImage":     "123456789012.dkr.ecr.us-east-1.amazonaws.com/image:latest",
			"TrainingInputMode": "File",
		},
		"RoleArn": "arn:aws:iam::123456789012:role/Role",
		"OutputDataConfig": map[string]any{
			"S3OutputPath": "s3://bucket/output",
		},
		"ResourceConfig": map[string]any{
			"InstanceType":  "ml.m5.large",
			"InstanceCount": 1,
			"VolumeSizeInGB": 50,
		},
		"StoppingCondition": map[string]any{
			"MaxRuntimeInSeconds": 3600,
		},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Contains(t, body["TrainingJobArn"], "train-1")

	// Initially InProgress or may have already transitioned
	descResp, err := s.HandleRequest(jsonCtx("DescribeTrainingJob", map[string]any{
		"TrainingJobName": "train-1",
	}))
	require.NoError(t, err)
	descBody := respBody(t, descResp)
	assert.Contains(t, []string{"InProgress", "Completed"}, descBody["TrainingJobStatus"])

	// Wait for lifecycle transition
	time.Sleep(4 * time.Second)
	descResp2, err := s.HandleRequest(jsonCtx("DescribeTrainingJob", map[string]any{
		"TrainingJobName": "train-1",
	}))
	require.NoError(t, err)
	descBody2 := respBody(t, descResp2)
	assert.Equal(t, "Completed", descBody2["TrainingJobStatus"])
}

// ---- Test 9: ListTrainingJobs ----

func TestListTrainingJobs(t *testing.T) {
	s := newService()
	for _, name := range []string{"tj-1", "tj-2"} {
		_, err := s.HandleRequest(jsonCtx("CreateTrainingJob", map[string]any{
			"TrainingJobName": name,
			"RoleArn":         "arn:aws:iam::123456789012:role/Role",
		}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListTrainingJobs", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	summaries := body["TrainingJobSummaries"].([]any)
	assert.Len(t, summaries, 2)
}

// ---- Test 10: StopTrainingJob ----

func TestStopTrainingJob(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateTrainingJob", map[string]any{
		"TrainingJobName": "tj-stop",
		"RoleArn":         "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("StopTrainingJob", map[string]any{
		"TrainingJobName": "tj-stop",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---- Test 11: CreateModel + DescribeModel + ListModels + DeleteModel ----

func TestModelCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateModel", map[string]any{
		"ModelName":        "model-1",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
		"PrimaryContainer": map[string]any{
			"Image":        "123456789012.dkr.ecr.us-east-1.amazonaws.com/image:latest",
			"ModelDataUrl": "s3://bucket/model.tar.gz",
		},
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	assert.Contains(t, body["ModelArn"], "model-1")

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeModel", map[string]any{"ModelName": "model-1"}))
	require.NoError(t, err)
	descBody := respBody(t, descResp)
	assert.Equal(t, "model-1", descBody["ModelName"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListModels", map[string]any{}))
	require.NoError(t, err)
	listBody := respBody(t, listResp)
	models := listBody["Models"].([]any)
	assert.Len(t, models, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteModel", map[string]any{"ModelName": "model-1"}))
	require.NoError(t, err)

	// Verify deleted
	_, err = s.HandleRequest(jsonCtx("DescribeModel", map[string]any{"ModelName": "model-1"}))
	require.Error(t, err)
}

// ---- Test 12: EndpointConfig + Endpoint lifecycle ----

func TestEndpointLifecycle(t *testing.T) {
	s := newService()
	// Create endpoint config
	_, err := s.HandleRequest(jsonCtx("CreateEndpointConfig", map[string]any{
		"EndpointConfigName": "ec-1",
		"ProductionVariants": []map[string]any{
			{"VariantName": "v1", "ModelName": "model-1", "InstanceType": "ml.m5.large", "InitialInstanceCount": 1},
		},
	}))
	require.NoError(t, err)

	// Create endpoint
	epResp, err := s.HandleRequest(jsonCtx("CreateEndpoint", map[string]any{
		"EndpointName":       "ep-1",
		"EndpointConfigName": "ec-1",
	}))
	require.NoError(t, err)
	epBody := respBody(t, epResp)
	assert.Contains(t, epBody["EndpointArn"], "ep-1")

	// Describe endpoint - initially Creating or may have transitioned
	descResp, err := s.HandleRequest(jsonCtx("DescribeEndpoint", map[string]any{"EndpointName": "ep-1"}))
	require.NoError(t, err)
	descBody := respBody(t, descResp)
	assert.Contains(t, []string{"Creating", "InService"}, descBody["EndpointStatus"])

	// Wait for InService
	time.Sleep(3 * time.Second)
	descResp2, err := s.HandleRequest(jsonCtx("DescribeEndpoint", map[string]any{"EndpointName": "ep-1"}))
	require.NoError(t, err)
	descBody2 := respBody(t, descResp2)
	assert.Equal(t, "InService", descBody2["EndpointStatus"])

	// List endpoints
	listResp, err := s.HandleRequest(jsonCtx("ListEndpoints", map[string]any{}))
	require.NoError(t, err)
	listBody := respBody(t, listResp)
	eps := listBody["Endpoints"].([]any)
	assert.Len(t, eps, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteEndpoint", map[string]any{"EndpointName": "ep-1"}))
	require.NoError(t, err)
}

// ---- Test 13: UpdateEndpoint ----

func TestUpdateEndpoint(t *testing.T) {
	s := newService()
	// Create two configs
	_, err := s.HandleRequest(jsonCtx("CreateEndpointConfig", map[string]any{"EndpointConfigName": "ec-a"}))
	require.NoError(t, err)
	_, err = s.HandleRequest(jsonCtx("CreateEndpointConfig", map[string]any{"EndpointConfigName": "ec-b"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("CreateEndpoint", map[string]any{
		"EndpointName":       "ep-upd",
		"EndpointConfigName": "ec-a",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("UpdateEndpoint", map[string]any{
		"EndpointName":       "ep-upd",
		"EndpointConfigName": "ec-b",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Contains(t, body["EndpointArn"], "ep-upd")
}

// ---- Test 14: ProcessingJob ----

func TestProcessingJob(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateProcessingJob", map[string]any{
		"ProcessingJobName": "pj-1",
		"RoleArn":           "arn:aws:iam::123456789012:role/Role",
		"AppSpecification":  map[string]any{"ImageUri": "image:latest"},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Contains(t, body["ProcessingJobArn"], "pj-1")

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListProcessingJobs", map[string]any{}))
	require.NoError(t, err)
	listBody := respBody(t, listResp)
	assert.Len(t, listBody["ProcessingJobSummaries"].([]any), 1)

	// Stop
	_, err = s.HandleRequest(jsonCtx("StopProcessingJob", map[string]any{"ProcessingJobName": "pj-1"}))
	require.NoError(t, err)
}

// ---- Test 15: TransformJob ----

func TestTransformJob(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateTransformJob", map[string]any{
		"TransformJobName": "xj-1",
		"ModelName":        "model-1",
		"TransformInput":   map[string]any{"DataSource": map[string]any{"S3DataSource": map[string]any{"S3Uri": "s3://input"}}},
		"TransformOutput":  map[string]any{"S3OutputPath": "s3://output"},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Contains(t, body["TransformJobArn"], "xj-1")

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeTransformJob", map[string]any{"TransformJobName": "xj-1"}))
	require.NoError(t, err)
	descBody := respBody(t, descResp)
	assert.Contains(t, []string{"InProgress", "Completed"}, descBody["TransformJobStatus"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListTransformJobs", map[string]any{}))
	require.NoError(t, err)
	listBody := respBody(t, listResp)
	assert.Len(t, listBody["TransformJobSummaries"].([]any), 1)

	// Stop
	_, err = s.HandleRequest(jsonCtx("StopTransformJob", map[string]any{"TransformJobName": "xj-1"}))
	require.NoError(t, err)
}

// ---- Test 16: Tags (AddTags, ListTags, DeleteTags) ----

func TestTags(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateModel", map[string]any{
		"ModelName":        "model-tagged",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
		"Tags": []any{
			map[string]any{"Key": "env", "Value": "test"},
		},
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	arn := body["ModelArn"].(string)

	// Add more tags
	_, err = s.HandleRequest(jsonCtx("AddTags", map[string]any{
		"ResourceArn": arn,
		"Tags": []any{
			map[string]any{"Key": "team", "Value": "ml"},
		},
	}))
	require.NoError(t, err)

	// List tags
	listResp, err := s.HandleRequest(jsonCtx("ListTags", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	listBody := respBody(t, listResp)
	tags := listBody["Tags"].([]any)
	assert.Len(t, tags, 2)

	// Delete tags
	_, err = s.HandleRequest(jsonCtx("DeleteTags", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	// Verify only one tag remains
	listResp2, err := s.HandleRequest(jsonCtx("ListTags", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	listBody2 := respBody(t, listResp2)
	tags2 := listBody2["Tags"].([]any)
	assert.Len(t, tags2, 1)
}

// ---- Test 17: Duplicate resource ----

func TestDuplicateNotebookInstance(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-dup",
		"RoleArn":              "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("CreateNotebookInstance", map[string]any{
		"NotebookInstanceName": "nb-dup",
		"RoleArn":              "arn:aws:iam::123456789012:role/Role",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceInUse", awsErr.Code)
}

// ---- Test 18: EndpointConfig CRUD ----

func TestEndpointConfigCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateEndpointConfig", map[string]any{
		"EndpointConfigName": "econfig-1",
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	assert.Contains(t, body["EndpointConfigArn"], "econfig-1")

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeEndpointConfig", map[string]any{"EndpointConfigName": "econfig-1"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, descResp.StatusCode)

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListEndpointConfigs", map[string]any{}))
	require.NoError(t, err)
	listBody := respBody(t, listResp)
	assert.Len(t, listBody["EndpointConfigs"].([]any), 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteEndpointConfig", map[string]any{"EndpointConfigName": "econfig-1"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DescribeEndpointConfig", map[string]any{"EndpointConfigName": "econfig-1"}))
	require.Error(t, err)
}

// ---- Test 19: Service Name ----

func TestServiceName(t *testing.T) {
	s := newService()
	assert.Equal(t, "sagemaker", s.Name())
}

// ---- Test 20: HealthCheck ----

func TestHealthCheck(t *testing.T) {
	s := newService()
	assert.NoError(t, s.HealthCheck())
}

// ---- Behavioral: InvokeEndpoint ----

func TestInvokeEndpoint(t *testing.T) {
	s := newService()

	// Create model, endpoint config, endpoint
	s.HandleRequest(jsonCtx("CreateModel", map[string]any{
		"ModelName":        "my-model",
		"PrimaryContainer": map[string]any{"Image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-image:latest"},
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/sagemaker-role",
	}))

	s.HandleRequest(jsonCtx("CreateEndpointConfig", map[string]any{
		"EndpointConfigName": "my-config",
		"ProductionVariants": []any{map[string]any{"ModelName": "my-model", "VariantName": "AllTraffic", "InstanceType": "ml.m4.xlarge", "InitialInstanceCount": 1}},
	}))

	s.HandleRequest(jsonCtx("CreateEndpoint", map[string]any{
		"EndpointName":       "my-endpoint",
		"EndpointConfigName": "my-config",
	}))

	// Invoke endpoint
	resp, err := s.HandleRequest(jsonCtx("InvokeEndpoint", map[string]any{
		"EndpointName": "my-endpoint",
		"Body":         map[string]any{"features": []float64{1.0, 2.0, 3.0}},
		"ContentType":  "application/json",
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotNil(t, m["Body"])
	assert.Equal(t, "application/json", m["ContentType"])

	// Body should contain predictions
	body := m["Body"].(map[string]any)
	predictions := body["predictions"].([]any)
	assert.Len(t, predictions, 2)
}

func TestInvokeEndpointNotInService(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("InvokeEndpoint", map[string]any{
		"EndpointName": "nonexistent",
	}))
	require.Error(t, err)
}

// ---- Behavioral: Training job model artifacts ----

func TestTrainingJobModelArtifacts(t *testing.T) {
	s := newService()

	resp, err := s.HandleRequest(jsonCtx("CreateTrainingJob", map[string]any{
		"TrainingJobName":        "my-training-job",
		"AlgorithmSpecification": map[string]any{"TrainingImage": "image:latest", "TrainingInputMode": "File"},
		"RoleArn":                "arn:aws:iam::123456789012:role/sagemaker-role",
		"OutputDataConfig":       map[string]any{"S3OutputPath": "s3://my-bucket/output"},
		"ResourceConfig":         map[string]any{"InstanceType": "ml.m4.xlarge", "InstanceCount": 1, "VolumeSizeInGB": 10},
		"StoppingCondition":      map[string]any{"MaxRuntimeInSeconds": 3600},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)

	// Describe to get artifacts
	descResp, err := s.HandleRequest(jsonCtx("DescribeTrainingJob", map[string]any{
		"TrainingJobName": "my-training-job",
	}))
	require.NoError(t, err)
	descData := respBody(t, descResp)
	artifacts := descData["ModelArtifacts"].(map[string]any)
	s3Path := artifacts["S3ModelArtifacts"].(string)
	assert.Contains(t, s3Path, "s3://my-bucket/output")
	assert.Contains(t, s3Path, "my-training-job")
	_ = m
}

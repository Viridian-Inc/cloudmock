package batch

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// BatchService is the cloudmock implementation of the AWS Batch API.
type BatchService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new BatchService for the given AWS account ID and region.
func New(accountID, region string) *BatchService {
	return &BatchService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *BatchService) Name() string { return "batch" }

// Actions returns the list of Batch API actions supported by this service.
func (s *BatchService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateComputeEnvironment", Method: http.MethodPost, IAMAction: "batch:CreateComputeEnvironment"},
		{Name: "DescribeComputeEnvironments", Method: http.MethodPost, IAMAction: "batch:DescribeComputeEnvironments"},
		{Name: "UpdateComputeEnvironment", Method: http.MethodPost, IAMAction: "batch:UpdateComputeEnvironment"},
		{Name: "DeleteComputeEnvironment", Method: http.MethodPost, IAMAction: "batch:DeleteComputeEnvironment"},
		{Name: "CreateJobQueue", Method: http.MethodPost, IAMAction: "batch:CreateJobQueue"},
		{Name: "DescribeJobQueues", Method: http.MethodPost, IAMAction: "batch:DescribeJobQueues"},
		{Name: "UpdateJobQueue", Method: http.MethodPost, IAMAction: "batch:UpdateJobQueue"},
		{Name: "DeleteJobQueue", Method: http.MethodPost, IAMAction: "batch:DeleteJobQueue"},
		{Name: "RegisterJobDefinition", Method: http.MethodPost, IAMAction: "batch:RegisterJobDefinition"},
		{Name: "DescribeJobDefinitions", Method: http.MethodPost, IAMAction: "batch:DescribeJobDefinitions"},
		{Name: "DeregisterJobDefinition", Method: http.MethodPost, IAMAction: "batch:DeregisterJobDefinition"},
		{Name: "SubmitJob", Method: http.MethodPost, IAMAction: "batch:SubmitJob"},
		{Name: "DescribeJobs", Method: http.MethodPost, IAMAction: "batch:DescribeJobs"},
		{Name: "ListJobs", Method: http.MethodPost, IAMAction: "batch:ListJobs"},
		{Name: "CancelJob", Method: http.MethodPost, IAMAction: "batch:CancelJob"},
		{Name: "TerminateJob", Method: http.MethodPost, IAMAction: "batch:TerminateJob"},
		{Name: "CreateSchedulingPolicy", Method: http.MethodPost, IAMAction: "batch:CreateSchedulingPolicy"},
		{Name: "DescribeSchedulingPolicies", Method: http.MethodPost, IAMAction: "batch:DescribeSchedulingPolicies"},
		{Name: "UpdateSchedulingPolicy", Method: http.MethodPost, IAMAction: "batch:UpdateSchedulingPolicy"},
		{Name: "DeleteSchedulingPolicy", Method: http.MethodPost, IAMAction: "batch:DeleteSchedulingPolicy"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "batch:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "batch:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "batch:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *BatchService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Batch request to the appropriate handler.
// Batch uses REST-JSON protocol with path-based routing.
func (s *BatchService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	// REST routes
	switch {
	case path == "/v1/createcomputeenvironment" && method == http.MethodPost:
		return handleCreateComputeEnvironment(params, s.store)
	case path == "/v1/describecomputeenvironments" && method == http.MethodPost:
		return handleDescribeComputeEnvironments(params, s.store)
	case path == "/v1/updatecomputeenvironment" && method == http.MethodPost:
		return handleUpdateComputeEnvironment(params, s.store)
	case path == "/v1/deletecomputeenvironment" && method == http.MethodPost:
		return handleDeleteComputeEnvironment(params, s.store)
	case path == "/v1/createjobqueue" && method == http.MethodPost:
		return handleCreateJobQueue(params, s.store)
	case path == "/v1/describejobqueues" && method == http.MethodPost:
		return handleDescribeJobQueues(params, s.store)
	case path == "/v1/updatejobqueue" && method == http.MethodPost:
		return handleUpdateJobQueue(params, s.store)
	case path == "/v1/deletejobqueue" && method == http.MethodPost:
		return handleDeleteJobQueue(params, s.store)
	case path == "/v1/registerjobdefinition" && method == http.MethodPost:
		return handleRegisterJobDefinition(params, s.store)
	case path == "/v1/describejobdefinitions" && method == http.MethodPost:
		return handleDescribeJobDefinitions(params, s.store)
	case path == "/v1/deregisterjobdefinition" && method == http.MethodPost:
		return handleDeregisterJobDefinition(params, s.store)
	case path == "/v1/submitjob" && method == http.MethodPost:
		return handleSubmitJob(params, s.store)
	case path == "/v1/describejobs" && method == http.MethodPost:
		return handleDescribeJobs(params, s.store)
	case path == "/v1/listjobs" && method == http.MethodPost:
		return handleListJobs(params, s.store)
	case path == "/v1/canceljob" && method == http.MethodPost:
		return handleCancelJob(params, s.store)
	case path == "/v1/terminatejob" && method == http.MethodPost:
		return handleTerminateJob(params, s.store)
	case path == "/v1/createschedulingpolicy" && method == http.MethodPost:
		return handleCreateSchedulingPolicy(params, s.store)
	case path == "/v1/describeschedulingpolicies" && method == http.MethodPost:
		return handleDescribeSchedulingPolicies(params, s.store)
	case path == "/v1/updateschedulingpolicy" && method == http.MethodPost:
		return handleUpdateSchedulingPolicy(params, s.store)
	case path == "/v1/deleteschedulingpolicy" && method == http.MethodPost:
		return handleDeleteSchedulingPolicy(params, s.store)
	// Tag operations use path params: /v1/tags/{resourceArn}
	case strings.HasPrefix(path, "/v1/tags/") && method == http.MethodGet:
		return handleListTagsForResource(path, s.store)
	case strings.HasPrefix(path, "/v1/tags/") && method == http.MethodPost:
		return handleTagResource(path, params, s.store)
	case strings.HasPrefix(path, "/v1/tags/") && method == http.MethodDelete:
		return handleUntagResource(path, params, s.store)
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}

package cloudwatchlogs

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// CloudWatchLogsService is the cloudmock implementation of the AWS CloudWatch Logs API.
type CloudWatchLogsService struct {
	store *Store
}

// New returns a new CloudWatchLogsService for the given AWS account ID and region.
func New(accountID, region string) *CloudWatchLogsService {
	return &CloudWatchLogsService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
// CloudWatch Logs uses "logs" in the credential scope and X-Amz-Target prefix.
func (s *CloudWatchLogsService) Name() string { return "logs" }

// Actions returns the list of CloudWatch Logs API actions supported by this service.
func (s *CloudWatchLogsService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateLogGroup", Method: http.MethodPost, IAMAction: "logs:CreateLogGroup"},
		{Name: "DeleteLogGroup", Method: http.MethodPost, IAMAction: "logs:DeleteLogGroup"},
		{Name: "DescribeLogGroups", Method: http.MethodPost, IAMAction: "logs:DescribeLogGroups"},
		{Name: "CreateLogStream", Method: http.MethodPost, IAMAction: "logs:CreateLogStream"},
		{Name: "DeleteLogStream", Method: http.MethodPost, IAMAction: "logs:DeleteLogStream"},
		{Name: "DescribeLogStreams", Method: http.MethodPost, IAMAction: "logs:DescribeLogStreams"},
		{Name: "PutLogEvents", Method: http.MethodPost, IAMAction: "logs:PutLogEvents"},
		{Name: "GetLogEvents", Method: http.MethodPost, IAMAction: "logs:GetLogEvents"},
		{Name: "FilterLogEvents", Method: http.MethodPost, IAMAction: "logs:FilterLogEvents"},
		{Name: "PutRetentionPolicy", Method: http.MethodPost, IAMAction: "logs:PutRetentionPolicy"},
		{Name: "DeleteRetentionPolicy", Method: http.MethodPost, IAMAction: "logs:DeleteRetentionPolicy"},
		{Name: "TagLogGroup", Method: http.MethodPost, IAMAction: "logs:TagLogGroup"},
		{Name: "UntagLogGroup", Method: http.MethodPost, IAMAction: "logs:UntagLogGroup"},
		{Name: "ListTagsLogGroup", Method: http.MethodPost, IAMAction: "logs:ListTagsLogGroup"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *CloudWatchLogsService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CloudWatch Logs request to the appropriate handler.
// CloudWatch Logs uses the JSON protocol with X-Amz-Target: Logs_20140328.<Action>.
func (s *CloudWatchLogsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateLogGroup":
		return handleCreateLogGroup(ctx, s.store)
	case "DeleteLogGroup":
		return handleDeleteLogGroup(ctx, s.store)
	case "DescribeLogGroups":
		return handleDescribeLogGroups(ctx, s.store)
	case "CreateLogStream":
		return handleCreateLogStream(ctx, s.store)
	case "DeleteLogStream":
		return handleDeleteLogStream(ctx, s.store)
	case "DescribeLogStreams":
		return handleDescribeLogStreams(ctx, s.store)
	case "PutLogEvents":
		return handlePutLogEvents(ctx, s.store)
	case "GetLogEvents":
		return handleGetLogEvents(ctx, s.store)
	case "FilterLogEvents":
		return handleFilterLogEvents(ctx, s.store)
	case "PutRetentionPolicy":
		return handlePutRetentionPolicy(ctx, s.store)
	case "DeleteRetentionPolicy":
		return handleDeleteRetentionPolicy(ctx, s.store)
	case "TagLogGroup":
		return handleTagLogGroup(ctx, s.store)
	case "UntagLogGroup":
		return handleUntagLogGroup(ctx, s.store)
	case "ListTagsLogGroup":
		return handleListTagsLogGroup(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}

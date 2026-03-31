package elasticbeanstalk

import (
	"net/http"
	"net/url"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ElasticBeanstalkService is the cloudmock implementation of the AWS Elastic Beanstalk API.
type ElasticBeanstalkService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ElasticBeanstalkService for the given AWS account ID and region.
func New(accountID, region string) *ElasticBeanstalkService {
	return &ElasticBeanstalkService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *ElasticBeanstalkService) Name() string { return "elasticbeanstalk" }

// Actions returns the list of Elastic Beanstalk API actions supported by this service.
func (s *ElasticBeanstalkService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateApplication", Method: http.MethodPost, IAMAction: "elasticbeanstalk:CreateApplication"},
		{Name: "DescribeApplications", Method: http.MethodPost, IAMAction: "elasticbeanstalk:DescribeApplications"},
		{Name: "DeleteApplication", Method: http.MethodPost, IAMAction: "elasticbeanstalk:DeleteApplication"},
		{Name: "CreateApplicationVersion", Method: http.MethodPost, IAMAction: "elasticbeanstalk:CreateApplicationVersion"},
		{Name: "DescribeApplicationVersions", Method: http.MethodPost, IAMAction: "elasticbeanstalk:DescribeApplicationVersions"},
		{Name: "CreateEnvironment", Method: http.MethodPost, IAMAction: "elasticbeanstalk:CreateEnvironment"},
		{Name: "DescribeEnvironments", Method: http.MethodPost, IAMAction: "elasticbeanstalk:DescribeEnvironments"},
		{Name: "TerminateEnvironment", Method: http.MethodPost, IAMAction: "elasticbeanstalk:TerminateEnvironment"},
		{Name: "CreateConfigurationTemplate", Method: http.MethodPost, IAMAction: "elasticbeanstalk:CreateConfigurationTemplate"},
		{Name: "DescribeConfigurationSettings", Method: http.MethodPost, IAMAction: "elasticbeanstalk:DescribeConfigurationSettings"},
		{Name: "DeleteConfigurationTemplate", Method: http.MethodPost, IAMAction: "elasticbeanstalk:DeleteConfigurationTemplate"},
	}
}

// HealthCheck always returns nil.
func (s *ElasticBeanstalkService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Elastic Beanstalk request to the appropriate handler.
// Elastic Beanstalk uses the query protocol: form-encoded input, XML output.
func (s *ElasticBeanstalkService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateApplication":
		return handleCreateApplication(ctx, s.store)
	case "DescribeApplications":
		return handleDescribeApplications(ctx, s.store)
	case "DeleteApplication":
		return handleDeleteApplication(ctx, s.store)
	case "CreateApplicationVersion":
		return handleCreateApplicationVersion(ctx, s.store)
	case "DescribeApplicationVersions":
		return handleDescribeApplicationVersions(ctx, s.store)
	case "CreateEnvironment":
		return handleCreateEnvironment(ctx, s.store)
	case "DescribeEnvironments":
		return handleDescribeEnvironments(ctx, s.store)
	case "TerminateEnvironment":
		return handleTerminateEnvironment(ctx, s.store)
	case "CreateConfigurationTemplate":
		return handleCreateConfigurationTemplate(ctx, s.store)
	case "DescribeConfigurationSettings":
		return handleDescribeConfigurationSettings(ctx, s.store)
	case "DeleteConfigurationTemplate":
		return handleDeleteConfigurationTemplate(ctx, s.store)
	default:
		return xmlErr(service.NewAWSError("InvalidAction",
			"The action "+action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}

func parseForm(ctx *service.RequestContext) url.Values {
	form, _ := url.ParseQuery(string(ctx.Body))
	return form
}

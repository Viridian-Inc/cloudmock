package elasticbeanstalk_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/elasticbeanstalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ElasticBeanstalkService { return svc.New("123456789012", "us-east-1") }

func queryCtx(action string, params map[string]string) *service.RequestContext {
	vals := url.Values{}
	vals.Set("Action", action)
	for k, v := range params {
		vals.Set(k, v)
	}
	return &service.RequestContext{
		Action: action, Region: "us-east-1", AccountID: "123456789012",
		Body:       []byte(vals.Encode()),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func TestEB_CreateAndDescribeApplication(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(queryCtx("CreateApplication", map[string]string{
		"ApplicationName": "my-app", "Description": "Test app",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Marshal XML to check response
	data, _ := xml.Marshal(resp.Body)
	body := string(data)
	assert.Contains(t, body, "my-app")

	descResp, _ := s.HandleRequest(queryCtx("DescribeApplications", nil))
	descData, _ := xml.Marshal(descResp.Body)
	assert.Contains(t, string(descData), "my-app")
}

func TestEB_DeleteApplication(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "del-app"}))
	resp, err := s.HandleRequest(queryCtx("DeleteApplication", map[string]string{"ApplicationName": "del-app"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	descResp, _ := s.HandleRequest(queryCtx("DescribeApplications", nil))
	descData, _ := xml.Marshal(descResp.Body)
	assert.NotContains(t, string(descData), "del-app")
}

func TestEB_CreateApplicationVersion(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "ver-app"}))

	resp, err := s.HandleRequest(queryCtx("CreateApplicationVersion", map[string]string{
		"ApplicationName": "ver-app", "VersionLabel": "v1.0", "Description": "First version",
		"SourceBundle.S3Bucket": "my-bucket", "SourceBundle.S3Key": "app.zip",
	}))
	require.NoError(t, err)
	data, _ := xml.Marshal(resp.Body)
	assert.Contains(t, string(data), "v1.0")

	descResp, _ := s.HandleRequest(queryCtx("DescribeApplicationVersions", map[string]string{"ApplicationName": "ver-app"}))
	descData, _ := xml.Marshal(descResp.Body)
	assert.Contains(t, string(descData), "v1.0")
}

func TestEB_CreateAndDescribeEnvironment(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "env-app"}))

	envResp, err := s.HandleRequest(queryCtx("CreateEnvironment", map[string]string{
		"ApplicationName": "env-app", "EnvironmentName": "my-env",
		"SolutionStackName": "64bit Amazon Linux 2 v3.5.0 running Docker",
	}))
	require.NoError(t, err)
	envData, _ := xml.Marshal(envResp.Body)
	envBody := string(envData)
	assert.Contains(t, envBody, "my-env")
	assert.Contains(t, envBody, "Launching")

	descResp, _ := s.HandleRequest(queryCtx("DescribeEnvironments", map[string]string{"ApplicationName": "env-app"}))
	descData, _ := xml.Marshal(descResp.Body)
	assert.Contains(t, string(descData), "my-env")
}

func TestEB_TerminateEnvironment(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "term-app"}))
	s.HandleRequest(queryCtx("CreateEnvironment", map[string]string{
		"ApplicationName": "term-app", "EnvironmentName": "term-env",
	}))

	resp, err := s.HandleRequest(queryCtx("TerminateEnvironment", map[string]string{"EnvironmentName": "term-env"}))
	require.NoError(t, err)
	data, _ := xml.Marshal(resp.Body)
	assert.Contains(t, string(data), "Terminating")
}

func TestEB_ConfigurationTemplate(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "tmpl-app"}))

	tmplResp, err := s.HandleRequest(queryCtx("CreateConfigurationTemplate", map[string]string{
		"ApplicationName": "tmpl-app", "TemplateName": "my-tmpl",
		"SolutionStackName": "64bit Amazon Linux 2", "Description": "Test template",
	}))
	require.NoError(t, err)
	tmplData, _ := xml.Marshal(tmplResp.Body)
	assert.Contains(t, string(tmplData), "my-tmpl")

	descResp, _ := s.HandleRequest(queryCtx("DescribeConfigurationSettings", map[string]string{
		"ApplicationName": "tmpl-app", "TemplateName": "my-tmpl",
	}))
	descData, _ := xml.Marshal(descResp.Body)
	assert.Contains(t, string(descData), "my-tmpl")

	s.HandleRequest(queryCtx("DeleteConfigurationTemplate", map[string]string{
		"ApplicationName": "tmpl-app", "TemplateName": "my-tmpl",
	}))
}

func TestEB_EnvironmentNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("TerminateEnvironment", map[string]string{"EnvironmentName": "nonexistent"}))
	require.Error(t, err)
}

func TestEB_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("BogusAction", nil))
	require.Error(t, err)
}

func TestEB_DuplicateApplication(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "dup-app"}))
	_, err := s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "dup-app"}))
	require.Error(t, err)
}

func TestEB_DuplicateVersionLabel(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "vl-app"}))
	_, err := s.HandleRequest(queryCtx("CreateApplicationVersion", map[string]string{
		"ApplicationName": "vl-app", "VersionLabel": "v1",
	}))
	require.NoError(t, err)

	// Same version label should fail
	_, err = s.HandleRequest(queryCtx("CreateApplicationVersion", map[string]string{
		"ApplicationName": "vl-app", "VersionLabel": "v1",
	}))
	require.Error(t, err)
}

func TestEB_EnvironmentURLFormat(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "url-app"}))
	envResp, err := s.HandleRequest(queryCtx("CreateEnvironment", map[string]string{
		"ApplicationName": "url-app", "EnvironmentName": "prod-env",
	}))
	require.NoError(t, err)
	data, _ := xml.Marshal(envResp.Body)
	body := string(data)
	// URL should follow pattern: {env-name}.{region}.elasticbeanstalk.com
	assert.Contains(t, body, "prod-env.us-east-1.elasticbeanstalk.com")
}

func TestEB_EnvironmentHealthTracking(t *testing.T) {
	s := newService()
	s.HandleRequest(queryCtx("CreateApplication", map[string]string{"ApplicationName": "health-app"}))
	envResp, err := s.HandleRequest(queryCtx("CreateEnvironment", map[string]string{
		"ApplicationName": "health-app", "EnvironmentName": "health-env",
	}))
	require.NoError(t, err)
	data, _ := xml.Marshal(envResp.Body)
	body := string(data)
	// Initial state should be Launching with Grey health
	assert.Contains(t, body, "<Health>Grey</Health>")
	assert.Contains(t, body, "<Status>Launching</Status>")
}

func TestEB_EnvironmentRequiresApp(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("CreateEnvironment", map[string]string{
		"ApplicationName": "nonexistent-app", "EnvironmentName": "env1",
	}))
	require.Error(t, err)
}

package elasticmapreduce_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/elasticmapreduce"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.EMRService {
	return svc.New("123456789012", "us-east-1")
}

func queryCtx(action string, params map[string]string) *service.RequestContext {
	vals := url.Values{}
	vals.Set("Action", action)
	for k, v := range params {
		vals.Set(k, v)
	}
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       []byte(vals.Encode()),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

type runJobFlowResult struct {
	XMLName xml.Name `xml:"RunJobFlowResponse"`
	Result  struct {
		JobFlowId string `xml:"JobFlowId"`
	} `xml:"RunJobFlowResult"`
}

func createCluster(t *testing.T, s *svc.EMRService, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(queryCtx("RunJobFlow", map[string]string{
		"Name": name, "ReleaseLabel": "emr-6.5.0", "ServiceRole": "EMR_DefaultRole",
		"Applications.member.1.Name": "Spark",
	}))
	require.NoError(t, err)
	data, _ := xml.Marshal(resp.Body)
	var result runJobFlowResult
	require.NoError(t, xml.Unmarshal(data, &result))
	return result.Result.JobFlowId
}

func TestServiceName(t *testing.T) {
	assert.Equal(t, "elasticmapreduce", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestRunJobFlow(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "test-cluster")
	assert.NotEmpty(t, id)
}

func TestRunJobFlowMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("RunJobFlow", map[string]string{}))
	require.Error(t, err)
}

func TestDescribeCluster(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "desc-cluster")
	resp, err := s.HandleRequest(queryCtx("DescribeCluster", map[string]string{"ClusterId": id}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("DescribeCluster", map[string]string{"ClusterId": "j-nonexistent"}))
	require.Error(t, err)
}

func TestListClusters(t *testing.T) {
	s := newService()
	createCluster(t, s, "lc1")
	createCluster(t, s, "lc2")
	resp, err := s.HandleRequest(queryCtx("ListClusters", map[string]string{}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTerminateJobFlows(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "term-cluster")
	resp, err := s.HandleRequest(queryCtx("TerminateJobFlows", map[string]string{"JobFlowIds.member.1": id}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTerminationProtection(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "prot-cluster")
	_, err := s.HandleRequest(queryCtx("SetTerminationProtection", map[string]string{
		"JobFlowIds.member.1": id, "TerminationProtected": "true",
	}))
	require.NoError(t, err)

	// Now termination should be blocked
	_, err = s.HandleRequest(queryCtx("TerminateJobFlows", map[string]string{"JobFlowIds.member.1": id}))
	require.NoError(t, err) // doesn't error, just doesn't terminate
}

func TestSetVisibleToAllUsers(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "vis-cluster")
	resp, err := s.HandleRequest(queryCtx("SetVisibleToAllUsers", map[string]string{
		"JobFlowIds.member.1": id, "VisibleToAllUsers": "false",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddJobFlowSteps(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "step-cluster")
	resp, err := s.HandleRequest(queryCtx("AddJobFlowSteps", map[string]string{
		"JobFlowId":                        id,
		"Steps.member.1.Name":              "MyStep",
		"Steps.member.1.HadoopJarStep.Jar": "s3://bucket/step.jar",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListSteps(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "liststep-cluster")
	_, _ = s.HandleRequest(queryCtx("AddJobFlowSteps", map[string]string{
		"JobFlowId": id, "Steps.member.1.Name": "s1", "Steps.member.1.HadoopJarStep.Jar": "jar",
	}))
	resp, err := s.HandleRequest(queryCtx("ListSteps", map[string]string{"ClusterId": id}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddInstanceGroups(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "ig-cluster")
	resp, err := s.HandleRequest(queryCtx("AddInstanceGroups", map[string]string{
		"JobFlowId":                                   id,
		"InstanceGroups.member.1.InstanceRole":         "CORE",
		"InstanceGroups.member.1.InstanceType":         "m5.xlarge",
		"InstanceGroups.member.1.InstanceCount":        "3",
		"InstanceGroups.member.1.Name":                 "CoreNodes",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListInstanceGroups(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "lig-cluster")
	_, _ = s.HandleRequest(queryCtx("AddInstanceGroups", map[string]string{
		"JobFlowId": id, "InstanceGroups.member.1.InstanceRole": "TASK",
		"InstanceGroups.member.1.InstanceType": "m5.xlarge", "InstanceGroups.member.1.InstanceCount": "2",
	}))
	resp, err := s.HandleRequest(queryCtx("ListInstanceGroups", map[string]string{"ClusterId": id}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddTags(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "tag-cluster")
	resp, err := s.HandleRequest(queryCtx("AddTags", map[string]string{
		"ResourceId": id, "Tags.member.1.Key": "env", "Tags.member.1.Value": "prod",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRemoveTags(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "rmtag-cluster")
	_, _ = s.HandleRequest(queryCtx("AddTags", map[string]string{
		"ResourceId": id, "Tags.member.1.Key": "env", "Tags.member.1.Value": "dev",
	}))
	resp, err := s.HandleRequest(queryCtx("RemoveTags", map[string]string{
		"ResourceId": id, "TagKeys.member.1": "env",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("BogusAction", map[string]string{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Behavioral tests ----

func TestStepExecution_TransitionsToCompleted(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "step-trans")

	// Add steps
	resp, err := s.HandleRequest(queryCtx("AddJobFlowSteps", map[string]string{
		"JobFlowId":                        id,
		"Steps.member.1.Name":              "Step1",
		"Steps.member.1.HadoopJarStep.Jar": "s3://bucket/step.jar",
		"Steps.member.2.Name":              "Step2",
		"Steps.member.2.HadoopJarStep.Jar": "s3://bucket/step2.jar",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse step IDs
	data, _ := xml.Marshal(resp.Body)
	var addResult struct {
		XMLName xml.Name `xml:"AddJobFlowStepsResponse"`
		Result  struct {
			StepIds []string `xml:"StepIds>member"`
		} `xml:"AddJobFlowStepsResult"`
	}
	require.NoError(t, xml.Unmarshal(data, &addResult))
	assert.Len(t, addResult.Result.StepIds, 2)

	// Verify steps are COMPLETED
	for _, stepID := range addResult.Result.StepIds {
		stepResp, err := s.HandleRequest(queryCtx("DescribeStep", map[string]string{
			"ClusterId": id, "StepId": stepID,
		}))
		require.NoError(t, err)
		stepData, _ := xml.Marshal(stepResp.Body)
		var descResult struct {
			XMLName xml.Name `xml:"DescribeStepResponse"`
			Result  struct {
				Step struct {
					Status struct {
						State string `xml:"State"`
					} `xml:"Status"`
				} `xml:"Step"`
			} `xml:"DescribeStepResult"`
		}
		require.NoError(t, xml.Unmarshal(stepData, &descResult))
		assert.Equal(t, "COMPLETED", descResult.Result.Step.Status.State)
	}
}

func TestTerminateJobFlows_NoLocator(t *testing.T) {
	// Termination should work even without EC2 locator (graceful degradation)
	s := newService()
	id := createCluster(t, s, "term-noloc")
	resp, err := s.HandleRequest(queryCtx("TerminateJobFlows", map[string]string{
		"JobFlowIds.member.1": id,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestModifyInstanceGroups_CountUpdate(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "modify-ig")

	// Add an instance group
	addResp, err := s.HandleRequest(queryCtx("AddInstanceGroups", map[string]string{
		"JobFlowId":                                   id,
		"InstanceGroups.member.1.InstanceRole":         "CORE",
		"InstanceGroups.member.1.InstanceType":         "m5.xlarge",
		"InstanceGroups.member.1.InstanceCount":        "2",
		"InstanceGroups.member.1.Name":                 "Core",
	}))
	require.NoError(t, err)

	// Parse IG ID
	data, _ := xml.Marshal(addResp.Body)
	var addResult struct {
		XMLName xml.Name `xml:"AddInstanceGroupsResponse"`
		Result  struct {
			InstanceGroupIds []string `xml:"InstanceGroupIds>member"`
		} `xml:"AddInstanceGroupsResult"`
	}
	require.NoError(t, xml.Unmarshal(data, &addResult))
	igID := addResult.Result.InstanceGroupIds[0]

	// Modify instance count
	_, err = s.HandleRequest(queryCtx("ModifyInstanceGroups", map[string]string{
		"ClusterId":                                       id,
		"InstanceGroups.member.1.InstanceGroupId":         igID,
		"InstanceGroups.member.1.InstanceCount":           "5",
	}))
	require.NoError(t, err)

	// Verify the count changed
	listResp, err := s.HandleRequest(queryCtx("ListInstanceGroups", map[string]string{"ClusterId": id}))
	require.NoError(t, err)
	listData, _ := xml.Marshal(listResp.Body)
	var listResult struct {
		XMLName xml.Name `xml:"ListInstanceGroupsResponse"`
		Result  struct {
			InstanceGroups []struct {
				Id                     string `xml:"Id"`
				RequestedInstanceCount int    `xml:"RequestedInstanceCount"`
			} `xml:"InstanceGroups>member"`
		} `xml:"ListInstanceGroupsResult"`
	}
	require.NoError(t, xml.Unmarshal(listData, &listResult))
	for _, ig := range listResult.Result.InstanceGroups {
		if ig.Id == igID {
			assert.Equal(t, 5, ig.RequestedInstanceCount)
		}
	}
}

func TestClusterLifecycle_NoLocator(t *testing.T) {
	s := newService()
	id := createCluster(t, s, "lifecycle-cluster")

	// With default instant transitions, cluster should reach final state
	resp, err := s.HandleRequest(queryCtx("DescribeCluster", map[string]string{"ClusterId": id}))
	require.NoError(t, err)
	data, _ := xml.Marshal(resp.Body)
	var result struct {
		XMLName xml.Name `xml:"DescribeClusterResponse"`
		Result  struct {
			Cluster struct {
				Status struct {
					State string `xml:"State"`
				} `xml:"Status"`
			} `xml:"Cluster"`
		} `xml:"DescribeClusterResult"`
	}
	require.NoError(t, xml.Unmarshal(data, &result))
	// With instant transitions: STARTING -> BOOTSTRAPPING -> RUNNING
	assert.Equal(t, "RUNNING", result.Result.Cluster.Status.State)
}

package tier2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/benchmarks/harness"
)

// serviceSpec defines a minimal service for tier2 benchmarking.
type serviceSpec struct {
	Name     string
	Protocol string // "json", "query", "rest-json", "rest-xml"
	Target   string // X-Amz-Target prefix for JSON protocol (empty for query/rest)
	// ops is the ordered list of [create, describe, list, delete] action names.
	// Empty string means "skip this op" (e.g., service has no describe).
	ops [4]string
}

// services is the hardcoded list of all non-tier1 AWS services benchmarked at tier 2.
// Tier 1 services (ec2, s3, iam, sqs, sns, lambda, dynamodb, kinesis, rds, cloudwatch,
// cloudwatchlogs, cloudformation, ecs, ecr, kms, ssm, stepfunctions, route53,
// eventbridge, firehose, sts, cognito, apigateway, eks, codebuild, codepipeline,
// cloudtrail) are excluded — they have dedicated tier1 suites.
var services = []serviceSpec{
	// ── Query protocol ──────────────────────────────────────────────────────────
	{Name: "autoscaling", Protocol: "query", ops: [4]string{"CreateAutoScalingGroup", "", "DescribeAutoScalingGroups", "DeleteAutoScalingGroup"}},
	{Name: "elasticloadbalancing", Protocol: "query", ops: [4]string{"CreateLoadBalancer", "", "DescribeLoadBalancers", "DeleteLoadBalancer"}},
	{Name: "elasticbeanstalk", Protocol: "query", ops: [4]string{"CreateApplication", "", "DescribeApplications", "DeleteApplication"}},
	{Name: "elasticache", Protocol: "query", ops: [4]string{"CreateCacheCluster", "", "DescribeCacheClusters", "DeleteCacheCluster"}},
	{Name: "redshift", Protocol: "query", ops: [4]string{"CreateCluster", "", "DescribeClusters", "DeleteCluster"}},
	{Name: "neptune", Protocol: "query", ops: [4]string{"CreateDBInstance", "", "DescribeDBInstances", "DeleteDBInstance"}},
	{Name: "es", Protocol: "query", ops: [4]string{"CreateElasticsearchDomain", "DescribeElasticsearchDomain", "", "DeleteElasticsearchDomain"}},
	{Name: "elasticmapreduce", Protocol: "query", ops: [4]string{"RunJobFlow", "DescribeCluster", "ListClusters", "TerminateJobFlows"}},
	{Name: "shield", Protocol: "query", ops: [4]string{"CreateProtection", "DescribeProtection", "ListProtections", "DeleteProtection"}},
	{Name: "waf-regional", Protocol: "query", ops: [4]string{"CreateWebACL", "GetWebACL", "ListWebACLs", "DeleteWebACL"}},
	{Name: "docdb", Protocol: "query", ops: [4]string{"CreateDBCluster", "", "DescribeDBClusters", "DeleteDBCluster"}},
	// ── JSON protocol ───────────────────────────────────────────────────────────
	{Name: "acm", Protocol: "json", Target: "CertificateManager", ops: [4]string{"RequestCertificate", "DescribeCertificate", "ListCertificates", "DeleteCertificate"}},
	{Name: "acm-pca", Protocol: "json", Target: "ACMPrivateCA", ops: [4]string{"CreateCertificateAuthority", "DescribeCertificateAuthority", "ListCertificateAuthorities", ""}},
	{Name: "appconfig", Protocol: "json", Target: "AppConfig", ops: [4]string{"CreateApplication", "GetApplication", "ListApplications", "DeleteApplication"}},
	{Name: "application-autoscaling", Protocol: "json", Target: "AnyScaleFrontendService", ops: [4]string{"RegisterScalableTarget", "", "DescribeScalableTargets", "DeregisterScalableTarget"}},
	{Name: "athena", Protocol: "json", Target: "AmazonAthena", ops: [4]string{"StartQueryExecution", "GetQueryExecution", "ListQueryExecutions", "StopQueryExecution"}},
	{Name: "backup", Protocol: "json", Target: "CryoControllerUserManager", ops: [4]string{"CreateBackupPlan", "DescribeBackupJob", "ListBackupPlans", "DeleteBackupPlan"}},
	{Name: "codebuild", Protocol: "json", Target: "CodeBuild_20161006", ops: [4]string{"CreateProject", "BatchGetProjects", "ListProjects", "DeleteProject"}},
	{Name: "codecommit", Protocol: "json", Target: "CodeCommit_20150413", ops: [4]string{"CreateRepository", "GetRepository", "ListRepositories", "DeleteRepository"}},
	{Name: "codedeploy", Protocol: "json", Target: "CodeDeploy_20141006", ops: [4]string{"CreateApplication", "GetApplication", "ListApplications", "DeleteApplication"}},
	{Name: "codepipeline", Protocol: "json", Target: "CodePipeline_20150709", ops: [4]string{"CreatePipeline", "GetPipeline", "ListPipelines", "DeletePipeline"}},
	{Name: "codeconnections", Protocol: "json", Target: "CodeConnections_20231201", ops: [4]string{"CreateConnection", "GetConnection", "ListConnections", "DeleteConnection"}},
	{Name: "config", Protocol: "json", Target: "StarlingDoveService", ops: [4]string{"PutConfigRule", "", "DescribeConfigRules", "DeleteConfigRule"}},
	{Name: "ce", Protocol: "json", Target: "AWSInsightsIndexService", ops: [4]string{"GetCostAndUsage", "GetCostForecast", "ListCostAllocationTags", ""}},
	{Name: "dms", Protocol: "json", Target: "AmazonDMSv20160101", ops: [4]string{"CreateReplicationInstance", "", "DescribeReplicationInstances", "DeleteReplicationInstance"}},
	{Name: "glue", Protocol: "json", Target: "AWSGlue", ops: [4]string{"CreateDatabase", "GetDatabase", "GetDatabases", "DeleteDatabase"}},
	{Name: "identitystore", Protocol: "json", Target: "AWSIdentityStore", ops: [4]string{"CreateUser", "DescribeUser", "ListUsers", "DeleteUser"}},
	{Name: "lakeformation", Protocol: "json", Target: "AWSLakeFormation", ops: [4]string{"RegisterResource", "DescribeResource", "ListResources", "DeregisterResource"}},
	{Name: "memorydb", Protocol: "json", Target: "AmazonMemoryDB", ops: [4]string{"CreateCluster", "", "DescribeClusters", "DeleteCluster"}},
	{Name: "organizations", Protocol: "json", Target: "AWSOrganizationsV20161128", ops: [4]string{"CreateOrganization", "DescribeOrganization", "ListAccounts", ""}},
	{Name: "ram", Protocol: "json", Target: "AWSRAMShareService", ops: [4]string{"CreateResourceShare", "", "GetResourceShares", "DeleteResourceShare"}},
	{Name: "tagging", Protocol: "json", Target: "ResourceGroupsTaggingAPI_20170126", ops: [4]string{"TagResources", "UntagResources", "GetResources", ""}},
	{Name: "sagemaker", Protocol: "json", Target: "SageMaker", ops: [4]string{"CreateNotebookInstance", "DescribeNotebookInstance", "ListNotebookInstances", "DeleteNotebookInstance"}},
	{Name: "servicediscovery", Protocol: "json", Target: "Route53AutoNaming_v20170314", ops: [4]string{"CreateService", "GetService", "ListServices", "DeleteService"}},
	{Name: "swf", Protocol: "json", Target: "SimpleWorkflowService", ops: [4]string{"RegisterDomain", "DescribeDomain", "ListDomains", "DeprecateDomain"}},
	{Name: "sso-admin", Protocol: "json", Target: "SWBExternalService", ops: [4]string{"CreatePermissionSet", "DescribePermissionSet", "ListPermissionSets", "DeletePermissionSet"}},
	{Name: "support", Protocol: "json", Target: "AWSSupport_20130415", ops: [4]string{"CreateCase", "DescribeTrustedAdvisorChecks", "DescribeCases", ""}},
	{Name: "textract", Protocol: "json", Target: "Textract", ops: [4]string{"StartDocumentTextDetection", "GetDocumentTextDetection", "DetectDocumentText", "AnalyzeDocument"}},
	{Name: "timestream-write", Protocol: "json", Target: "Timestream_20181101", ops: [4]string{"CreateDatabase", "DescribeDatabase", "ListDatabases", "DeleteDatabase"}},
	{Name: "transcribe", Protocol: "json", Target: "Transcribe", ops: [4]string{"StartTranscriptionJob", "GetTranscriptionJob", "ListTranscriptionJobs", "DeleteTranscriptionJob"}},
	{Name: "transfer", Protocol: "json", Target: "TransferService", ops: [4]string{"CreateServer", "DescribeServer", "ListServers", "DeleteServer"}},
	{Name: "verifiedpermissions", Protocol: "json", Target: "VerifiedPermissions", ops: [4]string{"CreatePolicyStore", "GetPolicyStore", "ListPolicyStores", "DeletePolicyStore"}},
	{Name: "cloudcontrol", Protocol: "json", Target: "CloudApiService", ops: [4]string{"CreateResource", "GetResource", "ListResources", "DeleteResource"}},
	{Name: "cloudtrail", Protocol: "json", Target: "CloudTrail_20131101", ops: [4]string{"CreateTrail", "GetTrail", "DescribeTrails", "DeleteTrail"}},
	{Name: "kinesisanalytics", Protocol: "json", Target: "KinesisAnalytics_20180523", ops: [4]string{"CreateApplication", "DescribeApplication", "ListApplications", "DeleteApplication"}},
	{Name: "xray", Protocol: "json", Target: "AWSXRay", ops: [4]string{"PutTraceSegments", "BatchGetTraces", "GetTraceSummaries", ""}},
	// ── REST-JSON protocol ───────────────────────────────────────────────────────
	{Name: "appsync", Protocol: "rest-json", ops: [4]string{"CreateGraphqlApi", "", "ListGraphqlApis", "DeleteGraphqlApi"}},
	{Name: "batch", Protocol: "rest-json", ops: [4]string{"CreateJobQueue", "", "ListJobQueues", "DeleteJobQueue"}},
	{Name: "bedrock", Protocol: "rest-json", ops: [4]string{"CreateModelCustomizationJob", "GetModelCustomizationJob", "ListModelCustomizationJobs", ""}},
	{Name: "codeartifact", Protocol: "rest-json", ops: [4]string{"CreateRepository", "", "ListRepositories", "DeleteRepository"}},
	{Name: "eks", Protocol: "rest-json", ops: [4]string{"CreateCluster", "", "ListClusters", "DeleteCluster"}},
	{Name: "fis", Protocol: "rest-json", ops: [4]string{"CreateExperimentTemplate", "", "ListExperimentTemplates", "DeleteExperimentTemplate"}},
	{Name: "iot", Protocol: "rest-json", ops: [4]string{"CreateThing", "", "ListThings", "DeleteThing"}},
	{Name: "iot-data", Protocol: "rest-json", ops: [4]string{"CreateThingShadow", "", "ListThingShadows", "DeleteThingShadow"}},
	{Name: "iot-wireless", Protocol: "rest-json", ops: [4]string{"CreateWirelessDevice", "", "ListWirelessDevices", "DeleteWirelessDevice"}},
	{Name: "managedblockchain", Protocol: "rest-json", ops: [4]string{"CreateNetwork", "", "ListNetworks", "DeleteNetwork"}},
	{Name: "kafka", Protocol: "rest-json", ops: [4]string{"CreateCluster", "", "ListClusters", "DeleteCluster"}},
	{Name: "airflow", Protocol: "rest-json", ops: [4]string{"CreateEnvironment", "", "ListEnvironments", "DeleteEnvironment"}},
	{Name: "mq", Protocol: "rest-json", ops: [4]string{"CreateBroker", "", "ListBrokers", "DeleteBroker"}},
	{Name: "opensearch", Protocol: "rest-json", ops: [4]string{"CreateDomain", "", "ListDomains", "DeleteDomain"}},
	{Name: "pinpoint", Protocol: "rest-json", ops: [4]string{"CreateApp", "", "ListApps", "DeleteApp"}},
	{Name: "resource-groups", Protocol: "rest-json", ops: [4]string{"CreateGroup", "", "ListGroups", "DeleteGroup"}},
	{Name: "serverlessrepo", Protocol: "rest-json", ops: [4]string{"CreateApplication", "", "ListApplications", "DeleteApplication"}},
	{Name: "amplify", Protocol: "rest-json", ops: [4]string{"CreateApp", "", "ListApps", "DeleteApp"}},
	{Name: "account", Protocol: "rest-json", ops: [4]string{"CreateContactInformation", "", "ListContactInformations", "DeleteContactInformation"}},
	{Name: "glacier", Protocol: "rest-json", ops: [4]string{"CreateVault", "", "ListVaults", "DeleteVault"}},
	{Name: "mediaconvert", Protocol: "rest-json", ops: [4]string{"CreateJob", "", "ListJobs", "DeleteJob"}},
	{Name: "pipes", Protocol: "rest-json", ops: [4]string{"CreatePipe", "", "ListPipes", "DeletePipe"}},
	{Name: "scheduler", Protocol: "rest-json", ops: [4]string{"CreateSchedule", "", "ListSchedules", "DeleteSchedule"}},
	{Name: "s3tables", Protocol: "rest-json", ops: [4]string{"CreateTableBucket", "", "ListTableBuckets", "DeleteTableBucket"}},
	{Name: "wafv2", Protocol: "rest-json", ops: [4]string{"CreateWebACL", "", "ListWebACLs", "DeleteWebACL"}},
	{Name: "route53resolver", Protocol: "rest-json", ops: [4]string{"CreateResolverEndpoint", "", "ListResolverEndpoints", "DeleteResolverEndpoint"}},
	{Name: "apprunner", Protocol: "rest-json", ops: [4]string{"CreateService", "", "ListServices", "DeleteService"}},
	{Name: "appmesh", Protocol: "rest-json", ops: [4]string{"CreateMesh", "", "ListMeshes", "DeleteMesh"}},
	{Name: "cloud9", Protocol: "rest-json", ops: [4]string{"CreateEnvironment", "", "ListEnvironments", "DeleteEnvironment"}},
	{Name: "codestar-connections", Protocol: "rest-json", ops: [4]string{"CreateConnection", "", "ListConnections", "DeleteConnection"}},
	{Name: "datasync", Protocol: "rest-json", ops: [4]string{"CreateTask", "", "ListTasks", "DeleteTask"}},
	{Name: "devicefarm", Protocol: "rest-json", ops: [4]string{"CreateProject", "", "ListProjects", "DeleteProject"}},
	{Name: "events", Protocol: "rest-json", ops: [4]string{"CreateEventBus", "", "ListEventBuses", "DeleteEventBus"}},
	{Name: "finspace", Protocol: "rest-json", ops: [4]string{"CreateEnvironment", "", "ListEnvironments", "DeleteEnvironment"}},
	{Name: "forecast", Protocol: "rest-json", ops: [4]string{"CreateDatasetGroup", "", "ListDatasetGroups", "DeleteDatasetGroup"}},
	{Name: "groundstation", Protocol: "rest-json", ops: [4]string{"CreateConfig", "", "ListConfigs", "DeleteConfig"}},
	{Name: "healthlake", Protocol: "rest-json", ops: [4]string{"CreateFHIRDatastore", "", "ListFHIRDatastores", "DeleteFHIRDatastore"}},
	{Name: "inspector2", Protocol: "rest-json", ops: [4]string{"CreateFilter", "", "ListFilters", "DeleteFilter"}},
	{Name: "cassandra", Protocol: "rest-json", ops: [4]string{"CreateKeyspace", "", "ListKeyspaces", "DeleteKeyspace"}},
	{Name: "location", Protocol: "rest-json", ops: [4]string{"CreateMap", "", "ListMaps", "DeleteMap"}},
	{Name: "lookoutmetrics", Protocol: "rest-json", ops: [4]string{"CreateAnomalyDetector", "", "ListAnomalyDetectors", "DeleteAnomalyDetector"}},
	{Name: "macie2", Protocol: "rest-json", ops: [4]string{"CreateFindingsFilter", "", "ListFindingsFilters", "DeleteFindingsFilter"}},
	{Name: "mgh", Protocol: "rest-json", ops: [4]string{"CreateProgressUpdateStream", "", "ListProgressUpdateStreams", "DeleteProgressUpdateStream"}},
	{Name: "nimble", Protocol: "rest-json", ops: [4]string{"CreateStudio", "", "ListStudios", "DeleteStudio"}},
	{Name: "outposts", Protocol: "rest-json", ops: [4]string{"CreateOutpost", "", "ListOutposts", "DeleteOutpost"}},
	{Name: "panorama", Protocol: "rest-json", ops: [4]string{"CreateDevice", "", "ListDevices", "DeleteDevice"}},
	{Name: "private-networks", Protocol: "rest-json", ops: [4]string{"CreateNetwork", "", "ListNetworks", "DeleteNetwork"}},
	{Name: "proton", Protocol: "rest-json", ops: [4]string{"CreateEnvironmentTemplate", "", "ListEnvironmentTemplates", "DeleteEnvironmentTemplate"}},
	{Name: "rekognition", Protocol: "rest-json", ops: [4]string{"CreateCollection", "", "ListCollections", "DeleteCollection"}},
	{Name: "robomaker", Protocol: "rest-json", ops: [4]string{"CreateSimulationApplication", "", "ListSimulationApplications", "DeleteSimulationApplication"}},
	{Name: "securityhub", Protocol: "rest-json", ops: [4]string{"CreateHub", "", "ListHubs", "DeleteHub"}},
	// ── REST-XML protocol ────────────────────────────────────────────────────────
	{Name: "cloudfront", Protocol: "rest-xml", ops: [4]string{"CreateDistribution", "GetDistribution", "ListDistributions", "DeleteDistribution"}},
}

// opLabels maps the four op-slot indices to human-readable names.
var opLabels = [4]string{"Create", "Describe", "List", "Delete"}

// GenerateAll returns one harness.Suite per tier-2 service.
func GenerateAll() []harness.Suite {
	suites := make([]harness.Suite, 0, len(services))
	for _, svc := range services {
		s := svc // capture
		suites = append(suites, &specSuite{spec: s})
	}
	return suites
}

type specSuite struct {
	spec serviceSpec
}

func (s *specSuite) Name() string { return s.spec.Name }
func (s *specSuite) Tier() int    { return 2 }

func (s *specSuite) Operations() []harness.Operation {
	var ops []harness.Operation
	for i, actionName := range s.spec.ops {
		if actionName == "" {
			continue
		}
		label := opLabels[i]
		name := actionName
		ops = append(ops, harness.Operation{
			Name: name,
			Run:  s.makeRunner(name),
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, label+"Response")}
			},
		})
	}
	return ops
}

func (s *specSuite) makeRunner(actionName string) func(ctx context.Context, endpoint string) (any, error) {
	return func(ctx context.Context, endpoint string) (any, error) {
		body := s.buildBody(actionName)

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}

		switch s.spec.Protocol {
		case "json":
			req.Header.Set("Content-Type", "application/x-amz-json-1.1")
			req.Header.Set("X-Amz-Target", s.spec.Target+"."+actionName)
		case "query", "rest-xml":
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		case "rest-json":
			req.Header.Set("Content-Type", "application/json")
		}

		req.Header.Set("Authorization",
			fmt.Sprintf("AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/%s/aws4_request, SignedHeaders=host, Signature=fake",
				s.spec.Name))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]any
		json.Unmarshal(respBody, &result) //nolint:errcheck

		if resp.StatusCode >= 400 {
			return result, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return result, nil
	}
}

// buildBody constructs a minimal request body for the given action.
func (s *specSuite) buildBody(actionName string) []byte {
	switch s.spec.Protocol {
	case "query":
		parts := []string{
			"Action=" + actionName,
			"Version=2012-11-05",
		}
		return []byte(strings.Join(parts, "&"))
	case "json", "rest-json":
		// Send an empty JSON object; required params will cause a service-level
		// error which still exercises the routing and parsing stack.
		return []byte("{}")
	default:
		return []byte{}
	}
}

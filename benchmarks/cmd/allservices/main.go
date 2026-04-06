// allservices benchmarks every registered CloudMock service by sending a
// simple "list" request to each one and measuring throughput + latency.
//
// Usage:
//   go run ./benchmarks/cmd/allservices --endpoint http://localhost:4577 --requests 1000 --concurrency 20
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// serviceSpec describes how to send a single request to a service.
type serviceSpec struct {
	Name        string
	ContentType string
	Target      string // X-Amz-Target header (JSON protocol services)
	Body        string
	Method      string
	Path        string // for REST services (e.g. S3)
	Query       string // for Query protocol services (e.g. EC2, SQS)
}

// allServices returns a representative request for every CloudMock service.
func allServices() []serviceSpec {
	// JSON 1.0 protocol services (X-Amz-Target with application/x-amz-json-1.0)
	json10 := []serviceSpec{
		{Name: "dynamodb", Target: "DynamoDB_20120810.ListTables", Body: `{}`},
		{Name: "kinesis", Target: "Kinesis_20131202.ListStreams", Body: `{}`},
		{Name: "kms", Target: "TrentService.ListKeys", Body: `{}`},
		{Name: "swf", Target: "SimpleWorkflowService.ListDomains", Body: `{"registrationStatus":"REGISTERED"}`},
		{Name: "dax", Target: "AmazonDAXV3.DescribeClusters", Body: `{}`},
	}
	for i := range json10 {
		json10[i].ContentType = "application/x-amz-json-1.0"
		json10[i].Method = "POST"
		json10[i].Path = "/"
	}

	// JSON 1.1 protocol services
	json11 := []serviceSpec{
		{Name: "ecs", Target: "AmazonEC2ContainerServiceV20141113.ListClusters", Body: `{}`},
		{Name: "cloudwatchlogs", Target: "Logs_20140328.DescribeLogGroups", Body: `{}`},
		{Name: "events", Target: "AWSEvents.ListEventBuses", Body: `{}`},
		{Name: "stepfunctions", Target: "AWSStepFunctions.ListStateMachines", Body: `{}`},
		{Name: "firehose", Target: "Firehose_20150804.ListDeliveryStreams", Body: `{}`},
		{Name: "codepipeline", Target: "CodePipeline_20150709.ListPipelines", Body: `{}`},
		{Name: "codebuild", Target: "CodeBuild_20161006.ListProjects", Body: `{}`},
		{Name: "codecommit", Target: "CodeCommit_20150413.ListRepositories", Body: `{}`},
		{Name: "codeartifact", Target: "CodeArtifact_20180901.ListDomains", Body: `{}`},
		{Name: "ssm", Target: "AmazonSSM.DescribeParameters", Body: `{}`},
		{Name: "secretsmanager", Target: "secretsmanager.ListSecrets", Body: `{}`},
		{Name: "glue", Target: "AWSGlue.GetDatabases", Body: `{}`},
		{Name: "athena", Target: "AmazonAthena.ListWorkGroups", Body: `{}`},
		{Name: "lambda", Target: "AWSLambda.ListFunctions20150331", Body: `{}`},
		{Name: "cognito", Target: "AWSCognitoIdentityProviderService.ListUserPools", Body: `{"MaxResults":10}`},
		{Name: "eks", Target: "AmazonEKS.ListClusters", Body: `{}`},
		{Name: "sagemaker", Target: "SageMaker.ListNotebookInstances", Body: `{}`},
		{Name: "bedrock", Target: "AmazonBedrock.ListFoundationModels", Body: `{}`},
		{Name: "organizations", Target: "AWSOrganizationsV20161128.ListAccounts", Body: `{}`},
		{Name: "cloudtrail", Target: "CloudTrail_20131101.DescribeTrails", Body: `{}`},
		{Name: "config", Target: "StarlingDoveService.DescribeConfigRules", Body: `{}`},
		{Name: "acm", Target: "CertificateManager.ListCertificates", Body: `{}`},
		{Name: "backup", Target: "CryoControllerUserManager.ListBackupPlans", Body: `{}`},
		{Name: "batch", Target: "AWSBatch.DescribeComputeEnvironments", Body: `{}`},
		{Name: "comprehend", Target: "Comprehend_20171127.ListEndpoints", Body: `{}`},
		{Name: "applicationautoscaling", Target: "AnyScaleFrontendService.DescribeScalableTargets", Body: `{"ServiceNamespace":"ecs"}`},
		{Name: "mediaconvert", Target: "MediaConvert.DescribeEndpoints", Body: `{}`},
		{Name: "kafka", Target: "Kafka.ListClustersV2", Body: `{}`},
		{Name: "wafv2", Target: "AWSWAF_20190729.ListWebACLs", Body: `{"Scope":"REGIONAL"}`},
		{Name: "shield", Target: "AWSShield_20160616.ListProtections", Body: `{}`},
		{Name: "guardduty", Target: "GuardDutyAPIService.ListDetectors", Body: `{}`},
		{Name: "securityhub", Target: "SecurityHubAPIService.GetFindings", Body: `{}`},
		{Name: "pinpoint", Target: "Pinpoint.GetApps", Body: `{}`},
		{Name: "rekognition", Target: "RekognitionService.ListCollections", Body: `{}`},
		{Name: "textract", Target: "Textract.DetectDocumentText", Body: `{}`},
		{Name: "transcribe", Target: "Transcribe.ListTranscriptionJobs", Body: `{}`},
		{Name: "translate", Target: "AWSShineFrontendService_20170701.ListTerminologies", Body: `{}`},
		{Name: "polly", Target: "Parrot_v1.DescribeVoices", Body: `{}`},
		{Name: "fis", Target: "FaultInjectionSimulator.ListExperimentTemplates", Body: `{}`},
		{Name: "pipes", Target: "Pipes.ListPipes", Body: `{}`},
		{Name: "scheduler", Target: "AWSChronos.ListSchedules", Body: `{}`},
		{Name: "identitystore", Target: "AWSIdentityStore.ListUsers", Body: `{"IdentityStoreId":"d-1234567890"}`},
		{Name: "lakeformation", Target: "AWSLakeFormation.ListResources", Body: `{}`},
		{Name: "ram", Target: "AmazonResourceSharing.GetResourceShares", Body: `{"resourceOwner":"SELF"}`},
		{Name: "quicksight", Target: "QuickSight_20180401.ListDashboards", Body: `{"AwsAccountId":"000000000000"}`},
		{Name: "servicecatalog", Target: "AWS242ServiceCatalogService.ListPortfolios", Body: `{}`},
		{Name: "support", Target: "AWSSupport_20130415.DescribeServices", Body: `{}`},
		{Name: "xray", Target: "AWSXRay.GetTraceSummaries", Body: `{"StartTime":0,"EndTime":9999999999}`},
		{Name: "inspector2", Target: "Inspector2.ListFindings", Body: `{}`},
		{Name: "cloudwatch", Target: "GraniteServiceVersion20100801.ListMetrics", Body: `{}`},
		{Name: "timestreamwrite", Target: "Timestream_20181101.ListDatabases", Body: `{}`},
		{Name: "iot", Target: "AWSIotService.ListThings", Body: `{}`},
		{Name: "appconfig", Target: "AmazonAppConfig.ListApplications", Body: `{}`},
		{Name: "appsync", Target: "AWSDeepdishControlPlaneService.ListGraphqlApis", Body: `{}`},
	}
	for i := range json11 {
		json11[i].ContentType = "application/x-amz-json-1.1"
		json11[i].Method = "POST"
		json11[i].Path = "/"
	}

	// Query protocol services (?Action=...)
	query := []serviceSpec{
		{Name: "ec2", Query: "Action=DescribeInstances&Version=2016-11-15", Body: ""},
		{Name: "sqs", Query: "Action=ListQueues", Body: ""},
		{Name: "sns", Query: "Action=ListTopics", Body: ""},
		{Name: "iam", Query: "Action=ListUsers&Version=2010-05-08", Body: ""},
		{Name: "sts", Query: "Action=GetCallerIdentity&Version=2011-06-15", Body: ""},
		{Name: "autoscaling", Query: "Action=DescribeAutoScalingGroups&Version=2011-01-01", Body: ""},
		{Name: "elasticloadbalancing", Query: "Action=DescribeLoadBalancers&Version=2012-06-01", Body: ""},
		{Name: "cloudformation", Query: "Action=ListStacks", Body: ""},
		{Name: "ses", Query: "Action=ListIdentities", Body: ""},
		{Name: "rds", Query: "Action=DescribeDBInstances&Version=2014-10-31", Body: ""},
		{Name: "redshift", Query: "Action=DescribeClusters", Body: ""},
		{Name: "elasticache", Query: "Action=DescribeCacheClusters&Version=2015-02-02", Body: ""},
		{Name: "cloudfront", Query: "Action=ListDistributions2020_05_31", Body: ""},
		{Name: "route53", Query: "Action=ListHostedZones", Body: ""},
		{Name: "elasticbeanstalk", Query: "Action=DescribeApplications", Body: ""},
		{Name: "elasticmapreduce", Query: "Action=ListClusters", Body: ""},
	}
	for i := range query {
		query[i].ContentType = "application/x-www-form-urlencoded"
		query[i].Method = "POST"
		query[i].Path = "/"
	}

	// REST services (S3, API Gateway, etc.)
	rest := []serviceSpec{
		{Name: "s3", Method: "GET", Path: "/", ContentType: ""},
		{Name: "apigateway", Method: "GET", Path: "/restapis", ContentType: "application/json"},
		{Name: "ecr", Method: "POST", Path: "/", ContentType: "application/x-amz-json-1.1", Target: "AmazonEC2ContainerRegistry_V20150921.DescribeRepositories", Body: `{}`},
		{Name: "glacier", Method: "GET", Path: "/-/vaults", ContentType: "application/json"},
	}

	all := make([]serviceSpec, 0, len(json10)+len(json11)+len(query)+len(rest))
	all = append(all, json10...)
	all = append(all, json11...)
	all = append(all, query...)
	all = append(all, rest...)
	return all
}

type result struct {
	Name    string
	ReqSec  float64
	AvgMs   float64
	P99Ms   float64
	Errors  int64
	Total   int64
}

func bench(endpoint string, spec serviceSpec, n, concurrency int) result {
	url := endpoint + spec.Path
	if spec.Query != "" {
		url += "?" + spec.Query
	}

	auth := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=test/20260406/us-east-1/%s/aws4_request, SignedHeaders=host, Signature=fake", spec.Name)

	var totalErrors atomic.Int64
	var totalRequests atomic.Int64
	latencies := make([]time.Duration, n)
	var mu sync.Mutex
	latIdx := 0

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: concurrency,
			MaxConnsPerHost:     concurrency,
		},
	}

	start := time.Now()

	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			var body io.Reader
			if spec.Body != "" {
				body = bytes.NewBufferString(spec.Body)
			}

			req, _ := http.NewRequest(spec.Method, url, body)
			req.Header.Set("Authorization", auth)
			if spec.ContentType != "" {
				req.Header.Set("Content-Type", spec.ContentType)
			}
			if spec.Target != "" {
				req.Header.Set("X-Amz-Target", spec.Target)
			}

			t0 := time.Now()
			resp, err := client.Do(req)
			d := time.Since(t0)

			totalRequests.Add(1)
			if err != nil || (resp != nil && resp.StatusCode >= 500) {
				totalErrors.Add(1)
			}
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}

			mu.Lock()
			if latIdx < n {
				latencies[latIdx] = d
				latIdx++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	mu.Lock()
	lats := latencies[:latIdx]
	mu.Unlock()

	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })

	var avg time.Duration
	for _, l := range lats {
		avg += l
	}
	if len(lats) > 0 {
		avg /= time.Duration(len(lats))
	}

	var p99 time.Duration
	if len(lats) > 0 {
		p99 = lats[int(float64(len(lats))*0.99)]
	}

	return result{
		Name:   spec.Name,
		ReqSec: float64(totalRequests.Load()) / elapsed.Seconds(),
		AvgMs:  float64(avg.Microseconds()) / 1000.0,
		P99Ms:  float64(p99.Microseconds()) / 1000.0,
		Errors: totalErrors.Load(),
		Total:  totalRequests.Load(),
	}
}

func main() {
	endpoint := flag.String("endpoint", "http://localhost:4566", "CloudMock endpoint")
	requests := flag.Int("requests", 1000, "requests per service")
	concurrency := flag.Int("concurrency", 20, "concurrent requests")
	filter := flag.String("filter", "", "comma-separated service names to test (empty = all)")
	flag.Parse()

	services := allServices()

	var filterSet map[string]bool
	if *filter != "" {
		filterSet = make(map[string]bool)
		for _, s := range strings.Split(*filter, ",") {
			filterSet[strings.TrimSpace(s)] = true
		}
	}

	fmt.Printf("Benchmarking %s — %d reqs/service, %d concurrent\n\n", *endpoint, *requests, *concurrency)
	fmt.Printf("%-28s %10s %8s %8s %6s\n", "Service", "req/s", "avg(ms)", "p99(ms)", "errs")
	fmt.Println(strings.Repeat("-", 66))

	var results []result
	for _, svc := range services {
		if filterSet != nil && !filterSet[svc.Name] {
			continue
		}
		r := bench(*endpoint, svc, *requests, *concurrency)
		results = append(results, r)
		errStr := fmt.Sprintf("%d", r.Errors)
		if r.Errors > 0 {
			errStr = fmt.Sprintf("!%d", r.Errors)
		}
		fmt.Printf("%-28s %10.0f %8.2f %8.2f %6s\n", r.Name, r.ReqSec, r.AvgMs, r.P99Ms, errStr)
	}

	// Summary
	var totalRPS float64
	var totalAvg float64
	for _, r := range results {
		totalRPS += r.ReqSec
		totalAvg += r.AvgMs
	}
	fmt.Println(strings.Repeat("-", 66))
	fmt.Printf("%-28s %10.0f %8.2f\n", "TOTAL (sum / avg)", totalRPS, totalAvg/float64(len(results)))
}

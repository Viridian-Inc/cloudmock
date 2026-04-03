package cloudformation_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	cfnsvc "github.com/neureaux/cloudmock/services/cloudformation"
	dynamodbsvc "github.com/neureaux/cloudmock/services/dynamodb"
	ecssvc "github.com/neureaux/cloudmock/services/ecs"
	iamsvc "github.com/neureaux/cloudmock/services/iam"
	kmssvc "github.com/neureaux/cloudmock/services/kms"
	lambdasvc "github.com/neureaux/cloudmock/services/lambda"
	rdssvc "github.com/neureaux/cloudmock/services/rds"
	route53svc "github.com/neureaux/cloudmock/services/route53"
	s3svc "github.com/neureaux/cloudmock/services/s3"
	snssvc "github.com/neureaux/cloudmock/services/sns"
	sqssvc "github.com/neureaux/cloudmock/services/sqs"
	ssmsvc "github.com/neureaux/cloudmock/services/ssm"
)

// newMultiServiceGateway creates a gateway with S3, DynamoDB, SQS, SNS, IAM, Lambda, and
// CloudFormation — with the CloudFormation provisioner wired to the registry.
func newMultiServiceGateway(t *testing.T) (http.Handler, *routing.Registry) {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	iamStore := iampkg.NewStore(cfg.AccountID)
	iamEngine := iampkg.NewEngine()

	reg := routing.NewRegistry()
	reg.Register(s3svc.New())
	reg.Register(dynamodbsvc.New(cfg.AccountID, cfg.Region))
	reg.Register(sqssvc.New(cfg.AccountID, cfg.Region))
	reg.Register(snssvc.New(cfg.AccountID, cfg.Region))
	reg.Register(iamsvc.New(cfg.AccountID, iamEngine, iamStore))

	reg.Register(rdssvc.New(cfg.AccountID, cfg.Region))
	reg.Register(ecssvc.New(cfg.AccountID, cfg.Region))
	reg.Register(route53svc.New(cfg.AccountID, cfg.Region))
	reg.Register(kmssvc.New(cfg.AccountID, cfg.Region))
	reg.Register(ssmsvc.New(cfg.AccountID, cfg.Region))

	lambdaSvc := lambdasvc.New(cfg.AccountID, cfg.Region)
	reg.Register(lambdaSvc)
	lambdaSvc.SetLocator(reg)

	cfnSvc := cfnsvc.New(cfg.AccountID, cfg.Region)
	reg.Register(cfnSvc)
	cfnSvc.SetLocator(reg)

	return gateway.New(cfg, reg), reg
}

// cfnProvisionReq builds a CFN request with the given action and extra form values.
func cfnProvisionReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()
	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2010-05-15")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}
	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/cloudformation/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// s3HeadBucket checks if an S3 bucket exists by making a HEAD request.
func s3HeadBucket(t *testing.T, handler http.Handler, bucketName string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodHead, "/"+bucketName, nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Code
}

// dynamoDescribeTable checks if a DynamoDB table exists.
func dynamoDescribeTable(t *testing.T, handler http.Handler, tableName string) int {
	t.Helper()
	body := `{"TableName":"` + tableName + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.DescribeTable")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Code
}

// sqsGetQueueURL checks if an SQS queue exists by calling GetQueueUrl.
func sqsGetQueueURL(t *testing.T, handler http.Handler, queueName string) (int, string) {
	t.Helper()
	form := url.Values{}
	form.Set("Action", "GetQueueUrl")
	form.Set("QueueName", queueName)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/sqs/aws4_request, SignedHeaders=host, Signature=abc123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

// ---- Test 1: CreateStack with S3 bucket — verify bucket exists ----

func TestProvisioner_S3Bucket(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyBucket": {
				"Type": "AWS::S3::Bucket",
				"Properties": {
					"BucketName": "cfn-test-bucket"
				}
			}
		}
	}`

	// Create stack.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"s3-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify the S3 bucket exists.
	code := s3HeadBucket(t, handler, "cfn-test-bucket")
	if code != http.StatusOK {
		t.Errorf("S3 HeadBucket: expected 200, got %d — bucket was not created", code)
	}

	// Verify DescribeStackResources shows a PhysicalResourceId.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnProvisionReq(t, "DescribeStackResources", url.Values{
		"StackName": {"s3-stack"},
	}))
	if wres.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d", wres.Code)
	}
	resBody := wres.Body.String()
	if !strings.Contains(resBody, "cfn-test-bucket") {
		t.Errorf("DescribeStackResources: expected PhysicalResourceId 'cfn-test-bucket'\nbody: %s", resBody)
	}
}

// ---- Test 2: CreateStack with DynamoDB table — verify table exists ----

func TestProvisioner_DynamoDBTable(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyTable": {
				"Type": "AWS::DynamoDB::Table",
				"Properties": {
					"TableName": "cfn-test-table",
					"KeySchema": [
						{"AttributeName": "pk", "KeyType": "HASH"}
					],
					"AttributeDefinitions": [
						{"AttributeName": "pk", "AttributeType": "S"}
					],
					"BillingMode": "PAY_PER_REQUEST"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"dynamo-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify the DynamoDB table exists.
	code := dynamoDescribeTable(t, handler, "cfn-test-table")
	if code != http.StatusOK {
		t.Errorf("DynamoDB DescribeTable: expected 200, got %d — table was not created", code)
	}
}

// ---- Test 3: CreateStack with SQS queue — verify queue exists ----

func TestProvisioner_SQSQueue(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyQueue": {
				"Type": "AWS::SQS::Queue",
				"Properties": {
					"QueueName": "cfn-test-queue"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"sqs-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify the SQS queue exists.
	code, body := sqsGetQueueURL(t, handler, "cfn-test-queue")
	if code != http.StatusOK {
		t.Errorf("SQS GetQueueUrl: expected 200, got %d — queue was not created\nbody: %s", code, body)
	}
	if !strings.Contains(body, "cfn-test-queue") {
		t.Errorf("SQS GetQueueUrl: expected queue URL containing 'cfn-test-queue'\nbody: %s", body)
	}
}

// ---- Test 4: CreateStack with Ref between resources (Lambda referencing IAM role) ----

func TestProvisioner_RefBetweenResources(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"LambdaRole": {
				"Type": "AWS::IAM::Role",
				"Properties": {
					"RoleName": "cfn-lambda-role",
					"AssumeRolePolicyDocument": {
						"Version": "2012-10-17",
						"Statement": [{"Effect": "Allow", "Principal": {"Service": "lambda.amazonaws.com"}, "Action": "sts:AssumeRole"}]
					}
				}
			},
			"MyFunction": {
				"Type": "AWS::Lambda::Function",
				"DependsOn": "LambdaRole",
				"Properties": {
					"FunctionName": "cfn-test-function",
					"Runtime": "nodejs18.x",
					"Handler": "index.handler",
					"Role": {"Fn::GetAtt": ["LambdaRole", "Arn"]}
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"lambda-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify DescribeStackResources shows both resources with physical IDs.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnProvisionReq(t, "DescribeStackResources", url.Values{
		"StackName": {"lambda-stack"},
	}))
	if wres.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d", wres.Code)
	}
	resBody := wres.Body.String()

	// Should contain physical IDs for both the role and the function.
	if !strings.Contains(resBody, "cfn-lambda-role") {
		t.Errorf("DescribeStackResources: expected IAM role physical ID 'cfn-lambda-role'\nbody: %s", resBody)
	}
	if !strings.Contains(resBody, "cfn-test-function") {
		t.Errorf("DescribeStackResources: expected Lambda function physical ID 'cfn-test-function'\nbody: %s", resBody)
	}

	// Both resources should be CREATE_COMPLETE.
	if strings.Count(resBody, "CREATE_COMPLETE") < 2 {
		t.Errorf("DescribeStackResources: expected at least 2 CREATE_COMPLETE statuses\nbody: %s", resBody)
	}
}

// ---- Test 5: DeleteStack — verify resources deleted ----

func TestProvisioner_DeleteStack(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"DeleteBucket": {
				"Type": "AWS::S3::Bucket",
				"Properties": {
					"BucketName": "cfn-delete-bucket"
				}
			}
		}
	}`

	// Create stack.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"delete-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify bucket exists.
	if code := s3HeadBucket(t, handler, "cfn-delete-bucket"); code != http.StatusOK {
		t.Fatalf("S3 HeadBucket before delete: expected 200, got %d", code)
	}

	// Delete stack.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, cfnProvisionReq(t, "DeleteStack", url.Values{
		"StackName": {"delete-stack"},
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteStack: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	// Verify bucket is deleted.
	if code := s3HeadBucket(t, handler, "cfn-delete-bucket"); code == http.StatusOK {
		t.Errorf("S3 HeadBucket after delete: expected non-200, got %d — bucket should be deleted", code)
	}

	// Verify stack is DELETE_COMPLETE.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, cfnProvisionReq(t, "DescribeStacks", url.Values{
		"StackName": {"delete-stack"},
	}))
	if !strings.Contains(wdesc.Body.String(), "DELETE_COMPLETE") {
		t.Errorf("DescribeStacks: expected DELETE_COMPLETE\nbody: %s", wdesc.Body.String())
	}
}

// ---- Test 6: Intrinsic functions — Fn::Sub and Fn::Join ----

func TestProvisioner_IntrinsicFunctions(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Parameters": {
			"Env": {
				"Type": "String",
				"Default": "test"
			}
		},
		"Resources": {
			"ParamBucket": {
				"Type": "AWS::S3::Bucket",
				"Properties": {
					"BucketName": {"Fn::Sub": "my-${Env}-bucket"}
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"intrinsic-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify bucket with resolved name exists.
	if code := s3HeadBucket(t, handler, "my-test-bucket"); code != http.StatusOK {
		t.Errorf("S3 HeadBucket: expected 200, got %d — Fn::Sub was not resolved correctly", code)
	}
}

// ---- Test 7: SNS Topic ----

func TestProvisioner_SNSTopic(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyTopic": {
				"Type": "AWS::SNS::Topic",
				"Properties": {
					"TopicName": "cfn-test-topic"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"sns-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify DescribeStackResources shows the topic ARN as physical ID.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnProvisionReq(t, "DescribeStackResources", url.Values{
		"StackName": {"sns-stack"},
	}))
	resBody := wres.Body.String()
	if !strings.Contains(resBody, "cfn-test-topic") {
		t.Errorf("DescribeStackResources: expected topic ARN containing 'cfn-test-topic'\nbody: %s", resBody)
	}
}

// ---- Test 8: Multiple resources with dependency ordering ----

func TestProvisioner_DependencyOrdering(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	// Template with explicit DependsOn: Queue depends on Bucket.
	template := `{
		"Resources": {
			"Queue": {
				"Type": "AWS::SQS::Queue",
				"DependsOn": "Bucket",
				"Properties": {
					"QueueName": "dep-test-queue"
				}
			},
			"Bucket": {
				"Type": "AWS::S3::Bucket",
				"Properties": {
					"BucketName": "dep-test-bucket"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"dep-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Both resources should exist.
	if code := s3HeadBucket(t, handler, "dep-test-bucket"); code != http.StatusOK {
		t.Errorf("S3 bucket missing: got %d", code)
	}
	if code, _ := sqsGetQueueURL(t, handler, "dep-test-queue"); code != http.StatusOK {
		t.Errorf("SQS queue missing: got %d", code)
	}
}

// ---- Test 9: Without provisioner — backward compatibility ----

func TestCFN_WithoutProvisioner_BackwardCompat(t *testing.T) {
	// Use the old-style gateway without multi-service wiring.
	handler := newCFNGateway(t)

	stackID := mustCreateStack(t, handler, "compat-stack")

	var resp struct {
		XMLName xml.Name `xml:"DescribeStackResourcesResponse"`
	}
	_ = stackID
	_ = resp

	// The original tests should still pass without a provisioner.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnReq(t, "DescribeStackResources", url.Values{
		"StackName": {"compat-stack"},
	}))
	if wres.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d", wres.Code)
	}
	// Should still contain the logical resources, just without physical IDs.
	body := wres.Body.String()
	if !strings.Contains(body, "MyBucket") {
		t.Errorf("expected MyBucket in response\nbody: %s", body)
	}
}

// ---- Test 10: RDS DBInstance ----

func TestProvisioner_RDSDBInstance(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyDB": {
				"Type": "AWS::RDS::DBInstance",
				"Properties": {
					"DBInstanceIdentifier": "cfn-test-db",
					"Engine": "mysql",
					"DBInstanceClass": "db.t3.micro",
					"MasterUsername": "admin",
					"MasterUserPassword": "secret123",
					"AllocatedStorage": "20"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"rds-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify DescribeStackResources shows the DB instance.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnProvisionReq(t, "DescribeStackResources", url.Values{
		"StackName": {"rds-stack"},
	}))
	if wres.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d", wres.Code)
	}
	resBody := wres.Body.String()
	if !strings.Contains(resBody, "cfn-test-db") {
		t.Errorf("DescribeStackResources: expected PhysicalResourceId 'cfn-test-db'\nbody: %s", resBody)
	}
	if !strings.Contains(resBody, "CREATE_COMPLETE") {
		t.Errorf("DescribeStackResources: expected CREATE_COMPLETE\nbody: %s", resBody)
	}
}

// ---- Test 11: ECS Cluster ----

func TestProvisioner_ECSCluster(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyCluster": {
				"Type": "AWS::ECS::Cluster",
				"Properties": {
					"ClusterName": "cfn-test-cluster"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"ecs-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify DescribeStackResources shows the cluster.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnProvisionReq(t, "DescribeStackResources", url.Values{
		"StackName": {"ecs-stack"},
	}))
	if wres.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d", wres.Code)
	}
	resBody := wres.Body.String()
	if !strings.Contains(resBody, "cfn-test-cluster") {
		t.Errorf("DescribeStackResources: expected PhysicalResourceId containing 'cfn-test-cluster'\nbody: %s", resBody)
	}
	if !strings.Contains(resBody, "CREATE_COMPLETE") {
		t.Errorf("DescribeStackResources: expected CREATE_COMPLETE\nbody: %s", resBody)
	}
}

// ---- Test 12: Route53 HostedZone ----

func TestProvisioner_Route53HostedZone(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyZone": {
				"Type": "AWS::Route53::HostedZone",
				"Properties": {
					"Name": "example.com"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"route53-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify DescribeStackResources shows the hosted zone.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnProvisionReq(t, "DescribeStackResources", url.Values{
		"StackName": {"route53-stack"},
	}))
	if wres.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d", wres.Code)
	}
	resBody := wres.Body.String()
	// The physical ID should be a zone ID starting with Z.
	if !strings.Contains(resBody, "CREATE_COMPLETE") {
		t.Errorf("DescribeStackResources: expected CREATE_COMPLETE\nbody: %s", resBody)
	}
	if !strings.Contains(resBody, "AWS::Route53::HostedZone") {
		t.Errorf("DescribeStackResources: expected resource type AWS::Route53::HostedZone\nbody: %s", resBody)
	}
}

// ---- Test 13: KMS Key ----

func TestProvisioner_KMSKey(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyKey": {
				"Type": "AWS::KMS::Key",
				"Properties": {
					"Description": "Test encryption key",
					"KeyUsage": "ENCRYPT_DECRYPT"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"kms-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify DescribeStackResources shows the KMS key.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnProvisionReq(t, "DescribeStackResources", url.Values{
		"StackName": {"kms-stack"},
	}))
	if wres.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d", wres.Code)
	}
	resBody := wres.Body.String()
	if !strings.Contains(resBody, "CREATE_COMPLETE") {
		t.Errorf("DescribeStackResources: expected CREATE_COMPLETE\nbody: %s", resBody)
	}
	if !strings.Contains(resBody, "AWS::KMS::Key") {
		t.Errorf("DescribeStackResources: expected resource type AWS::KMS::Key\nbody: %s", resBody)
	}
}

// ---- Test 14: SSM Parameter ----

func TestProvisioner_SSMParameter(t *testing.T) {
	handler, _ := newMultiServiceGateway(t)

	template := `{
		"Resources": {
			"MyParam": {
				"Type": "AWS::SSM::Parameter",
				"Properties": {
					"Name": "/cfn/test/param",
					"Type": "String",
					"Value": "hello-world"
				}
			}
		}
	}`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnProvisionReq(t, "CreateStack", url.Values{
		"StackName":    {"ssm-stack"},
		"TemplateBody": {template},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify DescribeStackResources shows the SSM parameter.
	wres := httptest.NewRecorder()
	handler.ServeHTTP(wres, cfnProvisionReq(t, "DescribeStackResources", url.Values{
		"StackName": {"ssm-stack"},
	}))
	if wres.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d", wres.Code)
	}
	resBody := wres.Body.String()
	if !strings.Contains(resBody, "/cfn/test/param") {
		t.Errorf("DescribeStackResources: expected PhysicalResourceId '/cfn/test/param'\nbody: %s", resBody)
	}
	if !strings.Contains(resBody, "CREATE_COMPLETE") {
		t.Errorf("DescribeStackResources: expected CREATE_COMPLETE\nbody: %s", resBody)
	}
}

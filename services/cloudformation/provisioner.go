package cloudformation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceLocator provides access to other cloudmock services for resource provisioning.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// ProvisionedResource holds the result of provisioning a single CFN resource.
type ProvisionedResource struct {
	LogicalId  string
	PhysicalId string            // e.g., bucket name, table name, VPC ID
	Type       string
	Attributes map[string]string // for Fn::GetAtt
	Status     string
}

// Provisioner creates real resources in cloudmock services when a stack is created.
type Provisioner struct {
	locator   ServiceLocator
	accountID string
	region    string
}

// NewProvisioner returns a new Provisioner.
func NewProvisioner(locator ServiceLocator, accountID, region string) *Provisioner {
	return &Provisioner{
		locator:   locator,
		accountID: accountID,
		region:    region,
	}
}

// ---- full template parsing types (with Properties and DependsOn) ----

type cfnFullTemplate struct {
	Description string                         `json:"Description"`
	Parameters  map[string]cfnParameter        `json:"Parameters"`
	Resources   map[string]cfnFullResource     `json:"Resources"`
	Outputs     map[string]cfnOutput           `json:"Outputs"`
}

type cfnFullResource struct {
	Type       string                 `json:"Type"`
	DependsOn  json.RawMessage        `json:"DependsOn"`
	Properties map[string]any `json:"Properties"`
}

// ProvisionStack parses a CFN template and creates real resources in dependency order.
// Returns the list of provisioned resources or an error.
func (p *Provisioner) ProvisionStack(templateBody string, params map[string]string) ([]ProvisionedResource, error) {
	var tmpl cfnFullTemplate
	if err := json.Unmarshal([]byte(templateBody), &tmpl); err != nil {
		// Not valid JSON — nothing to provision.
		return nil, nil
	}

	if len(tmpl.Resources) == 0 {
		return nil, nil
	}

	// Resolve parameters: merge provided values with defaults.
	resolvedParams := make(map[string]string)
	for key, defn := range tmpl.Parameters {
		if val, ok := params[key]; ok {
			resolvedParams[key] = val
		} else if defn.Default != nil {
			var defVal string
			if err := json.Unmarshal(defn.Default, &defVal); err != nil {
				defVal = string(defn.Default)
			}
			resolvedParams[key] = defVal
		}
	}
	// Also include pseudo-parameters.
	resolvedParams["AWS::AccountId"] = p.accountID
	resolvedParams["AWS::Region"] = p.region
	resolvedParams["AWS::StackName"] = "" // will be set by caller if needed
	resolvedParams["AWS::NoValue"] = ""

	// Build dependency graph and topological sort.
	order, err := topoSort(tmpl.Resources)
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Provision resources in order.
	provisioned := make(map[string]*ProvisionedResource)
	var result []ProvisionedResource

	for _, logicalId := range order {
		res := tmpl.Resources[logicalId]

		// Resolve intrinsic functions in properties.
		resolvedProps := resolveIntrinsics(res.Properties, resolvedParams, provisioned)
		propsMap, _ := resolvedProps.(map[string]any)
		if propsMap == nil {
			propsMap = make(map[string]any)
		}

		pr, err := p.createResource(logicalId, res.Type, propsMap)
		if err != nil {
			// Mark as failed but continue — best effort.
			pr = &ProvisionedResource{
				LogicalId:  logicalId,
				PhysicalId: "",
				Type:       res.Type,
				Attributes: make(map[string]string),
				Status:     "CREATE_FAILED",
			}
		}

		provisioned[logicalId] = pr
		result = append(result, *pr)
	}

	return result, nil
}

// DeleteResources deletes provisioned resources in reverse dependency order.
func (p *Provisioner) DeleteResources(resources []ProvisionedResource) {
	// Delete in reverse order.
	for i := len(resources) - 1; i >= 0; i-- {
		res := resources[i]
		if res.Status == "CREATE_FAILED" || res.PhysicalId == "" {
			continue
		}
		_ = p.deleteResource(res)
	}
}

// createResource dispatches to the appropriate service based on resource type.
func (p *Provisioner) createResource(logicalId, resourceType string, props map[string]any) (*ProvisionedResource, error) {
	switch resourceType {
	case "AWS::S3::Bucket":
		return p.createS3Bucket(logicalId, props)
	case "AWS::DynamoDB::Table":
		return p.createDynamoDBTable(logicalId, props)
	case "AWS::SQS::Queue":
		return p.createSQSQueue(logicalId, props)
	case "AWS::SNS::Topic":
		return p.createSNSTopic(logicalId, props)
	case "AWS::Lambda::Function":
		return p.createLambdaFunction(logicalId, props)
	case "AWS::IAM::Role":
		return p.createIAMRole(logicalId, props)
	case "AWS::IAM::Policy":
		return p.createIAMPolicy(logicalId, props)
	case "AWS::EC2::VPC":
		return p.createEC2VPC(logicalId, props)
	case "AWS::EC2::Subnet":
		return p.createEC2Subnet(logicalId, props)
	case "AWS::EC2::SecurityGroup":
		return p.createEC2SecurityGroup(logicalId, props)
	case "AWS::RDS::DBInstance":
		return p.createRDSDBInstance(logicalId, props)
	case "AWS::RDS::DBSubnetGroup":
		return p.createRDSDBSubnetGroup(logicalId, props)
	case "AWS::ECS::Cluster":
		return p.createECSCluster(logicalId, props)
	case "AWS::ECS::TaskDefinition":
		return p.createECSTaskDefinition(logicalId, props)
	case "AWS::ECS::Service":
		return p.createECSService(logicalId, props)
	case "AWS::Events::Rule":
		return p.createEventsRule(logicalId, props)
	case "AWS::StepFunctions::StateMachine":
		return p.createStepFunctionsStateMachine(logicalId, props)
	case "AWS::Logs::LogGroup":
		return p.createLogsLogGroup(logicalId, props)
	case "AWS::Route53::HostedZone":
		return p.createRoute53HostedZone(logicalId, props)
	case "AWS::KMS::Key":
		return p.createKMSKey(logicalId, props)
	case "AWS::SecretsManager::Secret":
		return p.createSecretsManagerSecret(logicalId, props)
	case "AWS::SSM::Parameter":
		return p.createSSMParameter(logicalId, props)
	case "AWS::ElastiCache::CacheCluster":
		return p.createElastiCacheCacheCluster(logicalId, props)
	case "AWS::EKS::Cluster":
		return p.createEKSCluster(logicalId, props)
	case "AWS::Kinesis::Stream":
		return p.createKinesisStream(logicalId, props)
	case "AWS::CloudFront::Distribution":
		return p.createCloudFrontDistribution(logicalId, props)
	case "AWS::Cognito::UserPool":
		return p.createCognitoUserPool(logicalId, props)
	default:
		// Unsupported resource type — return a stub with a generated physical ID.
		return &ProvisionedResource{
			LogicalId:  logicalId,
			PhysicalId: logicalId + "-" + newUUID()[:8],
			Type:       resourceType,
			Attributes: make(map[string]string),
			Status:     "CREATE_COMPLETE",
		}, nil
	}
}

// deleteResource dispatches deletion to the appropriate service.
func (p *Provisioner) deleteResource(res ProvisionedResource) error {
	switch res.Type {
	case "AWS::S3::Bucket":
		return p.deleteS3Bucket(res)
	case "AWS::SQS::Queue":
		return p.deleteSQSQueue(res)
	case "AWS::SNS::Topic":
		return p.deleteSNSTopic(res)
	case "AWS::IAM::Role":
		return p.deleteIAMRole(res)
	case "AWS::IAM::Policy":
		return p.deleteIAMPolicy(res)
	case "AWS::DynamoDB::Table":
		return p.deleteDynamoDBTable(res)
	case "AWS::RDS::DBInstance":
		return p.deleteRDSDBInstance(res)
	case "AWS::RDS::DBSubnetGroup":
		return p.deleteRDSDBSubnetGroup(res)
	case "AWS::ECS::Cluster":
		return p.deleteECSCluster(res)
	case "AWS::ECS::TaskDefinition":
		return p.deleteECSTaskDefinition(res)
	case "AWS::ECS::Service":
		return p.deleteECSService(res)
	case "AWS::Events::Rule":
		return p.deleteEventsRule(res)
	case "AWS::StepFunctions::StateMachine":
		return p.deleteStepFunctionsStateMachine(res)
	case "AWS::Logs::LogGroup":
		return p.deleteLogsLogGroup(res)
	case "AWS::Route53::HostedZone":
		return p.deleteRoute53HostedZone(res)
	case "AWS::KMS::Key":
		return p.deleteKMSKey(res)
	case "AWS::SecretsManager::Secret":
		return p.deleteSecretsManagerSecret(res)
	case "AWS::SSM::Parameter":
		return p.deleteSSMParameter(res)
	case "AWS::ElastiCache::CacheCluster":
		return p.deleteElastiCacheCacheCluster(res)
	case "AWS::EKS::Cluster":
		return p.deleteEKSCluster(res)
	case "AWS::Kinesis::Stream":
		return p.deleteKinesisStream(res)
	case "AWS::CloudFront::Distribution":
		return p.deleteCloudFrontDistribution(res)
	case "AWS::Cognito::UserPool":
		return p.deleteCognitoUserPool(res)
	default:
		// Best effort — many types don't need explicit cleanup.
		return nil
	}
}

// ---- S3 ----

func (p *Provisioner) createS3Bucket(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	bucketName := stringProp(props, "BucketName")
	if bucketName == "" {
		// Auto-generate a bucket name like CFN does.
		bucketName = strings.ToLower(logicalId) + "-" + newUUID()[:12]
	}

	svc, err := p.locator.Lookup("s3")
	if err != nil {
		return nil, fmt.Errorf("s3 service not found: %w", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/"+bucketName, nil)
	req.Header.Set("Authorization", p.fakeAuth("s3"))
	ctx := &service.RequestContext{
		Action:     "CreateBucket",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Service:    "s3",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create S3 bucket %s: %v", bucketName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:s3:::%s", bucketName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: bucketName,
		Type:       "AWS::S3::Bucket",
		Attributes: map[string]string{
			"Arn":        arn,
			"DomainName": bucketName + ".s3.amazonaws.com",
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteS3Bucket(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("s3")
	if err != nil {
		return err
	}
	req := httptest.NewRequest(http.MethodDelete, "/"+res.PhysicalId, nil)
	req.Header.Set("Authorization", p.fakeAuth("s3"))
	ctx := &service.RequestContext{
		Action:     "DeleteBucket",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Service:    "s3",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return fmt.Errorf("failed to delete S3 bucket %s: %v", res.PhysicalId, svcErr)
	}
	return nil
}

// ---- DynamoDB ----

func (p *Provisioner) createDynamoDBTable(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	tableName := stringProp(props, "TableName")
	if tableName == "" {
		tableName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("dynamodb")
	if err != nil {
		return nil, fmt.Errorf("dynamodb service not found: %w", err)
	}

	// Build a CreateTable JSON body.
	body := map[string]any{
		"TableName": tableName,
	}

	// KeySchema
	if ks, ok := props["KeySchema"]; ok {
		body["KeySchema"] = ks
	} else {
		// Default key schema if none provided.
		body["KeySchema"] = []map[string]string{
			{"AttributeName": "id", "KeyType": "HASH"},
		}
		body["AttributeDefinitions"] = []map[string]string{
			{"AttributeName": "id", "AttributeType": "S"},
		}
	}
	if ad, ok := props["AttributeDefinitions"]; ok {
		body["AttributeDefinitions"] = ad
	}
	if bm, ok := props["BillingMode"]; ok {
		body["BillingMode"] = bm
	} else {
		body["BillingMode"] = "PAY_PER_REQUEST"
	}
	if pt, ok := props["ProvisionedThroughput"]; ok {
		body["ProvisionedThroughput"] = pt
	}
	if gsi, ok := props["GlobalSecondaryIndexes"]; ok {
		body["GlobalSecondaryIndexes"] = gsi
	}
	if lsi, ok := props["LocalSecondaryIndexes"]; ok {
		body["LocalSecondaryIndexes"] = lsi
	}

	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.CreateTable")
	req.Header.Set("Authorization", p.fakeAuth("dynamodb"))
	ctx := &service.RequestContext{
		Action:     "CreateTable",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "dynamodb",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create DynamoDB table %s: %v", tableName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:dynamodb:%s:%s:table/%s", p.region, p.accountID, tableName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: tableName,
		Type:       "AWS::DynamoDB::Table",
		Attributes: map[string]string{
			"Arn":       arn,
			"TableName": tableName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteDynamoDBTable(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("dynamodb")
	if err != nil {
		return err
	}
	body := map[string]string{"TableName": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.DeleteTable")
	req.Header.Set("Authorization", p.fakeAuth("dynamodb"))
	ctx := &service.RequestContext{
		Action:     "DeleteTable",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "dynamodb",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return fmt.Errorf("failed to delete DynamoDB table %s: %v", res.PhysicalId, svcErr)
	}
	return nil
}

// ---- SQS ----

func (p *Provisioner) createSQSQueue(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	queueName := stringProp(props, "QueueName")
	if queueName == "" {
		queueName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("sqs")
	if err != nil {
		return nil, fmt.Errorf("sqs service not found: %w", err)
	}

	form := url.Values{}
	form.Set("Action", "CreateQueue")
	form.Set("QueueName", queueName)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("sqs"))
	ctx := &service.RequestContext{
		Action:     "CreateQueue",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Service:    "sqs",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create SQS queue %s: %v", queueName, svcErr)
	}

	queueURL := fmt.Sprintf("http://sqs.%s.localhost:4566/%s/%s", p.region, p.accountID, queueName)
	arn := fmt.Sprintf("arn:aws:sqs:%s:%s:%s", p.region, p.accountID, queueName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: queueURL,
		Type:       "AWS::SQS::Queue",
		Attributes: map[string]string{
			"Arn":       arn,
			"QueueName": queueName,
			"QueueUrl":  queueURL,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteSQSQueue(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("sqs")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("Action", "DeleteQueue")
	form.Set("QueueUrl", res.PhysicalId)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("sqs"))
	ctx := &service.RequestContext{
		Action:     "DeleteQueue",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Service:    "sqs",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- SNS ----

func (p *Provisioner) createSNSTopic(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	topicName := stringProp(props, "TopicName")
	if topicName == "" {
		topicName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("sns")
	if err != nil {
		return nil, fmt.Errorf("sns service not found: %w", err)
	}

	form := url.Values{}
	form.Set("Action", "CreateTopic")
	form.Set("Name", topicName)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("sns"))
	ctx := &service.RequestContext{
		Action:     "CreateTopic",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Service:    "sns",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create SNS topic %s: %v", topicName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:sns:%s:%s:%s", p.region, p.accountID, topicName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: arn,
		Type:       "AWS::SNS::Topic",
		Attributes: map[string]string{
			"TopicArn":  arn,
			"TopicName": topicName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteSNSTopic(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("sns")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("Action", "DeleteTopic")
	form.Set("TopicArn", res.PhysicalId)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("sns"))
	ctx := &service.RequestContext{
		Action:     "DeleteTopic",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Service:    "sns",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- Lambda ----

func (p *Provisioner) createLambdaFunction(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	funcName := stringProp(props, "FunctionName")
	if funcName == "" {
		funcName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("lambda")
	if err != nil {
		return nil, fmt.Errorf("lambda service not found: %w", err)
	}

	body := map[string]any{
		"FunctionName": funcName,
		"Runtime":      stringPropDefault(props, "Runtime", "nodejs18.x"),
		"Handler":      stringPropDefault(props, "Handler", "index.handler"),
		"Role":         stringPropDefault(props, "Role", fmt.Sprintf("arn:aws:iam::%s:role/cloudformation-placeholder", p.accountID)),
		"Code": map[string]string{
			"ZipFile": "UEsDBBQAAAAIAAAAAACKIYowEQAAABIAAAAIABwAaW5kZXguanNVVAkAAw==", // minimal zip
		},
	}
	if desc, ok := props["Description"]; ok {
		body["Description"] = desc
	}
	if timeout, ok := props["Timeout"]; ok {
		body["Timeout"] = timeout
	}
	if memSize, ok := props["MemorySize"]; ok {
		body["MemorySize"] = memSize
	}
	if env, ok := props["Environment"]; ok {
		body["Environment"] = env
	}

	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/2015-03-31/functions", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", p.fakeAuth("lambda"))
	ctx := &service.RequestContext{
		Action:     "CreateFunction",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "lambda",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create Lambda function %s: %v", funcName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s", p.region, p.accountID, funcName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: funcName,
		Type:       "AWS::Lambda::Function",
		Attributes: map[string]string{
			"Arn":          arn,
			"FunctionName": funcName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

// ---- IAM Role ----

func (p *Provisioner) createIAMRole(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	roleName := stringProp(props, "RoleName")
	if roleName == "" {
		roleName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("iam")
	if err != nil {
		return nil, fmt.Errorf("iam service not found: %w", err)
	}

	assumeDoc := "{}"
	if doc, ok := props["AssumeRolePolicyDocument"]; ok {
		b, _ := json.Marshal(doc)
		assumeDoc = string(b)
	}

	form := url.Values{}
	form.Set("Action", "CreateRole")
	form.Set("RoleName", roleName)
	form.Set("AssumeRolePolicyDocument", assumeDoc)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("iam"))
	ctx := &service.RequestContext{
		Action:     "CreateRole",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "iam",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create IAM role %s: %v", roleName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:iam::%s:role/%s", p.accountID, roleName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: roleName,
		Type:       "AWS::IAM::Role",
		Attributes: map[string]string{
			"Arn":      arn,
			"RoleName": roleName,
			"RoleId":   "AROA" + newUUID()[:12],
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteIAMRole(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("iam")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("Action", "DeleteRole")
	form.Set("RoleName", res.PhysicalId)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("iam"))
	ctx := &service.RequestContext{
		Action:     "DeleteRole",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "iam",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- IAM Policy ----

func (p *Provisioner) createIAMPolicy(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	policyName := stringProp(props, "PolicyName")
	if policyName == "" {
		policyName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("iam")
	if err != nil {
		return nil, fmt.Errorf("iam service not found: %w", err)
	}

	policyDoc := "{}"
	if doc, ok := props["PolicyDocument"]; ok {
		b, _ := json.Marshal(doc)
		policyDoc = string(b)
	}

	form := url.Values{}
	form.Set("Action", "CreatePolicy")
	form.Set("PolicyName", policyName)
	form.Set("PolicyDocument", policyDoc)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("iam"))
	ctx := &service.RequestContext{
		Action:     "CreatePolicy",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "iam",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create IAM policy %s: %v", policyName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:iam::%s:policy/%s", p.accountID, policyName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: arn,
		Type:       "AWS::IAM::Policy",
		Attributes: map[string]string{
			"Arn": arn,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteIAMPolicy(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("iam")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("Action", "DeletePolicy")
	form.Set("PolicyArn", res.PhysicalId)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("iam"))
	ctx := &service.RequestContext{
		Action:     "DeletePolicy",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "iam",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- EC2 VPC ----

func (p *Provisioner) createEC2VPC(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	cidrBlock := stringPropDefault(props, "CidrBlock", "10.0.0.0/16")

	svc, err := p.locator.Lookup("ec2")
	if err != nil {
		return nil, fmt.Errorf("ec2 service not found: %w", err)
	}

	form := url.Values{}
	form.Set("Action", "CreateVpc")
	form.Set("CidrBlock", cidrBlock)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("ec2"))
	ctx := &service.RequestContext{
		Action:     "CreateVpc",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Service:    "ec2",
	}
	resp, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create VPC: %v", svcErr)
	}

	// Extract VpcId from the response.
	vpcId := extractVpcIdFromResponse(resp)
	if vpcId == "" {
		vpcId = "vpc-" + newUUID()[:8]
	}

	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: vpcId,
		Type:       "AWS::EC2::VPC",
		Attributes: map[string]string{
			"VpcId":     vpcId,
			"CidrBlock": cidrBlock,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

// ---- EC2 Subnet ----

func (p *Provisioner) createEC2Subnet(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	vpcId := stringProp(props, "VpcId")
	cidrBlock := stringPropDefault(props, "CidrBlock", "10.0.1.0/24")

	svc, err := p.locator.Lookup("ec2")
	if err != nil {
		return nil, fmt.Errorf("ec2 service not found: %w", err)
	}

	form := url.Values{}
	form.Set("Action", "CreateSubnet")
	form.Set("VpcId", vpcId)
	form.Set("CidrBlock", cidrBlock)
	if az := stringProp(props, "AvailabilityZone"); az != "" {
		form.Set("AvailabilityZone", az)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("ec2"))
	ctx := &service.RequestContext{
		Action:     "CreateSubnet",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Service:    "ec2",
	}
	resp, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create Subnet: %v", svcErr)
	}

	subnetId := extractSubnetIdFromResponse(resp)
	if subnetId == "" {
		subnetId = "subnet-" + newUUID()[:8]
	}

	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: subnetId,
		Type:       "AWS::EC2::Subnet",
		Attributes: map[string]string{
			"SubnetId":  subnetId,
			"VpcId":     vpcId,
			"CidrBlock": cidrBlock,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

// ---- EC2 SecurityGroup ----

func (p *Provisioner) createEC2SecurityGroup(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	groupName := stringPropDefault(props, "GroupName", logicalId)
	description := stringPropDefault(props, "GroupDescription", logicalId+" security group")
	vpcId := stringProp(props, "VpcId")

	svc, err := p.locator.Lookup("ec2")
	if err != nil {
		return nil, fmt.Errorf("ec2 service not found: %w", err)
	}

	form := url.Values{}
	form.Set("Action", "CreateSecurityGroup")
	form.Set("GroupName", groupName)
	form.Set("GroupDescription", description)
	if vpcId != "" {
		form.Set("VpcId", vpcId)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("ec2"))
	ctx := &service.RequestContext{
		Action:     "CreateSecurityGroup",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Service:    "ec2",
	}
	resp, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create SecurityGroup: %v", svcErr)
	}

	sgId := extractSecurityGroupIdFromResponse(resp)
	if sgId == "" {
		sgId = "sg-" + newUUID()[:8]
	}

	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: sgId,
		Type:       "AWS::EC2::SecurityGroup",
		Attributes: map[string]string{
			"GroupId":   sgId,
			"GroupName": groupName,
			"VpcId":     vpcId,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

// ---- RDS DBInstance ----

func (p *Provisioner) createRDSDBInstance(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	dbInstanceId := stringProp(props, "DBInstanceIdentifier")
	if dbInstanceId == "" {
		dbInstanceId = strings.ToLower(logicalId) + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("rds")
	if err != nil {
		return nil, fmt.Errorf("rds service not found: %w", err)
	}

	form := url.Values{}
	form.Set("Action", "CreateDBInstance")
	form.Set("DBInstanceIdentifier", dbInstanceId)
	form.Set("Engine", stringPropDefault(props, "Engine", "mysql"))
	form.Set("DBInstanceClass", stringPropDefault(props, "DBInstanceClass", "db.t3.micro"))
	form.Set("MasterUsername", stringPropDefault(props, "MasterUsername", "admin"))
	form.Set("MasterUserPassword", stringPropDefault(props, "MasterUserPassword", "password"))
	form.Set("AllocatedStorage", stringPropDefault(props, "AllocatedStorage", "20"))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("rds"))
	ctx := &service.RequestContext{
		Action:     "CreateDBInstance",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "rds",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create RDS DB instance %s: %v", dbInstanceId, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:rds:%s:%s:db:%s", p.region, p.accountID, dbInstanceId)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: dbInstanceId,
		Type:       "AWS::RDS::DBInstance",
		Attributes: map[string]string{
			"Arn":                    arn,
			"DBInstanceIdentifier":   dbInstanceId,
			"Endpoint.Address":       dbInstanceId + "." + p.region + ".rds.amazonaws.com",
			"Endpoint.Port":          "3306",
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteRDSDBInstance(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("rds")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("Action", "DeleteDBInstance")
	form.Set("DBInstanceIdentifier", res.PhysicalId)
	form.Set("SkipFinalSnapshot", "true")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("rds"))
	ctx := &service.RequestContext{
		Action:     "DeleteDBInstance",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "rds",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- RDS DBSubnetGroup ----

func (p *Provisioner) createRDSDBSubnetGroup(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	groupName := stringProp(props, "DBSubnetGroupName")
	if groupName == "" {
		groupName = strings.ToLower(logicalId) + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("rds")
	if err != nil {
		return nil, fmt.Errorf("rds service not found: %w", err)
	}

	form := url.Values{}
	form.Set("Action", "CreateDBSubnetGroup")
	form.Set("DBSubnetGroupName", groupName)
	form.Set("DBSubnetGroupDescription", stringPropDefault(props, "DBSubnetGroupDescription", groupName+" subnet group"))
	if subnetIds, ok := props["SubnetIds"]; ok {
		if ids, ok := subnetIds.([]any); ok {
			for i, id := range ids {
				if s, ok := id.(string); ok {
					form.Set(fmt.Sprintf("SubnetIds.member.%d", i+1), s)
				}
			}
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("rds"))
	ctx := &service.RequestContext{
		Action:     "CreateDBSubnetGroup",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "rds",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create RDS DB subnet group %s: %v", groupName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:rds:%s:%s:subgrp:%s", p.region, p.accountID, groupName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: groupName,
		Type:       "AWS::RDS::DBSubnetGroup",
		Attributes: map[string]string{
			"Arn":                arn,
			"DBSubnetGroupName": groupName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteRDSDBSubnetGroup(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("rds")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("Action", "DeleteDBSubnetGroup")
	form.Set("DBSubnetGroupName", res.PhysicalId)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("rds"))
	ctx := &service.RequestContext{
		Action:     "DeleteDBSubnetGroup",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "rds",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- ECS Cluster ----

func (p *Provisioner) createECSCluster(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	clusterName := stringProp(props, "ClusterName")
	if clusterName == "" {
		clusterName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("ecs")
	if err != nil {
		return nil, fmt.Errorf("ecs service not found: %w", err)
	}

	body := map[string]any{
		"clusterName": clusterName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.CreateCluster")
	req.Header.Set("Authorization", p.fakeAuth("ecs"))
	ctx := &service.RequestContext{
		Action:     "CreateCluster",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "ecs",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create ECS cluster %s: %v", clusterName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", p.region, p.accountID, clusterName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: arn,
		Type:       "AWS::ECS::Cluster",
		Attributes: map[string]string{
			"Arn":         arn,
			"ClusterName": clusterName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteECSCluster(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("ecs")
	if err != nil {
		return err
	}

	body := map[string]string{"cluster": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.DeleteCluster")
	req.Header.Set("Authorization", p.fakeAuth("ecs"))
	ctx := &service.RequestContext{
		Action:     "DeleteCluster",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "ecs",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- ECS TaskDefinition ----

func (p *Provisioner) createECSTaskDefinition(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	family := stringProp(props, "Family")
	if family == "" {
		family = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("ecs")
	if err != nil {
		return nil, fmt.Errorf("ecs service not found: %w", err)
	}

	body := map[string]any{
		"family": family,
	}
	if cd, ok := props["ContainerDefinitions"]; ok {
		body["containerDefinitions"] = cd
	}
	if cpu, ok := props["Cpu"]; ok {
		body["cpu"] = cpu
	}
	if mem, ok := props["Memory"]; ok {
		body["memory"] = mem
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition")
	req.Header.Set("Authorization", p.fakeAuth("ecs"))
	ctx := &service.RequestContext{
		Action:     "RegisterTaskDefinition",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "ecs",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to register ECS task definition %s: %v", family, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s:1", p.region, p.accountID, family)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: arn,
		Type:       "AWS::ECS::TaskDefinition",
		Attributes: map[string]string{
			"TaskDefinitionArn": arn,
			"Family":            family,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteECSTaskDefinition(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("ecs")
	if err != nil {
		return err
	}

	body := map[string]string{"taskDefinition": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.DeregisterTaskDefinition")
	req.Header.Set("Authorization", p.fakeAuth("ecs"))
	ctx := &service.RequestContext{
		Action:     "DeregisterTaskDefinition",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "ecs",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- ECS Service ----

func (p *Provisioner) createECSService(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	serviceName := stringProp(props, "ServiceName")
	if serviceName == "" {
		serviceName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("ecs")
	if err != nil {
		return nil, fmt.Errorf("ecs service not found: %w", err)
	}

	body := map[string]any{
		"serviceName": serviceName,
	}
	if cluster, ok := props["Cluster"]; ok {
		body["cluster"] = cluster
	}
	if td, ok := props["TaskDefinition"]; ok {
		body["taskDefinition"] = td
	}
	if dc, ok := props["DesiredCount"]; ok {
		body["desiredCount"] = dc
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.CreateService")
	req.Header.Set("Authorization", p.fakeAuth("ecs"))
	ctx := &service.RequestContext{
		Action:     "CreateService",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "ecs",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create ECS service %s: %v", serviceName, svcErr)
	}

	clusterName := stringPropDefault(props, "Cluster", "default")
	arn := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", p.region, p.accountID, clusterName, serviceName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: arn,
		Type:       "AWS::ECS::Service",
		Attributes: map[string]string{
			"ServiceArn":  arn,
			"ServiceName": serviceName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteECSService(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("ecs")
	if err != nil {
		return err
	}

	body := map[string]string{"service": res.PhysicalId, "force": "true"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.DeleteService")
	req.Header.Set("Authorization", p.fakeAuth("ecs"))
	ctx := &service.RequestContext{
		Action:     "DeleteService",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "ecs",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- EventBridge Rule ----

func (p *Provisioner) createEventsRule(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	ruleName := stringProp(props, "Name")
	if ruleName == "" {
		ruleName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("events")
	if err != nil {
		return nil, fmt.Errorf("events service not found: %w", err)
	}

	body := map[string]any{
		"Name": ruleName,
	}
	if sched, ok := props["ScheduleExpression"]; ok {
		body["ScheduleExpression"] = sched
	}
	if pattern, ok := props["EventPattern"]; ok {
		body["EventPattern"] = pattern
	}
	if state, ok := props["State"]; ok {
		body["State"] = state
	} else {
		body["State"] = "ENABLED"
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSEvents.PutRule")
	req.Header.Set("Authorization", p.fakeAuth("events"))
	ctx := &service.RequestContext{
		Action:     "PutRule",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "events",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create EventBridge rule %s: %v", ruleName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:events:%s:%s:rule/%s", p.region, p.accountID, ruleName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: ruleName,
		Type:       "AWS::Events::Rule",
		Attributes: map[string]string{
			"Arn":  arn,
			"Name": ruleName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteEventsRule(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("events")
	if err != nil {
		return err
	}

	body := map[string]string{"Name": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSEvents.DeleteRule")
	req.Header.Set("Authorization", p.fakeAuth("events"))
	ctx := &service.RequestContext{
		Action:     "DeleteRule",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "events",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- Step Functions StateMachine ----

func (p *Provisioner) createStepFunctionsStateMachine(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	smName := stringProp(props, "StateMachineName")
	if smName == "" {
		smName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("stepfunctions")
	if err != nil {
		return nil, fmt.Errorf("stepfunctions service not found: %w", err)
	}

	body := map[string]any{
		"name": smName,
	}
	if def, ok := props["DefinitionString"]; ok {
		body["definition"] = def
	} else if def, ok := props["Definition"]; ok {
		b, _ := json.Marshal(def)
		body["definition"] = string(b)
	}
	if roleArn, ok := props["RoleArn"]; ok {
		body["roleArn"] = roleArn
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AWSStepFunctions.CreateStateMachine")
	req.Header.Set("Authorization", p.fakeAuth("stepfunctions"))
	ctx := &service.RequestContext{
		Action:     "CreateStateMachine",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "stepfunctions",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create Step Functions state machine %s: %v", smName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:states:%s:%s:stateMachine:%s", p.region, p.accountID, smName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: arn,
		Type:       "AWS::StepFunctions::StateMachine",
		Attributes: map[string]string{
			"Arn":  arn,
			"Name": smName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteStepFunctionsStateMachine(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("stepfunctions")
	if err != nil {
		return err
	}

	body := map[string]string{"stateMachineArn": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AWSStepFunctions.DeleteStateMachine")
	req.Header.Set("Authorization", p.fakeAuth("stepfunctions"))
	ctx := &service.RequestContext{
		Action:     "DeleteStateMachine",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "stepfunctions",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- CloudWatch Logs LogGroup ----

func (p *Provisioner) createLogsLogGroup(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	logGroupName := stringProp(props, "LogGroupName")
	if logGroupName == "" {
		logGroupName = "/" + logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("logs")
	if err != nil {
		return nil, fmt.Errorf("logs service not found: %w", err)
	}

	body := map[string]any{
		"logGroupName": logGroupName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.CreateLogGroup")
	req.Header.Set("Authorization", p.fakeAuth("logs"))
	ctx := &service.RequestContext{
		Action:     "CreateLogGroup",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "logs",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create CloudWatch log group %s: %v", logGroupName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:logs:%s:%s:log-group:%s", p.region, p.accountID, logGroupName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: logGroupName,
		Type:       "AWS::Logs::LogGroup",
		Attributes: map[string]string{
			"Arn":          arn,
			"LogGroupName": logGroupName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteLogsLogGroup(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("logs")
	if err != nil {
		return err
	}

	body := map[string]string{"logGroupName": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.DeleteLogGroup")
	req.Header.Set("Authorization", p.fakeAuth("logs"))
	ctx := &service.RequestContext{
		Action:     "DeleteLogGroup",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "logs",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- Route53 HostedZone ----

func (p *Provisioner) createRoute53HostedZone(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	zoneName := stringProp(props, "Name")
	if zoneName == "" {
		zoneName = logicalId + ".example.com"
	}

	svc, err := p.locator.Lookup("route53")
	if err != nil {
		return nil, fmt.Errorf("route53 service not found: %w", err)
	}

	callerRef := stringPropDefault(props, "CallerReference", newUUID())
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<CreateHostedZoneRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <Name>%s</Name>
  <CallerReference>%s</CallerReference>
</CreateHostedZoneRequest>`, zoneName, callerRef)

	req := httptest.NewRequest(http.MethodPost, "/2013-04-01/hostedzone", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Authorization", p.fakeAuth("route53"))
	ctx := &service.RequestContext{
		Action:     "CreateHostedZone",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(body),
		Service:    "route53",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create Route53 hosted zone %s: %v", zoneName, svcErr)
	}

	zoneId := "Z" + strings.ToUpper(newUUID()[:12])
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: zoneId,
		Type:       "AWS::Route53::HostedZone",
		Attributes: map[string]string{
			"Id":           zoneId,
			"HostedZoneId": zoneId,
			"Name":         zoneName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteRoute53HostedZone(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("route53")
	if err != nil {
		return err
	}

	req := httptest.NewRequest(http.MethodDelete, "/2013-04-01/hostedzone/"+res.PhysicalId, nil)
	req.Header.Set("Authorization", p.fakeAuth("route53"))
	ctx := &service.RequestContext{
		Action:     "DeleteHostedZone",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Service:    "route53",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- KMS Key ----

func (p *Provisioner) createKMSKey(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	svc, err := p.locator.Lookup("kms")
	if err != nil {
		return nil, fmt.Errorf("kms service not found: %w", err)
	}

	body := map[string]any{}
	if desc, ok := props["Description"]; ok {
		body["Description"] = desc
	}
	if usage, ok := props["KeyUsage"]; ok {
		body["KeyUsage"] = usage
	} else {
		body["KeyUsage"] = "ENCRYPT_DECRYPT"
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "TrentService.CreateKey")
	req.Header.Set("Authorization", p.fakeAuth("kms"))
	ctx := &service.RequestContext{
		Action:     "CreateKey",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "kms",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create KMS key: %v", svcErr)
	}

	keyId := newUUID()
	arn := fmt.Sprintf("arn:aws:kms:%s:%s:key/%s", p.region, p.accountID, keyId)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: keyId,
		Type:       "AWS::KMS::Key",
		Attributes: map[string]string{
			"Arn":   arn,
			"KeyId": keyId,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteKMSKey(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("kms")
	if err != nil {
		return err
	}

	body := map[string]any{"KeyId": res.PhysicalId, "PendingWindowInDays": 7}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "TrentService.ScheduleKeyDeletion")
	req.Header.Set("Authorization", p.fakeAuth("kms"))
	ctx := &service.RequestContext{
		Action:     "ScheduleKeyDeletion",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "kms",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- Secrets Manager Secret ----

func (p *Provisioner) createSecretsManagerSecret(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	secretName := stringProp(props, "Name")
	if secretName == "" {
		secretName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("secretsmanager")
	if err != nil {
		return nil, fmt.Errorf("secretsmanager service not found: %w", err)
	}

	body := map[string]any{
		"Name": secretName,
	}
	if ss, ok := props["SecretString"]; ok {
		body["SecretString"] = ss
	} else if gss, ok := props["GenerateSecretString"]; ok {
		// Pass through the generate config; the service will handle it.
		body["GenerateSecretString"] = gss
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager.CreateSecret")
	req.Header.Set("Authorization", p.fakeAuth("secretsmanager"))
	ctx := &service.RequestContext{
		Action:     "CreateSecret",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "secretsmanager",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create Secrets Manager secret %s: %v", secretName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:secretsmanager:%s:%s:secret:%s-%s", p.region, p.accountID, secretName, newUUID()[:6])
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: arn,
		Type:       "AWS::SecretsManager::Secret",
		Attributes: map[string]string{
			"Arn":  arn,
			"Name": secretName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteSecretsManagerSecret(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("secretsmanager")
	if err != nil {
		return err
	}

	body := map[string]any{"SecretId": res.PhysicalId, "ForceDeleteWithoutRecovery": true}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager.DeleteSecret")
	req.Header.Set("Authorization", p.fakeAuth("secretsmanager"))
	ctx := &service.RequestContext{
		Action:     "DeleteSecret",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "secretsmanager",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- SSM Parameter ----

func (p *Provisioner) createSSMParameter(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	paramName := stringProp(props, "Name")
	if paramName == "" {
		paramName = "/" + logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("ssm")
	if err != nil {
		return nil, fmt.Errorf("ssm service not found: %w", err)
	}

	body := map[string]any{
		"Name":  paramName,
		"Type":  stringPropDefault(props, "Type", "String"),
		"Value": stringPropDefault(props, "Value", ""),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonSSM.PutParameter")
	req.Header.Set("Authorization", p.fakeAuth("ssm"))
	ctx := &service.RequestContext{
		Action:     "PutParameter",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "ssm",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create SSM parameter %s: %v", paramName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:ssm:%s:%s:parameter%s", p.region, p.accountID, paramName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: paramName,
		Type:       "AWS::SSM::Parameter",
		Attributes: map[string]string{
			"Arn":   arn,
			"Name":  paramName,
			"Type":  stringPropDefault(props, "Type", "String"),
			"Value": stringPropDefault(props, "Value", ""),
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteSSMParameter(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("ssm")
	if err != nil {
		return err
	}

	body := map[string]string{"Name": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonSSM.DeleteParameter")
	req.Header.Set("Authorization", p.fakeAuth("ssm"))
	ctx := &service.RequestContext{
		Action:     "DeleteParameter",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "ssm",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- ElastiCache CacheCluster ----

func (p *Provisioner) createElastiCacheCacheCluster(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	clusterId := stringProp(props, "CacheClusterId")
	if clusterId == "" {
		clusterId = strings.ToLower(logicalId) + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("elasticache")
	if err != nil {
		return nil, fmt.Errorf("elasticache service not found: %w", err)
	}

	form := url.Values{}
	form.Set("Action", "CreateCacheCluster")
	form.Set("CacheClusterId", clusterId)
	form.Set("Engine", stringPropDefault(props, "Engine", "redis"))
	form.Set("CacheNodeType", stringPropDefault(props, "CacheNodeType", "cache.t3.micro"))
	form.Set("NumCacheNodes", stringPropDefault(props, "NumCacheNodes", "1"))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("elasticache"))
	ctx := &service.RequestContext{
		Action:     "CreateCacheCluster",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "elasticache",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create ElastiCache cluster %s: %v", clusterId, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:elasticache:%s:%s:cluster:%s", p.region, p.accountID, clusterId)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: clusterId,
		Type:       "AWS::ElastiCache::CacheCluster",
		Attributes: map[string]string{
			"Arn":                          arn,
			"CacheClusterId":               clusterId,
			"ConfigurationEndpoint.Address": clusterId + "." + p.region + ".cache.amazonaws.com",
			"ConfigurationEndpoint.Port":    "6379",
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteElastiCacheCacheCluster(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("elasticache")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("Action", "DeleteCacheCluster")
	form.Set("CacheClusterId", res.PhysicalId)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", p.fakeAuth("elasticache"))
	ctx := &service.RequestContext{
		Action:     "DeleteCacheCluster",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       []byte(form.Encode()),
		Params:     formToMap(form),
		Service:    "elasticache",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- EKS Cluster ----

func (p *Provisioner) createEKSCluster(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	clusterName := stringProp(props, "Name")
	if clusterName == "" {
		clusterName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("eks")
	if err != nil {
		return nil, fmt.Errorf("eks service not found: %w", err)
	}

	body := map[string]any{
		"name": clusterName,
	}
	if roleArn, ok := props["RoleArn"]; ok {
		body["roleArn"] = roleArn
	}
	if vpcConfig, ok := props["ResourcesVpcConfig"]; ok {
		body["resourcesVpcConfig"] = vpcConfig
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/clusters", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", p.fakeAuth("eks"))
	ctx := &service.RequestContext{
		Action:     "CreateCluster",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "eks",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create EKS cluster %s: %v", clusterName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", p.region, p.accountID, clusterName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: clusterName,
		Type:       "AWS::EKS::Cluster",
		Attributes: map[string]string{
			"Arn":                  arn,
			"ClusterName":         clusterName,
			"Endpoint":            fmt.Sprintf("https://%s.eks.%s.amazonaws.com", newUUID()[:12], p.region),
			"CertificateAuthorityData": "LS0tLS1CRUdJTi...",
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteEKSCluster(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("eks")
	if err != nil {
		return err
	}

	req := httptest.NewRequest(http.MethodDelete, "/clusters/"+res.PhysicalId, nil)
	req.Header.Set("Authorization", p.fakeAuth("eks"))
	ctx := &service.RequestContext{
		Action:     "DeleteCluster",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Service:    "eks",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- Kinesis Stream ----

func (p *Provisioner) createKinesisStream(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	streamName := stringProp(props, "Name")
	if streamName == "" {
		streamName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("kinesis")
	if err != nil {
		return nil, fmt.Errorf("kinesis service not found: %w", err)
	}

	body := map[string]any{
		"StreamName": streamName,
	}
	if sc, ok := props["ShardCount"]; ok {
		body["ShardCount"] = sc
	} else {
		body["ShardCount"] = 1
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Kinesis_20131202.CreateStream")
	req.Header.Set("Authorization", p.fakeAuth("kinesis"))
	ctx := &service.RequestContext{
		Action:     "CreateStream",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "kinesis",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create Kinesis stream %s: %v", streamName, svcErr)
	}

	arn := fmt.Sprintf("arn:aws:kinesis:%s:%s:stream/%s", p.region, p.accountID, streamName)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: streamName,
		Type:       "AWS::Kinesis::Stream",
		Attributes: map[string]string{
			"Arn":        arn,
			"StreamName": streamName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteKinesisStream(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("kinesis")
	if err != nil {
		return err
	}

	body := map[string]string{"StreamName": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Kinesis_20131202.DeleteStream")
	req.Header.Set("Authorization", p.fakeAuth("kinesis"))
	ctx := &service.RequestContext{
		Action:     "DeleteStream",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "kinesis",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- CloudFront Distribution ----

func (p *Provisioner) createCloudFrontDistribution(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	svc, err := p.locator.Lookup("cloudfront")
	if err != nil {
		return nil, fmt.Errorf("cloudfront service not found: %w", err)
	}

	body := map[string]any{}
	if dc, ok := props["DistributionConfig"]; ok {
		body["DistributionConfig"] = dc
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/2020-05-31/distribution", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", p.fakeAuth("cloudfront"))
	ctx := &service.RequestContext{
		Action:     "CreateDistribution",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "cloudfront",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create CloudFront distribution: %v", svcErr)
	}

	distId := "E" + strings.ToUpper(newUUID()[:13])
	domainName := strings.ToLower(distId) + ".cloudfront.net"
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: distId,
		Type:       "AWS::CloudFront::Distribution",
		Attributes: map[string]string{
			"Id":         distId,
			"DomainName": domainName,
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteCloudFrontDistribution(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("cloudfront")
	if err != nil {
		return err
	}

	req := httptest.NewRequest(http.MethodDelete, "/2020-05-31/distribution/"+res.PhysicalId, nil)
	req.Header.Set("Authorization", p.fakeAuth("cloudfront"))
	ctx := &service.RequestContext{
		Action:     "DeleteDistribution",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Service:    "cloudfront",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- Cognito UserPool ----

func (p *Provisioner) createCognitoUserPool(logicalId string, props map[string]any) (*ProvisionedResource, error) {
	poolName := stringProp(props, "UserPoolName")
	if poolName == "" {
		poolName = logicalId + "-" + newUUID()[:8]
	}

	svc, err := p.locator.Lookup("cognito-idp")
	if err != nil {
		return nil, fmt.Errorf("cognito-idp service not found: %w", err)
	}

	body := map[string]any{
		"PoolName": poolName,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService.CreateUserPool")
	req.Header.Set("Authorization", p.fakeAuth("cognito-idp"))
	ctx := &service.RequestContext{
		Action:     "CreateUserPool",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "cognito-idp",
	}
	_, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		return nil, fmt.Errorf("failed to create Cognito user pool %s: %v", poolName, svcErr)
	}

	poolId := p.region + "_" + newUUID()[:9]
	arn := fmt.Sprintf("arn:aws:cognito-idp:%s:%s:userpool/%s", p.region, p.accountID, poolId)
	return &ProvisionedResource{
		LogicalId:  logicalId,
		PhysicalId: poolId,
		Type:       "AWS::Cognito::UserPool",
		Attributes: map[string]string{
			"Arn":          arn,
			"UserPoolId":   poolId,
			"UserPoolName": poolName,
			"ProviderName": fmt.Sprintf("cognito-idp.%s.amazonaws.com/%s", p.region, poolId),
		},
		Status: "CREATE_COMPLETE",
	}, nil
}

func (p *Provisioner) deleteCognitoUserPool(res ProvisionedResource) error {
	svc, err := p.locator.Lookup("cognito-idp")
	if err != nil {
		return err
	}

	body := map[string]string{"UserPoolId": res.PhysicalId}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService.DeleteUserPool")
	req.Header.Set("Authorization", p.fakeAuth("cognito-idp"))
	ctx := &service.RequestContext{
		Action:     "DeleteUserPool",
		Region:     p.region,
		AccountID:  p.accountID,
		RawRequest: req,
		Body:       bodyBytes,
		Service:    "cognito-idp",
	}
	_, _ = svc.HandleRequest(ctx)
	return nil
}

// ---- Intrinsic Function Resolution ----

// resolveIntrinsics recursively resolves Ref, Fn::Sub, Fn::GetAtt, Fn::Join, Fn::Select
// in a template value tree.
func resolveIntrinsics(value any, params map[string]string, resources map[string]*ProvisionedResource) any {
	switch v := value.(type) {
	case map[string]any:
		// Check for intrinsic functions.
		if ref, ok := v["Ref"]; ok {
			if refStr, ok := ref.(string); ok {
				return resolveRef(refStr, params, resources)
			}
		}
		if sub, ok := v["Fn::Sub"]; ok {
			return resolveFnSub(sub, params, resources)
		}
		if getAtt, ok := v["Fn::GetAtt"]; ok {
			return resolveFnGetAtt(getAtt, resources)
		}
		if join, ok := v["Fn::Join"]; ok {
			return resolveFnJoin(join, params, resources)
		}
		if sel, ok := v["Fn::Select"]; ok {
			return resolveFnSelect(sel, params, resources)
		}

		// Not an intrinsic — recurse into all keys.
		result := make(map[string]any, len(v))
		for key, val := range v {
			result[key] = resolveIntrinsics(val, params, resources)
		}
		return result

	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = resolveIntrinsics(item, params, resources)
		}
		return result

	default:
		return value
	}
}

// resolveRef resolves a Ref to a physical ID or parameter value.
func resolveRef(refName string, params map[string]string, resources map[string]*ProvisionedResource) string {
	// Check parameters first (including pseudo-parameters).
	if val, ok := params[refName]; ok {
		return val
	}
	// Check provisioned resources.
	if res, ok := resources[refName]; ok {
		return res.PhysicalId
	}
	return refName
}

// resolveFnSub resolves Fn::Sub with ${VarName} substitution.
func resolveFnSub(value any, params map[string]string, resources map[string]*ProvisionedResource) string {
	switch v := value.(type) {
	case string:
		return substituteVars(v, params, resources)
	case []any:
		// Fn::Sub with explicit variable map: ["template", {"Var": "value"}]
		if len(v) >= 1 {
			tmplStr, _ := v[0].(string)
			if len(v) >= 2 {
				if varMap, ok := v[1].(map[string]any); ok {
					// Merge variable map into params.
					merged := make(map[string]string, len(params))
					for k, pv := range params {
						merged[k] = pv
					}
					for k, pv := range varMap {
						resolved := resolveIntrinsics(pv, params, resources)
						merged[k] = fmt.Sprintf("%v", resolved)
					}
					return substituteVars(tmplStr, merged, resources)
				}
			}
			return substituteVars(tmplStr, params, resources)
		}
	}
	return fmt.Sprintf("%v", value)
}

// substituteVars replaces ${VarName} in a string.
func substituteVars(template string, params map[string]string, resources map[string]*ProvisionedResource) string {
	result := template
	// Find all ${...} patterns.
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start
		varName := result[start+2 : end]
		replacement := resolveRef(varName, params, resources)
		result = result[:start] + replacement + result[end+1:]
	}
	return result
}

// resolveFnGetAtt resolves Fn::GetAtt to an attribute of a provisioned resource.
func resolveFnGetAtt(value any, resources map[string]*ProvisionedResource) string {
	switch v := value.(type) {
	case []any:
		if len(v) >= 2 {
			logicalId, _ := v[0].(string)
			attrName, _ := v[1].(string)
			return resolveGetAtt(logicalId, attrName, resources)
		}
	case string:
		// "LogicalId.Attribute" shorthand
		parts := strings.SplitN(v, ".", 2)
		if len(parts) == 2 {
			return resolveGetAtt(parts[0], parts[1], resources)
		}
	}
	return ""
}

// resolveGetAtt gets an attribute from a provisioned resource.
func resolveGetAtt(logicalId, attrName string, resources map[string]*ProvisionedResource) string {
	res, ok := resources[logicalId]
	if !ok {
		return ""
	}
	if val, ok := res.Attributes[attrName]; ok {
		return val
	}
	// Fallback: return PhysicalId for "Ref"-like attributes.
	return res.PhysicalId
}

// resolveFnJoin resolves Fn::Join: [delimiter, [values...]].
func resolveFnJoin(value any, params map[string]string, resources map[string]*ProvisionedResource) string {
	arr, ok := value.([]any)
	if !ok || len(arr) < 2 {
		return ""
	}
	delimiter, _ := arr[0].(string)
	values, ok := arr[1].([]any)
	if !ok {
		return ""
	}

	parts := make([]string, 0, len(values))
	for _, v := range values {
		resolved := resolveIntrinsics(v, params, resources)
		parts = append(parts, fmt.Sprintf("%v", resolved))
	}
	return strings.Join(parts, delimiter)
}

// resolveFnSelect resolves Fn::Select: [index, [values...]].
func resolveFnSelect(value any, params map[string]string, resources map[string]*ProvisionedResource) any {
	arr, ok := value.([]any)
	if !ok || len(arr) < 2 {
		return ""
	}
	var index int
	switch idx := arr[0].(type) {
	case float64:
		index = int(idx)
	case string:
		fmt.Sscanf(idx, "%d", &index)
	}
	values, ok := arr[1].([]any)
	if !ok || index < 0 || index >= len(values) {
		return ""
	}
	return resolveIntrinsics(values[index], params, resources)
}

// ---- Dependency Graph / Topological Sort ----

// topoSort returns resource logical IDs in dependency order.
func topoSort(resources map[string]cfnFullResource) ([]string, error) {
	// Build adjacency list: edges[A] = [B, C] means A depends on B and C.
	deps := make(map[string]map[string]bool)
	for logicalId := range resources {
		deps[logicalId] = make(map[string]bool)
	}

	for logicalId, res := range resources {
		// Explicit DependsOn.
		if res.DependsOn != nil {
			var single string
			var multi []string
			if err := json.Unmarshal(res.DependsOn, &single); err == nil {
				if _, ok := resources[single]; ok {
					deps[logicalId][single] = true
				}
			} else if err := json.Unmarshal(res.DependsOn, &multi); err == nil {
				for _, dep := range multi {
					if _, ok := resources[dep]; ok {
						deps[logicalId][dep] = true
					}
				}
			}
		}

		// Implicit dependencies from Ref and Fn::GetAtt in Properties.
		findRefs(res.Properties, resources, deps[logicalId])
	}

	// Kahn's algorithm for topological sort.
	inDegree := make(map[string]int)
	for logicalId := range resources {
		inDegree[logicalId] = 0
	}
	for _, depSet := range deps {
		for dep := range depSet {
			inDegree[dep] = inDegree[dep] // ensure key exists
		}
	}
	// Reverse: for each A that depends on B, B has an outgoing edge to A.
	// inDegree[A] = number of dependencies A has.
	for logicalId, depSet := range deps {
		inDegree[logicalId] = len(depSet)
		_ = logicalId
	}

	// Find all nodes with inDegree 0.
	var queue []string
	for logicalId := range resources {
		if inDegree[logicalId] == 0 {
			queue = append(queue, logicalId)
		}
	}
	// Sort queue for deterministic ordering.
	sort.Strings(queue)

	var order []string
	for len(queue) > 0 {
		// Pop first.
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		// For each resource that depends on this node, reduce inDegree.
		for logicalId, depSet := range deps {
			if depSet[node] {
				inDegree[logicalId]--
				if inDegree[logicalId] == 0 {
					queue = append(queue, logicalId)
					sort.Strings(queue)
				}
			}
		}
	}

	if len(order) != len(resources) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return order, nil
}

// findRefs recursively scans a value tree for Ref and Fn::GetAtt references to other resources.
func findRefs(value any, resources map[string]cfnFullResource, deps map[string]bool) {
	switch v := value.(type) {
	case map[string]any:
		if ref, ok := v["Ref"]; ok {
			if refStr, ok := ref.(string); ok {
				if _, isResource := resources[refStr]; isResource {
					deps[refStr] = true
				}
			}
		}
		if getAtt, ok := v["Fn::GetAtt"]; ok {
			switch ga := getAtt.(type) {
			case []any:
				if len(ga) >= 1 {
					if logicalId, ok := ga[0].(string); ok {
						if _, isResource := resources[logicalId]; isResource {
							deps[logicalId] = true
						}
					}
				}
			case string:
				parts := strings.SplitN(ga, ".", 2)
				if len(parts) >= 1 {
					if _, isResource := resources[parts[0]]; isResource {
						deps[parts[0]] = true
					}
				}
			}
		}
		// Also check Fn::Sub for ${ResourceName} references.
		if sub, ok := v["Fn::Sub"]; ok {
			findSubRefs(sub, resources, deps)
		}
		for _, val := range v {
			findRefs(val, resources, deps)
		}
	case []any:
		for _, item := range v {
			findRefs(item, resources, deps)
		}
	}
}

// findSubRefs extracts resource references from Fn::Sub template strings.
func findSubRefs(value any, resources map[string]cfnFullResource, deps map[string]bool) {
	var tmplStr string
	switch v := value.(type) {
	case string:
		tmplStr = v
	case []any:
		if len(v) >= 1 {
			tmplStr, _ = v[0].(string)
		}
	}

	// Find all ${VarName} patterns.
	for {
		start := strings.Index(tmplStr, "${")
		if start == -1 {
			break
		}
		end := strings.Index(tmplStr[start:], "}")
		if end == -1 {
			break
		}
		end += start
		varName := tmplStr[start+2 : end]
		// Check if it references a resource.
		if _, isResource := resources[varName]; isResource {
			deps[varName] = true
		}
		tmplStr = tmplStr[end+1:]
	}
}

// ---- Response parsing helpers ----

// extractVpcIdFromResponse tries to extract a VpcId from an EC2 CreateVpc response.
func extractVpcIdFromResponse(resp *service.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	// Marshal the body to XML and search for VpcId.
	return extractXMLField(resp, "vpcId")
}

func extractSubnetIdFromResponse(resp *service.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	return extractXMLField(resp, "subnetId")
}

func extractSecurityGroupIdFromResponse(resp *service.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	return extractXMLField(resp, "groupId")
}

// extractXMLField marshals the response body to XML and searches for a field.
// This is a best-effort approach — it searches the XML string for <fieldName>value</fieldName>.
func extractXMLField(resp *service.Response, fieldName string) string {
	if resp.RawBody != nil {
		return findXMLTag(string(resp.RawBody), fieldName)
	}
	// Try marshaling Body.
	if resp.Body != nil {
		// Use JSON as a fallback for structured data.
		b, err := json.Marshal(resp.Body)
		if err == nil {
			// Search for the field in various casings.
			var data map[string]any
			if json.Unmarshal(b, &data) == nil {
				return findInMap(data, fieldName)
			}
		}
	}
	return ""
}

// findXMLTag does a simple search for <tag>value</tag> in an XML string.
func findXMLTag(xml, tag string) string {
	// Try exact case and common variants.
	for _, t := range []string{tag, strings.Title(tag)} { //nolint:staticcheck
		openTag := "<" + t + ">"
		closeTag := "</" + t + ">"
		start := strings.Index(xml, openTag)
		if start == -1 {
			continue
		}
		start += len(openTag)
		end := strings.Index(xml[start:], closeTag)
		if end == -1 {
			continue
		}
		return xml[start : start+end]
	}
	return ""
}

// findInMap recursively searches a map for a key (case-insensitive).
func findInMap(data map[string]any, key string) string {
	lowerKey := strings.ToLower(key)
	for k, v := range data {
		if strings.ToLower(k) == lowerKey {
			if s, ok := v.(string); ok {
				return s
			}
		}
		if sub, ok := v.(map[string]any); ok {
			if found := findInMap(sub, key); found != "" {
				return found
			}
		}
	}
	return ""
}

// ---- Utility helpers ----

func (p *Provisioner) fakeAuth(svcName string) string {
	return fmt.Sprintf("AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/%s/%s/aws4_request, SignedHeaders=host, Signature=abc123",
		p.region, svcName)
}

// stringProp extracts a string property from a CFN Properties map.
func stringProp(props map[string]any, key string) string {
	v, ok := props[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// stringPropDefault extracts a string property with a default fallback.
func stringPropDefault(props map[string]any, key, defaultVal string) string {
	if s := stringProp(props, key); s != "" {
		return s
	}
	return defaultVal
}

// formToMap converts url.Values to a map[string]string (first value only).
func formToMap(form url.Values) map[string]string {
	m := make(map[string]string, len(form))
	for k, vs := range form {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	return m
}

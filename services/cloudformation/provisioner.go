package cloudformation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
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

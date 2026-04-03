package cloudtrail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Protocol types for AWS services.
const (
	protoJSON  = "json"
	protoQuery = "query"
	protoREST  = "rest"
)

// serviceProtocols maps service names to their wire protocol.
var serviceProtocols = map[string]string{
	"dynamodb":       protoJSON,
	"kms":            protoJSON,
	"kinesis":        protoJSON,
	"logs":           protoJSON,
	"lambda":         protoREST,
	"sqs":            protoQuery,
	"sns":            protoQuery,
	"iam":            protoQuery,
	"sts":            protoQuery,
	"ec2":            protoQuery,
	"autoscaling":    protoQuery,
	"elasticloadbalancing": protoQuery,
	"cloudformation": protoQuery,
	"s3":             protoREST,
}

// targetPrefixes maps service names to the X-Amz-Target prefix used for JSON protocol services.
var targetPrefixes = map[string]string{
	"dynamodb": "DynamoDB_20120810",
	"kms":      "TrentService",
	"kinesis":  "Kinesis_20131202",
	"logs":     "Logs_20140328",
}

// s3MethodMap maps S3 CloudTrail event names to HTTP methods.
var s3MethodMap = map[string]string{
	"CreateBucket":      "PUT",
	"DeleteBucket":      "DELETE",
	"PutObject":         "PUT",
	"DeleteObject":      "DELETE",
	"GetObject":         "GET",
	"HeadObject":        "HEAD",
	"HeadBucket":        "HEAD",
	"ListBuckets":       "GET",
	"ListObjects":       "GET",
	"GetBucketLocation": "GET",
}

// lambdaPathMap maps Lambda CloudTrail event names to HTTP method + path patterns.
var lambdaPathMap = map[string]struct {
	method string
	path   string // %s is replaced with function name from requestParameters
}{
	"CreateFunction20150331":  {"POST", "/2015-03-31/functions"},
	"Invoke":                  {"POST", "/2015-03-31/functions/%s/invocations"},
	"DeleteFunction20150331":  {"DELETE", "/2015-03-31/functions/%s"},
	"GetFunction":             {"GET", "/2015-03-31/functions/%s"},
	"ListFunctions":           {"GET", "/2015-03-31/functions"},
	"UpdateFunctionCode20150331v2": {"PUT", "/2015-03-31/functions/%s/code"},
}

// supportedEvents lists the top events we handle. Unknown events return an error.
var supportedEvents = map[string]bool{
	// DynamoDB
	"CreateTable": true, "DeleteTable": true, "PutItem": true, "GetItem": true, "UpdateItem": true,
	// S3
	"CreateBucket": true, "DeleteBucket": true, "PutObject": true, "DeleteObject": true, "GetObject": true,
	// SQS
	"CreateQueue": true, "SendMessage": true, "DeleteQueue": true,
	// SNS
	"CreateTopic": true, "Subscribe": true, "Publish": true,
	// IAM
	"CreateRole": true, "CreateUser": true,
	// KMS
	"CreateKey": true,
	// Lambda
	"CreateFunction20150331": true, "Invoke": true,
	// CloudWatch Logs
	"CreateLogGroup": true,
	// Kinesis
	"CreateStream": true,
	// STS
	"GetCallerIdentity": true,
	// EC2
	"RunInstances": true,
	// CloudFormation
	"CreateStack": true,
}

// ConvertToRequest converts a CloudTrail event into an HTTP request suitable for
// sending to a CloudMock endpoint.
func ConvertToRequest(event CloudTrailEvent, endpoint string) (*http.Request, error) {
	svc := event.ServiceName()

	proto, ok := serviceProtocols[svc]
	if !ok {
		return nil, fmt.Errorf("unsupported service: %s", svc)
	}

	if !supportedEvents[event.EventName] {
		return nil, fmt.Errorf("unsupported event: %s.%s", svc, event.EventName)
	}

	var req *http.Request
	var err error

	switch proto {
	case protoJSON:
		req, err = buildJSONRequest(event, svc, endpoint)
	case protoQuery:
		req, err = buildQueryRequest(event, svc, endpoint)
	case protoREST:
		if svc == "s3" {
			req, err = buildS3Request(event, endpoint)
		} else if svc == "lambda" {
			req, err = buildLambdaRequest(event, endpoint)
		} else {
			return nil, fmt.Errorf("unsupported REST service: %s", svc)
		}
	}

	if err != nil {
		return nil, err
	}

	// Set fake Authorization header for gateway service detection.
	region := event.AWSRegion
	if region == "" {
		region = "us-east-1"
	}
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=test/20260101/%s/%s/aws4_request, SignedHeaders=host, Signature=fake",
		region, svc))

	// Set fast-route header for the gateway.
	req.Header.Set("X-Cloudmock-Service", svc)

	return req, nil
}

func buildJSONRequest(event CloudTrailEvent, svc, endpoint string) (*http.Request, error) {
	prefix, ok := targetPrefixes[svc]
	if !ok {
		return nil, fmt.Errorf("no target prefix for JSON service: %s", svc)
	}

	body, err := json.Marshal(event.RequestParameters)
	if err != nil {
		return nil, fmt.Errorf("marshal request parameters: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Amz-Target", prefix+"."+event.EventName)
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")

	return req, nil
}

func buildQueryRequest(event CloudTrailEvent, svc, endpoint string) (*http.Request, error) {
	form := url.Values{}
	form.Set("Action", event.EventName)

	for k, v := range event.RequestParameters {
		form.Set(k, fmt.Sprint(v))
	}

	body := form.Encode()
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

func buildS3Request(event CloudTrailEvent, endpoint string) (*http.Request, error) {
	method, ok := s3MethodMap[event.EventName]
	if !ok {
		method = "GET"
	}

	bucket, _ := event.RequestParameters["bucketName"].(string)
	key, _ := event.RequestParameters["key"].(string)

	var reqURL string
	switch {
	case bucket != "" && key != "":
		reqURL = endpoint + "/" + bucket + "/" + key
	case bucket != "":
		reqURL = endpoint + "/" + bucket
	default:
		reqURL = endpoint
	}

	var bodyReader *bytes.Reader
	if method == "PUT" && key != "" {
		// For PutObject, send a placeholder body.
		bodyReader = bytes.NewReader([]byte("cloudtrail-replay-placeholder"))
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func buildLambdaRequest(event CloudTrailEvent, endpoint string) (*http.Request, error) {
	mapping, ok := lambdaPathMap[event.EventName]
	if !ok {
		return nil, fmt.Errorf("unsupported Lambda event: %s", event.EventName)
	}

	funcName, _ := event.RequestParameters["functionName"].(string)

	var path string
	if strings.Contains(mapping.path, "%s") {
		path = fmt.Sprintf(mapping.path, funcName)
	} else {
		path = mapping.path
	}

	var body []byte
	if mapping.method == "POST" || mapping.method == "PUT" {
		var err error
		body, err = json.Marshal(event.RequestParameters)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(mapping.method, endpoint+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

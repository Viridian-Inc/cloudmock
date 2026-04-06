package sdk

import (
	"net/http"
	"testing"
)

func TestDetectServiceAction_XAmzTarget(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://dynamodb.us-east-1.amazonaws.com/", nil)
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")

	service, action := DetectServiceAction(req)
	if service != "dynamodb" {
		t.Errorf("service = %q, want dynamodb", service)
	}
	if action != "GetItem" {
		t.Errorf("action = %q, want GetItem", action)
	}
}

func TestDetectServiceAction_S3Host(t *testing.T) {
	req, _ := http.NewRequest("PUT", "https://s3.us-east-1.amazonaws.com/bucket/key", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKIA.../20260403/us-east-1/s3/aws4_request, ...")

	service, action := DetectServiceAction(req)
	if service != "s3" {
		t.Errorf("service = %q, want s3", service)
	}
	if action != "PutObject" {
		t.Errorf("action = %q, want PutObject", action)
	}
}

func TestExtractRegion(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://dynamodb.us-west-2.amazonaws.com/", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKIA.../20260403/us-west-2/dynamodb/aws4_request, ...")

	region := ExtractRegion(req)
	if region != "us-west-2" {
		t.Errorf("region = %q, want us-west-2", region)
	}
}

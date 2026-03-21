package routing

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectService_AuthorizationHeader(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantService string
	}{
		{
			name:        "s3 from credential scope",
			authHeader:  "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=abcdef",
			wantService: "s3",
		},
		{
			name:        "dynamodb from credential scope",
			authHeader:  "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/dynamodb/aws4_request, SignedHeaders=host;x-amz-date, Signature=abcdef",
			wantService: "dynamodb",
		},
		{
			name:        "sts from credential scope",
			authHeader:  "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/sts/aws4_request, SignedHeaders=host;x-amz-date, Signature=abcdef",
			wantService: "sts",
		},
		{
			name:        "empty header returns empty string",
			authHeader:  "",
			wantService: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", nil)
			if tt.authHeader != "" {
				r.Header.Set("Authorization", tt.authHeader)
			}
			got := DetectService(r)
			assert.Equal(t, tt.wantService, got)
		})
	}
}

func TestDetectService_XAmzTargetHeader(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantService string
	}{
		{
			name:        "dynamodb from X-Amz-Target",
			target:      "DynamoDB_20120810.CreateTable",
			wantService: "dynamodb",
		},
		{
			name:        "kinesis from X-Amz-Target",
			target:      "Kinesis_20131202.ListStreams",
			wantService: "kinesis",
		},
		{
			name:        "no version suffix",
			target:      "SomeService.SomeAction",
			wantService: "someservice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", nil)
			r.Header.Set("X-Amz-Target", tt.target)
			got := DetectService(r)
			assert.Equal(t, tt.wantService, got)
		})
	}
}

func TestDetectService_AuthorizationTakesPrecedence(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20130524/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	r.Header.Set("X-Amz-Target", "DynamoDB_20120810.CreateTable")
	got := DetectService(r)
	assert.Equal(t, "s3", got)
}

func TestDetectAction_XAmzTargetHeader(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantAction string
	}{
		{
			name:       "CreateTable from X-Amz-Target",
			target:     "DynamoDB_20120810.CreateTable",
			wantAction: "CreateTable",
		},
		{
			name:       "PutItem from X-Amz-Target",
			target:     "DynamoDB_20120810.PutItem",
			wantAction: "PutItem",
		},
		{
			name:       "ListStreams from kinesis target",
			target:     "Kinesis_20131202.ListStreams",
			wantAction: "ListStreams",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", nil)
			r.Header.Set("X-Amz-Target", tt.target)
			got := DetectAction(r)
			assert.Equal(t, tt.wantAction, got)
		})
	}
}

func TestDetectAction_QueryStringAction(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantAction string
	}{
		{
			name:       "DescribeInstances from query string",
			url:        "/?Action=DescribeInstances",
			wantAction: "DescribeInstances",
		},
		{
			name:       "CreateBucket from query string",
			url:        "/?Action=CreateBucket&Version=2006-03-01",
			wantAction: "CreateBucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.url, nil)
			got := DetectAction(r)
			assert.Equal(t, tt.wantAction, got)
		})
	}
}

func TestDetectAction_XAmzTargetTakesPrecedence(t *testing.T) {
	r := httptest.NewRequest("POST", "/?Action=DescribeInstances", nil)
	r.Header.Set("X-Amz-Target", "DynamoDB_20120810.CreateTable")
	got := DetectAction(r)
	assert.Equal(t, "CreateTable", got)
}

func TestDetectAction_NoHeader_NoQuery(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	got := DetectAction(r)
	assert.Equal(t, "", got)
}

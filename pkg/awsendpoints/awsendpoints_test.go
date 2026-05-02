package awsendpoints

import (
	"net/http/httptest"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		service string
		region  string
		want    string
	}{
		{"s3", "us-east-1", "s3.us-east-1.amazonaws.com"},
		{"iam", "us-east-1", "iam.amazonaws.com"},
		{"dynamodb", "eu-west-1", "dynamodb.eu-west-1.amazonaws.com"},
		{"route53", "us-east-1", "route53.amazonaws.com"},
		{"cloudfront", "any", "cloudfront.amazonaws.com"},
		{"ses", "eu-west-1", "email.eu-west-1.amazonaws.com"},
		{"cognito-idp", "us-east-1", "cognito-idp.us-east-1.amazonaws.com"},
		{"stepfunctions", "us-east-1", "states.us-east-1.amazonaws.com"},
		{"cloudwatch", "us-east-1", "monitoring.us-east-1.amazonaws.com"},
		{"unknown-service", "us-east-1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			got := Resolve(tt.service, tt.region)
			if got != tt.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", tt.service, tt.region, got, tt.want)
			}
		})
	}
}

func TestServiceFromAuth(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
	}{
		{
			name: "s3 from auth header",
			auth: "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc",
			want: "s3",
		},
		{
			name: "dynamodb from auth header",
			auth: "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc",
			want: "dynamodb",
		},
		{
			name: "sts from auth header",
			auth: "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/sts/aws4_request, SignedHeaders=host, Signature=abc",
			want: "sts",
		},
		{
			name: "uppercase service is normalized",
			auth: "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/S3/aws4_request, SignedHeaders=host, Signature=abc",
			want: "s3",
		},
		{
			name: "empty auth",
			auth: "",
			want: "",
		},
		{
			name: "non-sigv4 auth",
			auth: "Basic Zm9vOmJhcg==",
			want: "",
		},
		{
			name: "truncated credential scope",
			auth: "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1, SignedHeaders=host, Signature=abc",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			got := ServiceFromAuth(req)
			if got != tt.want {
				t.Errorf("ServiceFromAuth() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAction(t *testing.T) {
	tests := []struct {
		name   string
		target string
		query  string
		want   string
	}{
		{"x-amz-target json-rpc", "DynamoDB_20120810.GetItem", "", "GetItem"},
		{"x-amz-target nested", "Logs_20140328.CreateLogGroup", "", "CreateLogGroup"},
		{"query Action param", "", "Action=SendMessage", "SendMessage"},
		{"target wins over query", "DynamoDB_20120810.PutItem", "Action=Other", "PutItem"},
		{"neither present", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/"
			if tt.query != "" {
				url = "/?" + tt.query
			}
			req := httptest.NewRequest("POST", url, nil)
			if tt.target != "" {
				req.Header.Set("X-Amz-Target", tt.target)
			}
			got := Action(req)
			if got != tt.want {
				t.Errorf("Action() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOverride_RestoresKnownService(t *testing.T) {
	const svc = "s3"
	original := Resolve(svc, "us-east-1")
	if original == "" {
		t.Fatalf("expected %s to be a known service before the test", svc)
	}

	cleanup := Override(svc, "fake-host")
	if got := Resolve(svc, "us-east-1"); got != "fake-host" {
		t.Errorf("after Override, Resolve = %q, want %q", got, "fake-host")
	}

	cleanup()
	if got := Resolve(svc, "us-east-1"); got != original {
		t.Errorf("after cleanup, Resolve = %q, want %q", got, original)
	}
}

func TestOverride_RemovesUnknownService(t *testing.T) {
	const svc = "this-service-does-not-exist"
	if Resolve(svc, "us-east-1") != "" {
		t.Fatalf("precondition: %s should not be in the table", svc)
	}

	cleanup := Override(svc, "fake-host")
	if got := Resolve(svc, "us-east-1"); got != "fake-host" {
		t.Errorf("after Override, Resolve = %q, want %q", got, "fake-host")
	}

	cleanup()
	if got := Resolve(svc, "us-east-1"); got != "" {
		t.Errorf("after cleanup of an originally-unknown service, Resolve = %q, want \"\"", got)
	}
}

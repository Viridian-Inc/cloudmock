package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	cmschema "github.com/neureaux/cloudmock/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock cloudmock server ───────────────────────────────────────────────────

func newTestCloudmockServer() *httptest.Server {
	store := make(map[string]map[string]any)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		service := extractServiceFromAuth(auth)

		target := r.Header.Get("X-Amz-Target")
		var action string
		if target != "" {
			if dot := strings.LastIndex(target, "."); dot >= 0 {
				action = target[dot+1:]
			}
		}

		if action == "" {
			body := make([]byte, 4096)
			n, _ := r.Body.Read(body)
			bodyStr := string(body[:n])
			values, _ := url.ParseQuery(bodyStr)
			action = values.Get("Action")

			if action == "" && service == "s3" {
				handleS3(w, r, store)
				return
			}

			if action != "" {
				handleQueryAction(w, service, action, values, store)
				return
			}
		}

		if action != "" {
			var params map[string]any
			if r.Body != nil {
				bodyBytes := make([]byte, 4096)
				n, _ := r.Body.Read(bodyBytes)
				json.Unmarshal(bodyBytes[:n], &params)
			}
			if params == nil {
				params = make(map[string]any)
			}
			handleJSONAction(w, service, action, params, store)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"__type":"InvalidAction","Message":"no action detected"}`))
	}))
}

func handleS3(w http.ResponseWriter, r *http.Request, store map[string]map[string]any) {
	path := r.URL.Path
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	bucket := parts[0]
	key := fmt.Sprintf("s3/bucket/%s", bucket)

	switch r.Method {
	case "PUT":
		store[key] = map[string]any{
			"bucket": bucket,
			"arn":    fmt.Sprintf("arn:aws:s3:::%s", bucket),
			"region": "us-east-1",
		}
		w.WriteHeader(http.StatusOK)
	case "HEAD":
		if _, ok := store[key]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("X-Amz-Bucket-Region", "us-east-1")
		w.WriteHeader(http.StatusOK)
	case "DELETE":
		if _, ok := store[key]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		delete(store, key)
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleJSONAction(w http.ResponseWriter, service, action string, params map[string]any, store map[string]map[string]any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")

	switch {
	case strings.HasPrefix(action, "Create"):
		switch service {
		case "dynamodb":
			tableName, _ := params["TableName"].(string)
			if tableName != "" {
				key := fmt.Sprintf("dynamodb/table/%s", tableName)
				data := map[string]any{}
				for k, v := range params {
					data[k] = v
				}
				data["TableArn"] = fmt.Sprintf("arn:aws:dynamodb:us-east-1:000000000000:table/%s", tableName)
				data["TableStatus"] = "ACTIVE"
				store[key] = data
				json.NewEncoder(w).Encode(map[string]any{"TableDescription": data})
				return
			}
		}
		id := fmt.Sprintf("%s-%s-001", service, strings.ToLower(action[6:]))
		key := fmt.Sprintf("%s/%s/%s", service, action, id)
		data := map[string]any{}
		for k, v := range params {
			data[k] = v
		}
		store[key] = data
		json.NewEncoder(w).Encode(data)

	case strings.HasPrefix(action, "Describe"):
		switch service {
		case "dynamodb":
			tableName, _ := params["TableName"].(string)
			key := fmt.Sprintf("dynamodb/table/%s", tableName)
			if data, ok := store[key]; ok {
				json.NewEncoder(w).Encode(map[string]any{"Table": data})
			} else {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"__type":  "ResourceNotFoundException",
					"Message": fmt.Sprintf("Table not found: %s", tableName),
				})
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"__type": "ResourceNotFoundException"})

	case strings.HasPrefix(action, "Delete"):
		switch service {
		case "dynamodb":
			tableName, _ := params["TableName"].(string)
			key := fmt.Sprintf("dynamodb/table/%s", tableName)
			if data, ok := store[key]; ok {
				delete(store, key)
				json.NewEncoder(w).Encode(map[string]any{"TableDescription": data})
			} else {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"__type":  "ResourceNotFoundException",
					"Message": fmt.Sprintf("Table not found: %s", tableName),
				})
			}
			return
		}
		json.NewEncoder(w).Encode(map[string]any{})

	default:
		json.NewEncoder(w).Encode(map[string]any{})
	}
}

func handleQueryAction(w http.ResponseWriter, service, action string, values url.Values, store map[string]map[string]any) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case action == "CreateVpc":
		cidr := values.Get("CidrBlock")
		vpcID := "vpc-test001"
		key := fmt.Sprintf("ec2/vpc/%s", vpcID)
		data := map[string]any{
			"VpcId":              vpcID,
			"CidrBlock":         cidr,
			"Arn":               fmt.Sprintf("arn:aws:ec2:us-east-1:000000000000:vpc/%s", vpcID),
			"EnableDnsSupport":  true,
			"EnableDnsHostnames": false,
			"IsDefault":         false,
		}
		store[key] = data
		json.NewEncoder(w).Encode(data)

	case action == "DescribeVpcs":
		vpcID := values.Get("VpcId")
		key := fmt.Sprintf("ec2/vpc/%s", vpcID)
		if data, ok := store[key]; ok {
			json.NewEncoder(w).Encode(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"__type":  "ResourceNotFoundException",
				"Message": fmt.Sprintf("VPC not found: %s", vpcID),
			})
		}

	case action == "DeleteVpc":
		vpcID := values.Get("VpcId")
		key := fmt.Sprintf("ec2/vpc/%s", vpcID)
		if _, ok := store[key]; ok {
			delete(store, key)
			json.NewEncoder(w).Encode(map[string]any{})
		} else {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"__type":  "ResourceNotFoundException",
				"Message": fmt.Sprintf("VPC not found: %s", vpcID),
			})
		}

	default:
		json.NewEncoder(w).Encode(map[string]any{})
	}
}

func extractServiceFromAuth(auth string) string {
	idx := strings.Index(auth, "Credential=")
	if idx < 0 {
		return ""
	}
	rest := auth[idx+len("Credential="):]
	end := strings.IndexAny(rest, ", ")
	if end >= 0 {
		rest = rest[:end]
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 4 {
		return ""
	}
	return parts[3]
}

// ── S3 Bucket Tests ─────────────────────────────────────────────────────────

func s3BucketSchema() *cmschema.ResourceSchema {
	return &cmschema.ResourceSchema{
		ServiceName:   "s3",
		TerraformType: "cloudmock_s3_bucket",
		AWSType:       "AWS::S3::Bucket",
		CreateAction:  "CreateBucket",
		ReadAction:    "HeadBucket",
		DeleteAction:  "DeleteBucket",
		ImportID:      "bucket",
		Attributes: []cmschema.AttributeSchema{
			{Name: "bucket", Type: "string", Required: true, ForceNew: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "region", Type: "string", Computed: true},
		},
	}
}

func TestReconciler_CreateObserve_S3Bucket(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	r := NewReconciler(server.URL, "us-east-1", "test", "test", s3BucketSchema())

	// Create.
	id, state, err := r.Create(map[string]any{
		"bucket": "test-bucket",
	})
	require.NoError(t, err)
	assert.Equal(t, "test-bucket", id)
	assert.Equal(t, "test-bucket", state["bucket"])

	// Observe.
	exists, obsState, err := r.Observe("test-bucket")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "test-bucket", obsState["bucket"])
	assert.Equal(t, "us-east-1", obsState["region"])
	assert.Contains(t, obsState["arn"], "arn:aws:s3:::test-bucket")
}

func TestReconciler_DeleteObserve_S3Bucket(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	r := NewReconciler(server.URL, "us-east-1", "test", "test", s3BucketSchema())

	// Create.
	_, _, err := r.Create(map[string]any{"bucket": "delete-me"})
	require.NoError(t, err)

	// Delete.
	err = r.Delete("delete-me")
	require.NoError(t, err)

	// Observe after delete should return exists=false.
	exists, _, err := r.Observe("delete-me")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestReconciler_ObserveNonExistent_S3Bucket(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	r := NewReconciler(server.URL, "us-east-1", "test", "test", s3BucketSchema())

	exists, _, err := r.Observe("nonexistent-bucket")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestReconciler_DeleteNonExistent_S3Bucket(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	r := NewReconciler(server.URL, "us-east-1", "test", "test", s3BucketSchema())

	// Deleting a nonexistent resource should not error.
	err := r.Delete("nonexistent-bucket")
	require.NoError(t, err)
}

// ── EC2 VPC Tests ───────────────────────────────────────────────────────────

func ec2VPCSchema() *cmschema.ResourceSchema {
	return &cmschema.ResourceSchema{
		ServiceName:   "ec2",
		TerraformType: "cloudmock_ec2_vpc",
		AWSType:       "AWS::EC2::VPC",
		CreateAction:  "CreateVpc",
		ReadAction:    "DescribeVpcs",
		DeleteAction:  "DeleteVpc",
		ImportID:      "vpc_id",
		Attributes: []cmschema.AttributeSchema{
			{Name: "vpc_id", Type: "string", Computed: true},
			{Name: "cidr_block", Type: "string", Required: true, ForceNew: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "enable_dns_support", Type: "bool", Default: true},
			{Name: "enable_dns_hostnames", Type: "bool", Default: false},
			{Name: "is_default", Type: "bool", Computed: true},
		},
	}
}

func TestReconciler_CreateObserve_EC2VPC(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	r := NewReconciler(server.URL, "us-east-1", "test", "test", ec2VPCSchema())

	// Create.
	id, state, err := r.Create(map[string]any{
		"cidr_block": "10.0.0.0/16",
	})
	require.NoError(t, err)
	assert.Equal(t, "vpc-test001", id)
	assert.Equal(t, "10.0.0.0/16", state["cidr_block"])

	// Observe.
	exists, obsState, err := r.Observe("vpc-test001")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "vpc-test001", obsState["VpcId"])
}

func TestReconciler_DeleteObserve_EC2VPC(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	r := NewReconciler(server.URL, "us-east-1", "test", "test", ec2VPCSchema())

	// Create.
	id, _, err := r.Create(map[string]any{"cidr_block": "10.0.0.0/16"})
	require.NoError(t, err)

	// Delete.
	err = r.Delete(id)
	require.NoError(t, err)

	// Observe after delete should return exists=false.
	exists, _, err := r.Observe(id)
	require.NoError(t, err)
	assert.False(t, exists)
}

// ── DynamoDB Table Tests ────────────────────────────────────────────────────

func dynamoDBTableSchema() *cmschema.ResourceSchema {
	return &cmschema.ResourceSchema{
		ServiceName:   "dynamodb",
		TerraformType: "cloudmock_dynamodb_table",
		AWSType:       "AWS::DynamoDB::Table",
		CreateAction:  "CreateTable",
		ReadAction:    "DescribeTable",
		DeleteAction:  "DeleteTable",
		ImportID:      "table_name",
		Attributes: []cmschema.AttributeSchema{
			{Name: "table_name", Type: "string", Required: true, ForceNew: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "hash_key", Type: "string", Required: true, ForceNew: true},
			{Name: "billing_mode", Type: "string", Default: "PROVISIONED"},
		},
	}
}

func TestReconciler_CreateObserve_DynamoDBTable(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	r := NewReconciler(server.URL, "us-east-1", "test", "test", dynamoDBTableSchema())

	// Create.
	id, state, err := r.Create(map[string]any{
		"table_name": "my-table",
		"hash_key":   "id",
	})
	require.NoError(t, err)
	assert.Equal(t, "my-table", id)
	assert.Equal(t, "my-table", state["table_name"])
}

func TestReconciler_DeleteObserve_DynamoDBTable(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	r := NewReconciler(server.URL, "us-east-1", "test", "test", dynamoDBTableSchema())

	// Create.
	_, _, err := r.Create(map[string]any{
		"table_name": "delete-table",
		"hash_key":   "pk",
	})
	require.NoError(t, err)

	// Delete.
	err = r.Delete("delete-table")
	require.NoError(t, err)

	// Observe after delete should return exists=false.
	exists, _, err := r.Observe("delete-table")
	require.NoError(t, err)
	assert.False(t, exists)
}

// ── Helper Function Tests ───────────────────────────────────────────────────

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"table_name", "TableName"},
		{"vpc_id", "VpcId"},
		{"cidr_block", "CidrBlock"},
		{"arn", "Arn"},
		{"bucket", "Bucket"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, snakeToPascal(tt.input), "snakeToPascal(%q)", tt.input)
	}
}

func TestIsNotFoundError(t *testing.T) {
	assert.True(t, isNotFoundError(fmt.Errorf("resource not found")))
	assert.True(t, isNotFoundError(fmt.Errorf("ResourceNotFoundException: table foo")))
	assert.True(t, isNotFoundError(fmt.Errorf("HTTP 404: Not Found")))
	assert.False(t, isNotFoundError(fmt.Errorf("internal server error")))
	assert.False(t, isNotFoundError(nil))
}

func TestPascalCase(t *testing.T) {
	assert.Equal(t, "Bucket", pascalCase("bucket"))
	assert.Equal(t, "FancyThing", pascalCase("fancy_thing"))
	assert.Equal(t, "SecurityGroup", pascalCase("security_group"))
}

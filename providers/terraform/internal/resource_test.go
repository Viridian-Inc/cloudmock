package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	tfschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmschema "github.com/neureaux/cloudmock/pkg/schema"
)

// ── Unit Tests: Schema Building ───────────────────────────────────────────────

func TestBuildTFSchema_StringAttribute(t *testing.T) {
	rs := cmschema.ResourceSchema{
		TerraformType: "cloudmock_test_thing",
		ResourceType:  "aws_test_thing",
		Attributes: []cmschema.AttributeSchema{
			{Name: "name", Type: "string", Required: true},
		},
	}

	tfRes := buildResource(rs)
	s, ok := tfRes.Schema["name"]
	require.True(t, ok)
	assert.Equal(t, tfschema.TypeString, s.Type)
	assert.True(t, s.Required)
	assert.False(t, s.Computed)
	assert.False(t, s.Optional)
}

func TestBuildTFSchema_ComputedAttribute(t *testing.T) {
	rs := cmschema.ResourceSchema{
		TerraformType: "cloudmock_test_thing",
		ResourceType:  "aws_test_thing",
		Attributes: []cmschema.AttributeSchema{
			{Name: "arn", Type: "string", Computed: true},
		},
	}

	tfRes := buildResource(rs)
	s, ok := tfRes.Schema["arn"]
	require.True(t, ok)
	assert.True(t, s.Computed)
	assert.False(t, s.Required)
}

func TestBuildTFSchema_ForceNew(t *testing.T) {
	rs := cmschema.ResourceSchema{
		TerraformType: "cloudmock_test_thing",
		ResourceType:  "aws_test_thing",
		Attributes: []cmschema.AttributeSchema{
			{Name: "bucket", Type: "string", Required: true, ForceNew: true},
		},
	}

	tfRes := buildResource(rs)
	s := tfRes.Schema["bucket"]
	assert.True(t, s.ForceNew)
	assert.True(t, s.Required)
}

func TestBuildTFSchema_AllTypes(t *testing.T) {
	rs := cmschema.ResourceSchema{
		TerraformType: "cloudmock_test_thing",
		ResourceType:  "aws_test_thing",
		Attributes: []cmschema.AttributeSchema{
			{Name: "name", Type: "string"},
			{Name: "count", Type: "int"},
			{Name: "enabled", Type: "bool"},
			{Name: "ratio", Type: "float"},
			{Name: "items", Type: "list"},
			{Name: "rules", Type: "set"},
			{Name: "tags", Type: "map"},
		},
	}

	tfRes := buildResource(rs)
	assert.Equal(t, tfschema.TypeString, tfRes.Schema["name"].Type)
	assert.Equal(t, tfschema.TypeInt, tfRes.Schema["count"].Type)
	assert.Equal(t, tfschema.TypeBool, tfRes.Schema["enabled"].Type)
	assert.Equal(t, tfschema.TypeFloat, tfRes.Schema["ratio"].Type)
	assert.Equal(t, tfschema.TypeList, tfRes.Schema["items"].Type)
	assert.Equal(t, tfschema.TypeSet, tfRes.Schema["rules"].Type)
	assert.Equal(t, tfschema.TypeMap, tfRes.Schema["tags"].Type)
}

func TestBuildTFSchema_DefaultValue(t *testing.T) {
	rs := cmschema.ResourceSchema{
		TerraformType: "cloudmock_test_thing",
		ResourceType:  "aws_test_thing",
		Attributes: []cmschema.AttributeSchema{
			{Name: "mode", Type: "string", Default: "standard"},
		},
	}

	tfRes := buildResource(rs)
	assert.Equal(t, "standard", tfRes.Schema["mode"].Default)
}

func TestBuildTFSchema_MapWithElem(t *testing.T) {
	rs := cmschema.ResourceSchema{
		TerraformType: "cloudmock_test_thing",
		ResourceType:  "aws_test_thing",
		Attributes: []cmschema.AttributeSchema{
			{Name: "tags", Type: "map"},
		},
	}

	tfRes := buildResource(rs)
	s := tfRes.Schema["tags"]
	assert.NotNil(t, s.Elem)
	elem, ok := s.Elem.(*tfschema.Schema)
	require.True(t, ok)
	assert.Equal(t, tfschema.TypeString, elem.Type)
}

func TestBuildTFSchema_S3BucketFullSchema(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "s3",
		TerraformType: "cloudmock_s3_bucket",
		ResourceType:  "aws_s3_bucket",
		CreateAction:  "CreateBucket",
		ReadAction:    "HeadBucket",
		DeleteAction:  "DeleteBucket",
		ImportID:      "bucket",
		Attributes: []cmschema.AttributeSchema{
			{Name: "bucket", Type: "string", Required: true, ForceNew: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "region", Type: "string", Computed: true},
			{Name: "acl", Type: "string", Default: "private"},
			{Name: "tags", Type: "map"},
		},
	}

	tfRes := buildResource(rs)
	require.NotNil(t, tfRes)
	assert.NotNil(t, tfRes.CreateContext)
	assert.NotNil(t, tfRes.ReadContext)
	assert.NotNil(t, tfRes.DeleteContext)
	assert.Nil(t, tfRes.UpdateContext) // S3 bucket has no UpdateAction

	// Verify schema attributes
	assert.True(t, tfRes.Schema["bucket"].Required)
	assert.True(t, tfRes.Schema["bucket"].ForceNew)
	assert.True(t, tfRes.Schema["arn"].Computed)
	assert.True(t, tfRes.Schema["region"].Computed)
	assert.Equal(t, "private", tfRes.Schema["acl"].Default)
	assert.Equal(t, tfschema.TypeMap, tfRes.Schema["tags"].Type)
}

func TestBuildTFSchema_DynamoDBTableFullSchema(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "dynamodb",
		TerraformType: "cloudmock_dynamodb_table",
		ResourceType:  "aws_dynamodb_table",
		CreateAction:  "CreateTable",
		ReadAction:    "DescribeTable",
		DeleteAction:  "DeleteTable",
		ImportID:      "table_name",
		Attributes: []cmschema.AttributeSchema{
			{Name: "table_name", Type: "string", Required: true, ForceNew: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "billing_mode", Type: "string", Default: "PROVISIONED"},
			{Name: "read_capacity", Type: "int"},
			{Name: "write_capacity", Type: "int"},
			{Name: "hash_key", Type: "string", Required: true, ForceNew: true},
			{Name: "range_key", Type: "string", ForceNew: true},
			{Name: "attribute", Type: "set", Required: true},
			{Name: "tags", Type: "map"},
		},
	}

	tfRes := buildResource(rs)
	require.NotNil(t, tfRes)

	assert.True(t, tfRes.Schema["table_name"].Required)
	assert.True(t, tfRes.Schema["table_name"].ForceNew)
	assert.True(t, tfRes.Schema["hash_key"].Required)
	assert.Equal(t, tfschema.TypeInt, tfRes.Schema["read_capacity"].Type)
	assert.Equal(t, "PROVISIONED", tfRes.Schema["billing_mode"].Default)
}

func TestBuildTFSchema_EC2VPCFullSchema(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "ec2",
		TerraformType: "cloudmock_ec2_vpc",
		ResourceType:  "aws_vpc",
		CreateAction:  "CreateVpc",
		ReadAction:    "DescribeVpcs",
		DeleteAction:  "DeleteVpc",
		UpdateAction:  "ModifyVpcAttribute",
		ImportID:      "vpc_id",
		Attributes: []cmschema.AttributeSchema{
			{Name: "vpc_id", Type: "string", Computed: true},
			{Name: "cidr_block", Type: "string", Required: true, ForceNew: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "enable_dns_support", Type: "bool", Default: true},
			{Name: "enable_dns_hostnames", Type: "bool", Default: false},
			{Name: "is_default", Type: "bool", Computed: true},
			{Name: "tags", Type: "map"},
		},
	}

	tfRes := buildResource(rs)
	require.NotNil(t, tfRes)
	assert.NotNil(t, tfRes.UpdateContext) // VPC has update action

	assert.True(t, tfRes.Schema["vpc_id"].Computed)
	assert.True(t, tfRes.Schema["cidr_block"].Required)
	assert.True(t, tfRes.Schema["cidr_block"].ForceNew)
	assert.Equal(t, true, tfRes.Schema["enable_dns_support"].Default)
	assert.Equal(t, false, tfRes.Schema["enable_dns_hostnames"].Default)
}

// ── Unit Tests: Helper Functions ──────────────────────────────────────────────

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"table_name", "TableName"},
		{"vpc_id", "VpcId"},
		{"cidr_block", "CidrBlock"},
		{"arn", "Arn"},
		{"enable_dns_support", "EnableDnsSupport"},
		{"bucket", "Bucket"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, snakeToPascal(tt.input), "snakeToPascal(%q)", tt.input)
	}
}

func TestIsNotFoundError(t *testing.T) {
	assert.True(t, isNotFoundError(fmt.Errorf("resource not found")))
	assert.True(t, isNotFoundError(fmt.Errorf("ResourceNotFoundException: table foo")))
	assert.True(t, isNotFoundError(fmt.Errorf("NoSuchBucket: test-bucket")))
	assert.True(t, isNotFoundError(fmt.Errorf("HTTP 404: Not Found")))
	assert.False(t, isNotFoundError(fmt.Errorf("internal server error")))
	assert.False(t, isNotFoundError(nil))
}

func TestMapAttrType(t *testing.T) {
	assert.Equal(t, tfschema.TypeString, mapAttrType("string"))
	assert.Equal(t, tfschema.TypeInt, mapAttrType("int"))
	assert.Equal(t, tfschema.TypeBool, mapAttrType("bool"))
	assert.Equal(t, tfschema.TypeFloat, mapAttrType("float"))
	assert.Equal(t, tfschema.TypeList, mapAttrType("list"))
	assert.Equal(t, tfschema.TypeSet, mapAttrType("set"))
	assert.Equal(t, tfschema.TypeMap, mapAttrType("map"))
	assert.Equal(t, tfschema.TypeString, mapAttrType("unknown"))
}

// ── Unit Tests: Provider ──────────────────────────────────────────────────────

func TestProviderSchema(t *testing.T) {
	p := providerSchema()
	assert.Contains(t, p, "endpoint")
	assert.Contains(t, p, "region")
	assert.Contains(t, p, "access_key")
	assert.Contains(t, p, "secret_key")
	assert.True(t, p["secret_key"].Sensitive)
}

func TestProviderHasResources(t *testing.T) {
	p := Provider()
	// Should have at least the S3, DynamoDB, and EC2 resources from Tier 1 schemas.
	assert.Contains(t, p.ResourcesMap, "cloudmock_s3_bucket")
	assert.Contains(t, p.ResourcesMap, "cloudmock_dynamodb_table")
	assert.Contains(t, p.ResourcesMap, "cloudmock_ec2_vpc")
}

func TestProviderFromSchemas(t *testing.T) {
	schemas := []cmschema.ResourceSchema{
		{
			TerraformType: "cloudmock_test_widget",
			ResourceType:  "aws_test_widget",
			ServiceName:   "test",
			CreateAction:  "CreateWidget",
			DeleteAction:  "DeleteWidget",
			Attributes: []cmschema.AttributeSchema{
				{Name: "name", Type: "string", Required: true},
			},
		},
	}
	p := ProviderFromSchemas(schemas)
	assert.Contains(t, p.ResourcesMap, "cloudmock_test_widget")
	assert.Len(t, p.ResourcesMap, 1)
}

func TestRegistryResourceCount(t *testing.T) {
	count := RegistryResourceCount()
	// We expect at least S3 bucket, DynamoDB table, EC2 VPC, subnet, security group, instance.
	assert.GreaterOrEqual(t, count, 3)
}

// ── Unit Tests: API Client ────────────────────────────────────────────────────

func TestAPIClientSetAuthHeader(t *testing.T) {
	client := NewAPIClient("http://localhost:4566", "us-east-1", "AKID", "secret")
	req, _ := http.NewRequest("GET", "http://localhost:4566/", nil)
	client.setAuthHeader(req, "s3")

	auth := req.Header.Get("Authorization")
	assert.Contains(t, auth, "AWS4-HMAC-SHA256")
	assert.Contains(t, auth, "AKID")
	assert.Contains(t, auth, "us-east-1")
	assert.Contains(t, auth, "s3")
	assert.Contains(t, auth, "aws4_request")
}

func TestAPIClientDoJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/x-amz-json-1.1", r.Header.Get("Content-Type"))
		assert.Equal(t, "TestService.TestAction", r.Header.Get("X-Amz-Target"))
		assert.Contains(t, r.Header.Get("Authorization"), "testservice")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"Id": "test-123"})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "us-east-1", "test", "test")
	result, err := client.DoJSON("testservice", "TestService", "TestAction", map[string]interface{}{"Name": "foo"})
	require.NoError(t, err)
	assert.Equal(t, "test-123", result["Id"])
}

func TestAPIClientDoQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		values, _ := url.ParseQuery(string(body))
		assert.Equal(t, "CreateVpc", values.Get("Action"))
		assert.Equal(t, "10.0.0.0/16", values.Get("CidrBlock"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"VpcId": "vpc-123"})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "us-east-1", "test", "test")
	result, err := client.DoQuery("ec2", "CreateVpc", map[string]string{"CidrBlock": "10.0.0.0/16"})
	require.NoError(t, err)
	assert.Equal(t, "vpc-123", result["VpcId"])
}

func TestAPIClientDoRESTRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test-bucket", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "us-east-1", "test", "test")
	resp, err := client.DoRESTRaw("s3", "PUT", "/test-bucket", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAPIClientErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"__type":"ResourceNotFoundException","Message":"Table not found"}`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "us-east-1", "test", "test")
	_, err := client.DoJSON("dynamodb", "DynamoDB_20120810", "DescribeTable", map[string]interface{}{"TableName": "missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// ── Acceptance Tests ──────────────────────────────────────────────────────────
//
// These tests run against a mock cloudmock server started in-process.
// They verify the full Terraform resource lifecycle: Create, Read, Delete.

func newTestCloudmockServer() *httptest.Server {
	// In-memory store for resources, keyed by service/resourceType/id.
	store := make(map[string]map[string]interface{})

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Detect service from Authorization header.
		auth := r.Header.Get("Authorization")
		service := extractServiceFromAuth(auth)

		// Detect action.
		target := r.Header.Get("X-Amz-Target")
		var action string
		if target != "" {
			if dot := strings.LastIndex(target, "."); dot >= 0 {
				action = target[dot+1:]
			}
		}

		// For query protocol, read action from form body.
		if action == "" {
			body := make([]byte, 4096)
			n, _ := r.Body.Read(body)
			bodyStr := string(body[:n])
			values, _ := url.ParseQuery(bodyStr)
			action = values.Get("Action")

			// For REST protocol S3 operations.
			if action == "" && service == "s3" {
				handleS3(w, r, store)
				return
			}

			// Process query-protocol request with parsed values.
			if action != "" {
				handleQueryAction(w, service, action, values, store)
				return
			}
		}

		// Process JSON-protocol request.
		if action != "" {
			var params map[string]interface{}
			if r.Body != nil {
				// Re-read body for JSON parsing.
				bodyBytes := make([]byte, 4096)
				n, _ := r.Body.Read(bodyBytes)
				json.Unmarshal(bodyBytes[:n], &params)
			}
			if params == nil {
				params = make(map[string]interface{})
			}
			handleJSONAction(w, service, action, params, store)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"__type":"InvalidAction","Message":"no action detected"}`))
	}))
}

func handleS3(w http.ResponseWriter, r *http.Request, store map[string]map[string]interface{}) {
	path := r.URL.Path
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	bucket := parts[0]
	key := fmt.Sprintf("s3/bucket/%s", bucket)

	switch r.Method {
	case "PUT":
		store[key] = map[string]interface{}{
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
	case "GET":
		if path == "/" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"Buckets": []interface{}{}})
			return
		}
		if data, ok := store[key]; ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func handleJSONAction(w http.ResponseWriter, service, action string, params map[string]interface{}, store map[string]map[string]interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")

	switch {
	case strings.HasPrefix(action, "Create"):
		id := fmt.Sprintf("%s-%s-001", service, strings.ToLower(action[6:]))
		key := fmt.Sprintf("%s/%s/%s", service, action, id)
		data := map[string]interface{}{}
		for k, v := range params {
			data[k] = v
		}
		// Generate ID based on service conventions.
		switch service {
		case "dynamodb":
			tableName, _ := params["TableName"].(string)
			if tableName != "" {
				key = fmt.Sprintf("dynamodb/table/%s", tableName)
				data["TableName"] = tableName
				data["TableArn"] = fmt.Sprintf("arn:aws:dynamodb:us-east-1:000000000000:table/%s", tableName)
				data["TableStatus"] = "ACTIVE"
			}
			store[key] = data
			json.NewEncoder(w).Encode(map[string]interface{}{"TableDescription": data})
			return
		}
		store[key] = data
		json.NewEncoder(w).Encode(data)

	case strings.HasPrefix(action, "Describe"):
		switch service {
		case "dynamodb":
			tableName, _ := params["TableName"].(string)
			key := fmt.Sprintf("dynamodb/table/%s", tableName)
			if data, ok := store[key]; ok {
				json.NewEncoder(w).Encode(map[string]interface{}{"Table": data})
			} else {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"__type":  "ResourceNotFoundException",
					"Message": fmt.Sprintf("Table not found: %s", tableName),
				})
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"__type": "ResourceNotFoundException"})

	case strings.HasPrefix(action, "Delete"):
		switch service {
		case "dynamodb":
			tableName, _ := params["TableName"].(string)
			key := fmt.Sprintf("dynamodb/table/%s", tableName)
			if data, ok := store[key]; ok {
				delete(store, key)
				json.NewEncoder(w).Encode(map[string]interface{}{"TableDescription": data})
			} else {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"__type":  "ResourceNotFoundException",
					"Message": fmt.Sprintf("Table not found: %s", tableName),
				})
			}
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{})

	default:
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}
}

func handleQueryAction(w http.ResponseWriter, service, action string, values url.Values, store map[string]map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case action == "CreateVpc":
		cidr := values.Get("CidrBlock")
		vpcID := "vpc-test001"
		key := fmt.Sprintf("ec2/vpc/%s", vpcID)
		data := map[string]interface{}{
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
			json.NewEncoder(w).Encode(map[string]interface{}{
				"__type":  "ResourceNotFoundException",
				"Message": fmt.Sprintf("VPC not found: %s", vpcID),
			})
		}

	case action == "DeleteVpc":
		vpcID := values.Get("VpcId")
		key := fmt.Sprintf("ec2/vpc/%s", vpcID)
		if _, ok := store[key]; ok {
			delete(store, key)
			json.NewEncoder(w).Encode(map[string]interface{}{})
		} else {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"__type":  "ResourceNotFoundException",
				"Message": fmt.Sprintf("VPC not found: %s", vpcID),
			})
		}

	default:
		json.NewEncoder(w).Encode(map[string]interface{}{})
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

// testAccProvider creates a configured test provider.
func testAccProvider(endpoint string) map[string]*tfschema.Provider {
	p := Provider()
	return map[string]*tfschema.Provider{
		"cloudmock": p,
	}
}

func testAccProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "cloudmock" {
  endpoint   = %q
  region     = "us-east-1"
  access_key = "test"
  secret_key = "test"
}
`, endpoint)
}

func TestAccS3Bucket(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	providers := testAccProvider(server.URL)

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: map[string]func() (*tfschema.Provider, error){
			"cloudmock": func() (*tfschema.Provider, error) {
				return providers["cloudmock"], nil
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(server.URL) + `
resource "cloudmock_s3_bucket" "test" {
  bucket = "test-bucket"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudmock_s3_bucket.test", "bucket", "test-bucket"),
					resource.TestCheckResourceAttr("cloudmock_s3_bucket.test", "id", "test-bucket"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["cloudmock_s3_bucket.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						if rs.Primary.ID == "" {
							return fmt.Errorf("no ID set")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccDynamoDBTable(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	providers := testAccProvider(server.URL)

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: map[string]func() (*tfschema.Provider, error){
			"cloudmock": func() (*tfschema.Provider, error) {
				return providers["cloudmock"], nil
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(server.URL) + `
resource "cloudmock_dynamodb_table" "test" {
  table_name   = "test-table"
  hash_key     = "id"
  billing_mode = "PAY_PER_REQUEST"
  attribute    = ["id"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudmock_dynamodb_table.test", "table_name", "test-table"),
					resource.TestCheckResourceAttr("cloudmock_dynamodb_table.test", "hash_key", "id"),
					resource.TestCheckResourceAttr("cloudmock_dynamodb_table.test", "id", "test-table"),
				),
			},
		},
	})
}

func TestAccEC2VPC(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	providers := testAccProvider(server.URL)

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: map[string]func() (*tfschema.Provider, error){
			"cloudmock": func() (*tfschema.Provider, error) {
				return providers["cloudmock"], nil
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(server.URL) + `
resource "cloudmock_ec2_vpc" "test" {
  cidr_block = "10.0.0.0/16"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudmock_ec2_vpc.test", "cidr_block", "10.0.0.0/16"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["cloudmock_ec2_vpc.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						if rs.Primary.ID == "" {
							return fmt.Errorf("no ID set")
						}
						return nil
					},
				),
			},
		},
	})
}

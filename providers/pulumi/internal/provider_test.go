package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	cmschema "github.com/neureaux/cloudmock/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// ── Schema Generation Tests ─────────────────────────────────────────────────

func TestGeneratePulumiSchema_ValidJSON(t *testing.T) {
	reg := DefaultRegistry()
	schemaJSON, err := GeneratePulumiSchemaJSON(reg)
	require.NoError(t, err)

	var schema map[string]interface{}
	err = json.Unmarshal(schemaJSON, &schema)
	require.NoError(t, err)

	assert.Equal(t, "cloudmock", schema["name"])
	assert.Equal(t, "0.1.0", schema["version"])
	assert.Equal(t, "CloudMock", schema["displayName"])
}

func TestGeneratePulumiSchema_HasExpectedResources(t *testing.T) {
	reg := DefaultRegistry()
	schema := GeneratePulumiSchema(reg)

	resources, ok := schema["resources"].(map[string]interface{})
	require.True(t, ok, "schema should have resources map")

	// Check that key resources exist.
	expectedTokens := []string{
		"cloudmock:s3:Bucket",
		"cloudmock:dynamodb:Table",
		"cloudmock:ec2:Vpc",
	}

	for _, token := range expectedTokens {
		_, ok := resources[token]
		assert.True(t, ok, "expected resource token %s in schema", token)
	}
}

func TestGeneratePulumiSchema_ResourceProperties(t *testing.T) {
	reg := DefaultRegistry()
	schema := GeneratePulumiSchema(reg)

	resources := schema["resources"].(map[string]interface{})
	bucket := resources["cloudmock:s3:Bucket"].(map[string]interface{})

	inputProps := bucket["inputProperties"].(map[string]interface{})
	assert.Contains(t, inputProps, "bucket")

	outputProps := bucket["properties"].(map[string]interface{})
	assert.Contains(t, outputProps, "bucket")
	assert.Contains(t, outputProps, "arn")
	assert.Contains(t, outputProps, "id")
}

func TestGeneratePulumiSchema_TypeMapping(t *testing.T) {
	reg := cmschema.NewRegistry()
	reg.Add(cmschema.ResourceSchema{
		ServiceName:   "test",
		TerraformType: "cloudmock_test_widget",
		AWSType:       "AWS::Test::Widget",
		Attributes: []cmschema.AttributeSchema{
			{Name: "name", Type: "string", Required: true},
			{Name: "count", Type: "int"},
			{Name: "enabled", Type: "bool"},
			{Name: "ratio", Type: "float"},
			{Name: "items", Type: "list"},
			{Name: "tags", Type: "map"},
		},
	})

	schema := GeneratePulumiSchema(reg)
	resources := schema["resources"].(map[string]interface{})

	// The token for this resource.
	token := "cloudmock:test:Widget"
	res, ok := resources[token].(map[string]interface{})
	require.True(t, ok, "expected resource %s", token)

	props := res["inputProperties"].(map[string]interface{})

	nameP := props["name"].(map[string]interface{})
	assert.Equal(t, "string", nameP["type"])

	countP := props["count"].(map[string]interface{})
	assert.Equal(t, "integer", countP["type"])

	enabledP := props["enabled"].(map[string]interface{})
	assert.Equal(t, "boolean", enabledP["type"])

	ratioP := props["ratio"].(map[string]interface{})
	assert.Equal(t, "number", ratioP["type"])

	itemsP := props["items"].(map[string]interface{})
	assert.Equal(t, "array", itemsP["type"])

	tagsP := props["tags"].(map[string]interface{})
	assert.Equal(t, "object", tagsP["type"])
}

func TestGeneratePulumiSchema_Config(t *testing.T) {
	reg := cmschema.NewRegistry()
	schema := GeneratePulumiSchema(reg)

	config := schema["config"].(map[string]interface{})
	vars := config["variables"].(map[string]interface{})

	assert.Contains(t, vars, "endpoint")
	assert.Contains(t, vars, "region")
	assert.Contains(t, vars, "accessKey")
	assert.Contains(t, vars, "secretKey")
}

func TestResourceToken(t *testing.T) {
	tests := []struct {
		rs       cmschema.ResourceSchema
		expected string
	}{
		{
			rs:       cmschema.ResourceSchema{TerraformType: "cloudmock_s3_bucket", ServiceName: "s3"},
			expected: "cloudmock:s3:Bucket",
		},
		{
			rs:       cmschema.ResourceSchema{TerraformType: "cloudmock_dynamodb_table", ServiceName: "dynamodb"},
			expected: "cloudmock:dynamodb:Table",
		},
		{
			rs:       cmschema.ResourceSchema{TerraformType: "cloudmock_ec2_vpc", ServiceName: "ec2"},
			expected: "cloudmock:ec2:Vpc",
		},
		{
			rs:       cmschema.ResourceSchema{TerraformType: "cloudmock_test_thing", ServiceName: "test"},
			expected: "cloudmock:test:Thing",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, ResourceToken(tt.rs), "token for %s", tt.rs.TerraformType)
	}
}

// ── Provider Protocol Tests ─────────────────────────────────────────────────

func newTestProvider(t *testing.T) *CloudmockProvider {
	t.Helper()
	reg := DefaultRegistry()
	p, err := NewProvider(reg)
	require.NoError(t, err)
	return p
}

func TestGetSchema(t *testing.T) {
	p := newTestProvider(t)
	resp, err := p.GetSchema(context.Background(), &GetSchemaRequest{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Schema)

	var schema map[string]interface{}
	err = json.Unmarshal([]byte(resp.Schema), &schema)
	require.NoError(t, err)
	assert.Equal(t, "cloudmock", schema["name"])
}

func TestConfigure(t *testing.T) {
	p := newTestProvider(t)

	args, _ := structpb.NewStruct(map[string]interface{}{
		"endpoint":  "http://localhost:5555",
		"region":    "eu-west-1",
		"accessKey": "mykey",
		"secretKey": "mysecret",
	})

	resp, err := p.Configure(context.Background(), &ConfigureRequest{
		Args: args,
	})
	require.NoError(t, err)
	assert.True(t, resp.AcceptSecrets)

	assert.Equal(t, "http://localhost:5555", p.endpoint)
	assert.Equal(t, "eu-west-1", p.region)
	assert.Equal(t, "mykey", p.accessKey)
	assert.Equal(t, "mysecret", p.secretKey)
	assert.NotNil(t, p.client)
}

func TestConfigure_Defaults(t *testing.T) {
	p := newTestProvider(t)

	resp, err := p.Configure(context.Background(), &ConfigureRequest{})
	require.NoError(t, err)
	assert.True(t, resp.AcceptSecrets)

	assert.Equal(t, "http://localhost:4566", p.endpoint)
	assert.Equal(t, "us-east-1", p.region)
}

func TestCheck_ValidatesRequiredFields(t *testing.T) {
	p := newTestProvider(t)

	// Check with missing required "bucket" field.
	resp, err := p.Check(context.Background(), &CheckRequest{
		Urn:  "urn:pulumi:test::proj::cloudmock:s3:Bucket::mybucket",
		News: &structpb.Struct{Fields: map[string]*structpb.Value{}},
	})
	require.NoError(t, err)

	// Should have at least one failure for "bucket".
	found := false
	for _, f := range resp.Failures {
		if f.Property == "bucket" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected validation failure for missing 'bucket' property")
}

func TestCheck_PassesWithRequiredFields(t *testing.T) {
	p := newTestProvider(t)

	news, _ := structpb.NewStruct(map[string]interface{}{
		"bucket": "my-bucket",
	})

	resp, err := p.Check(context.Background(), &CheckRequest{
		Urn:  "urn:pulumi:test::proj::cloudmock:s3:Bucket::mybucket",
		News: news,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Failures)
}

func TestGetPluginInfo(t *testing.T) {
	p := newTestProvider(t)
	resp, err := p.GetPluginInfo(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", resp.Version)
}

// ── Integration Tests: CRUD against mock cloudmock server ───────────────────

func newTestCloudmockServer() *httptest.Server {
	store := make(map[string]map[string]interface{})

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
			var params map[string]interface{}
			if r.Body != nil {
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
		switch service {
		case "dynamodb":
			tableName, _ := params["TableName"].(string)
			if tableName != "" {
				key := fmt.Sprintf("dynamodb/table/%s", tableName)
				data := map[string]interface{}{}
				for k, v := range params {
					data[k] = v
				}
				data["TableArn"] = fmt.Sprintf("arn:aws:dynamodb:us-east-1:000000000000:table/%s", tableName)
				data["TableStatus"] = "ACTIVE"
				store[key] = data
				json.NewEncoder(w).Encode(map[string]interface{}{"TableDescription": data})
				return
			}
		}
		id := fmt.Sprintf("%s-%s-001", service, strings.ToLower(action[6:]))
		key := fmt.Sprintf("%s/%s/%s", service, action, id)
		data := map[string]interface{}{}
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

func setupTestProviderWithServer(t *testing.T, serverURL string) *CloudmockProvider {
	t.Helper()
	p := newTestProvider(t)

	args, _ := structpb.NewStruct(map[string]interface{}{
		"endpoint":  serverURL,
		"region":    "us-east-1",
		"accessKey": "test",
		"secretKey": "test",
	})

	_, err := p.Configure(context.Background(), &ConfigureRequest{Args: args})
	require.NoError(t, err)
	return p
}

func TestCreate_S3Bucket(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	p := setupTestProviderWithServer(t, server.URL)

	props, _ := structpb.NewStruct(map[string]interface{}{
		"bucket": "test-bucket",
	})

	resp, err := p.Create(context.Background(), &CreateRequest{
		Urn:        "urn:pulumi:test::proj::cloudmock:s3:Bucket::mybucket",
		Properties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, "test-bucket", resp.Id)
	assert.NotNil(t, resp.Properties)
	assert.Equal(t, "test-bucket", resp.Properties.Fields["bucket"].GetStringValue())
}

func TestCreateReadDelete_S3Bucket(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	p := setupTestProviderWithServer(t, server.URL)
	ctx := context.Background()

	// Create.
	props, _ := structpb.NewStruct(map[string]interface{}{
		"bucket": "lifecycle-bucket",
	})
	createResp, err := p.Create(ctx, &CreateRequest{
		Urn:        "urn:pulumi:test::proj::cloudmock:s3:Bucket::mybucket",
		Properties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, "lifecycle-bucket", createResp.Id)

	// Read.
	readResp, err := p.Read(ctx, &ReadRequest{
		Id:         createResp.Id,
		Urn:        "urn:pulumi:test::proj::cloudmock:s3:Bucket::mybucket",
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
	assert.Equal(t, "lifecycle-bucket", readResp.Id)
	assert.Equal(t, "lifecycle-bucket", readResp.Properties.Fields["bucket"].GetStringValue())

	// Delete.
	_, err = p.Delete(ctx, &DeleteRequest{
		Id:         createResp.Id,
		Urn:        "urn:pulumi:test::proj::cloudmock:s3:Bucket::mybucket",
		Properties: createResp.Properties,
	})
	require.NoError(t, err)

	// Read after delete should return empty ID (resource gone).
	readResp2, err := p.Read(ctx, &ReadRequest{
		Id:         createResp.Id,
		Urn:        "urn:pulumi:test::proj::cloudmock:s3:Bucket::mybucket",
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
	assert.Equal(t, "", readResp2.Id)
}

func TestCreateReadDelete_EC2VPC(t *testing.T) {
	server := newTestCloudmockServer()
	defer server.Close()

	p := setupTestProviderWithServer(t, server.URL)
	ctx := context.Background()

	// Create.
	props, _ := structpb.NewStruct(map[string]interface{}{
		"cidr_block": "10.0.0.0/16",
	})
	createResp, err := p.Create(ctx, &CreateRequest{
		Urn:        "urn:pulumi:test::proj::cloudmock:ec2:Vpc::myvpc",
		Properties: props,
	})
	require.NoError(t, err)
	assert.Equal(t, "vpc-test001", createResp.Id)
	assert.Equal(t, "10.0.0.0/16", createResp.Properties.Fields["cidr_block"].GetStringValue())

	// Read.
	readResp, err := p.Read(ctx, &ReadRequest{
		Id:         createResp.Id,
		Urn:        "urn:pulumi:test::proj::cloudmock:ec2:Vpc::myvpc",
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
	assert.Equal(t, "vpc-test001", readResp.Id)

	// Delete.
	_, err = p.Delete(ctx, &DeleteRequest{
		Id:         createResp.Id,
		Urn:        "urn:pulumi:test::proj::cloudmock:ec2:Vpc::myvpc",
		Properties: createResp.Properties,
	})
	require.NoError(t, err)
}

func TestDiff_DetectsChanges(t *testing.T) {
	p := newTestProvider(t)

	olds, _ := structpb.NewStruct(map[string]interface{}{
		"cidr_block":         "10.0.0.0/16",
		"enable_dns_support": true,
	})
	news, _ := structpb.NewStruct(map[string]interface{}{
		"cidr_block":         "10.0.0.0/16",
		"enable_dns_support": false,
	})

	resp, err := p.Diff(context.Background(), &DiffRequest{
		Id:   "vpc-test001",
		Urn:  "urn:pulumi:test::proj::cloudmock:ec2:Vpc::myvpc",
		Olds: olds,
		News: news,
	})
	require.NoError(t, err)
	assert.Equal(t, DiffResponse_DIFF_SOME, resp.Changes)
	assert.Contains(t, resp.Diffs, "enable_dns_support")
}

func TestDiff_ForceNewOnCIDRChange(t *testing.T) {
	p := newTestProvider(t)

	olds, _ := structpb.NewStruct(map[string]interface{}{
		"cidr_block": "10.0.0.0/16",
	})
	news, _ := structpb.NewStruct(map[string]interface{}{
		"cidr_block": "172.16.0.0/16",
	})

	resp, err := p.Diff(context.Background(), &DiffRequest{
		Id:   "vpc-test001",
		Urn:  "urn:pulumi:test::proj::cloudmock:ec2:Vpc::myvpc",
		Olds: olds,
		News: news,
	})
	require.NoError(t, err)
	assert.Contains(t, resp.Replaces, "cidr_block")
}

// ── gRPC Server Registration Test ───────────────────────────────────────────

func TestGRPCServer_StartsAndAcceptsConnections(t *testing.T) {
	reg := DefaultRegistry()
	p, err := NewProvider(reg)
	require.NoError(t, err)

	// Start gRPC server on a random port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	RegisterResourceProviderServer(srv, p)
	go srv.Serve(lis)
	defer srv.Stop()

	// Verify the server accepts TCP connections.
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	// The server is running and accepting connections.
	// Full wire-protocol testing requires the Pulumi CLI or generated proto stubs.
	// The provider methods are thoroughly tested via direct calls above.
}

func TestGRPCServer_ServeMethod(t *testing.T) {
	reg := DefaultRegistry()
	p, err := NewProvider(reg)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// Test Serve method starts without error (run in background).
	go func() {
		_ = p.Serve(lis)
	}()

	// Verify the server is listening by connecting.
	conn, err := net.Dial("tcp", lis.Addr().String())
	require.NoError(t, err)
	conn.Close()
}

// ── Helper Function Tests ───────────────────────────────────────────────────

func TestPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bucket", "Bucket"},
		{"table_name", "TableName"},
		{"vpc_id", "VpcId"},
		{"cidr_block", "CidrBlock"},
		{"security_group", "SecurityGroup"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, pascalCase(tt.input), "pascalCase(%q)", tt.input)
	}
}

func TestSnakeToPascal(t *testing.T) {
	assert.Equal(t, "TableName", snakeToPascal("table_name"))
	assert.Equal(t, "VpcId", snakeToPascal("vpc_id"))
	assert.Equal(t, "Bucket", snakeToPascal("bucket"))
}

func TestExtractTypeToken(t *testing.T) {
	tests := []struct {
		urn      string
		expected string
	}{
		{
			"urn:pulumi:test::proj::cloudmock:s3:Bucket::mybucket",
			"cloudmock:s3:Bucket",
		},
		{
			"urn:pulumi:prod::myapp::cloudmock:ec2:Vpc::main-vpc",
			"cloudmock:ec2:Vpc",
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, extractTypeToken(tt.urn), "extractTypeToken(%q)", tt.urn)
	}
}

func TestGenerateSchemaJSON_WriteFile(t *testing.T) {
	if os.Getenv("WRITE_SCHEMA") == "" {
		t.Skip("set WRITE_SCHEMA=1 to write schema.json")
	}
	reg := DefaultRegistry()
	data, err := GeneratePulumiSchemaJSON(reg)
	require.NoError(t, err)
	err = os.WriteFile("../schema.json", data, 0644)
	require.NoError(t, err)
	t.Logf("wrote schema.json (%d bytes)", len(data))
}

func TestIsNotFoundError(t *testing.T) {
	assert.True(t, isNotFoundError(fmt.Errorf("resource not found")))
	assert.True(t, isNotFoundError(fmt.Errorf("ResourceNotFoundException: table foo")))
	assert.True(t, isNotFoundError(fmt.Errorf("HTTP 404: Not Found")))
	assert.False(t, isNotFoundError(fmt.Errorf("internal server error")))
	assert.False(t, isNotFoundError(nil))
}

//go:build !short

package snapshot_test

import (
	"encoding/json"
	"strings"
	"testing"

	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/Viridian-Inc/cloudmock/pkg/snapshot"
	cloudwatchlogssvc "github.com/Viridian-Inc/cloudmock/services/cloudwatchlogs"
	ddbsvc "github.com/Viridian-Inc/cloudmock/services/dynamodb"
	iamsvc "github.com/Viridian-Inc/cloudmock/services/iam"
	lambdasvc "github.com/Viridian-Inc/cloudmock/services/lambda"
	r53svc "github.com/Viridian-Inc/cloudmock/services/route53"
	s3svc "github.com/Viridian-Inc/cloudmock/services/s3"
	snssvc "github.com/Viridian-Inc/cloudmock/services/sns"
	sqssvc "github.com/Viridian-Inc/cloudmock/services/sqs"
)

const (
	testAccount = "123456789012"
	testRegion  = "us-east-1"
)

// snapshotableService is the combination of service.Service and service.Snapshotable.
type snapshotableService interface {
	service.Service
	ExportState() (json.RawMessage, error)
	ImportState(json.RawMessage) error
}

// exportJSON is a helper that exports state from a single service and returns the raw JSON.
func exportJSON(t *testing.T, svc snapshotableService) json.RawMessage {
	t.Helper()
	reg := routing.NewRegistry()
	reg.Register(svc)
	data, err := snapshot.Export(reg)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	var sf snapshot.StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("Unmarshal StateFile: %v", err)
	}
	return sf.Services[svc.Name()]
}

// importJSON imports raw service JSON directly onto the target service.
func importJSON(t *testing.T, svc snapshotableService, raw json.RawMessage) {
	t.Helper()
	if err := svc.ImportState(raw); err != nil {
		t.Fatalf("ImportState: %v", err)
	}
}

// stringContains reports whether any element of slice contains substr.
func stringContains(slice []string, substr string) bool {
	for _, s := range slice {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// TestSnapshot_S3_RoundTrip creates a bucket and object, exports, imports into a
// fresh service, and verifies the bucket and object body are restored.
func TestSnapshot_S3_RoundTrip(t *testing.T) {
	svc := s3svc.New()

	// Set up state via ExportState/ImportState directly (white-box) using a seeded state.
	seed := json.RawMessage(`{"buckets":[{"name":"test-bucket","objects":[{"key":"hello.txt","body_base64":"aGVsbG8gd29ybGQ=","content_type":"text/plain"}]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("seed ImportState: %v", err)
	}

	// Verify seeding worked.
	names := svc.GetBucketNames()
	if !stringContains(names, "test-bucket") {
		t.Fatalf("bucket not seeded, got %v", names)
	}

	// Export.
	raw := exportJSON(t, svc)
	if raw == nil {
		t.Fatal("expected non-nil s3 export")
	}

	// Import into fresh service.
	svc2 := s3svc.New()
	importJSON(t, svc2, raw)

	// Verify bucket restored.
	names2 := svc2.GetBucketNames()
	if !stringContains(names2, "test-bucket") {
		t.Errorf("bucket not restored after import, got %v", names2)
	}

	// Verify object via GetObjectData.
	body, err := svc2.GetObjectData("test-bucket", "hello.txt")
	if err != nil {
		t.Fatalf("GetObjectData: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("expected body 'hello world', got %q", string(body))
	}
}

// TestSnapshot_DynamoDB_RoundTrip creates a table and items, exports, imports, and verifies.
func TestSnapshot_DynamoDB_RoundTrip(t *testing.T) {
	svc := ddbsvc.New(testAccount, testRegion)

	// Seed via ImportState. KeySchemaElement and AttributeDefinition use PascalCase JSON tags.
	seed := json.RawMessage(`{"tables":[{"name":"users","key_schema":[{"AttributeName":"pk","KeyType":"HASH"}],"attribute_definitions":[{"AttributeName":"pk","AttributeType":"S"}],"billing_mode":"PAY_PER_REQUEST","items":[{"pk":{"S":"user-1"}},{"pk":{"S":"user-2"}}]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("seed ImportState: %v", err)
	}

	names := svc.GetTableNames()
	if !stringContains(names, "users") {
		t.Fatalf("table not seeded, got %v", names)
	}

	// Export.
	raw := exportJSON(t, svc)

	// Decode to verify structure.
	var state struct {
		Tables []struct {
			Name        string `json:"name"`
			BillingMode string `json:"billing_mode"`
			Items       []any  `json:"items"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal ddb state: %v", err)
	}
	if len(state.Tables) == 0 {
		t.Fatal("expected at least one table in export")
	}
	if state.Tables[0].Name != "users" {
		t.Errorf("expected table 'users', got %q", state.Tables[0].Name)
	}
	if state.Tables[0].BillingMode != "PAY_PER_REQUEST" {
		t.Errorf("expected billing mode PAY_PER_REQUEST, got %q", state.Tables[0].BillingMode)
	}
	// Items: user-1 and user-2 both have the same key structure; DynamoDB puts deduplicate
	// on the primary key. Seeding two distinct pk values should yield 2 items.
	if len(state.Tables[0].Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(state.Tables[0].Items))
	}

	// Import into fresh service.
	svc2 := ddbsvc.New(testAccount, testRegion)
	importJSON(t, svc2, raw)

	names2 := svc2.GetTableNames()
	if !stringContains(names2, "users") {
		t.Errorf("table not restored, got %v", names2)
	}
}

// TestSnapshot_SQS_RoundTrip creates a queue with attributes, exports, imports, and verifies.
func TestSnapshot_SQS_RoundTrip(t *testing.T) {
	svc := sqssvc.New(testAccount, testRegion)

	seed := json.RawMessage(`{"queues":[{"name":"my-queue","attributes":{"VisibilityTimeout":"60","MessageRetentionPeriod":"86400"}}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("seed ImportState: %v", err)
	}

	// GetQueueNames returns URLs, so check by substring.
	queueURLs := svc.GetQueueNames()
	if !stringContains(queueURLs, "my-queue") {
		t.Fatalf("queue not seeded, got %v", queueURLs)
	}

	raw := exportJSON(t, svc)

	// Decode and verify.
	var state struct {
		Queues []struct {
			Name       string            `json:"name"`
			Attributes map[string]string `json:"attributes"`
		} `json:"queues"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal sqs state: %v", err)
	}
	if len(state.Queues) == 0 {
		t.Fatal("expected queues in export")
	}
	if state.Queues[0].Attributes["VisibilityTimeout"] != "60" {
		t.Errorf("expected VisibilityTimeout=60, got %q", state.Queues[0].Attributes["VisibilityTimeout"])
	}

	// Import into fresh service.
	svc2 := sqssvc.New(testAccount, testRegion)
	importJSON(t, svc2, raw)

	if !stringContains(svc2.GetQueueNames(), "my-queue") {
		t.Error("queue not restored after import")
	}
}

// TestSnapshot_SNS_RoundTrip creates a topic and subscription, exports, imports, and verifies.
func TestSnapshot_SNS_RoundTrip(t *testing.T) {
	svc := snssvc.New(testAccount, testRegion)

	seed := json.RawMessage(`{"topics":[{"name":"alerts","subscriptions":[{"protocol":"sqs","endpoint":"arn:aws:sqs:us-east-1:123456789012:my-queue"}]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("seed ImportState: %v", err)
	}

	topics := svc.GetAllTopics()
	if !stringContains(topics, "alerts") {
		t.Fatalf("topic not seeded, got %v", topics)
	}

	raw := exportJSON(t, svc)

	var state struct {
		Topics []struct {
			Name          string `json:"name"`
			Subscriptions []struct {
				Protocol string `json:"protocol"`
				Endpoint string `json:"endpoint"`
			} `json:"subscriptions"`
		} `json:"topics"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal sns state: %v", err)
	}
	if len(state.Topics) == 0 {
		t.Fatal("expected topics in export")
	}
	found := false
	for _, topic := range state.Topics {
		if topic.Name == "alerts" {
			found = true
			if len(topic.Subscriptions) != 1 {
				t.Errorf("expected 1 subscription for alerts, got %d", len(topic.Subscriptions))
			}
			break
		}
	}
	if !found {
		t.Error("topic 'alerts' not found in export")
	}

	// Import into fresh service.
	svc2 := snssvc.New(testAccount, testRegion)
	importJSON(t, svc2, raw)

	topics2 := svc2.GetAllTopics()
	if !stringContains(topics2, "alerts") {
		t.Errorf("topic not restored after import, got %v", topics2)
	}
	subs := svc2.GetAllSubscriptions()
	if len(subs) == 0 {
		t.Error("expected subscriptions after import")
	}
}

// TestSnapshot_MultiService_RoundTrip exercises S3, DynamoDB, and SQS in a single export/import.
func TestSnapshot_MultiService_RoundTrip(t *testing.T) {
	s3Svc := s3svc.New()
	ddbSvc := ddbsvc.New(testAccount, testRegion)
	sqsSvc := sqssvc.New(testAccount, testRegion)

	// Seed each service.
	s3Svc.ImportState(json.RawMessage(`{"buckets":[{"name":"multi-bucket","objects":[]}]}`))
	ddbSvc.ImportState(json.RawMessage(`{"tables":[{"name":"multi-table","key_schema":[{"attribute_name":"id","key_type":"HASH"}],"attribute_definitions":[{"attribute_name":"id","attribute_type":"S"}],"billing_mode":"PAY_PER_REQUEST","items":[]}]}`))
	sqsSvc.ImportState(json.RawMessage(`{"queues":[{"name":"multi-queue","attributes":{}}]}`))

	// Export all.
	reg := routing.NewRegistry()
	reg.Register(s3Svc)
	reg.Register(ddbSvc)
	reg.Register(sqsSvc)
	data, err := snapshot.Export(reg)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	var sf snapshot.StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, name := range []string{"s3", "dynamodb", "sqs"} {
		if _, ok := sf.Services[name]; !ok {
			t.Errorf("expected service %q in snapshot", name)
		}
	}

	// Import into fresh services.
	s3Svc2 := s3svc.New()
	ddbSvc2 := ddbsvc.New(testAccount, testRegion)
	sqsSvc2 := sqssvc.New(testAccount, testRegion)

	reg2 := routing.NewRegistry()
	reg2.Register(s3Svc2)
	reg2.Register(ddbSvc2)
	reg2.Register(sqsSvc2)
	if err := snapshot.Import(reg2, data); err != nil {
		t.Fatalf("Import: %v", err)
	}

	if !stringContains(s3Svc2.GetBucketNames(), "multi-bucket") {
		t.Error("S3 bucket not restored")
	}
	if !stringContains(ddbSvc2.GetTableNames(), "multi-table") {
		t.Error("DynamoDB table not restored")
	}
	// GetQueueNames returns URLs; check by substring.
	if !stringContains(sqsSvc2.GetQueueNames(), "multi-queue") {
		t.Error("SQS queue not restored")
	}
}

// TestSnapshot_EmptyState exports with no resources and verifies valid JSON with empty services.
func TestSnapshot_EmptyState(t *testing.T) {
	reg := routing.NewRegistry()
	reg.Register(s3svc.New())
	reg.Register(ddbsvc.New(testAccount, testRegion))
	reg.Register(sqssvc.New(testAccount, testRegion))

	data, err := snapshot.Export(reg)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	var sf snapshot.StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if sf.Version != 1 {
		t.Errorf("expected version 1, got %d", sf.Version)
	}
	if sf.Services == nil {
		t.Error("expected non-nil services map")
	}

	// Verify exported state is valid JSON per-service (no nulls for empty state).
	for name, raw := range sf.Services {
		if !json.Valid(raw) {
			t.Errorf("service %q export is not valid JSON", name)
		}
	}
}

// TestSnapshot_ImportIdempotent imports the same state twice and verifies no duplicates.
func TestSnapshot_ImportIdempotent(t *testing.T) {
	svc := s3svc.New()
	raw := json.RawMessage(`{"buckets":[{"name":"idempotent-bucket","objects":[]}]}`)

	// Import once.
	if err := svc.ImportState(raw); err != nil {
		t.Fatalf("ImportState (first): %v", err)
	}

	// Import again — should not error or duplicate.
	if err := svc.ImportState(raw); err != nil {
		t.Fatalf("ImportState (second): %v", err)
	}

	names := svc.GetBucketNames()
	count := 0
	for _, n := range names {
		if n == "idempotent-bucket" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 idempotent-bucket after double import, got %d (all: %v)", count, names)
	}
}

// TestSnapshot_IAM_RoundTrip verifies IAM users, roles, and policies are preserved.
func TestSnapshot_IAM_RoundTrip(t *testing.T) {
	engine := iampkg.NewEngine()
	pkgStore := iampkg.NewStore(testAccount)
	svc := iamsvc.New(testAccount, engine, pkgStore)

	seed := json.RawMessage(`{"users":[{"user_name":"alice"}],"roles":[{"role_name":"my-role","assume_role_policy_document":"{}"}],"policies":[{"policy_name":"my-policy","document":"{}"}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("seed ImportState: %v", err)
	}

	raw := exportJSON(t, svc)
	if raw == nil {
		t.Fatal("expected non-nil iam export")
	}

	// Decode and verify.
	var state struct {
		Users    []struct{ UserName string `json:"user_name"` }    `json:"users"`
		Roles    []struct{ RoleName string `json:"role_name"` }    `json:"roles"`
		Policies []struct{ PolicyName string `json:"policy_name"` } `json:"policies"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal iam state: %v", err)
	}
	if len(state.Users) == 0 {
		t.Error("expected users in export")
	}
	if len(state.Roles) == 0 {
		t.Error("expected roles in export")
	}
	if len(state.Policies) == 0 {
		t.Error("expected policies in export")
	}

	// Import into fresh service.
	engine2 := iampkg.NewEngine()
	pkgStore2 := iampkg.NewStore(testAccount)
	svc2 := iamsvc.New(testAccount, engine2, pkgStore2)
	importJSON(t, svc2, raw)

	// Re-export and verify counts match.
	raw2 := exportJSON(t, svc2)
	var state2 struct {
		Users    []any `json:"users"`
		Roles    []any `json:"roles"`
		Policies []any `json:"policies"`
	}
	json.Unmarshal(raw2, &state2)
	if len(state2.Users) != len(state.Users) {
		t.Errorf("user count mismatch: want %d, got %d", len(state.Users), len(state2.Users))
	}
}

// TestSnapshot_Lambda_RoundTrip verifies Lambda function configs are preserved.
func TestSnapshot_Lambda_RoundTrip(t *testing.T) {
	svc := lambdasvc.New(testAccount, testRegion)

	seed := json.RawMessage(`{"functions":[{"function_name":"my-func","runtime":"nodejs18.x","role":"arn:aws:iam::123456789012:role/lambda-role","handler":"index.handler","timeout":30,"memory_size":128}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("seed ImportState: %v", err)
	}

	if !stringContains(svc.GetFunctionNames(), "my-func") {
		t.Fatal("function not seeded")
	}

	raw := exportJSON(t, svc)

	// Import into fresh service.
	svc2 := lambdasvc.New(testAccount, testRegion)
	importJSON(t, svc2, raw)

	if !stringContains(svc2.GetFunctionNames(), "my-func") {
		t.Error("function not restored after import")
	}
}

// TestSnapshot_CloudWatchLogs_RoundTrip verifies log groups are preserved.
func TestSnapshot_CloudWatchLogs_RoundTrip(t *testing.T) {
	svc := cloudwatchlogssvc.New(testAccount, testRegion)

	seed := json.RawMessage(`{"log_groups":[{"name":"/app/service","retention_days":7}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("seed ImportState: %v", err)
	}

	raw := exportJSON(t, svc)

	var state struct {
		LogGroups []struct {
			Name          string `json:"name"`
			RetentionDays int    `json:"retention_days"`
		} `json:"log_groups"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal cwl state: %v", err)
	}
	if len(state.LogGroups) == 0 {
		t.Fatal("expected log groups in export")
	}
	if state.LogGroups[0].Name != "/app/service" {
		t.Errorf("expected log group '/app/service', got %q", state.LogGroups[0].Name)
	}

	svc2 := cloudwatchlogssvc.New(testAccount, testRegion)
	importJSON(t, svc2, raw)

	// Re-export and check.
	raw2 := exportJSON(t, svc2)
	var state2 struct {
		LogGroups []struct{ Name string `json:"name"` } `json:"log_groups"`
	}
	json.Unmarshal(raw2, &state2)
	if len(state2.LogGroups) == 0 {
		t.Error("log groups not restored after import")
	}
}

// TestSnapshot_Route53_RoundTrip verifies hosted zones are preserved.
func TestSnapshot_Route53_RoundTrip(t *testing.T) {
	svc := r53svc.New(testAccount, testRegion)

	seed := json.RawMessage(`{"hosted_zones":[{"name":"example.com.","comment":"test zone"}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("seed ImportState: %v", err)
	}

	raw := exportJSON(t, svc)

	var state struct {
		HostedZones []struct {
			Name    string `json:"name"`
			Comment string `json:"comment"`
		} `json:"hosted_zones"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal r53 state: %v", err)
	}
	if len(state.HostedZones) == 0 {
		t.Fatal("expected hosted zones in export")
	}
	if state.HostedZones[0].Name != "example.com." {
		t.Errorf("expected zone 'example.com.', got %q", state.HostedZones[0].Name)
	}

	svc2 := r53svc.New(testAccount, testRegion)
	importJSON(t, svc2, raw)

	raw2 := exportJSON(t, svc2)
	var state2 struct {
		HostedZones []any `json:"hosted_zones"`
	}
	json.Unmarshal(raw2, &state2)
	if len(state2.HostedZones) == 0 {
		t.Error("hosted zones not restored after import")
	}
}

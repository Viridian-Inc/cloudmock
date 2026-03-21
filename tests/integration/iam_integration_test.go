package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	s3svc "github.com/neureaux/cloudmock/services/s3"
)

// setupFullStack constructs a fully wired gateway + IAM engine + IAM store
// with an S3 service registered. iamMode is written into cfg.IAM.Mode.
func setupFullStack(t *testing.T, iamMode string) (*gateway.Gateway, *iampkg.Store, *iampkg.Engine) {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = iamMode

	store := iampkg.NewStore(cfg.AccountID)
	if err := store.InitRoot("ROOTKEY", "ROOTSECRET"); err != nil {
		t.Fatalf("InitRoot: %v", err)
	}

	engine := iampkg.NewEngine()

	reg := routing.NewRegistry()
	reg.Register(s3svc.New())

	gw := gateway.NewWithIAM(cfg, reg, store, engine)

	return gw, store, engine
}

// s3AuthHeader returns a minimal AWS SigV4 Authorization header that embeds
// the given accessKeyID and points to the s3 service scope. The signature
// value is intentionally fake — cloudmock validates credential existence but
// does not verify the HMAC.
func s3AuthHeader(accessKeyID string) string {
	return "AWS4-HMAC-SHA256 Credential=" + accessKeyID +
		"/20240101/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef"
}

// TestFullStack_RootCanCreateBucket verifies that the root credential is
// allowed to create a bucket in enforce mode.
func TestFullStack_RootCanCreateBucket(t *testing.T) {
	gw, _, _ := setupFullStack(t, "enforce")

	req := httptest.NewRequest(http.MethodPut, "/test-bucket", nil)
	req.Header.Set("Authorization", s3AuthHeader("ROOTKEY"))

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for root CreateBucket, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestFullStack_UserWithPolicyCanListBuckets verifies that a user with an
// explicit Allow policy for s3:ListAllMyBuckets can call GET /.
func TestFullStack_UserWithPolicyCanListBuckets(t *testing.T) {
	gw, store, engine := setupFullStack(t, "enforce")

	// Create user and access key.
	if _, err := store.CreateUser("reader"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	key, err := store.CreateAccessKey("reader")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}

	// Build the policy. The gateway's DetectAction reads the ?Action= query
	// parameter, so the IAM action evaluated is "s3:ListBuckets". We allow
	// s3:ListAllMyBuckets (the real AWS IAM action name) via a wildcard on the
	// action so both naming conventions are covered, but we use the simpler
	// "s3:ListBuckets" that matches DetectAction's output. A wildcard policy
	// on s3:* is also valid; here we use the exact token the gateway produces.
	policy := &iampkg.Policy{
		Version: "2012-10-17",
		Statements: []iampkg.Statement{
			{
				Effect:    "Allow",
				Actions:   []string{"s3:ListBuckets"},
				Resources: []string{"*"},
			},
		},
	}

	// Attach to store records and register with engine.
	if err := store.AttachUserPolicy("reader", "ListBucketsPolicy", policy); err != nil {
		t.Fatalf("AttachUserPolicy: %v", err)
	}
	engine.AddPolicy("reader", policy)

	// Send GET /?Action=ListBuckets with user credentials.
	// The ?Action= parameter is required so that DetectAction returns
	// "ListBuckets", making the evaluated IAM action "s3:ListBuckets".
	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization", s3AuthHeader(key.AccessKeyID))

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for user ListBuckets with policy, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestFullStack_UserWithoutPolicyDenied verifies that a user with no attached
// policies receives a 403 in enforce mode.
func TestFullStack_UserWithoutPolicyDenied(t *testing.T) {
	gw, store, _ := setupFullStack(t, "enforce")

	// Create a user with an access key but no policies.
	if _, err := store.CreateUser("noperms"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	key, err := store.CreateAccessKey("noperms")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization", s3AuthHeader(key.AccessKeyID))

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user without policy, got %d (body: %s)", w.Code, w.Body.String())
	}
}

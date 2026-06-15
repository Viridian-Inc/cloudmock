package iam_test

import (
	"testing"

	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	iamsvc "github.com/Viridian-Inc/cloudmock/services/iam"
)

// Attaching a managed policy must register it with the IAM engine so the
// engine actually evaluates it (registerPolicyWithEngine was previously a
// no-op, so attached policies were silently ignored).
func TestStore_AttachUserPolicy_RegistersWithEngine(t *testing.T) {
	const account = "000000000000"
	engine := iampkg.NewEngine()
	store := iamsvc.NewStore(account, engine, iampkg.NewStore(account))

	if _, err := store.CreateUser("alice"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	// Single-string Action/Resource (the common AWS form the parser must tolerate).
	pol, err := store.CreatePolicy("AllowS3Get",
		`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`, "")
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if err := store.AttachUserPolicy("alice", pol.Arn); err != nil {
		t.Fatalf("AttachUserPolicy: %v", err)
	}

	principal := "arn:aws:iam::" + account + ":user/alice"

	if res := engine.Evaluate(&iampkg.EvalRequest{
		Principal: principal, Action: "s3:GetObject", Resource: "arn:aws:s3:::bucket/key",
	}); res.Decision != iampkg.Allow {
		t.Fatalf("allowed action: decision = %v, want Allow (reason=%s)", res.Decision, res.Reason)
	}

	if res := engine.Evaluate(&iampkg.EvalRequest{
		Principal: principal, Action: "dynamodb:PutItem", Resource: "*",
	}); res.Decision != iampkg.Deny {
		t.Errorf("unrelated action: decision = %v, want implicit Deny", res.Decision)
	}
}

func TestParsePolicyDocument_Tolerant(t *testing.T) {
	// Array-form Action plus an explicit Deny — both must register.
	const account = "000000000000"
	engine := iampkg.NewEngine()
	store := iamsvc.NewStore(account, engine, iampkg.NewStore(account))

	if _, err := store.CreateUser("bob"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	pol, err := store.CreatePolicy("Mixed",
		`{"Statement":[{"Effect":"Allow","Action":["s3:GetObject","s3:PutObject"],"Resource":["*"]},{"Effect":"Deny","Action":"s3:DeleteObject","Resource":"*"}]}`, "")
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if err := store.AttachUserPolicy("bob", pol.Arn); err != nil {
		t.Fatalf("AttachUserPolicy: %v", err)
	}
	principal := "arn:aws:iam::" + account + ":user/bob"

	if res := engine.Evaluate(&iampkg.EvalRequest{Principal: principal, Action: "s3:PutObject", Resource: "*"}); res.Decision != iampkg.Allow {
		t.Errorf("array action s3:PutObject: decision = %v, want Allow", res.Decision)
	}
	if res := engine.Evaluate(&iampkg.EvalRequest{Principal: principal, Action: "s3:DeleteObject", Resource: "*"}); res.Decision != iampkg.Deny {
		t.Errorf("explicit deny s3:DeleteObject: decision = %v, want Deny", res.Decision)
	}
}

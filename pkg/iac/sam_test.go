package iac

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

const testSAMTemplate = `
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31

Resources:
  UsersTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: users
      AttributeDefinitions:
        - AttributeName: pk
          AttributeType: S
        - AttributeName: sk
          AttributeType: S
      KeySchema:
        - AttributeName: pk
          KeyType: HASH
        - AttributeName: sk
          KeyType: RANGE
      BillingMode: PAY_PER_REQUEST

  ApiHandler:
    Type: AWS::Serverless::Function
    Properties:
      FunctionName: api-handler
      Runtime: nodejs20.x
      Handler: index.handler
      Timeout: 30
      MemorySize: 256
    DependsOn: UsersTable

  TaskQueue:
    Type: AWS::SQS::Queue
    Properties:
      QueueName: task-queue

  AlertTopic:
    Type: AWS::SNS::Topic
    Properties:
      TopicName: alert-topic

  UploadsBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: uploads-bucket
`

func TestImportSAMTemplate(t *testing.T) {
	dir := t.TempDir()
	tplPath := filepath.Join(dir, "template.yaml")
	os.WriteFile(tplPath, []byte(testSAMTemplate), 0644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	result, graph, err := ImportSAMTemplate(tplPath, "dev", logger)
	if err != nil {
		t.Fatalf("ImportSAMTemplate: %v", err)
	}

	if len(result.Tables) != 1 {
		t.Fatalf("tables: got %d, want 1", len(result.Tables))
	}
	tbl := result.Tables[0]
	if tbl.Name != "users" {
		t.Errorf("table name = %q, want users", tbl.Name)
	}
	if tbl.HashKey != "pk" {
		t.Errorf("hash key = %q, want pk", tbl.HashKey)
	}
	if tbl.RangeKey != "sk" {
		t.Errorf("range key = %q, want sk", tbl.RangeKey)
	}
	if len(tbl.Attributes) != 2 {
		t.Errorf("attributes: got %d, want 2", len(tbl.Attributes))
	}

	if len(result.Lambdas) != 1 || result.Lambdas[0].Name != "api-handler" {
		t.Errorf("lambdas: got %v, want [{api-handler}]", result.Lambdas)
	}
	lam := result.Lambdas[0]
	if lam.Runtime != "nodejs20.x" {
		t.Errorf("lambda runtime = %q, want nodejs20.x", lam.Runtime)
	}
	if lam.Timeout != 30 {
		t.Errorf("lambda timeout = %d, want 30", lam.Timeout)
	}
	if lam.Memory != 256 {
		t.Errorf("lambda memory = %d, want 256", lam.Memory)
	}

	if len(result.SQSQueues) != 1 || result.SQSQueues[0].Name != "task-queue" {
		t.Errorf("queues: got %v", result.SQSQueues)
	}
	if len(result.SNSTopics) != 1 || result.SNSTopics[0].Name != "alert-topic" {
		t.Errorf("topics: got %v", result.SNSTopics)
	}
	if len(result.S3Buckets) != 1 || result.S3Buckets[0].Name != "uploads-bucket" {
		t.Errorf("buckets: got %v", result.S3Buckets)
	}

	// Dependency graph.
	if graph == nil {
		t.Fatal("graph is nil")
	}
	if len(graph.Nodes) != 5 {
		t.Errorf("graph nodes: got %d, want 5", len(graph.Nodes))
	}

	// Check explicit DependsOn edge.
	var hasDep bool
	for _, e := range graph.Edges {
		if e.Source == "ApiHandler" && e.Target == "UsersTable" && e.Type == "dependsOn" {
			hasDep = true
		}
	}
	if !hasDep {
		t.Error("missing DependsOn edge: ApiHandler → UsersTable")
	}
}

func TestFindSAMTemplate(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "template.yaml"), []byte("Resources: {}"), 0644)
	if got := FindSAMTemplate(dir); got == "" {
		t.Error("FindSAMTemplate returned empty for dir with template.yaml")
	}

	emptyDir := t.TempDir()
	if got := FindSAMTemplate(emptyDir); got != "" {
		t.Errorf("FindSAMTemplate returned %q for empty dir", got)
	}
}

func TestDetectIaCType_SAM(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "template.yaml"), []byte("Resources: {}"), 0644)
	if got := DetectIaCType(dir); got != "sam" {
		t.Errorf("DetectIaCType = %q, want sam", got)
	}
}

func TestDetectIaCType_CDK(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "cdk.json"), []byte(`{}`), 0644)
	if got := DetectIaCType(dir); got != "cdk" {
		t.Errorf("DetectIaCType = %q, want cdk", got)
	}
}

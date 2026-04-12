package iac

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

const testTerraformHCL = `
provider "aws" {
  region = "us-east-1"
}

resource "aws_dynamodb_table" "users" {
  name         = "users-table"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pk"
  range_key    = "sk"

  attribute {
    name = "pk"
    type = "S"
  }
  attribute {
    name = "sk"
    type = "S"
  }

  global_secondary_index {
    name            = "email-index"
    hash_key        = "email"
    projection_type = "ALL"
  }
}

resource "aws_lambda_function" "handler" {
  function_name = "api-handler"
  runtime       = "nodejs20.x"
  handler       = "index.handler"
  timeout       = 30
  memory_size   = 256

  environment {
    variables = {
      TABLE_NAME = aws_dynamodb_table.users.name
    }
  }

  depends_on = [aws_dynamodb_table.users]
}

resource "aws_sqs_queue" "tasks" {
  name = "task-queue"
}

resource "aws_sns_topic" "alerts" {
  name = "alert-topic"
}

resource "aws_s3_bucket" "uploads" {
  bucket = "my-uploads-bucket"
}

resource "aws_iam_role" "lambda_role" {
  name = "lambda-exec-role"
  assume_role_policy = "{}"
}
`

func TestImportTerraformDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(testTerraformHCL), 0644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	result, graph, err := ImportTerraformDir(dir, "dev", logger)
	if err != nil {
		t.Fatalf("ImportTerraformDir: %v", err)
	}

	// --- Resource extraction ---

	if len(result.Tables) != 1 {
		t.Fatalf("tables: got %d, want 1", len(result.Tables))
	}
	tbl := result.Tables[0]
	if tbl.Name != "users-table" {
		t.Errorf("table name = %q, want users-table", tbl.Name)
	}
	if tbl.HashKey != "pk" {
		t.Errorf("hash_key = %q, want pk", tbl.HashKey)
	}
	if tbl.RangeKey != "sk" {
		t.Errorf("range_key = %q, want sk", tbl.RangeKey)
	}
	if len(tbl.Attributes) != 2 {
		t.Errorf("attributes: got %d, want 2", len(tbl.Attributes))
	}
	if len(tbl.GSIs) != 1 {
		t.Errorf("GSIs: got %d, want 1", len(tbl.GSIs))
	} else if tbl.GSIs[0].Name != "email-index" {
		t.Errorf("GSI name = %q, want email-index", tbl.GSIs[0].Name)
	}

	if len(result.Lambdas) != 1 {
		t.Fatalf("lambdas: got %d, want 1", len(result.Lambdas))
	}
	lam := result.Lambdas[0]
	if lam.Name != "api-handler" {
		t.Errorf("lambda name = %q, want api-handler", lam.Name)
	}
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
		t.Errorf("sqs queues: got %v, want [{task-queue}]", result.SQSQueues)
	}
	if len(result.SNSTopics) != 1 || result.SNSTopics[0].Name != "alert-topic" {
		t.Errorf("sns topics: got %v, want [{alert-topic}]", result.SNSTopics)
	}
	if len(result.S3Buckets) != 1 || result.S3Buckets[0].Name != "my-uploads-bucket" {
		t.Errorf("s3 buckets: got %v, want [{my-uploads-bucket}]", result.S3Buckets)
	}

	// --- Dependency graph ---

	if graph == nil {
		t.Fatal("graph is nil")
	}

	// Should have 6 nodes: dynamodb, lambda, sqs, sns, s3, iam
	if len(graph.Nodes) != 6 {
		t.Errorf("graph nodes: got %d, want 6", len(graph.Nodes))
	}

	// Check edges: lambda depends_on dynamodb (explicit) + references dynamodb (implicit)
	var hasExplicitDep, hasImplicitRef bool
	for _, e := range graph.Edges {
		if e.Source == "aws_lambda_function.handler" && e.Target == "aws_dynamodb_table.users" {
			if e.Type == "dependsOn" {
				hasExplicitDep = true
			}
			if e.Type == "reference" {
				hasImplicitRef = true
			}
		}
	}
	if !hasExplicitDep {
		t.Error("missing explicit depends_on edge: lambda → dynamodb")
	}
	if !hasImplicitRef {
		t.Error("missing implicit reference edge: lambda → dynamodb (from aws_dynamodb_table.users.name)")
	}
}

func TestDetectIaCType(t *testing.T) {
	// Terraform
	tfDir := t.TempDir()
	os.WriteFile(filepath.Join(tfDir, "main.tf"), []byte("resource {}"), 0644)
	if got := DetectIaCType(tfDir); got != "terraform" {
		t.Errorf("terraform dir: got %q, want terraform", got)
	}

	// Pulumi
	pulumiDir := t.TempDir()
	os.WriteFile(filepath.Join(pulumiDir, "Pulumi.yaml"), []byte("name: test"), 0644)
	if got := DetectIaCType(pulumiDir); got != "pulumi" {
		t.Errorf("pulumi dir: got %q, want pulumi", got)
	}

	// Empty
	emptyDir := t.TempDir()
	if got := DetectIaCType(emptyDir); got != "" {
		t.Errorf("empty dir: got %q, want empty", got)
	}
}

func TestExtractImplicitRefs(t *testing.T) {
	body := `
    TABLE_NAME = aws_dynamodb_table.users.name
    QUEUE_URL  = aws_sqs_queue.tasks.url
    SELF_REF   = aws_lambda_function.handler.arn
  `
	refs := extractImplicitRefs(body)
	want := map[string]bool{
		"aws_dynamodb_table.users":      true,
		"aws_sqs_queue.tasks":           true,
		"aws_lambda_function.handler":   true,
	}
	for _, ref := range refs {
		if !want[ref] {
			t.Errorf("unexpected ref: %q", ref)
		}
		delete(want, ref)
	}
	for missing := range want {
		t.Errorf("missing ref: %q", missing)
	}
}

func TestExtractDependsOn(t *testing.T) {
	body := `
    depends_on = [aws_dynamodb_table.users, aws_sqs_queue.tasks]
  `
	deps := extractDependsOn(body)
	if len(deps) != 2 {
		t.Fatalf("depends_on: got %d deps, want 2", len(deps))
	}
	if deps[0] != "aws_dynamodb_table.users" {
		t.Errorf("dep[0] = %q, want aws_dynamodb_table.users", deps[0])
	}
	if deps[1] != "aws_sqs_queue.tasks" {
		t.Errorf("dep[1] = %q, want aws_sqs_queue.tasks", deps[1])
	}
}

package iac

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

const testCDKSource = `
import * as cdk from 'aws-cdk-lib';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as s3 from 'aws-cdk-lib/aws-s3';

export class MyStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string) {
    super(scope, id);

    const table = new dynamodb.Table(this, 'UsersTable', {
      tableName: 'users',
      partitionKey: { name: 'pk', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'sk', type: dynamodb.AttributeType.STRING },
    });

    const handler = new lambda.Function(this, 'ApiHandler', {
      functionName: 'api-handler',
      runtime: lambda.Runtime.NODEJS_20_X,
      handler: 'index.handler',
    });

    const queue = new sqs.Queue(this, 'TaskQueue', {
      queueName: 'task-queue',
    });

    const topic = new sns.Topic(this, 'AlertTopic', {
      topicName: 'alert-topic',
    });

    const bucket = new s3.Bucket(this, 'UploadsBucket', {
      bucketName: 'uploads-bucket',
    });
  }
}
`

func TestImportCDKDir(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	os.MkdirAll(libDir, 0755)
	os.WriteFile(filepath.Join(libDir, "stack.ts"), []byte(testCDKSource), 0644)
	// CDK detection requires cdk.json.
	os.WriteFile(filepath.Join(dir, "cdk.json"), []byte(`{"app":"npx ts-node lib/stack.ts"}`), 0644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	result, graph, err := ImportCDKDir(dir, "dev", logger)
	if err != nil {
		t.Fatalf("ImportCDKDir: %v", err)
	}

	if len(result.Tables) != 1 {
		t.Fatalf("tables: got %d, want 1", len(result.Tables))
	}
	if result.Tables[0].Name != "users" {
		t.Errorf("table name = %q, want users", result.Tables[0].Name)
	}
	if result.Tables[0].HashKey != "pk" {
		t.Errorf("hash key = %q, want pk", result.Tables[0].HashKey)
	}
	if result.Tables[0].RangeKey != "sk" {
		t.Errorf("range key = %q, want sk", result.Tables[0].RangeKey)
	}

	if len(result.Lambdas) != 1 || result.Lambdas[0].Name != "api-handler" {
		t.Errorf("lambdas: got %v, want [{api-handler}]", result.Lambdas)
	}
	if len(result.SQSQueues) != 1 || result.SQSQueues[0].Name != "task-queue" {
		t.Errorf("queues: got %v, want [{task-queue}]", result.SQSQueues)
	}
	if len(result.SNSTopics) != 1 || result.SNSTopics[0].Name != "alert-topic" {
		t.Errorf("topics: got %v, want [{alert-topic}]", result.SNSTopics)
	}
	if len(result.S3Buckets) != 1 || result.S3Buckets[0].Name != "uploads-bucket" {
		t.Errorf("buckets: got %v, want [{uploads-bucket}]", result.S3Buckets)
	}

	if graph == nil {
		t.Fatal("graph is nil")
	}
	if len(graph.Nodes) != 5 {
		t.Errorf("graph nodes: got %d, want 5", len(graph.Nodes))
	}
}

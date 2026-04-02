package sdk

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---------------------------------------------------------------------------
// Functional tests
// ---------------------------------------------------------------------------

func TestInProcess_Health(t *testing.T) {
	cm := New()
	defer cm.Close()

	// The gateway exposes /_cloudmock/health — verify the transport works at all.
	cfg := cm.Config()
	_ = cfg // just verify New() doesn't panic
}

func TestInProcess_STS(t *testing.T) {
	cm := New()
	defer cm.Close()

	client := sts.NewFromConfig(cm.Config())
	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	require.NoError(t, err)
	assert.Equal(t, "000000000000", *out.Account)
}

func TestInProcess_S3(t *testing.T) {
	cm := New()
	defer cm.Close()
	cfg := cm.Config()

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("test-bucket"),
	})
	require.NoError(t, err)

	// Put object
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("hello.txt"),
		Body:   strings.NewReader("world"),
	})
	require.NoError(t, err)

	// Get object
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("hello.txt"),
	})
	require.NoError(t, err)
	body, err := io.ReadAll(out.Body)
	require.NoError(t, err)
	assert.Equal(t, "world", string(body))

	// List buckets
	listOut, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, err)
	found := false
	for _, b := range listOut.Buckets {
		if *b.Name == "test-bucket" {
			found = true
		}
	}
	assert.True(t, found, "test-bucket should appear in bucket list")
}

func TestInProcess_DynamoDB(t *testing.T) {
	cm := New()
	defer cm.Close()

	client := dynamodb.NewFromConfig(cm.Config())

	// Create table
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("test"),
		KeySchema: []ddbTypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: ddbTypes.KeyTypeHash},
		},
		AttributeDefinitions: []ddbTypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: ddbTypes.ScalarAttributeTypeS},
		},
		BillingMode: ddbTypes.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	// Put item
	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("test"),
		Item: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "hello"},
		},
	})
	require.NoError(t, err)

	// Get item
	getOut, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("test"),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "hello"},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, getOut.Item)
	assert.Equal(t, "hello", getOut.Item["pk"].(*ddbTypes.AttributeValueMemberS).Value)
}

func TestInProcess_SQS(t *testing.T) {
	cm := New()
	defer cm.Close()

	client := sqs.NewFromConfig(cm.Config())

	// Create queue
	createOut, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("test-queue"),
	})
	require.NoError(t, err)
	queueURL := createOut.QueueUrl
	require.NotNil(t, queueURL)

	// Send message
	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    queueURL,
		MessageBody: aws.String("hello from in-process"),
	})
	require.NoError(t, err)

	// Receive message
	recvOut, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl: queueURL,
	})
	require.NoError(t, err)
	require.Len(t, recvOut.Messages, 1)
	assert.Equal(t, "hello from in-process", *recvOut.Messages[0].Body)
}

func TestInProcess_WithOptions(t *testing.T) {
	cm := New(
		WithRegion("eu-west-1"),
		WithAccountID("123456789012"),
		WithIAMMode("none"),
		WithProfile("minimal"),
	)
	defer cm.Close()

	cfg := cm.Config()
	assert.Equal(t, "eu-west-1", cfg.Region)
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkInProcess_DynamoDB_GetItem(b *testing.B) {
	cm := New()
	defer cm.Close()

	client := dynamodb.NewFromConfig(cm.Config())

	// Setup: create table + put item
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("bench"),
		KeySchema: []ddbTypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: ddbTypes.KeyTypeHash},
		},
		AttributeDefinitions: []ddbTypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: ddbTypes.ScalarAttributeTypeS},
		},
		BillingMode: ddbTypes.BillingModePayPerRequest,
	})
	if err != nil {
		b.Fatal(err)
	}
	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("bench"),
		Item: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "hello"},
		},
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String("bench"),
			Key: map[string]ddbTypes.AttributeValue{
				"pk": &ddbTypes.AttributeValueMemberS{Value: "hello"},
			},
		})
	}
}

func BenchmarkInProcess_S3_PutObject(b *testing.B) {
	cm := New()
	defer cm.Close()

	client := s3.NewFromConfig(cm.Config(), func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Setup: create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("bench"),
	})
	if err != nil {
		b.Fatal(err)
	}

	payload := strings.NewReader("benchmark-payload-data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload.Reset("benchmark-payload-data")
		_, _ = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String("bench"),
			Key:    aws.String("key"),
			Body:   payload,
		})
	}
}

func BenchmarkInProcess_SQS_SendMessage(b *testing.B) {
	cm := New()
	defer cm.Close()

	client := sqs.NewFromConfig(cm.Config())

	// Setup: create queue
	createOut, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("bench-queue"),
	})
	if err != nil {
		b.Fatal(err)
	}
	queueURL := createOut.QueueUrl

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    queueURL,
			MessageBody: aws.String("bench"),
		})
	}
}

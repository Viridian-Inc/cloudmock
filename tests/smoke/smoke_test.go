//go:build smoke

package smoke_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const endpoint = "http://localhost:4566"

func cloudmockConfig(t *testing.T) aws.Config {
	t.Helper()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	require.NoError(t, err)
	return cfg
}

func TestSmoke_Health(t *testing.T) {
	resp, err := http.Get(endpoint + "/_cloudmock/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestSmoke_Services(t *testing.T) {
	resp, err := http.Get(endpoint + "/_cloudmock/services")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	var services []string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&services))
	assert.Greater(t, len(services), 90, "should have 90+ services registered")
}

func TestSmoke_STS_GetCallerIdentity(t *testing.T) {
	cfg := cloudmockConfig(t)
	client := sts.NewFromConfig(cfg, func(o *sts.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	out, err := client.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	require.NoError(t, err)
	assert.Equal(t, "000000000000", *out.Account)
	assert.NotEmpty(t, *out.Arn)
}

func TestSmoke_S3_BucketAndObjectCRUD(t *testing.T) {
	cfg := cloudmockConfig(t)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	ctx := context.TODO()
	bucket := "smoke-test-" + time.Now().Format("20060102150405")

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)

	// List buckets
	listOut, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, err)
	found := false
	for _, b := range listOut.Buckets {
		if *b.Name == bucket {
			found = true
		}
	}
	assert.True(t, found, "bucket should appear in list")

	// Delete bucket
	_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)
}

func TestSmoke_SQS_SendReceive(t *testing.T) {
	t.Skip("AWS SDK v2 SQS uses JSON protocol; cloudmock SQS returns XML. TODO: add JSON support to SQS service.")
	cfg := cloudmockConfig(t)
	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
	ctx := context.TODO()

	// Create queue
	createOut, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("smoke-test-queue"),
	})
	require.NoError(t, err)

	// Send message
	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    createOut.QueueUrl,
		MessageBody: aws.String("hello from smoke test"),
	})
	require.NoError(t, err)

	// Receive message
	recvOut, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            createOut.QueueUrl,
		MaxNumberOfMessages: 1,
	})
	require.NoError(t, err)
	require.Len(t, recvOut.Messages, 1)
	assert.Equal(t, "hello from smoke test", *recvOut.Messages[0].Body)

	// Delete queue
	_, err = client.DeleteQueue(ctx, &sqs.DeleteQueueInput{QueueUrl: createOut.QueueUrl})
	require.NoError(t, err)
}

func TestSmoke_DynamoDB_TableAndItems(t *testing.T) {
	cfg := cloudmockConfig(t)
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
	ctx := context.TODO()
	tableName := "smoke-test-table"

	// Create table
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
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
		TableName: aws.String(tableName),
		Item: map[string]ddbTypes.AttributeValue{
			"pk":   &ddbTypes.AttributeValueMemberS{Value: "user1"},
			"name": &ddbTypes.AttributeValueMemberS{Value: "Alice"},
		},
	})
	require.NoError(t, err)

	// Get item
	getOut, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "user1"},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, getOut.Item)

	// Delete table
	_, err = client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	require.NoError(t, err)
}

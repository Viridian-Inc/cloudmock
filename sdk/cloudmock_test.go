package sdk

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
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
// DynamoDB – additional tests
// ---------------------------------------------------------------------------

func TestInProcess_DynamoDB_Query(t *testing.T) {
	cm := New()
	defer cm.Close()

	client := dynamodb.NewFromConfig(cm.Config())

	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("query-table"),
		KeySchema: []ddbTypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: ddbTypes.KeyTypeHash},
			{AttributeName: aws.String("sk"), KeyType: ddbTypes.KeyTypeRange},
		},
		AttributeDefinitions: []ddbTypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: ddbTypes.ScalarAttributeTypeS},
			{AttributeName: aws.String("sk"), AttributeType: ddbTypes.ScalarAttributeTypeS},
		},
		BillingMode: ddbTypes.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	// Put three items under the same pk, different sk values.
	for _, sk := range []string{"a", "b", "c"} {
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String("query-table"),
			Item: map[string]ddbTypes.AttributeValue{
				"pk":   &ddbTypes.AttributeValueMemberS{Value: "user#1"},
				"sk":   &ddbTypes.AttributeValueMemberS{Value: sk},
				"data": &ddbTypes.AttributeValueMemberS{Value: "val-" + sk},
			},
		})
		require.NoError(t, err)
	}

	// Query by pk only.
	out, err := client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String("query-table"),
		KeyConditionExpression: aws.String("pk = :pk"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":pk": &ddbTypes.AttributeValueMemberS{Value: "user#1"},
		},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, out.Count, "expected 3 items for pk=user#1")
}

func TestInProcess_DynamoDB_Scan(t *testing.T) {
	cm := New()
	defer cm.Close()

	client := dynamodb.NewFromConfig(cm.Config())

	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("scan-table"),
		KeySchema: []ddbTypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: ddbTypes.KeyTypeHash},
		},
		AttributeDefinitions: []ddbTypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: ddbTypes.ScalarAttributeTypeS},
		},
		BillingMode: ddbTypes.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String("scan-table"),
			Item: map[string]ddbTypes.AttributeValue{
				"pk": &ddbTypes.AttributeValueMemberS{Value: fmt.Sprintf("item-%d", i)},
			},
		})
		require.NoError(t, err)
	}

	out, err := client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String("scan-table"),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 5, out.Count, "expected 5 items from scan")
}

// ---------------------------------------------------------------------------
// S3 – additional tests
// ---------------------------------------------------------------------------

func newS3Client(cm *CloudMock) *s3.Client {
	return s3.NewFromConfig(cm.Config(), func(o *s3.Options) {
		o.UsePathStyle = true
	})
}

func TestInProcess_S3_ListObjects(t *testing.T) {
	cm := New()
	defer cm.Close()
	client := newS3Client(cm)

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("list-bucket")})
	require.NoError(t, err)

	keys := []string{"alpha.txt", "beta.txt", "gamma.txt"}
	for _, k := range keys {
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String("list-bucket"),
			Key:    aws.String(k),
			Body:   strings.NewReader("content"),
		})
		require.NoError(t, err)
	}

	out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String("list-bucket"),
	})
	require.NoError(t, err)
	assert.Len(t, out.Contents, 3)

	found := make(map[string]bool)
	for _, obj := range out.Contents {
		found[*obj.Key] = true
	}
	for _, k := range keys {
		assert.True(t, found[k], "key %q not in listing", k)
	}
}

func TestInProcess_S3_DeleteObject(t *testing.T) {
	cm := New()
	defer cm.Close()
	client := newS3Client(cm)

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("del-bucket")})
	require.NoError(t, err)

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String("del-bucket"),
		Key:    aws.String("to-delete.txt"),
		Body:   strings.NewReader("bye"),
	})
	require.NoError(t, err)

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String("del-bucket"),
		Key:    aws.String("to-delete.txt"),
	})
	require.NoError(t, err)

	// Object should no longer exist.
	out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String("del-bucket"),
	})
	require.NoError(t, err)
	assert.Len(t, out.Contents, 0, "bucket should be empty after delete")
}

func TestInProcess_S3_CopyObject(t *testing.T) {
	cm := New()
	defer cm.Close()
	client := newS3Client(cm)

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("copy-bucket")})
	require.NoError(t, err)

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String("copy-bucket"),
		Key:    aws.String("src.txt"),
		Body:   strings.NewReader("copy-me"),
	})
	require.NoError(t, err)

	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String("copy-bucket"),
		Key:        aws.String("dst.txt"),
		CopySource: aws.String("copy-bucket/src.txt"),
	})
	require.NoError(t, err)

	// Read the copy and verify content matches.
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String("copy-bucket"),
		Key:    aws.String("dst.txt"),
	})
	require.NoError(t, err)
	body, err := io.ReadAll(out.Body)
	require.NoError(t, err)
	assert.Equal(t, "copy-me", string(body))
}

// ---------------------------------------------------------------------------
// SQS – additional tests
// ---------------------------------------------------------------------------

func TestInProcess_SQS_BatchSend(t *testing.T) {
	cm := New()
	defer cm.Close()
	client := sqs.NewFromConfig(cm.Config())

	createOut, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("batch-queue"),
	})
	require.NoError(t, err)
	queueURL := createOut.QueueUrl

	entries := []sqsTypes.SendMessageBatchRequestEntry{
		{Id: aws.String("1"), MessageBody: aws.String("msg-one")},
		{Id: aws.String("2"), MessageBody: aws.String("msg-two")},
		{Id: aws.String("3"), MessageBody: aws.String("msg-three")},
	}

	batchOut, err := client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: queueURL,
		Entries:  entries,
	})
	require.NoError(t, err)
	assert.Len(t, batchOut.Successful, 3, "all 3 messages should succeed")
	assert.Empty(t, batchOut.Failed, "no messages should fail")
}

func TestInProcess_SQS_ChangeVisibility(t *testing.T) {
	cm := New()
	defer cm.Close()
	client := sqs.NewFromConfig(cm.Config())

	createOut, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("vis-queue"),
	})
	require.NoError(t, err)
	queueURL := createOut.QueueUrl

	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    queueURL,
		MessageBody: aws.String("visibility-test"),
	})
	require.NoError(t, err)

	recvOut, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl: queueURL,
	})
	require.NoError(t, err)
	require.Len(t, recvOut.Messages, 1)

	receiptHandle := recvOut.Messages[0].ReceiptHandle
	require.NotNil(t, receiptHandle)

	// Extend visibility timeout — should not error.
	_, err = client.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          queueURL,
		ReceiptHandle:     receiptHandle,
		VisibilityTimeout: 30,
	})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Multi-service and concurrency tests
// ---------------------------------------------------------------------------

func TestInProcess_MultiService(t *testing.T) {
	cm := New()
	defer cm.Close()

	ddbClient := dynamodb.NewFromConfig(cm.Config())
	s3Client := newS3Client(cm)
	sqsClient := sqs.NewFromConfig(cm.Config())

	// DynamoDB: create table + put item.
	_, err := ddbClient.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("multi-table"),
		KeySchema: []ddbTypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: ddbTypes.KeyTypeHash},
		},
		AttributeDefinitions: []ddbTypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: ddbTypes.ScalarAttributeTypeS},
		},
		BillingMode: ddbTypes.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("multi-table"),
		Item: map[string]ddbTypes.AttributeValue{
			"pk":  &ddbTypes.AttributeValueMemberS{Value: "row1"},
			"val": &ddbTypes.AttributeValueMemberS{Value: "fromDDB"},
		},
	})
	require.NoError(t, err)

	// S3: create bucket + put object.
	_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("multi-bucket")})
	require.NoError(t, err)

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String("multi-bucket"),
		Key:    aws.String("data.txt"),
		Body:   strings.NewReader("multiservice"),
	})
	require.NoError(t, err)

	// SQS: create queue + send message.
	createOut, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("multi-queue"),
	})
	require.NoError(t, err)

	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    createOut.QueueUrl,
		MessageBody: aws.String("multi-service-msg"),
	})
	require.NoError(t, err)

	// Verify all three services round-trip correctly.
	getOut, err := ddbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("multi-table"),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "row1"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "fromDDB", getOut.Item["val"].(*ddbTypes.AttributeValueMemberS).Value)

	s3Out, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String("multi-bucket"),
		Key:    aws.String("data.txt"),
	})
	require.NoError(t, err)
	body, _ := io.ReadAll(s3Out.Body)
	assert.Equal(t, "multiservice", string(body))

	recvOut, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl: createOut.QueueUrl,
	})
	require.NoError(t, err)
	require.Len(t, recvOut.Messages, 1)
	assert.Equal(t, "multi-service-msg", *recvOut.Messages[0].Body)
}

func TestInProcess_Concurrent(t *testing.T) {
	cm := New()
	defer cm.Close()

	s3Client := newS3Client(cm)
	sqsClient := sqs.NewFromConfig(cm.Config())

	// Pre-create bucket and queue so goroutines only do reads/writes.
	_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("concurrent-bucket")})
	require.NoError(t, err)

	createOut, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("concurrent-queue"),
	})
	require.NoError(t, err)
	queueURL := createOut.QueueUrl

	const goroutines = 100
	var wg sync.WaitGroup
	errs := make(chan error, goroutines*2)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			// S3 put.
			_, e := s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: aws.String("concurrent-bucket"),
				Key:    aws.String(fmt.Sprintf("obj-%d.txt", n)),
				Body:   strings.NewReader(fmt.Sprintf("data-%d", n)),
			})
			if e != nil {
				errs <- fmt.Errorf("s3 put goroutine %d: %w", n, e)
			}

			// SQS send.
			_, e = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
				QueueUrl:    queueURL,
				MessageBody: aws.String(fmt.Sprintf("msg-%d", n)),
			})
			if e != nil {
				errs <- fmt.Errorf("sqs send goroutine %d: %w", n, e)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	// Spot-check: list objects — expect 100.
	listOut, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String("concurrent-bucket"),
	})
	require.NoError(t, err)
	assert.Len(t, listOut.Contents, goroutines, "expected %d objects", goroutines)
}

func TestInProcess_StateSnapshot(t *testing.T) {
	// Verify that two separate CloudMock instances have completely isolated state.
	// Instance A creates resources; instance B must not see them.
	cmA := New()
	defer cmA.Close()

	cmB := New()
	defer cmB.Close()

	clientA := dynamodb.NewFromConfig(cmA.Config())
	clientB := dynamodb.NewFromConfig(cmB.Config())

	// Create a table in A.
	_, err := clientA.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("isolated-table"),
		KeySchema: []ddbTypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: ddbTypes.KeyTypeHash},
		},
		AttributeDefinitions: []ddbTypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: ddbTypes.ScalarAttributeTypeS},
		},
		BillingMode: ddbTypes.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	// Put an item in A.
	_, err = clientA.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("isolated-table"),
		Item: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "secret"},
		},
	})
	require.NoError(t, err)

	// Verify the item exists in A.
	getA, err := clientA.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("isolated-table"),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "secret"},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, getA.Item, "item should exist in instance A")

	// The same table must NOT exist in B (requesting it should error).
	_, err = clientB.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String("isolated-table"),
	})
	assert.Error(t, err, "instance B should not see instance A's table")
}

// ---------------------------------------------------------------------------
// Chaos injection tests
// ---------------------------------------------------------------------------

func TestChaos_InjectError(t *testing.T) {
	cm := New()
	defer cm.Close()

	// Inject a 503 error on all S3 actions.
	err := cm.InjectFault("s3", "*", "error", WithStatusCode(503), WithMessage("injected service unavailable"))
	require.NoError(t, err)

	client := newS3Client(cm)

	// Create a bucket first (pre-fault for comparison).
	// Now that the fault is active, any S3 call should return 503.
	_, callErr := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.Error(t, callErr, "expected S3 call to fail with injected 503 fault")
}

func TestChaos_InjectThrottle(t *testing.T) {
	cm := New()
	defer cm.Close()

	// Inject a throttle fault on DynamoDB.
	err := cm.InjectFault("dynamodb", "*", "throttle", WithMessage("too many requests"))
	require.NoError(t, err)

	client := dynamodb.NewFromConfig(cm.Config())

	// Any DynamoDB call should return 429 ThrottlingException.
	_, callErr := client.ListTables(ctx, &dynamodb.ListTablesInput{})
	require.Error(t, callErr, "expected DynamoDB call to fail with throttle fault")
	assert.Contains(t, callErr.Error(), "429", "expected 429 status in error")
}

func TestChaos_ClearFaults(t *testing.T) {
	cm := New()
	defer cm.Close()

	// Inject an S3 fault.
	err := cm.InjectFault("s3", "*", "error", WithStatusCode(503))
	require.NoError(t, err)

	client := newS3Client(cm)

	// Verify the fault is active.
	_, callErr := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.Error(t, callErr, "fault should be active before clearing")

	// Clear all faults.
	err = cm.ClearFaults()
	require.NoError(t, err)

	// Now S3 calls should succeed.
	_, callErr = client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, callErr, "S3 should work normally after clearing faults")
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

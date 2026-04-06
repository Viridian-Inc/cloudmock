package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

func main() {
	ctx := context.Background()
	endpoint := "http://localhost:4566"

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		panic(err)
	}

	ddb := dynamodb.NewFromConfig(cfg)
	s3c := s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true })
	snsc := sns.NewFromConfig(cfg)
	sqsc := sqs.NewFromConfig(cfg)

	fmt.Println("=== CloudMock Todo Demo (Go) ===")
	fmt.Println()

	// 1. Create resources
	fmt.Println("Creating AWS resources...")
	ddb.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:            aws.String("todos"),
		KeySchema:            []ddbtypes.KeySchemaElement{{AttributeName: aws.String("id"), KeyType: ddbtypes.KeyTypeHash}},
		AttributeDefinitions: []ddbtypes.AttributeDefinition{{AttributeName: aws.String("id"), AttributeType: ddbtypes.ScalarAttributeTypeS}},
		BillingMode:          ddbtypes.BillingModePayPerRequest,
	})
	s3c.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("todo-attachments")})
	topicOut, _ := snsc.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String("todo-completed")})
	topicArn := topicOut.TopicArn
	queueOut, _ := sqsc.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String("todo-notifications")})
	queueURL := queueOut.QueueUrl
	attrsOut, _ := sqsc.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       queueURL,
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	snsc.Subscribe(ctx, &sns.SubscribeInput{TopicArn: topicArn, Protocol: aws.String("sqs"), Endpoint: aws.String(attrsOut.Attributes["QueueArn"])})
	fmt.Println("  DynamoDB table, S3 bucket, SNS topic, SQS queue created.")
	fmt.Println()

	// 2. Create a todo
	todoID := uuid.NewString()
	fmt.Printf("Creating todo: \"%s...\"\n", todoID[:8])
	now := time.Now().UTC().Format(time.RFC3339)
	ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("todos"),
		Item: map[string]ddbtypes.AttributeValue{
			"id":          &ddbtypes.AttributeValueMemberS{Value: todoID},
			"title":       &ddbtypes.AttributeValueMemberS{Value: "Try CloudMock"},
			"description": &ddbtypes.AttributeValueMemberS{Value: "Run the quickstart and explore the DevTools dashboard"},
			"status":      &ddbtypes.AttributeValueMemberS{Value: "pending"},
			"createdAt":   &ddbtypes.AttributeValueMemberS{Value: now},
		},
	})
	fmt.Println("  Todo created in DynamoDB.")
	fmt.Println()

	// 3. Attach an image
	fmt.Println("Uploading image to S3...")
	imageKey := fmt.Sprintf("todos/%s/image.jpg", todoID)
	imageData, _ := os.ReadFile("../sample.jpg")
	s3c.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String("todo-attachments"),
		Key:         aws.String(imageKey),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String("image/jpeg"),
	})
	ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String("todos"),
		Key:                       map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: todoID}},
		UpdateExpression:          aws.String("SET imageKey = :key"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{":key": &ddbtypes.AttributeValueMemberS{Value: imageKey}},
	})
	fmt.Printf("  Image uploaded to s3://todo-attachments/%s\n\n", imageKey)

	// 4. Complete the todo
	fmt.Println("Marking todo as complete...")
	ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String("todos"),
		Key:                       map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: todoID}},
		UpdateExpression:          aws.String("SET #s = :s"),
		ExpressionAttributeNames:  map[string]string{"#s": "status"},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{":s": &ddbtypes.AttributeValueMemberS{Value: "completed"}},
	})
	completedAt := time.Now().UTC().Format(time.RFC3339)
	msgBody, _ := json.Marshal(map[string]string{"todoId": todoID, "title": "Try CloudMock", "completedAt": completedAt})
	snsc.Publish(ctx, &sns.PublishInput{TopicArn: topicArn, Subject: aws.String("Todo Completed"), Message: aws.String(string(msgBody))})
	fmt.Println("  Todo completed. Notification sent via SNS.")
	fmt.Println()

	// 5. Poll for notification
	fmt.Println("Polling SQS for notifications...")
	time.Sleep(500 * time.Millisecond)
	msgOut, _ := sqsc.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{QueueUrl: queueURL, MaxNumberOfMessages: 10, WaitTimeSeconds: 2})
	for _, msg := range msgOut.Messages {
		var snsEnvelope struct{ Message string }
		json.Unmarshal([]byte(*msg.Body), &snsEnvelope)
		var notification map[string]string
		json.Unmarshal([]byte(snsEnvelope.Message), &notification)
		fmt.Printf("  Notification received: \"%s\" completed at %s\n", notification["title"], notification["completedAt"])
		sqsc.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: queueURL, ReceiptHandle: msg.ReceiptHandle})
	}

	// 6. Verify
	fmt.Println()
	fmt.Println("Final state:")
	getOut, _ := ddb.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String("todos"), Key: map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: todoID}}})
	title := getOut.Item["title"].(*ddbtypes.AttributeValueMemberS).Value
	status := getOut.Item["status"].(*ddbtypes.AttributeValueMemberS).Value
	imgKey := ""
	if v, ok := getOut.Item["imageKey"]; ok {
		imgKey = v.(*ddbtypes.AttributeValueMemberS).Value
	}
	fmt.Printf("  Todo: \"%s\" — status: %s, image: %s\n", title, status, imgKey)

	fmt.Println()
	fmt.Println("=== Done! Open http://localhost:4500 to explore the DevTools. ===")
}

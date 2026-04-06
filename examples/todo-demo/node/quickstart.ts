import { DynamoDBClient, CreateTableCommand, PutItemCommand, GetItemCommand, UpdateItemCommand } from "@aws-sdk/client-dynamodb";
import { S3Client, CreateBucketCommand, PutObjectCommand } from "@aws-sdk/client-s3";
import { SNSClient, CreateTopicCommand, SubscribeCommand, PublishCommand } from "@aws-sdk/client-sns";
import { SQSClient, CreateQueueCommand, GetQueueAttributesCommand, ReceiveMessageCommand, DeleteMessageCommand } from "@aws-sdk/client-sqs";
import { readFileSync } from "fs";
import { randomUUID } from "crypto";

const endpoint = "http://localhost:4566";
const config = { region: "us-east-1", endpoint, credentials: { accessKeyId: "test", secretAccessKey: "test" } };

const dynamodb = new DynamoDBClient(config);
const s3 = new S3Client({ ...config, forcePathStyle: true });
const sns = new SNSClient(config);
const sqs = new SQSClient(config);

async function main() {
  console.log("=== CloudMock Todo Demo (Node.js) ===\n");

  // 1. Create resources
  console.log("Creating AWS resources...");
  try { await dynamodb.send(new CreateTableCommand({
    TableName: "todos",
    KeySchema: [{ AttributeName: "id", KeyType: "HASH" }],
    AttributeDefinitions: [{ AttributeName: "id", AttributeType: "S" }],
    BillingMode: "PAY_PER_REQUEST",
  })); } catch (e: any) { if (e.name !== "ResourceInUseException") throw e; }
  try { await s3.send(new CreateBucketCommand({ Bucket: "todo-attachments" })); } catch { /* bucket may already exist */ }
  const { TopicArn } = await sns.send(new CreateTopicCommand({ Name: "todo-completed" }));
  const { QueueUrl } = await sqs.send(new CreateQueueCommand({ QueueName: "todo-notifications" }));
  const { Attributes } = await sqs.send(new GetQueueAttributesCommand({ QueueUrl, AttributeNames: ["QueueArn"] }));
  await sns.send(new SubscribeCommand({ TopicArn, Protocol: "sqs", Endpoint: Attributes!.QueueArn }));
  console.log("  DynamoDB table, S3 bucket, SNS topic, SQS queue created.\n");

  // 2. Create a todo
  const todoId = randomUUID();
  console.log(`Creating todo: "${todoId.slice(0, 8)}..."`);
  await dynamodb.send(new PutItemCommand({
    TableName: "todos",
    Item: {
      id: { S: todoId },
      title: { S: "Try CloudMock" },
      description: { S: "Run the quickstart and explore the DevTools dashboard" },
      status: { S: "pending" },
      createdAt: { S: new Date().toISOString() },
    },
  }));
  console.log("  Todo created in DynamoDB.\n");

  // 3. Attach an image
  console.log("Uploading image to S3...");
  const imageKey = `todos/${todoId}/image.jpg`;
  const imageData = readFileSync("../sample.jpg");
  await s3.send(new PutObjectCommand({ Bucket: "todo-attachments", Key: imageKey, Body: imageData, ContentType: "image/jpeg" }));
  await dynamodb.send(new UpdateItemCommand({
    TableName: "todos",
    Key: { id: { S: todoId } },
    UpdateExpression: "SET imageKey = :key",
    ExpressionAttributeValues: { ":key": { S: imageKey } },
  }));
  console.log(`  Image uploaded to s3://todo-attachments/${imageKey}\n`);

  // 4. Complete the todo
  console.log("Marking todo as complete...");
  await dynamodb.send(new UpdateItemCommand({
    TableName: "todos",
    Key: { id: { S: todoId } },
    UpdateExpression: "SET #s = :s",
    ExpressionAttributeNames: { "#s": "status" },
    ExpressionAttributeValues: { ":s": { S: "completed" } },
  }));
  await sns.send(new PublishCommand({
    TopicArn,
    Subject: "Todo Completed",
    Message: JSON.stringify({ todoId, title: "Try CloudMock", completedAt: new Date().toISOString() }),
  }));
  console.log("  Todo completed. Notification sent via SNS.\n");

  // 5. Poll for notification
  console.log("Polling SQS for notifications...");
  await new Promise((r) => setTimeout(r, 500));
  const { Messages } = await sqs.send(new ReceiveMessageCommand({ QueueUrl, MaxNumberOfMessages: 10, WaitTimeSeconds: 2 }));
  if (Messages && Messages.length > 0) {
    for (const msg of Messages) {
      const body = JSON.parse(msg.Body!);
      const notification = JSON.parse(body.Message);
      console.log(`  Notification received: "${notification.title}" completed at ${notification.completedAt}`);
      await sqs.send(new DeleteMessageCommand({ QueueUrl, ReceiptHandle: msg.ReceiptHandle }));
    }
  }

  // 6. Verify
  console.log("\nFinal state:");
  const { Item } = await dynamodb.send(new GetItemCommand({ TableName: "todos", Key: { id: { S: todoId } } }));
  console.log(`  Todo: "${Item!.title.S}" — status: ${Item!.status.S}, image: ${Item!.imageKey?.S || "none"}`);

  console.log("\n=== Done! Open http://localhost:4500 to explore the DevTools. ===");
}

main().catch(console.error);

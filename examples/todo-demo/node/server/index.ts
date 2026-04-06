import express from "express";
import { DynamoDBClient, CreateTableCommand, PutItemCommand, GetItemCommand, UpdateItemCommand, ScanCommand } from "@aws-sdk/client-dynamodb";
import { S3Client, CreateBucketCommand, PutObjectCommand, GetObjectCommand } from "@aws-sdk/client-s3";
import { SNSClient, CreateTopicCommand, SubscribeCommand, PublishCommand } from "@aws-sdk/client-sns";
import { SQSClient, CreateQueueCommand, GetQueueAttributesCommand, ReceiveMessageCommand, DeleteMessageCommand } from "@aws-sdk/client-sqs";
import { randomUUID } from "crypto";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const endpoint = process.env.CLOUDMOCK_ENDPOINT || "http://localhost:4566";
const config = { region: "us-east-1", endpoint, credentials: { accessKeyId: "test", secretAccessKey: "test" } };

const dynamodb = new DynamoDBClient(config);
const s3 = new S3Client({ ...config, forcePathStyle: true });
const sns = new SNSClient(config);
const sqs = new SQSClient(config);

let topicArn: string;
let queueUrl: string;

async function setup() {
  try { await dynamodb.send(new CreateTableCommand({
    TableName: "todos",
    KeySchema: [{ AttributeName: "id", KeyType: "HASH" }],
    AttributeDefinitions: [{ AttributeName: "id", AttributeType: "S" }],
    BillingMode: "PAY_PER_REQUEST",
  })); } catch (e: any) { if (e.name !== "ResourceInUseException") throw e; }

  try { await s3.send(new CreateBucketCommand({ Bucket: "todo-attachments" })); } catch { /* bucket may already exist */ }

  const topic = await sns.send(new CreateTopicCommand({ Name: "todo-completed" }));
  topicArn = topic.TopicArn!;

  const queue = await sqs.send(new CreateQueueCommand({ QueueName: "todo-notifications" }));
  queueUrl = queue.QueueUrl!;

  const attrs = await sqs.send(new GetQueueAttributesCommand({ QueueUrl: queueUrl, AttributeNames: ["QueueArn"] }));
  await sns.send(new SubscribeCommand({ TopicArn: topicArn, Protocol: "sqs", Endpoint: attrs.Attributes!.QueueArn }));
}

const app = express();
app.use((_req, res, next) => {
  res.header("Access-Control-Allow-Origin", "*");
  res.header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
  res.header("Access-Control-Allow-Headers", "Content-Type");
  if (_req.method === "OPTIONS") return res.sendStatus(204);
  next();
});
app.use(express.json());

app.post("/todos", async (req, res) => {
  const { title, description } = req.body;
  const id = randomUUID();
  await dynamodb.send(new PutItemCommand({
    TableName: "todos",
    Item: {
      id: { S: id },
      title: { S: title },
      description: { S: description || "" },
      status: { S: "pending" },
      createdAt: { S: new Date().toISOString() },
    },
  }));
  res.status(201).json({ id, title, description, status: "pending" });
});

app.get("/todos", async (_req, res) => {
  const { Items } = await dynamodb.send(new ScanCommand({ TableName: "todos" }));
  const todos = (Items || []).map((item) => ({
    id: item.id.S, title: item.title.S, description: item.description?.S,
    status: item.status.S, imageKey: item.imageKey?.S, createdAt: item.createdAt.S,
  }));
  res.json(todos);
});

app.get("/todos/:id", async (req, res) => {
  const { Item } = await dynamodb.send(new GetItemCommand({ TableName: "todos", Key: { id: { S: req.params.id } } }));
  if (!Item) return res.status(404).json({ error: "Not found" });
  res.json({
    id: Item.id.S, title: Item.title.S, description: Item.description?.S,
    status: Item.status.S, imageKey: Item.imageKey?.S, createdAt: Item.createdAt.S,
  });
});

app.get("/todos/:id/image", async (req, res) => {
  const imageKey = `todos/${req.params.id}/image.jpg`;
  try {
    const obj = await s3.send(new GetObjectCommand({ Bucket: "todo-attachments", Key: imageKey }));
    res.header("Content-Type", obj.ContentType || "image/jpeg");
    const chunks: Buffer[] = [];
    for await (const chunk of obj.Body as any) chunks.push(chunk);
    res.send(Buffer.concat(chunks));
  } catch {
    res.status(404).json({ error: "No image" });
  }
});

app.put("/todos/:id/image", express.raw({ type: "image/*", limit: "10mb" }), async (req, res) => {
  const imageKey = `todos/${req.params.id}/image.jpg`;
  await s3.send(new PutObjectCommand({ Bucket: "todo-attachments", Key: imageKey, Body: req.body, ContentType: req.headers["content-type"] }));
  await dynamodb.send(new UpdateItemCommand({
    TableName: "todos",
    Key: { id: { S: req.params.id } },
    UpdateExpression: "SET imageKey = :key",
    ExpressionAttributeValues: { ":key": { S: imageKey } },
  }));
  res.json({ imageKey });
});

app.put("/todos/:id/complete", async (req, res) => {
  const { Item } = await dynamodb.send(new GetItemCommand({ TableName: "todos", Key: { id: { S: req.params.id } } }));
  if (!Item) return res.status(404).json({ error: "Not found" });

  await dynamodb.send(new UpdateItemCommand({
    TableName: "todos",
    Key: { id: { S: req.params.id } },
    UpdateExpression: "SET #s = :s",
    ExpressionAttributeNames: { "#s": "status" },
    ExpressionAttributeValues: { ":s": { S: "completed" } },
  }));
  await sns.send(new PublishCommand({
    TopicArn: topicArn,
    Subject: "Todo Completed",
    Message: JSON.stringify({ todoId: req.params.id, title: Item.title.S, completedAt: new Date().toISOString() }),
  }));
  res.json({ status: "completed" });
});

app.get("/notifications", async (_req, res) => {
  const { Messages } = await sqs.send(new ReceiveMessageCommand({ QueueUrl: queueUrl, MaxNumberOfMessages: 10, WaitTimeSeconds: 1 }));
  const notifications = [];
  for (const msg of Messages || []) {
    const body = JSON.parse(msg.Body!);
    notifications.push(JSON.parse(body.Message));
    await sqs.send(new DeleteMessageCommand({ QueueUrl: queueUrl, ReceiptHandle: msg.ReceiptHandle }));
  }
  res.json(notifications);
});

const __dirname = dirname(fileURLToPath(import.meta.url));
app.use(express.static(join(__dirname, "../../web")));

setup().then(() => {
  app.listen(3000, () => console.log("Todo API running on http://localhost:3000\nWeb UI:  http://localhost:3000"));
});

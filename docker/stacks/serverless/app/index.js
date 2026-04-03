const express = require("express");
const { DynamoDBClient, PutItemCommand, GetItemCommand } = require("@aws-sdk/client-dynamodb");
const { SQSClient, SendMessageCommand } = require("@aws-sdk/client-sqs");

const app = express();
app.use(express.json());

const endpoint = process.env.AWS_ENDPOINT_URL || "http://cloudmock:4566";
const region = "us-east-1";
const credentials = { accessKeyId: "test", secretAccessKey: "test" };

const dynamo = new DynamoDBClient({ endpoint, region, credentials });
const sqs = new SQSClient({ endpoint, region, credentials });

const TABLE = "items";
const QUEUE_URL = `${endpoint}/000000000000/jobs`;

app.get("/health", (_req, res) => res.json({ status: "ok" }));

app.post("/items", async (req, res) => {
  const { id, data } = req.body;
  await dynamo.send(new PutItemCommand({
    TableName: TABLE,
    Item: { id: { S: id }, data: { S: JSON.stringify(data) } },
  }));
  res.json({ id });
});

app.get("/items/:id", async (req, res) => {
  const result = await dynamo.send(new GetItemCommand({
    TableName: TABLE,
    Key: { id: { S: req.params.id } },
  }));
  if (!result.Item) return res.status(404).json({ error: "not found" });
  res.json({ id: result.Item.id.S, data: JSON.parse(result.Item.data.S) });
});

app.post("/jobs", async (req, res) => {
  await sqs.send(new SendMessageCommand({
    QueueUrl: QUEUE_URL,
    MessageBody: JSON.stringify(req.body),
  }));
  res.json({ queued: true });
});

app.listen(3000, () => console.log("API listening on :3000"));

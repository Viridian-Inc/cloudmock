const express = require("express");
const { DynamoDBClient, PutItemCommand } = require("@aws-sdk/client-dynamodb");
const { SNSClient, PublishCommand } = require("@aws-sdk/client-sns");
const { randomUUID } = require("crypto");

const app = express();
app.use(express.json());

const endpoint = process.env.AWS_ENDPOINT_URL || "http://cloudmock:4566";
const cfg = { endpoint, region: "us-east-1", credentials: { accessKeyId: "test", secretAccessKey: "test" } };

const dynamo = new DynamoDBClient(cfg);
const sns = new SNSClient(cfg);

app.get("/health", (_req, res) => res.json({ status: "ok" }));

app.post("/orders", async (req, res) => {
  const id = randomUUID();
  const order = { id, ...req.body, status: "pending", createdAt: new Date().toISOString() };

  await dynamo.send(new PutItemCommand({
    TableName: "orders",
    Item: {
      id: { S: order.id },
      customerId: { S: order.customerId || "unknown" },
      amount: { N: String(order.amount || 0) },
      status: { S: order.status },
      createdAt: { S: order.createdAt },
    },
  }));

  await sns.send(new PublishCommand({
    TopicArn: process.env.ORDERS_TOPIC_ARN,
    Message: JSON.stringify(order),
    Subject: "order.created",
  }));

  res.status(201).json(order);
});

app.listen(3001, () => console.log("Order service on :3001"));

const express = require("express");
const { DynamoDBClient, PutItemCommand, GetItemCommand, ScanCommand } = require("@aws-sdk/client-dynamodb");
const { S3Client, PutObjectCommand, GetObjectCommand } = require("@aws-sdk/client-s3");
const { getSignedUrl } = require("@aws-sdk/s3-request-presigner");
const { randomUUID } = require("crypto");

const app = express();
app.use(express.json());
app.use((_req, res, next) => {
  res.header("Access-Control-Allow-Origin", "*");
  res.header("Access-Control-Allow-Headers", "Content-Type");
  next();
});

const endpoint = process.env.AWS_ENDPOINT_URL || "http://cloudmock:4566";
const awsCfg = {
  endpoint,
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
  forcePathStyle: true,
};
const dynamo = new DynamoDBClient(awsCfg);
const s3 = new S3Client(awsCfg);

const TABLE = "notes";
const BUCKET = "note-attachments";

app.get("/health", (_req, res) => res.json({ status: "ok" }));

app.get("/notes", async (_req, res) => {
  const result = await dynamo.send(new ScanCommand({ TableName: TABLE }));
  const notes = (result.Items || []).map(item => ({
    id: item.id.S,
    title: item.title.S,
    body: item.body.S,
    createdAt: item.createdAt.S,
  }));
  res.json(notes);
});

app.post("/notes", async (req, res) => {
  const id = randomUUID();
  const note = { id, title: req.body.title || "Untitled", body: req.body.body || "", createdAt: new Date().toISOString() };
  await dynamo.send(new PutItemCommand({
    TableName: TABLE,
    Item: { id: { S: note.id }, title: { S: note.title }, body: { S: note.body }, createdAt: { S: note.createdAt } },
  }));
  res.status(201).json(note);
});

app.get("/notes/:id", async (req, res) => {
  const result = await dynamo.send(new GetItemCommand({
    TableName: TABLE,
    Key: { id: { S: req.params.id } },
  }));
  if (!result.Item) return res.status(404).json({ error: "not found" });
  const item = result.Item;
  res.json({ id: item.id.S, title: item.title.S, body: item.body.S, createdAt: item.createdAt.S });
});

app.post("/upload", async (req, res) => {
  const key = `attachments/${randomUUID()}-${req.body.filename}`;
  await s3.send(new PutObjectCommand({ Bucket: BUCKET, Key: key, Body: req.body.content || "" }));
  res.json({ key });
});

app.listen(3000, () => console.log("API on :3000"));

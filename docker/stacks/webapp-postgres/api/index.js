const express = require("express");
const { Pool } = require("pg");
const { S3Client, PutObjectCommand, GetObjectCommand } = require("@aws-sdk/client-s3");
const { SQSClient, SendMessageCommand } = require("@aws-sdk/client-sqs");
const { randomUUID } = require("crypto");

const app = express();
app.use(express.json());

const pool = new Pool({ connectionString: process.env.DATABASE_URL });

const endpoint = process.env.AWS_ENDPOINT_URL || "http://cloudmock:4566";
const awsCfg = {
  endpoint,
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
  forcePathStyle: true,
};
const s3 = new S3Client(awsCfg);
const sqs = new SQSClient(awsCfg);

const BUCKET = "user-files";
const QUEUE_URL = `${endpoint}/000000000000/jobs`;

async function initDb() {
  await pool.query(`
    CREATE TABLE IF NOT EXISTS users (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      email TEXT UNIQUE NOT NULL,
      name TEXT,
      created_at TIMESTAMPTZ DEFAULT NOW()
    )
  `);
}

app.get("/health", async (_req, res) => {
  try {
    await pool.query("SELECT 1");
    res.json({ status: "ok", db: "connected" });
  } catch {
    res.status(503).json({ status: "error", db: "disconnected" });
  }
});

// Users — stored in Postgres
app.post("/users", async (req, res) => {
  const { email, name } = req.body;
  const result = await pool.query(
    "INSERT INTO users (email, name) VALUES ($1, $2) RETURNING *",
    [email, name]
  );
  res.status(201).json(result.rows[0]);
});

app.get("/users/:id", async (req, res) => {
  const result = await pool.query("SELECT * FROM users WHERE id = $1", [req.params.id]);
  if (!result.rows.length) return res.status(404).json({ error: "not found" });
  res.json(result.rows[0]);
});

// File uploads — stored in S3
app.post("/users/:id/files", async (req, res) => {
  const key = `users/${req.params.id}/${randomUUID()}.json`;
  await s3.send(new PutObjectCommand({
    Bucket: BUCKET,
    Key: key,
    Body: JSON.stringify(req.body),
    ContentType: "application/json",
  }));
  res.json({ key });
});

// Background jobs — queued in SQS
app.post("/jobs", async (req, res) => {
  await sqs.send(new SendMessageCommand({
    QueueUrl: QUEUE_URL,
    MessageBody: JSON.stringify({ ...req.body, enqueuedAt: new Date().toISOString() }),
  }));
  res.json({ queued: true });
});

initDb().then(() => {
  app.listen(3000, () => console.log("API on :3000"));
}).catch(err => { console.error(err); process.exit(1); });

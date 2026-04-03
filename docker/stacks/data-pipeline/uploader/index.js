const { S3Client, PutObjectCommand } = require("@aws-sdk/client-s3");
const { SQSClient, SendMessageCommand } = require("@aws-sdk/client-sqs");
const { randomUUID } = require("crypto");

const endpoint = process.env.AWS_ENDPOINT_URL || "http://cloudmock:4566";
const cfg = {
  endpoint,
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
  forcePathStyle: true,
};

const s3 = new S3Client(cfg);
const sqs = new SQSClient(cfg);

const BUCKET = process.env.S3_BUCKET || "data-ingestion";
const QUEUE_URL = process.env.QUEUE_URL || `${endpoint}/000000000000/ingest`;

async function uploadFile(content, filename) {
  const key = `uploads/${new Date().toISOString().slice(0, 10)}/${filename}`;
  await s3.send(new PutObjectCommand({
    Bucket: BUCKET,
    Key: key,
    Body: content,
    ContentType: "application/json",
  }));

  await sqs.send(new SendMessageCommand({
    QueueUrl: QUEUE_URL,
    MessageBody: JSON.stringify({ bucket: BUCKET, key, filename, uploadedAt: new Date().toISOString() }),
  }));

  console.log(`Uploaded s3://${BUCKET}/${key} and notified queue`);
  return key;
}

async function main() {
  // Simulate uploading a batch of data files
  for (let i = 1; i <= 5; i++) {
    const id = randomUUID();
    const record = { id, index: i, value: Math.random() * 100, timestamp: new Date().toISOString() };
    await uploadFile(JSON.stringify(record), `record-${id}.json`);
    await new Promise(r => setTimeout(r, 500));
  }
  console.log("Upload batch complete");
}

main().catch(err => { console.error(err); process.exit(1); });

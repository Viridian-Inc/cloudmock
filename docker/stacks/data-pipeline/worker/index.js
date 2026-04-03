const { S3Client, GetObjectCommand } = require("@aws-sdk/client-s3");
const { SQSClient, ReceiveMessageCommand, DeleteMessageCommand } = require("@aws-sdk/client-sqs");
const { DynamoDBClient, PutItemCommand } = require("@aws-sdk/client-dynamodb");

const endpoint = process.env.AWS_ENDPOINT_URL || "http://cloudmock:4566";
const cfg = {
  endpoint,
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
  forcePathStyle: true,
};

const s3 = new S3Client(cfg);
const sqs = new SQSClient(cfg);
const dynamo = new DynamoDBClient(cfg);

const QUEUE_URL = process.env.QUEUE_URL || `${endpoint}/000000000000/ingest`;
const TABLE = process.env.DYNAMO_TABLE || "processed-records";

async function processMessage(msg) {
  const { bucket, key, filename } = JSON.parse(msg.Body);

  // Fetch file from S3
  const obj = await s3.send(new GetObjectCommand({ Bucket: bucket, Key: key }));
  const body = await obj.Body.transformToString();
  const record = JSON.parse(body);

  // Transform: add computed fields
  const processed = {
    ...record,
    processedAt: new Date().toISOString(),
    normalized: (record.value / 100).toFixed(4),
    status: record.value > 50 ? "high" : "low",
  };

  // Write result to DynamoDB
  await dynamo.send(new PutItemCommand({
    TableName: TABLE,
    Item: {
      id: { S: processed.id },
      filename: { S: filename },
      normalized: { N: processed.normalized },
      status: { S: processed.status },
      processedAt: { S: processed.processedAt },
    },
  }));

  console.log(`Processed ${key}: status=${processed.status}, normalized=${processed.normalized}`);
}

async function poll() {
  console.log("Worker polling for messages...");
  while (true) {
    const resp = await sqs.send(new ReceiveMessageCommand({
      QueueUrl: QUEUE_URL,
      WaitTimeSeconds: 5,
      MaxNumberOfMessages: 10,
    }));

    for (const msg of resp.Messages || []) {
      try {
        await processMessage(msg);
        await sqs.send(new DeleteMessageCommand({ QueueUrl: QUEUE_URL, ReceiptHandle: msg.ReceiptHandle }));
      } catch (err) {
        console.error("Error processing message:", err.message);
      }
    }
  }
}

poll().catch(err => { console.error(err); process.exit(1); });

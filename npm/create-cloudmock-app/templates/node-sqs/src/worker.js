/**
 * SQS consumer worker — polls the queue and processes messages.
 * Run alongside the API server: node src/worker.js
 */
import {
  SQSClient,
  ReceiveMessageCommand,
  DeleteMessageCommand,
} from '@aws-sdk/client-sqs';

const QUEUE_URL = process.env.SQS_QUEUE_URL || 'http://localhost:4566/000000000000/messages';
const POLL_INTERVAL_MS = Number(process.env.POLL_INTERVAL_MS) || 2000;

const sqs = new SQSClient({
  endpoint: process.env.AWS_ENDPOINT_URL || 'http://localhost:4566',
  region: process.env.AWS_REGION || 'us-east-1',
  credentials: {
    accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
    secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
  },
});

async function processMessage(msg) {
  let payload;
  try {
    payload = JSON.parse(msg.Body);
  } catch {
    payload = { body: msg.Body };
  }
  console.log('[worker] processing message:', payload);
  // TODO: add your business logic here
}

async function poll() {
  while (true) {
    const result = await sqs.send(
      new ReceiveMessageCommand({
        QueueUrl: QUEUE_URL,
        MaxNumberOfMessages: 10,
        WaitTimeSeconds: 5,
      })
    );

    const messages = result.Messages ?? [];
    for (const msg of messages) {
      try {
        await processMessage(msg);
        await sqs.send(
          new DeleteMessageCommand({
            QueueUrl: QUEUE_URL,
            ReceiptHandle: msg.ReceiptHandle,
          })
        );
      } catch (err) {
        console.error('[worker] failed to process message:', err);
      }
    }

    if (messages.length === 0) {
      await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
    }
  }
}

console.log('[worker] starting SQS consumer...');
poll().catch((err) => {
  console.error('[worker] fatal error:', err);
  process.exit(1);
});

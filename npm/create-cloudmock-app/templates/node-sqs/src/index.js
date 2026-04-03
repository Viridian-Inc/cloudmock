import express from 'express';
import {
  SQSClient,
  SendMessageCommand,
  ReceiveMessageCommand,
  DeleteMessageCommand,
} from '@aws-sdk/client-sqs';

const QUEUE_URL = process.env.SQS_QUEUE_URL || 'http://localhost:4566/000000000000/messages';
const PORT = process.env.PORT || 3000;

export const sqs = new SQSClient({
  endpoint: process.env.AWS_ENDPOINT_URL || 'http://localhost:4566',
  region: process.env.AWS_REGION || 'us-east-1',
  credentials: {
    accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
    secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
  },
});

export const app = express();
app.use(express.json());

// POST /messages — send a message to SQS
app.post('/messages', async (req, res) => {
  const body = req.body;
  if (!body.message) return res.status(400).json({ error: 'message is required' });

  const result = await sqs.send(
    new SendMessageCommand({
      QueueUrl: QUEUE_URL,
      MessageBody: JSON.stringify(body),
    })
  );
  res.status(202).json({ messageId: result.MessageId });
});

// GET /messages — receive and delete up to 10 messages
app.get('/messages', async (req, res) => {
  const received = await sqs.send(
    new ReceiveMessageCommand({
      QueueUrl: QUEUE_URL,
      MaxNumberOfMessages: 10,
      WaitTimeSeconds: 1,
    })
  );

  const messages = received.Messages ?? [];
  // Delete each message after reading
  await Promise.all(
    messages.map((msg) =>
      sqs.send(
        new DeleteMessageCommand({
          QueueUrl: QUEUE_URL,
          ReceiptHandle: msg.ReceiptHandle,
        })
      )
    )
  );

  res.json(
    messages.map((m) => {
      try {
        return JSON.parse(m.Body);
      } catch {
        return { body: m.Body };
      }
    })
  );
});

if (process.argv[1] === new URL(import.meta.url).pathname) {
  app.listen(PORT, () => console.log(`Listening on http://localhost:${PORT}`));
}

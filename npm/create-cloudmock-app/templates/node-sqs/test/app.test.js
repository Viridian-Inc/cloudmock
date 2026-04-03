import { mockAWS } from '@cloudmock/sdk';
import { SQSClient, CreateQueueCommand } from '@aws-sdk/client-sqs';
import request from 'supertest';
import { app } from '../src/index.js';

let cm;
let queueUrl;

beforeAll(async () => {
  cm = await mockAWS();
  process.env.AWS_ENDPOINT_URL = cm.endpoint();

  const sqs = new SQSClient(cm.clientConfig());
  const q = await sqs.send(new CreateQueueCommand({ QueueName: 'messages' }));
  queueUrl = q.QueueUrl;
  process.env.SQS_QUEUE_URL = queueUrl;
});

afterAll(async () => {
  await cm.stop();
});

test('POST /messages sends a message', async () => {
  const res = await request(app)
    .post('/messages')
    .send({ message: 'hello', priority: 'high' });
  expect(res.status).toBe(202);
  expect(res.body.messageId).toBeTruthy();
});

test('GET /messages receives messages', async () => {
  await request(app).post('/messages').send({ message: 'first' });
  await request(app).post('/messages').send({ message: 'second' });

  const res = await request(app).get('/messages');
  expect(res.status).toBe(200);
  expect(Array.isArray(res.body)).toBe(true);
  expect(res.body.length).toBeGreaterThanOrEqual(1);
});

test('POST /messages returns 400 without message field', async () => {
  const res = await request(app).post('/messages').send({ data: 'no message key' });
  expect(res.status).toBe(400);
});

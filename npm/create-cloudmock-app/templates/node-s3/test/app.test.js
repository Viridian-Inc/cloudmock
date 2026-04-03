import { mockAWS } from '@cloudmock/sdk';
import { S3Client, CreateBucketCommand } from '@aws-sdk/client-s3';
import request from 'supertest';
import { app } from '../src/index.js';

let cm;

beforeAll(async () => {
  cm = await mockAWS();
  process.env.AWS_ENDPOINT_URL = cm.endpoint();
  process.env.S3_BUCKET = 'uploads';

  const s3 = new S3Client({ ...cm.clientConfig(), forcePathStyle: true });
  await s3.send(new CreateBucketCommand({ Bucket: 'uploads' }));
});

afterAll(async () => {
  await cm.stop();
});

test('POST /upload stores a file', async () => {
  const res = await request(app)
    .post('/upload?key=hello.txt')
    .set('Content-Type', 'text/plain')
    .send(Buffer.from('Hello, CloudMock!'));
  expect(res.status).toBe(201);
  expect(res.body.key).toBe('hello.txt');
});

test('GET /files/:key retrieves the file', async () => {
  await request(app)
    .post('/upload?key=data.json')
    .set('Content-Type', 'application/json')
    .send(Buffer.from(JSON.stringify({ msg: 'test' })));

  const res = await request(app).get('/files/data.json');
  expect(res.status).toBe(200);
});

test('GET /files lists objects', async () => {
  const res = await request(app).get('/files');
  expect(res.status).toBe(200);
  expect(Array.isArray(res.body)).toBe(true);
  expect(res.body.length).toBeGreaterThanOrEqual(1);
});

test('GET /files/:key returns 404 for missing key', async () => {
  const res = await request(app).get('/files/nonexistent.txt');
  expect(res.status).toBe(404);
});

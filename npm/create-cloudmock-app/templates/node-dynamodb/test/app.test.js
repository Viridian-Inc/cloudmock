import { mockAWS } from '@cloudmock/sdk';
import { DynamoDBClient, CreateTableCommand } from '@aws-sdk/client-dynamodb';
import { DynamoDBDocumentClient, PutCommand, GetCommand } from '@aws-sdk/lib-dynamodb';
import request from 'supertest';
import { app } from '../src/index.js';

let cm;
let server;

beforeAll(async () => {
  cm = await mockAWS();
  process.env.AWS_ENDPOINT_URL = cm.endpoint();
  process.env.TABLE_NAME = 'items';

  const dynamo = new DynamoDBClient(cm.clientConfig());
  await dynamo.send(
    new CreateTableCommand({
      TableName: 'items',
      KeySchema: [{ AttributeName: 'id', KeyType: 'HASH' }],
      AttributeDefinitions: [{ AttributeName: 'id', AttributeType: 'S' }],
      BillingMode: 'PAY_PER_REQUEST',
    })
  );

  server = app.listen(0);
});

afterAll(async () => {
  server.close();
  await cm.stop();
});

test('POST /items creates an item', async () => {
  const res = await request(app).post('/items').send({ id: '1', name: 'Widget' });
  expect(res.status).toBe(201);
  expect(res.body.id).toBe('1');
});

test('GET /items/:id retrieves item', async () => {
  await request(app).post('/items').send({ id: '2', name: 'Gadget' });
  const res = await request(app).get('/items/2');
  expect(res.status).toBe(200);
  expect(res.body.name).toBe('Gadget');
});

test('GET /items returns all items', async () => {
  const res = await request(app).get('/items');
  expect(res.status).toBe(200);
  expect(Array.isArray(res.body)).toBe(true);
  expect(res.body.length).toBeGreaterThanOrEqual(2);
});

test('DELETE /items/:id removes item', async () => {
  await request(app).post('/items').send({ id: '3', name: 'Doomed' });
  const del = await request(app).delete('/items/3');
  expect(del.status).toBe(204);
  const get = await request(app).get('/items/3');
  expect(get.status).toBe(404);
});

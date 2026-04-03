import express from 'express';
import { DynamoDBClient } from '@aws-sdk/client-dynamodb';
import {
  DynamoDBDocumentClient,
  PutCommand,
  GetCommand,
  ScanCommand,
  DeleteCommand,
} from '@aws-sdk/lib-dynamodb';

const TABLE_NAME = process.env.TABLE_NAME || 'items';
const PORT = process.env.PORT || 3000;

const dynamo = new DynamoDBClient({
  endpoint: process.env.AWS_ENDPOINT_URL || 'http://localhost:4566',
  region: process.env.AWS_REGION || 'us-east-1',
  credentials: {
    accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
    secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
  },
});

const docClient = DynamoDBDocumentClient.from(dynamo);

export const app = express();
app.use(express.json());

// POST /items — create or replace an item
app.post('/items', async (req, res) => {
  const item = req.body;
  if (!item.id) {
    return res.status(400).json({ error: 'id is required' });
  }
  await docClient.send(new PutCommand({ TableName: TABLE_NAME, Item: item }));
  res.status(201).json(item);
});

// GET /items/:id — fetch a single item
app.get('/items/:id', async (req, res) => {
  const result = await docClient.send(
    new GetCommand({ TableName: TABLE_NAME, Key: { id: req.params.id } })
  );
  if (!result.Item) return res.status(404).json({ error: 'Not found' });
  res.json(result.Item);
});

// GET /items — scan all items
app.get('/items', async (req, res) => {
  const result = await docClient.send(new ScanCommand({ TableName: TABLE_NAME }));
  res.json(result.Items ?? []);
});

// DELETE /items/:id — remove an item
app.delete('/items/:id', async (req, res) => {
  await docClient.send(
    new DeleteCommand({ TableName: TABLE_NAME, Key: { id: req.params.id } })
  );
  res.status(204).send();
});

if (process.argv[1] === new URL(import.meta.url).pathname) {
  app.listen(PORT, () => console.log(`Listening on http://localhost:${PORT}`));
}

import express from 'express';
import {
  S3Client,
  PutObjectCommand,
  GetObjectCommand,
  ListObjectsV2Command,
} from '@aws-sdk/client-s3';

const BUCKET = process.env.S3_BUCKET || 'uploads';
const PORT = process.env.PORT || 3000;

const s3 = new S3Client({
  endpoint: process.env.AWS_ENDPOINT_URL || 'http://localhost:4566',
  region: process.env.AWS_REGION || 'us-east-1',
  credentials: {
    accessKeyId: process.env.AWS_ACCESS_KEY_ID || 'test',
    secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || 'test',
  },
  forcePathStyle: true,
});

export const app = express();
app.use(express.raw({ type: '*/*', limit: '50mb' }));

// POST /upload?key=filename — upload a file
app.post('/upload', async (req, res) => {
  const key = req.query.key;
  if (!key) return res.status(400).json({ error: 'key query param required' });
  await s3.send(
    new PutObjectCommand({
      Bucket: BUCKET,
      Key: key,
      Body: req.body,
      ContentType: req.headers['content-type'] || 'application/octet-stream',
    })
  );
  res.status(201).json({ key, bucket: BUCKET });
});

// GET /files/:key — download a file
app.get('/files/:key', async (req, res) => {
  try {
    const result = await s3.send(
      new GetObjectCommand({ Bucket: BUCKET, Key: req.params.key })
    );
    res.setHeader('Content-Type', result.ContentType || 'application/octet-stream');
    result.Body.pipe(res);
  } catch (err) {
    if (err.name === 'NoSuchKey') return res.status(404).json({ error: 'Not found' });
    throw err;
  }
});

// GET /files — list all objects
app.get('/files', async (req, res) => {
  const result = await s3.send(
    new ListObjectsV2Command({ Bucket: BUCKET })
  );
  res.json((result.Contents ?? []).map((o) => ({ key: o.Key, size: o.Size })));
});

if (process.argv[1] === new URL(import.meta.url).pathname) {
  app.listen(PORT, () => console.log(`Listening on http://localhost:${PORT}`));
}

/**
 * Example: Node.js API server with CloudMock SDK
 *
 * This demonstrates how an app connects to neureaux devtools.
 * Run with: node examples/node-api-server.mjs
 *
 * Prerequisites:
 *   - neureaux devtools running (pnpm tauri dev)
 *   - cloudmock running on localhost:4566
 */
import http from 'node:http';
import { init } from '../sdk/node/dist/index.mjs';

// Initialize CloudMock SDK — connects to devtools on :4580
init({ appName: 'example-api' });

const PORT = 3456;

// Simple API server that uses cloudmock services
const server = http.createServer(async (req, res) => {
  console.log(`${req.method} ${req.url}`);

  try {
    if (req.url === '/s3/buckets') {
      // List S3 buckets via cloudmock
      const s3Res = await fetch('http://localhost:4566/', {
        headers: {
          'Authorization': 'AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=fake',
        },
      });
      const body = await s3Res.text();
      console.info('S3 ListBuckets:', s3Res.status);
      res.writeHead(200, { 'Content-Type': 'application/xml' });
      res.end(body);

    } else if (req.url === '/dynamo/tables') {
      // List DynamoDB tables via cloudmock
      const ddbRes = await fetch('http://localhost:4566/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-amz-json-1.0',
          'X-Amz-Target': 'DynamoDB_20120810.ListTables',
          'Authorization': 'AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=fake',
        },
        body: JSON.stringify({}),
      });
      const data = await ddbRes.json();
      console.info('DynamoDB ListTables:', data);
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(data, null, 2));

    } else if (req.url === '/error') {
      // Trigger an error
      console.error('Intentional error for testing');
      throw new Error('Something went wrong!');

    } else {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({
        message: 'Example API server',
        endpoints: ['/s3/buckets', '/dynamo/tables', '/error'],
      }));
    }
  } catch (err) {
    console.error('Request failed:', err.message);
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: err.message }));
  }
});

server.listen(PORT, () => {
  console.log(`Example API server running on http://localhost:${PORT}`);
  console.log('Endpoints: /s3/buckets, /dynamo/tables, /error');
  console.log('CloudMock SDK connected — events stream to devtools');
});

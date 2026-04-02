---
title: Troubleshooting
description: Common issues and solutions when using CloudMock
---

This page covers the most frequently encountered issues when using CloudMock, along with their causes and fixes.

## No requests showing in the Activity view

**Symptoms:** CloudMock is running and accepting API calls, but the Activity view in the dashboard or devtools shows no entries.

**Cause 1: Level filter is set to `app` instead of `all`.**

The Activity view defaults to showing `app`-level requests, which filters out internal and infrastructure traffic. If your requests are not tagged with the devtools SDK, they may not appear.

Fix: Change the level filter to `all`:

```bash
curl "http://localhost:4599/api/requests?level=all"
```

In the dashboard, use the level dropdown in the Activity view toolbar and select "All".

**Cause 2: Your SDK is not pointing at CloudMock.**

Verify that `AWS_ENDPOINT_URL` is set correctly in the environment where your application is running:

```bash
echo $AWS_ENDPOINT_URL
# Should print: http://localhost:4566
```

If you are setting the endpoint in code, double-check the port number. The AWS API endpoint is **4566**, not 4599 (admin) or 4500 (dashboard).

**Cause 3: CloudMock is not running.**

Check that the process is alive and the ports are open:

```bash
curl http://localhost:4599/api/health
```

If this returns a connection error, CloudMock is not running. Start it with `npx cloudmock start` or `docker compose up cloudmock`.

## CORS errors in the browser

**Symptoms:** Browser console shows `Access-Control-Allow-Origin` errors when your frontend makes requests to CloudMock.

**Cause:** Your frontend is running on a different origin (e.g., `http://localhost:3000`) from CloudMock (`http://localhost:4566`), and the browser is enforcing the same-origin policy.

**Fix 1: Use single-port mode.**

If your frontend and backend both go through the same proxy, configure your dev server to proxy `/aws` or a similar path to CloudMock. This avoids cross-origin requests entirely.

For Vite:

```typescript
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      "/aws": {
        target: "http://localhost:4566",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/aws/, ""),
      },
    },
  },
});
```

**Fix 2: Check Content Security Policy.**

If your application sets a `Content-Security-Policy` header, make sure `connect-src` includes `http://localhost:4566`:

```
Content-Security-Policy: connect-src 'self' http://localhost:4566
```

## Service not found

**Symptoms:** Your AWS SDK returns errors like `UnknownServiceException` or `Service not available`, or CloudMock returns a `404` for a valid AWS API call.

**Cause:** The service you are trying to use is not enabled in the current profile.

**Fix:** Check which profile CloudMock is running with:

```bash
curl http://localhost:4599/api/config | jq .profile
```

The `minimal` profile only includes 8 services: `iam`, `sts`, `s3`, `dynamodb`, `sqs`, `sns`, `lambda`, `cloudwatch-logs`.

If you need additional services, switch to a larger profile:

```bash
# Use the standard profile (20 services)
CLOUDMOCK_PROFILE=standard npx cloudmock start

# Use the full profile (all 98 services)
CLOUDMOCK_PROFILE=full npx cloudmock start

# Or enable specific services on top of minimal
CLOUDMOCK_SERVICES=s3,dynamodb,cognito-idp,events npx cloudmock start
```

You can also set the profile in `cloudmock.yml`:

```yaml
profile: standard
```

Or enable individual services:

```yaml
profile: minimal
services:
  cognito-idp:
    enabled: true
  events:
    enabled: true
```

See the [Configuration Reference](/docs/reference/configuration/) for the full list of profile contents.

## Connection refused

**Symptoms:** `curl: (7) Failed to connect to localhost port 4566: Connection refused` or equivalent errors from AWS SDKs.

**Cause 1: CloudMock is not running.**

Start it:

```bash
npx cloudmock start
```

Or check if it is running under Docker:

```bash
docker ps | grep cloudmock
```

**Cause 2: Port conflict.**

Another process is using port 4566. Check what is listening:

```bash
# macOS / Linux
lsof -i :4566
```

If the port is taken, start CloudMock on a different port:

```bash
CLOUDMOCK_GATEWAY_PORT=5566 npx cloudmock start
```

Then update your SDK endpoint accordingly:

```bash
export AWS_ENDPOINT_URL=http://localhost:5566
```

**Cause 3: Docker networking.**

If CloudMock is running in Docker and your application is running on the host (or vice versa), `localhost` may not resolve correctly.

- **App on host, CloudMock in Docker:** Use `localhost:4566` (with `-p 4566:4566` on the container). This is the default setup.
- **Both in Docker Compose:** Use the service name (`http://cloudmock:4566`) as the endpoint, not `localhost`.
- **App in Docker, CloudMock on host:** Use `host.docker.internal:4566` (macOS/Windows) or the host's IP address (Linux).

**Cause 4: Firewall.**

On Linux, check that the firewall allows traffic on port 4566:

```bash
sudo iptables -L -n | grep 4566
```

## Stale data after restart

**Symptoms:** After restarting CloudMock, resources from a previous session still exist, or resources you just created are missing.

**Cause 1: Persistence is enabled and loading old state.**

If `persistence.enabled: true` in your config, CloudMock saves state on shutdown and restores it on startup. To start fresh:

```bash
# Delete the persistence directory
rm -rf /var/lib/cloudmock/data

# Or reset via the API after startup
curl -X POST http://localhost:4599/api/reset
```

**Cause 2: Persistence is disabled and you expected data to survive a restart.**

By default, all state is in memory and lost when CloudMock exits. If you need data to persist, enable persistence:

```yaml
persistence:
  enabled: true
  path: ./cloudmock-data
```

**Cause 3: Browser cache showing old dashboard state.**

The dashboard may show cached data. Hard-refresh the browser (`Cmd+Shift+R` on macOS, `Ctrl+Shift+R` on Linux/Windows) or open an incognito window.

## AWS CLI returns signature errors

**Symptoms:** The AWS CLI returns `SignatureDoesNotMatch` or `InvalidSignatureException`.

**Cause:** The credentials in your environment do not match the credentials CloudMock expects.

**Fix:** CloudMock's default root credentials are `test` / `test`. Make sure your environment matches:

```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
```

If you have configured custom credentials in `cloudmock.yml`, use those instead. Or disable IAM entirely for development:

```bash
CLOUDMOCK_IAM_MODE=none npx cloudmock start
```

## S3 operations fail with "bucket not found" for virtual-hosted-style URLs

**Symptoms:** S3 requests fail because the SDK is using `http://my-bucket.localhost:4566` instead of `http://localhost:4566/my-bucket`.

**Cause:** The SDK is using virtual-hosted-style addressing, which does not work with localhost.

**Fix:** Enable path-style addressing in your S3 client:

```typescript
// Node.js (AWS SDK v3)
const s3 = new S3Client({ forcePathStyle: true, /* ... */ });
```

```python
# Python (boto3)
s3 = boto3.client("s3", endpoint_url="http://localhost:4566",
                  config=botocore.config.Config(s3={"addressing_style": "path"}))
```

```go
// Go (aws-sdk-go-v2)
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.UsePathStyle = true
})
```

```java
// Java (AWS SDK v2)
S3Client s3 = S3Client.builder()
    .forcePathStyle(true)
    .build();
```

## Devtools SDK not connecting

**Symptoms:** You initialized the `@cloudmock/node`, Go, or Python SDK but events are not appearing in the Activity or Topology views.

**Cause 1: CloudMock admin API is not reachable.**

The SDK connects to the admin API on port **4599** (HTTP) or the source server on port **4580** (TCP). Verify the admin API is up:

```bash
curl http://localhost:4599/api/source/status
```

**Cause 2: Wrong host or port.**

If CloudMock is running on a non-default port, pass the correct values to `init()`:

```typescript
init({ host: "192.168.1.100", port: 4580 });
```

**Cause 3: The SDK initialized after HTTP clients were created.**

The SDK works by monkey-patching HTTP libraries at `init()` time. If you create AWS SDK clients before calling `init()`, those clients will not be intercepted. Always call `init()` first.

## FAQ

**Does CloudMock support service X?**

Check the [Compatibility Matrix](/docs/reference/compatibility-matrix/). CloudMock supports 98 fully emulated AWS services.

**Can I use CloudMock with Terraform/CDK/Pulumi?**

Yes. Point Terraform's AWS provider at CloudMock by setting `AWS_ENDPOINT_URL=http://localhost:4566`. CloudFormation is a Tier 1 service with full emulation, so `aws cloudformation` commands work against CloudMock. CDK and Pulumi both work via the standard AWS SDK endpoint override.

**Does CloudMock work offline?**

Yes. CloudMock has zero external dependencies and does not make any network calls to AWS or any other remote service. It works fully offline.

**How do I see debug logs?**

Set the log level to `debug`:

```bash
CLOUDMOCK_LOG_LEVEL=debug npx cloudmock start
```

This prints every incoming request, the service routing decision, and the response. Useful for understanding why a particular API call is failing.

**How do I report a bug?**

Open an issue at [github.com/Viridian-Inc/cloudmock](https://github.com/Viridian-Inc/cloudmock/issues) with:

1. CloudMock version (`curl http://localhost:4599/api/version`)
2. The AWS service and action
3. The request you sent (redact credentials)
4. The response you received
5. The response you expected

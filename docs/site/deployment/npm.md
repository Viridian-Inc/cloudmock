# npm / npx Deployment

The simplest way to run CloudMock on any machine with Node.js.

## npx (No Install)

```bash
npx cloudmock
```

This downloads the platform-specific CloudMock binary on first run and starts it immediately. Subsequent runs use the cached binary.

## Global Install

```bash
npm install -g cloudmock
```

Then use the `cmk` CLI:

```bash
cmk start           # Start in background
cmk status           # Check status
cmk logs             # Tail logs
cmk stop             # Stop
```

## How It Works

The `cloudmock` npm package is a lightweight wrapper that:

1. Detects your platform (macOS/Linux/Windows, x64/arm64)
2. Downloads the correct Go binary from GitHub releases
3. Runs it with the appropriate arguments

The actual CloudMock binary is a single Go executable (~25MB) with the DevTools dashboard embedded.

## Configuration

Create `.cloudmock.yaml` in your project:

```yaml
profile: standard
region: us-east-1
gateway:
  port: 4566
dashboard:
  port: 4500
```

Or use environment variables:

```bash
CLOUDMOCK_PROFILE=full npx cloudmock
CLOUDMOCK_GATEWAY_PORT=5566 cmk start
```

## package.json Scripts

Add CloudMock to your development workflow:

```json
{
  "scripts": {
    "dev": "concurrently \"cmk start --fg\" \"npm run start\"",
    "pretest": "cmk start",
    "test": "jest --runInBand",
    "posttest": "cmk stop",
    "seed": "node scripts/seed-cloudmock.js"
  },
  "devDependencies": {
    "cloudmock": "^1.0.0",
    "concurrently": "^9.0.0"
  }
}
```

## CI/CD

Use `npx` in CI pipelines:

```yaml
# GitHub Actions
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20

      - name: Start CloudMock
        run: |
          npx cloudmock &
          sleep 3
          curl -f http://localhost:4599/api/health

      - name: Run tests
        run: npm test
        env:
          AWS_ENDPOINT_URL: http://localhost:4566
          AWS_ACCESS_KEY_ID: test
          AWS_SECRET_ACCESS_KEY: test

      - name: Stop CloudMock
        if: always()
        run: cmk stop
```

## Supported Platforms

| OS | Architecture | Supported |
|----|-------------|-----------|
| macOS | Apple Silicon (arm64) | Yes |
| macOS | Intel (amd64) | Yes |
| Linux | x86_64 (amd64) | Yes |
| Linux | ARM64 | Yes |
| Windows | x86_64 | Yes |

## Troubleshooting

### npx hangs on first run

The first run downloads the binary (~25MB). If you're behind a proxy:

```bash
HTTPS_PROXY=http://proxy:8080 npx cloudmock
```

### Permission denied

```bash
# If the downloaded binary isn't executable
chmod +x ~/.npm/_npx/*/node_modules/cloudmock/bin/gateway
```

### Port conflicts

```bash
# Check what's using port 4566
lsof -i :4566

# Use a different port
CLOUDMOCK_GATEWAY_PORT=5566 cmk start
```

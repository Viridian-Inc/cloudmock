# Installation

## npx (Recommended)

The fastest way to run CloudMock. No global install required.

```bash
npx cloudmock
```

This downloads the platform-specific binary and starts CloudMock immediately. Subsequent runs use the cached binary.

## npm (Global Install)

```bash
npm install -g cloudmock
cmk start
```

## Homebrew (macOS / Linux)

```bash
brew install cloudmock
cmk start
```

## Docker

```bash
docker run -d \
  --name cloudmock \
  -p 4566:4566 \
  -p 4500:4500 \
  -p 4599:4599 \
  -p 4318:4318 \
  ghcr.io/neureaux/cloudmock:latest
```

With persistence:

```bash
docker run -d \
  --name cloudmock \
  -p 4566:4566 \
  -p 4500:4500 \
  -p 4599:4599 \
  -p 4318:4318 \
  -v cloudmock-data:/data \
  -e CLOUDMOCK_PERSIST=true \
  -e CLOUDMOCK_PERSIST_PATH=/data \
  ghcr.io/neureaux/cloudmock:latest
```

See [Docker deployment](../deployment/docker.md) for Docker Compose examples.

## Binary Download

Download prebuilt binaries from the [releases page](https://github.com/Viridian-Inc/cloudmock/releases):

| Platform | Architecture | File |
|----------|-------------|------|
| macOS | Apple Silicon (arm64) | `cloudmock-darwin-arm64.tar.gz` |
| macOS | Intel (amd64) | `cloudmock-darwin-amd64.tar.gz` |
| Linux | x86_64 | `cloudmock-linux-amd64.tar.gz` |
| Linux | ARM64 | `cloudmock-linux-arm64.tar.gz` |
| Windows | x86_64 | `cloudmock-windows-amd64.zip` |

```bash
# Example: macOS Apple Silicon
curl -L https://github.com/Viridian-Inc/cloudmock/releases/latest/download/cloudmock-darwin-arm64.tar.gz | tar xz
sudo mv cloudmock /usr/local/bin/
cmk start
```

## Build from Source

Requires Go 1.26+ and Node.js 20+.

```bash
git clone https://github.com/Viridian-Inc/cloudmock.git
cd cloudmock
make build
./bin/cmk start
```

The `make build` target compiles both the Go gateway and the DevTools dashboard (Preact).

## Verify Installation

```bash
cmk version
# cmk version 1.0.0 (darwin/arm64)

cmk start
cmk status
# CloudMock is running (PID 12345)
#   Gateway:   http://localhost:4566
#   Admin API: http://localhost:4599
#   Dashboard: http://localhost:4500
#   Profile:   standard
#   Region:    us-east-1
#   Health:    ok (20/20 services healthy)
```

## Uninstall

**npm:**
```bash
cmk stop
npm uninstall -g cloudmock
```

**Homebrew:**
```bash
cmk stop
brew uninstall cloudmock
```

**Docker:**
```bash
docker stop cloudmock && docker rm cloudmock
```

**Manual:**
```bash
cmk stop
rm /usr/local/bin/cloudmock
rm -rf ~/.cloudmock
```

## Troubleshooting

### Port already in use

```bash
cmk stop                           # Stop existing instance
# Or change the port:
CLOUDMOCK_GATEWAY_PORT=5566 cmk start
```

### Permission denied

```bash
# If installed via binary download, ensure it's executable:
chmod +x /usr/local/bin/cloudmock
```

### Gateway binary not found

If you installed `cmk` but get "gateway binary not found":

```bash
# The gateway binary must be alongside cmk or in your PATH
which cloudmock-gateway    # Check if it's in PATH
ls ./bin/gateway           # Check local build
make build                 # Rebuild if building from source
```

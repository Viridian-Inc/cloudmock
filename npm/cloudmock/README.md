# cloudmock

Local AWS emulation. 25 services. One command.

## Install

```bash
npx cloudmock
```

Or install globally:

```bash
npm install -g cloudmock
cloudmock
```

## What this does

This npm package downloads the pre-built CloudMock binary for your platform
(macOS, Linux, Windows on arm64/x64) and runs it. The binary is cached at
`~/.cloudmock/bin/` so subsequent runs start instantly.

## Usage

```bash
# Start CloudMock
npx cloudmock

# Point your AWS SDK at it
aws --endpoint-url=http://localhost:4566 s3 ls
```

## Links

- [Documentation](https://cloudmock.io/docs)
- [GitHub](https://github.com/neureaux/cloudmock)
- [License](https://github.com/neureaux/cloudmock/blob/main/LICENSE) (Apache-2.0)

# CloudMock GitHub Action

Start CloudMock for local AWS emulation in CI. 99 fully emulated AWS services, one line.

## Usage

```yaml
steps:
  - uses: viridian-inc/cloudmock-action@v1
  - run: npm test  # AWS_ENDPOINT_URL is auto-set
```

### With options

```yaml
steps:
  - uses: viridian-inc/cloudmock-action@v1
    with:
      profile: full           # all 99 services
      state: fixtures/state.json  # pre-load state
  - run: npm test
```

### Access outputs

```yaml
steps:
  - uses: viridian-inc/cloudmock-action@v1
    id: cloudmock
  - run: |
      echo "Endpoint: ${{ steps.cloudmock.outputs.endpoint }}"
      echo "Version: ${{ steps.cloudmock.outputs.version }}"
      aws --endpoint ${{ steps.cloudmock.outputs.endpoint }} s3 ls
```

## Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `version` | `latest` | CloudMock version (npm tag or semver) |
| `profile` | `minimal` | Service profile: `minimal`, `standard`, `full` |
| `services` | | Comma-separated services (overrides profile) |
| `port` | `4566` | Gateway port |
| `state` | | State file to load on startup |
| `iam-mode` | `none` | IAM mode: `none`, `log`, `enforce` |

## Outputs

| Output | Description |
|--------|-------------|
| `endpoint` | Gateway URL (e.g., `http://localhost:4566`) |
| `admin-url` | Admin API URL (e.g., `http://localhost:4599`) |
| `version` | Installed version |

## Environment Variables

The action automatically sets these for subsequent steps:
- `AWS_ENDPOINT_URL` — points to CloudMock
- `AWS_ACCESS_KEY_ID` — `test`
- `AWS_SECRET_ACCESS_KEY` — `test`
- `AWS_DEFAULT_REGION` — `us-east-1`

## Examples

### Node.js + Jest

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: viridian-inc/cloudmock-action@v1
      - uses: actions/setup-node@v4
        with: { node-version: 22 }
      - run: npm ci && npm test
```

### Python + pytest

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: viridian-inc/cloudmock-action@v1
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - run: pip install -r requirements.txt && pytest
```

### Go

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: viridian-inc/cloudmock-action@v1
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: go test ./...
```

### Terraform

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: viridian-inc/cloudmock-action@v1
        with: { profile: full }
      - uses: hashicorp/setup-terraform@v3
      - run: |
          cd infra
          terraform init
          terraform apply -auto-approve
```

### With state file

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: viridian-inc/cloudmock-action@v1
        with:
          state: fixtures/cloudmock-state.json
      - run: npm test  # tables, queues, buckets already exist
```

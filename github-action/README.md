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
      services: 's3, sqs'         # explicit AWS service selection
      port: '4567'               # also used by AWS_ENDPOINT_URL
      state: 'fixtures/seed state.json'  # paths may contain spaces
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

An empty `services` input keeps the profile's default behavior. A nonempty list
limits AWS endpoints to those services; surrounding spaces are ignored. Follow-up
requests should use the emitted `endpoint` or `AWS_ENDPOINT_URL` so they reach the
configured port.

State is restored before the gateway begins serving requests. The repository
action at `.github/actions/cloudmock` also supports `test-mode` (default `true`)
and `iac`. Test mode loads the snapshot without saving changes back. Normal mode
can update the state file during execution and shutdown, so use a copy when the
original fixture must remain unchanged. State and IaC paths are passed as literal
values, including spaces and shell metacharacters.

## Testing this repository from source

The repository action can build the checked-out gateway instead of installing
the published npm package. Set up Go first, then opt into `source`:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
  - uses: ./.github/actions/cloudmock
    id: cloudmock
    with:
      source: 'true'
  - run: echo "Tested source $CM_SOURCE_SHA"
    env:
      CM_SOURCE_SHA: ${{ steps.cloudmock.outputs.source-sha }}
```

Source mode skips npm, builds `cmd/gateway` from the checkout, and places that
binary first on the following steps' PATH. The `source-sha` output identifies the
tested commit; `version` is `source-<sha>`. This option belongs to the repository
action at `.github/actions/cloudmock`; the distributable action keeps its released
package behavior. The default `source: 'false'` continues to use `version` from npm.

The compatibility workflow uses source mode for SDK, Terraform, and external-repo
checks. SDK reports include `source_sha` and compare operation results against a
fresh run of the base commit. A previously passing operation becoming failed,
skipped, or missing fails the job; existing base-commit failures remain visible in
the report. Pull-request and test-branch runs upload reports without publishing
them back to the repository.

External checks pin their upstream commits and adapt only test setup to use the
CloudMock endpoint. They retain the upstream application code and assertions,
and reject missing or skipped tests so a run cannot pass without exercising the
selected Python and DynamoMQ tests.

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

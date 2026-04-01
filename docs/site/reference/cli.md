# CLI Reference

The `cmk` CLI manages CloudMock instances -- starting, stopping, and inspecting the running gateway.

## Installation

```bash
# Via npm
npm install -g cloudmock

# Via Homebrew
brew install cloudmock

# Or use npx (no install)
npx cloudmock
```

## Commands

### cmk start

Start the CloudMock gateway, admin API, and dashboard.

```bash
cmk start [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--config <path>` | auto-discovered | Path to `.cloudmock.yaml` |
| `--profile <name>` | `standard` | Service profile: `minimal`, `standard`, `full` |
| `--port <number>` | `4566` | Gateway port |
| `--fg` | `false` | Run in foreground (don't daemonize) |

**Examples:**

```bash
# Start with defaults
cmk start

# Start with minimal profile for fast startup
cmk start --profile minimal

# Start on a custom port
cmk start --port 5566

# Start in foreground (useful for debugging)
cmk start --fg

# Start with explicit config file
cmk start --config ./my-config.yaml
```

**Output:**

```
Starting CloudMock...
  Config:    /home/user/project/.cloudmock.yaml
  Discovered pulumi project at ./infra
  Gateway:   http://localhost:4566
  Admin API: http://localhost:4599
  Dashboard: http://localhost:4500
  Profile:   standard
  Region:    us-east-1

CloudMock started (PID 12345)
  Logs: /home/user/.cloudmock/cloudmock.log

Run 'cmk status' to check health, 'cmk logs' to tail output.
```

**Behavior:**

1. Resolves configuration: explicit `--config` > auto-discovered `.cloudmock.yaml` > defaults
2. Applies environment variable overrides
3. Auto-discovers IaC projects (Pulumi, Terraform) in current and parent directories
4. Checks if CloudMock is already running (reads PID file)
5. Locates the gateway binary
6. Starts the gateway as a background daemon (or foreground with `--fg`)
7. Writes PID file to `~/.cloudmock/cloudmock.pid`
8. Logs to `~/.cloudmock/cloudmock.log`

### cmk stop

Stop the running CloudMock instance.

```bash
cmk stop
```

Sends SIGTERM for graceful shutdown. If the process doesn't exit within 10 seconds, sends SIGKILL. Cleans up the PID file.

```
Stopping CloudMock (PID 12345)...
CloudMock stopped.
```

### cmk status

Show the status of the running CloudMock instance.

```bash
cmk status
```

Output:

```
CloudMock is running (PID 12345)

  Gateway:   http://localhost:4566
  Admin API: http://localhost:4599
  Dashboard: http://localhost:4500
  Profile:   standard
  Region:    us-east-1

  Health:    ok (20/20 services healthy)
```

Queries the `/api/health` endpoint to report service health. Exit code 1 if CloudMock is not running.

### cmk logs

Tail CloudMock log output.

```bash
cmk logs [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-n <lines>` | `50` | Number of lines to show (0 for all) |
| `-f` | `true` | Follow log output (like `tail -f`) |

**Examples:**

```bash
# Tail logs (default: last 50 lines, follow)
cmk logs

# Show last 100 lines without following
cmk logs -n 100 -f=false

# Show all logs
cmk logs -n 0 -f=false
```

Reads from `~/.cloudmock/cloudmock.log`. On Windows, uses PowerShell's `Get-Content` instead of `tail`.

### cmk config

Print the resolved configuration as YAML.

```bash
cmk config
```

Output:

```yaml
gateway:
  port: 4566
admin:
  port: 4599
dashboard:
  port: 4500
  enabled: true
otlp:
  port: 4318
iac:
  path: ""
  env: ""
profile: standard
region: us-east-1
account_id: "000000000000"
```

Shows the effective configuration after merging defaults, config file, and environment variables.

### cmk version

Print version information.

```bash
cmk version
```

```
cmk version 1.0.0 (darwin/arm64)
```

### cmk help

Show usage information.

```bash
cmk help
cmk --help
cmk -h
```

## File Locations

| File | Path | Purpose |
|------|------|---------|
| Config | `.cloudmock.yaml` (project root) | Project configuration |
| PID | `~/.cloudmock/cloudmock.pid` | Running instance PID |
| Logs | `~/.cloudmock/cloudmock.log` | Server logs |
| Data | `~/.cloudmock/data/` | Persistent data (when enabled) |

## Environment Variables

The CLI respects all `CLOUDMOCK_*` environment variables. See [Configuration](configuration.md) for the full list.

Key variables:

```bash
# Change ports
CLOUDMOCK_GATEWAY_PORT=5566 cmk start
CLOUDMOCK_ADMIN_PORT=5599 cmk start
CLOUDMOCK_DASHBOARD_PORT=5500 cmk start

# Change region
CLOUDMOCK_REGION=eu-west-1 cmk start

# Change profile
CLOUDMOCK_PROFILE=full cmk start

# Enable specific services only
CLOUDMOCK_SERVICES=s3,dynamodb,lambda cmk start
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (already running, port in use, binary not found, etc.) |

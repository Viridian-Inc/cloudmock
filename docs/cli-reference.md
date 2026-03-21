# CLI Reference

The `cloudmock` binary is the control-plane client. It communicates with the running gateway via the admin API (default: `http://localhost:4599`).

```
Usage: cloudmock <command> [options]

Commands:
  start      Start the cloudmock gateway
  stop       Stop the cloudmock gateway
  status     Show health status of all services
  reset      Reset service state (all or specific)
  services   List registered services
  config     Show current configuration
  version    Print version information
  help       Show this help message
```

---

## `cloudmock start`

Launches the gateway binary. The gateway binary must be present at `./bin/gateway`, `bin/gateway`, or on `PATH` (build with `make build`).

```
cloudmock start [flags]

Flags:
  -config string    Path to config file (default: cloudmock.yml)
  -profile string   Service profile: minimal | standard | full
  -services string  Comma-separated list of services to enable
```

### Examples

```bash
# Start with defaults
cloudmock start

# Start with a specific config file
cloudmock start -config /etc/cloudmock/prod.yml

# Start with the standard profile
cloudmock start -profile standard

# Start with specific services only
cloudmock start -services s3,dynamodb,sqs
```

The `-profile` and `-services` flags set the `CLOUDMOCK_PROFILE` and `CLOUDMOCK_SERVICES` environment variables before launching the gateway. They take precedence over the config file.

---

## `cloudmock stop`

Prints instructions for stopping the gateway. A full PID-tracking implementation is planned; currently the gateway must be stopped with Ctrl+C or by killing the process.

```bash
cloudmock stop
```

---

## `cloudmock status`

Queries the admin API health endpoint and prints the health of all registered services.

```
cloudmock status

Status: ok

SERVICE             HEALTHY
-------             -------
s3                  yes
dynamodb            yes
sqs                 yes
sns                 yes
sts                 yes
```

The admin API address can be overridden with `CLOUDMOCK_ADMIN_ADDR`:

```bash
CLOUDMOCK_ADMIN_ADDR=http://remote-host:4599 cloudmock status
```

Exits with a non-zero status code if the admin API is unreachable.

---

## `cloudmock reset`

Resets the in-memory state of one or all services. This is equivalent to deleting all resources without restarting the process.

```
cloudmock reset [flags]

Flags:
  -service string   Name of the service to reset (omit to reset all)
```

### Examples

```bash
# Reset all services
cloudmock reset

# Reset only S3
cloudmock reset -service s3

# Reset DynamoDB
cloudmock reset -service dynamodb
```

Output:

```
Reset 5 services: s3, dynamodb, sqs, sns, sts
```

Exits with a non-zero status code if the named service is not registered.

---

## `cloudmock services`

Lists all registered services with their action count and health status.

```bash
cloudmock services

SERVICE             ACTIONS  HEALTHY
-------             -------  -------
s3                  10       yes
dynamodb            12       yes
sqs                 13       yes
sns                 12       yes
sts                 3        yes
kms                 10       yes
```

---

## `cloudmock config`

Fetches and prints the active configuration from the running gateway as JSON.

```bash
cloudmock config
{
  "region": "us-east-1",
  "account_id": "000000000000",
  "profile": "minimal",
  "iam": {
    "mode": "enforce",
    "root_access_key": "test",
    "root_secret_key": "test",
    "seed_file": ""
  },
  "gateway": { "port": 4566 },
  "dashboard": { "enabled": true, "port": 4500 },
  "admin": { "port": 4599 },
  "logging": { "level": "info", "format": "text" },
  "persistence": { "enabled": false, "path": "" }
}
```

---

## `cloudmock version`

Prints the version of the cloudmock binary.

```bash
cloudmock version
cloudmock version 0.1.0
```

---

## `cloudmock help`

Prints the usage summary.

```bash
cloudmock help
cloudmock --help
cloudmock -h
```

---

## Admin API Endpoints

The CLI wraps these admin API endpoints. You can also call them directly:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health status of all services |
| `/api/services` | GET | List of registered services |
| `/api/reset` | POST | Reset all services |
| `/api/services/{name}/reset` | POST | Reset a specific service |
| `/api/config` | GET | Active configuration |

Example:

```bash
curl http://localhost:4599/api/health
{"status":"ok","services":{"s3":true,"dynamodb":true}}
```

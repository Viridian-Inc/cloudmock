# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.5.x   | Yes       |
| 1.4.x   | Security fixes only |
| < 1.4   | No        |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please report them privately:

1. **Email:** security@viridian.dev
2. **GitHub Security Advisories:** Use the [Report a vulnerability](https://github.com/Viridian-Inc/cloudmock/security/advisories/new) button

### What to include

- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Any potential impact assessment

### Response timeline

- **Acknowledgment:** Within 48 hours
- **Initial assessment:** Within 1 week
- **Fix timeline:** Depends on severity
  - Critical: Patch within 72 hours
  - High: Patch within 1 week
  - Medium/Low: Next scheduled release

### Scope

CloudMock is a local development tool. Security issues we care about:

- **In scope:** Command injection, path traversal, arbitrary file access, denial of service, dependency vulnerabilities
- **Out of scope:** Issues that require local machine access (CloudMock runs locally by design), AWS API compatibility differences

## Security Best Practices

When using CloudMock:

- Do not expose CloudMock ports to the public internet
- Do not use real AWS credentials with CloudMock
- Use `test`/`test` as access key/secret key for local development
- Keep CloudMock updated to the latest version

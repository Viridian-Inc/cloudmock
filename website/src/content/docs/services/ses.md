---
title: SES
description: Amazon SES (Simple Email Service) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon SES, supporting email sending (recorded but not delivered), identity verification, and identity management. Sent emails are stored in memory for test assertions.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| SendEmail | Supported | Records a send operation; no email is delivered |
| SendRawEmail | Supported | Records a raw MIME send; no email is delivered |
| VerifyEmailIdentity | Supported | Marks an email address as verified |
| ListIdentities | Supported | Returns all verified identities |
| DeleteIdentity | Supported | Removes a verified identity |
| GetIdentityVerificationAttributes | Supported | Returns verification status for identities |
| ListVerifiedEmailAddresses | Supported | Returns all verified email addresses |

## Quick Start

### curl

```bash
# Verify a sender identity
curl -X POST "http://localhost:4566/?Action=VerifyEmailIdentity&EmailAddress=sender@example.com"

# Send an email
curl -X POST "http://localhost:4566/?Action=SendEmail&Source=sender@example.com&Destination.ToAddresses.member.1=recipient@example.com&Message.Subject.Data=Hello&Message.Body.Text.Data=Hello+from+CloudMock"

# List identities
curl -X POST "http://localhost:4566/?Action=ListIdentities"
```

### Node.js

```typescript
import { SESClient, SendEmailCommand, VerifyEmailIdentityCommand } from '@aws-sdk/client-ses';

const ses = new SESClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await ses.send(new VerifyEmailIdentityCommand({ EmailAddress: 'noreply@myapp.com' }));
const result = await ses.send(new SendEmailCommand({
  Source: 'noreply@myapp.com',
  Destination: { ToAddresses: ['user@example.com'] },
  Message: {
    Subject: { Data: 'Welcome!' },
    Body: { Html: { Data: '<h1>Welcome to our app</h1>' } },
  },
}));
console.log(result.MessageId);
```

### Python

```python
import boto3

ses = boto3.client('ses', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

ses.verify_email_identity(EmailAddress='noreply@myapp.com')

response = ses.send_email(
    Source='noreply@myapp.com',
    Destination={'ToAddresses': ['user@example.com']},
    Message={
        'Subject': {'Data': 'Welcome!'},
        'Body': {'Html': {'Data': '<h1>Welcome to our app</h1>'}},
    },
)
print(response['MessageId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  ses:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- **No SMTP delivery** occurs. Email messages are stored in memory and accessible for test assertions via the dashboard.
- All identities are **immediately reported as verified** (`Success` status).
- **SES v2 API** (`sesv2` service name) is not implemented.
- **Configuration sets**, suppression lists, and dedicated IPs are not implemented.
- **Sending quotas** are not enforced.
- **Templates** (`SendTemplatedEmail`) are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| MessageRejected | 400 | The message could not be sent |
| MailFromDomainNotVerifiedException | 400 | The sender's domain is not verified |
| ConfigurationSetDoesNotExistException | 400 | The configuration set does not exist |
| AccountSendingPausedException | 400 | Email sending is paused for this account |

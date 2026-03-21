# SES — Simple Email Service

**Tier:** 1 (Full Emulation)
**Protocol:** Query (`Action=<Action>`)
**Service Name:** `email`

## Supported Actions

| Action | Notes |
|--------|-------|
| `SendEmail` | Records a send operation; no email is delivered |
| `SendRawEmail` | Records a raw MIME send; no email is delivered |
| `VerifyEmailIdentity` | Marks an email address as verified |
| `ListIdentities` | Returns all verified identities |
| `DeleteIdentity` | Removes a verified identity |
| `GetIdentityVerificationAttributes` | Returns verification status for identities |
| `ListVerifiedEmailAddresses` | Returns all verified email addresses |

## Examples

### AWS CLI

```bash
# Verify a sender address
aws ses verify-email-identity --email-address sender@example.com

# List verified identities
aws ses list-identities

# Send an email (recorded, not delivered)
aws ses send-email \
  --from sender@example.com \
  --destination '{"ToAddresses":["recipient@example.com"]}' \
  --message '{
    "Subject": {"Data": "Hello"},
    "Body": {"Text": {"Data": "Hello from cloudmock"}}
  }'

# Check verification status
aws ses get-identity-verification-attributes \
  --identities sender@example.com
```

### Python (boto3)

```python
import boto3

ses = boto3.client("ses", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Verify sender
ses.verify_email_identity(EmailAddress="noreply@myapp.com")

# Send (no actual delivery)
response = ses.send_email(
    Source="noreply@myapp.com",
    Destination={"ToAddresses": ["user@example.com"]},
    Message={
        "Subject": {"Data": "Welcome!"},
        "Body": {"Html": {"Data": "<h1>Welcome to our app</h1>"}},
    },
)
print(response["MessageId"])

# Verify email was recorded as sent
identities = ses.list_verified_email_addresses()
print(identities["VerifiedEmailAddresses"])
```

## Notes

- Email messages are stored in memory and accessible for test assertions via `SendEmail` return values and inspection through the dashboard. No SMTP delivery occurs.
- All identities are immediately reported as `Success` in verification status.
- SES v2 API (`sesv2` service name) is not implemented.
- Configuration sets, suppression lists, and dedicated IPs are not implemented.

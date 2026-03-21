# Route 53

**Tier:** 1 (Full Emulation)
**Protocol:** REST-XML
**Service Name:** `route53`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateHostedZone` | Creates a public or private hosted zone |
| `ListHostedZones` | Returns all hosted zones |
| `GetHostedZone` | Returns details for a specific zone |
| `DeleteHostedZone` | Deletes a hosted zone |
| `ChangeResourceRecordSets` | Creates, updates, or deletes DNS records |
| `ListResourceRecordSets` | Returns all DNS records in a zone |

## Examples

### AWS CLI

```bash
# Create a hosted zone
aws route53 create-hosted-zone \
  --name example.com \
  --caller-reference unique-ref-1

# List zones
aws route53 list-hosted-zones

# Create an A record
aws route53 change-resource-record-sets \
  --hosted-zone-id /hostedzone/Z1234567890ABC \
  --change-batch '{
    "Changes": [{
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "api.example.com",
        "Type": "A",
        "TTL": 300,
        "ResourceRecords": [{"Value": "10.0.0.1"}]
      }
    }]
  }'

# List records
aws route53 list-resource-record-sets \
  --hosted-zone-id /hostedzone/Z1234567890ABC
```

### Python (boto3)

```python
import boto3

r53 = boto3.client("route53", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Create zone
zone = r53.create_hosted_zone(Name="internal.example.com", CallerReference="ref-1")
zone_id = zone["HostedZone"]["Id"]

# Upsert record
r53.change_resource_record_sets(
    HostedZoneId=zone_id,
    ChangeBatch={
        "Changes": [{
            "Action": "UPSERT",
            "ResourceRecordSet": {
                "Name": "db.internal.example.com",
                "Type": "CNAME",
                "TTL": 60,
                "ResourceRecords": [{"Value": "rds-endpoint.us-east-1.rds.amazonaws.com"}],
            },
        }]
    },
)

# List records
records = r53.list_resource_record_sets(HostedZoneId=zone_id)
for rrs in records["ResourceRecordSets"]:
    print(rrs["Name"], rrs["Type"])
```

## Notes

- DNS resolution is not performed — records are stored for reference only.
- `ChangeResourceRecordSets` supports `CREATE`, `DELETE`, and `UPSERT` actions.
- Alias records are accepted and stored but not resolved.
- Traffic policies, health checks, and DNSSEC are not implemented.

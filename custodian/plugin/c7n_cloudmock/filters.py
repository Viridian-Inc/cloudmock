"""Custom Cloud Custodian filters for cloudmock."""
from c7n.filters.core import Filter
from c7n import utils

class TierFilter(Filter):
    """Filter resources by cloudmock service tier."""
    schema = utils.type_schema("cloudmock-tier", tier={"type": "integer", "enum": [1, 2]})

    def process(self, resources, event=None):
        tier = self.data.get("tier", 1)
        # Tier 1 services are the fully emulated ones
        tier1_services = {
            "s3", "dynamodb", "sqs", "sns", "lambda", "iam", "sts",
            "cognito", "apigateway", "cloudformation", "cloudwatch",
            "logs", "events", "states", "secretsmanager", "kms",
            "ssm", "route53", "ecr", "ecs", "rds", "ses", "kinesis",
            "firehose", "ec2", "monitoring"
        }
        if tier == 1:
            return [r for r in resources if r.get("service", "") in tier1_services]
        return [r for r in resources if r.get("service", "") not in tier1_services]

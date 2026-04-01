#!/usr/bin/env bash
#
# create-dynamodb-table.sh — Creates the cloudmock-data DynamoDB table
# with a feature-time-index GSI and TTL enabled.
#
# Usage:
#   ./scripts/create-dynamodb-table.sh                          # AWS
#   ./scripts/create-dynamodb-table.sh --endpoint http://localhost:4566  # LocalStack / CloudMock
#
# Environment variables:
#   CLOUDMOCK_DYNAMODB_TABLE  — table name (default: cloudmock-data)
#   AWS_REGION                — AWS region (default: us-east-1)
set -euo pipefail

TABLE_NAME="${CLOUDMOCK_DYNAMODB_TABLE:-cloudmock-data}"
REGION="${AWS_REGION:-us-east-1}"
ENDPOINT_ARGS=()

# Parse --endpoint flag
while [[ $# -gt 0 ]]; do
  case $1 in
    --endpoint)
      ENDPOINT_ARGS=(--endpoint-url "$2")
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

echo "Creating DynamoDB table: ${TABLE_NAME} in ${REGION}"

aws dynamodb create-table \
  --region "${REGION}" \
  "${ENDPOINT_ARGS[@]}" \
  --table-name "${TABLE_NAME}" \
  --attribute-definitions \
    AttributeName=pk,AttributeType=S \
    AttributeName=sk,AttributeType=S \
    AttributeName=feature,AttributeType=S \
    AttributeName=created_at,AttributeType=S \
  --key-schema \
    AttributeName=pk,KeyType=HASH \
    AttributeName=sk,KeyType=RANGE \
  --global-secondary-indexes \
    '[{
      "IndexName": "feature-time-index",
      "KeySchema": [
        {"AttributeName": "feature", "KeyType": "HASH"},
        {"AttributeName": "created_at", "KeyType": "RANGE"}
      ],
      "Projection": {"ProjectionType": "ALL"}
    }]' \
  --billing-mode PAY_PER_REQUEST \
  --tags Key=Service,Value=cloudmock

echo "Enabling TTL on attribute 'ttl'..."

aws dynamodb update-time-to-live \
  --region "${REGION}" \
  "${ENDPOINT_ARGS[@]}" \
  --table-name "${TABLE_NAME}" \
  --time-to-live-specification "Enabled=true,AttributeName=ttl"

echo "Done. Table '${TABLE_NAME}' is ready."
echo ""
echo "To use with CloudMock:"
echo "  CLOUDMOCK_DATAPLANE_MODE=dynamodb \\"
echo "  CLOUDMOCK_DYNAMODB_TABLE=${TABLE_NAME} \\"
echo "  ./gateway"
echo ""
echo "For local development (dogfooding CloudMock's own DynamoDB):"
echo "  CLOUDMOCK_DATAPLANE_MODE=dynamodb \\"
echo "  CLOUDMOCK_DYNAMODB_TABLE=${TABLE_NAME} \\"
echo "  CLOUDMOCK_DYNAMODB_ENDPOINT=http://localhost:4566 \\"
echo "  ./gateway"

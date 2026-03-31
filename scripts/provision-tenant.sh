#!/usr/bin/env bash
# Usage: ./scripts/provision-tenant.sh <slug> <clerk-org-id>
# Manually provision a tenant (useful for testing)
set -euo pipefail
SLUG="${1:?Usage: provision-tenant.sh <slug> <clerk-org-id>}"
CLERK_ORG="${2:?Usage: provision-tenant.sh <slug> <clerk-org-id>}"
ADMIN_URL="${CLOUDMOCK_ADMIN_URL:-http://localhost:4599}"

curl -s -X POST "${ADMIN_URL}/api/tenants" \
  -H "Content-Type: application/json" \
  -d "{\"clerk_org_id\": \"${CLERK_ORG}\", \"name\": \"${SLUG}\", \"slug\": \"${SLUG}\", \"tier\": \"pro\"}"

echo "Tenant ${SLUG} provisioned."

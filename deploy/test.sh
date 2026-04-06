#!/usr/bin/env bash
set -euo pipefail

# Test the Pulumi deployment against a local CloudMock instance.
# This validates that all AWS resources (ECS, ALB, Route53, IAM, etc.)
# can be created successfully before deploying to real AWS.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLOUDMOCK_PID=""
STACK_NAME="test"

cleanup() {
    echo ""
    echo "--- Cleanup ---"
    if [ -n "$CLOUDMOCK_PID" ] && kill -0 "$CLOUDMOCK_PID" 2>/dev/null; then
        echo "Stopping CloudMock (pid $CLOUDMOCK_PID)..."
        kill "$CLOUDMOCK_PID" 2>/dev/null || true
        wait "$CLOUDMOCK_PID" 2>/dev/null || true
    fi
    # Destroy test stack resources (ignore errors during cleanup)
    cd "$SCRIPT_DIR"
    pulumi destroy --stack "$STACK_NAME" --yes --non-interactive 2>/dev/null || true
}
trap cleanup EXIT

echo "=== CloudMock Deploy Test ==="
echo ""

# ------------------------------------------------------------------
# Step 1: Build and start CloudMock
# ------------------------------------------------------------------
echo "--- Step 1: Start CloudMock ---"
cd "$REPO_ROOT"
go build -o /tmp/cloudmock-test-binary ./cmd/gateway/
/tmp/cloudmock-test-binary --profile full &
CLOUDMOCK_PID=$!

# Wait for CloudMock to be ready
echo "Waiting for CloudMock..."
for i in $(seq 1 30); do
    if curl -sf http://localhost:4566/ > /dev/null 2>&1; then
        echo "  CloudMock ready (${i}s)"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "  ERROR: CloudMock failed to start within 30s"
        exit 1
    fi
    sleep 1
done
echo ""

# ------------------------------------------------------------------
# Step 2: Install dependencies
# ------------------------------------------------------------------
echo "--- Step 2: Install dependencies ---"
cd "$SCRIPT_DIR"
npm install --silent
echo "  Dependencies installed"
echo ""

# ------------------------------------------------------------------
# Step 3: Initialize test stack
# ------------------------------------------------------------------
echo "--- Step 3: Initialize test stack ---"
pulumi stack select "$STACK_NAME" 2>/dev/null || pulumi stack init "$STACK_NAME" --non-interactive
echo "  Stack: $STACK_NAME"
echo ""

# ------------------------------------------------------------------
# Step 4: Preview (dry-run against CloudMock)
# ------------------------------------------------------------------
echo "--- Step 4: Preview ---"
if pulumi preview --stack "$STACK_NAME" --non-interactive --diff 2>&1; then
    echo "  Preview: PASS"
else
    echo "  Preview: FAIL"
    exit 1
fi
echo ""

# ------------------------------------------------------------------
# Step 5: Deploy to CloudMock
# ------------------------------------------------------------------
echo "--- Step 5: Deploy to CloudMock ---"
if pulumi up --stack "$STACK_NAME" --yes --non-interactive 2>&1; then
    echo "  Deploy: PASS"
else
    echo "  Deploy: FAIL"
    exit 1
fi
echo ""

# ------------------------------------------------------------------
# Step 6: Verify resources were created
# ------------------------------------------------------------------
echo "--- Step 6: Verify resources ---"
ENDPOINT="http://localhost:4566"
FAIL=0

# Verify ECS cluster
echo -n "  ECS cluster: "
if curl -sf -X POST "$ENDPOINT" \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
    -d '{}' | grep -q "cloudmock-test"; then
    echo "OK"
else
    echo "MISSING"
    FAIL=1
fi

# Verify ECS service
echo -n "  ECS service: "
if curl -sf -X POST "$ENDPOINT" \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListServices" \
    -d '{"cluster":"cloudmock-test-cluster"}' | grep -q "cloudmock-test"; then
    echo "OK"
else
    echo "MISSING"
    FAIL=1
fi

# Verify ECS task definition
echo -n "  ECS task def: "
if curl -sf -X POST "$ENDPOINT" \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListTaskDefinitions" \
    -d '{}' | grep -q "cloudmock-test"; then
    echo "OK"
else
    echo "MISSING"
    FAIL=1
fi

# Verify Route 53 hosted zone
echo -n "  Route 53 zone: "
if curl -sf "$ENDPOINT/2013-04-01/hostedzone" | grep -q "cloudmock.test"; then
    echo "OK"
else
    echo "MISSING"
    FAIL=1
fi

# Verify CloudWatch log group
echo -n "  Log group: "
if curl -sf -X POST "$ENDPOINT" \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: Logs_20140328.DescribeLogGroups" \
    -d '{"logGroupNamePrefix":"/ecs/cloudmock-test"}' | grep -q "cloudmock-test"; then
    echo "OK"
else
    echo "MISSING"
    FAIL=1
fi

echo ""
if [ "$FAIL" -eq 0 ]; then
    echo "=== ALL TESTS PASSED ==="
else
    echo "=== SOME TESTS FAILED ==="
    exit 1
fi

# ------------------------------------------------------------------
# Step 7: Verify stack outputs
# ------------------------------------------------------------------
echo ""
echo "--- Stack outputs ---"
pulumi stack output --stack "$STACK_NAME" --json
echo ""

echo "=== Test complete ==="

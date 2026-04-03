#!/bin/bash
set -e
for stack in minimal serverless data-pipeline; do
  echo "Testing $stack..."
  cd "$(dirname "$0")/$stack"
  docker compose up -d
  sleep 5
  curl -sf http://localhost:4566/ > /dev/null
  echo "  $stack: OK"
  docker compose down
  cd ..
done

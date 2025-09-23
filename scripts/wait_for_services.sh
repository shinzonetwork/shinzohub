#!/bin/bash

set -e

# Wait for services to be ready before running integration tests
# This script checks that both registrar and DefraDB are responding

echo "===> Waiting for services to be ready..."

READY_FILE=".shinzohub/ready"

# Wait for ready file from bootstrap.sh
echo "===> Waiting for $READY_FILE to be created by bootstrap.sh..."
for i in {1..30}; do
  if [ -f "$READY_FILE" ]; then
    echo "===> $READY_FILE found."
    break
  fi
  sleep 1
done

# Wait for sourcehub to be ready
echo "Checking sourcehub at http://localhost:26657..."
for i in {1..60}; do
    if curl -s http://localhost:26657 > /dev/null 2>&1; then
        echo "✓ Sourcehub is responding"
        break
    fi
    
    if [ $i -eq 60 ]; then
        echo "✗ Sourcehub failed to start within 60 seconds"
        exit 1
    fi
    
    echo "Waiting for sourcehub... (attempt $i/60)"
    sleep 1
done

# Wait for DefraDB to be ready
GRAPHQL_URL="http://localhost:9181/api/v0/graphql"
echo "===> Waiting for GraphQL endpoint at $GRAPHQL_URL to be ready (schema applied)..."
for i in {1..30}; do
  RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
    --data '{"query":"{ Block { __typename } }"}' "$GRAPHQL_URL" || true)

  if echo "$RESPONSE" | grep -q '"Block"'; then
    echo "✓ GraphQL endpoint is up and schema is applied."
    break
  fi

  if [ $i -eq 30 ]; then
    echo "✗ GraphQL endpoint did not become ready within 30 seconds"
    exit 1
  fi

  sleep 1
done

# Create the ready file to indicate services are up
echo "✓ All services are ready!"

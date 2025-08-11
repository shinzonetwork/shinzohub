#!/bin/bash

set -e

# Test script for Access Control Policies
# This script will:
# 1. Bootstrap the system if not already running
# 2. Run the ACP integration tests
# 3. Report results

echo "=== Shinzo ACP Testing ==="

# Check if services are already running
if [[ -f ".shinzohub/ready" ]]; then
    echo "Services appear to be running. Checking health..."
    
    # Check registrar health
    if curl -s http://localhost:8081/registrar/ > /dev/null 2>&1; then
        echo "✓ Registrar is responding"
    else
        echo "✗ Registrar is not responding"
        echo "Please ensure services are running with: make bootstrap"
        exit 1
    fi
    
    # Check DefraDB health
    if curl -s http://localhost:9181/graphql > /dev/null 2>&1; then
        echo "✓ DefraDB is responding"
    else
        echo "✗ DefraDB is not responding"
        echo "Please ensure services are running with: make bootstrap"
        exit 1
    fi
else
    echo "Services not running. Please start them first with:"
    echo "  make bootstrap SOURCEHUB_PATH=/path/to/sourcehub INDEXER_PATH=/path/to/indexer"
    exit 1
fi

echo ""
echo "=== Running Integration Tests ==="

echo ""
echo "=== Running ACP Integration Tests ==="

# Run the Go tests
cd "$(dirname "$0")/.."
go test -v ./tests -run TestAccessControl

if [[ $? -eq 0 ]]; then
    echo ""
    echo "=== Test Results ==="
    echo "✓ All ACP tests passed!"
    echo ""
    echo "The access control system is working correctly."
    echo "Users can only access resources they have permission for."
else
    echo ""
    echo "=== Test Results ==="
    echo "✗ Some ACP tests failed!"
    echo ""
    echo "Please check the test output above for details."
    exit 1
fi 
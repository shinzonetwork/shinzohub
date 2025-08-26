#!/bin/bash

set -e

echo "===> Waiting for sourcehub to be ready..."

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

echo "✓ Sourcehub ready!"

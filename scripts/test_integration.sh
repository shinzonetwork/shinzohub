#!/bin/bash
set -e

# Usage: ./scripts/test_integration.sh /path/to/sourcehub
# Or:   SOURCEHUB_PATH=/path/to/sourcehub ./scripts/test_integration.sh

if [[ -z "$SOURCEHUB_PATH" && -z "$1" ]]; then
  echo "ERROR: You must provide SOURCEHUB_PATH as an env variable or first argument."
  echo "Usage: ./scripts/test_integration.sh /path/to/sourcehub"
  exit 1
fi

SOURCEHUB_PATH_ARG="$SOURCEHUB_PATH"
if [[ -z "$SOURCEHUB_PATH_ARG" ]]; then
  SOURCEHUB_PATH_ARG="$1"
  shift
fi

READY_FILE=".shinzohub/ready"
if [ -f "$READY_FILE" ]; then
  rm -f "$READY_FILE"
fi

echo "===> Bootstrapping system in background..."
make bootstrap SOURCEHUB_PATH="$SOURCEHUB_PATH_ARG" "$@" &
BOOTSTRAP_PID=$!

# Cleanup on exit
cleanup() {
  echo "===> Cleaning up bootstrap process (PID $BOOTSTRAP_PID)..."
  kill $BOOTSTRAP_PID 2>/dev/null || true
  wait $BOOTSTRAP_PID 2>/dev/null || true
}
trap cleanup EXIT INT TERM

scripts/wait_for_services.sh

echo "===> Running ACP integration tests..."
go test -v ./tests -run TestAccessControl > logs/integration_test_output.txt 2>&1 || true

echo -e "\n\n===> Integration test output:"
cat logs/integration_test_output.txt
echo "===> Tests finished. Inspect test output at logs/integration_test_output.txt"

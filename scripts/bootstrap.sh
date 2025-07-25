#!/bin/bash

set -e

# Usage: ./scripts/bootstrap.sh /path/to/sourcehub [INDEXER_PATH=/path/to/indexer]  (or set INDEXER_PATH env var)

SOURCEHUB_PATH="$1"
INDEXER_PATH="${INDEXER_PATH:-$2}"
ROOTDIR="$(pwd)/.shinzohub"
LOGDIR="logs"
SOURCEHUB_LOG_PATH="$LOGDIR/sourcehub_logs.txt"
SHINZOHUBD_LOG_PATH="$LOGDIR/shinzohubd_logs.txt"
REGISTRAR_LOG_PATH="$LOGDIR/registrar_logs.txt"
INDEXER_BOOTSTRAP_LOG_PATH="$LOGDIR/indexer_bootstrap_logs.txt"

# Expand ~ to $HOME if present
SOURCEHUB_PATH="${SOURCEHUB_PATH/#\~/$HOME}"
INDEXER_PATH="${INDEXER_PATH/#\~/$HOME}"

if [[ -z "$SOURCEHUB_PATH" ]]; then
  echo "ERROR: You must pass SOURCEHUB_PATH. Usage:"
  echo "  make bootstrap SOURCEHUB_PATH=/path/to/sourcehub INDEXER_PATH=/path/to/indexer"
  exit 1
fi
if [[ -z "$INDEXER_PATH" ]]; then
  echo "ERROR: You must pass INDEXER_PATH (as env var or 2nd arg). Usage:"
  echo "  make bootstrap SOURCEHUB_PATH=/path/to/sourcehub INDEXER_PATH=/path/to/indexer"
  exit 1
fi

SOURCEHUB_ROOT="$(cd "$SOURCEHUB_PATH" && pwd)"
INDEXER_ROOT="$(cd "$INDEXER_PATH" && pwd)"

mkdir -p "$LOGDIR"
mkdir -p "$ROOTDIR"

# Build and run SourceHub
echo "===> Building SourceHub from $SOURCEHUB_ROOT"
cd "$SOURCEHUB_ROOT"
./build/sourcehubd start > "$OLDPWD/$SOURCEHUB_LOG_PATH" 2>&1 &
SOURCEHUB_PID=$!
echo "$SOURCEHUB_PID" > "$ROOTDIR/sourcehubd.pid"
echo "Started sourcehubd (PID $SOURCEHUB_PID). Logs at $SOURCEHUB_LOG_PATH"
cd "$OLDPWD"

# Build and run shinzohubd
echo "===> Building shinzohubd"
go build -o bin/shinzohubd cmd/shinzohubd/main.go
./bin/shinzohubd start > "$SHINZOHUBD_LOG_PATH" 2>&1 &
SHINZOHUBD_PID=$!
echo "$SHINZOHUBD_PID" > "$ROOTDIR/shinzohubd.pid"
echo "Started shinzohubd (PID $SHINZOHUBD_PID). Logs at $SHINZOHUBD_LOG_PATH"

# Build and run registrar
echo "===> Building registrar"
go build -o bin/registrar cmd/registrar/main.go
./bin/registrar > "$REGISTRAR_LOG_PATH" 2>&1 &
REGISTRAR_PID=$!
echo "$REGISTRAR_PID" > "$ROOTDIR/registrar.pid"
echo "Started registrar (PID $REGISTRAR_PID). Logs at $REGISTRAR_LOG_PATH"

# Inject policy id into indexer schema before starting indexer bootstrap
POLICY_ID_FILE="$ROOTDIR/policy_id"
SCHEMA_FILE="$INDEXER_ROOT/schema/schema.graphql"
if [[ ! -f "$POLICY_ID_FILE" ]]; then
  echo "ERROR: Policy ID file not found at $POLICY_ID_FILE. Cannot update schema."
  exit 1
fi
POLICY_ID=$(cat "$POLICY_ID_FILE")
if [[ -z "$POLICY_ID" ]]; then
  echo "ERROR: Policy ID is empty in $POLICY_ID_FILE. Cannot update schema."
  exit 1
fi
if [[ ! -f "$SCHEMA_FILE" ]]; then
  echo "ERROR: Schema file not found at $SCHEMA_FILE."
  exit 1
fi
# Replace <replace_with_policy_id> with actual policy id (removing chevrons)
sed -i.bak "s/<replace_with_policy_id>/$POLICY_ID/g" "$SCHEMA_FILE"

# Start indexer bootstrap (DefraDB + block_poster)
echo "===> Bootstrapping indexer (DefraDB/block_poster) from $INDEXER_ROOT"
(cd "$INDEXER_ROOT" && ./scripts/bootstrap.sh "$INDEXER_ROOT/../defradb" > "$OLDPWD/$INDEXER_BOOTSTRAP_LOG_PATH" 2>&1 &)
INDEXER_BOOTSTRAP_PID=$!
echo "$INDEXER_BOOTSTRAP_PID" > "$ROOTDIR/indexer_bootstrap.pid"
echo "Started indexer bootstrap (PID $INDEXER_BOOTSTRAP_PID). Logs at $INDEXER_BOOTSTRAP_LOG_PATH"

sleep 3

# Check if processes are running
if ! kill -0 $SOURCEHUB_PID 2>/dev/null; then
  echo "ERROR: sourcehubd failed to start (PID $SOURCEHUB_PID not running)" >&2
  echo "--- sourcehubd log errors ---"
  grep -iE 'error|fail|panic|fatal' "$SOURCEHUB_LOG_PATH" || echo "No error lines found in $SOURCEHUB_LOG_PATH"
  cleanup
  exit 1
fi
if ! kill -0 $SHINZOHUBD_PID 2>/dev/null; then
  echo "ERROR: shinzohubd failed to start (PID $SHINZOHUBD_PID not running)" >&2
  echo "--- shinzohubd log errors ---"
  grep -iE 'error|fail|panic|fatal' "$SHINZOHUBD_LOG_PATH" || echo "No error lines found in $SHINZOHUBD_LOG_PATH"
  # cleanup
  # exit 1
fi
if ! kill -0 $REGISTRAR_PID 2>/dev/null; then
  echo "ERROR: registrar failed to start (PID $REGISTRAR_PID not running)" >&2
  echo "--- registrar log errors ---"
  grep -iE 'error|fail|panic|fatal' "$REGISTRAR_LOG_PATH" || echo "No error lines found in $REGISTRAR_LOG_PATH"
  cleanup
  exit 1
fi
if ! kill -0 $INDEXER_BOOTSTRAP_PID 2>/dev/null; then
  echo "ERROR: indexer bootstrap failed to start (PID $INDEXER_BOOTSTRAP_PID not running)" >&2
  echo "--- indexer bootstrap log errors ---"
  grep -iE 'error|fail|panic|fatal' "$INDEXER_BOOTSTRAP_LOG_PATH" || echo "No error lines found in $INDEXER_BOOTSTRAP_LOG_PATH"
  cleanup
  exit 1
fi

# Run setup_policy.sh to upload policy and create groups
echo "===> Setting up policy and groups"
if ! scripts/setup_policy.sh; then
  echo "ERROR: setup_policy.sh failed. Exiting bootstrap."
  cleanup
  exit 1
fi

# Create an empty file to indicate that services are ready
READY_FILE="$ROOTDIR/ready"
echo "===> Ready"
touch "$READY_FILE"

# Define cleanup function for robust process cleanup
cleanup() {
  echo "Stopping sourcehubd..."
  if [[ -f "$ROOTDIR/sourcehubd.pid" ]]; then
    kill -9 $(cat "$ROOTDIR/sourcehubd.pid") 2>/dev/null || true
    rm -f "$ROOTDIR/sourcehubd.pid"
  fi
  # Failsafe: kill any remaining sourcehubd processes
  SOURCEHUB_PIDS=$(ps aux | grep '[s]ourcehubd' | awk '{print $2}')
  if [[ -n "$SOURCEHUB_PIDS" ]]; then
    echo "Killing remaining sourcehubd PIDs: $SOURCEHUB_PIDS"
    echo "$SOURCEHUB_PIDS" | xargs -r kill -9 2>/dev/null || true
  fi
  echo "Stopping shinzohubd..."
  if [[ -f "$ROOTDIR/shinzohubd.pid" ]]; then
    kill -9 $(cat "$ROOTDIR/shinzohubd.pid") 2>/dev/null || true
    rm -f "$ROOTDIR/shinzohubd.pid"
  fi
  echo "Stopping registrar..."
  if [[ -f "$ROOTDIR/registrar.pid" ]]; then
    kill -9 $(cat "$ROOTDIR/registrar.pid") 2>/dev/null || true
    rm -f "$ROOTDIR/registrar.pid"
  fi
  echo "Stopping indexer bootstrap..."
  if [[ -f "$ROOTDIR/indexer_bootstrap.pid" ]]; then
    kill -9 $(cat "$ROOTDIR/indexer_bootstrap.pid") 2>/dev/null || true
    rm -f "$ROOTDIR/indexer_bootstrap.pid"
  fi
  # Failsafe: kill any remaining defradb/block_poster processes
  DEFRA_PIDS=$(ps aux | grep '[d]efradb start --rootdir ' | awk '{print $2}')
  if [[ -n "$DEFRA_PIDS" ]]; then
    echo "Killing remaining defradb PIDs: $DEFRA_PIDS"
    echo "$DEFRA_PIDS" | xargs -r kill -9 2>/dev/null || true
  fi
  POSTER_PIDS=$(ps aux | grep '[b]lock_poster' | awk '{print $2}')
  if [[ -n "$POSTER_PIDS" ]]; then
    echo "Killing remaining block_poster PIDs: $POSTER_PIDS"
    echo "$POSTER_PIDS" | xargs -r kill -9 2>/dev/null || true
  fi
  # Restore schema file to original state
  if [[ -f "$SCHEMA_FILE" && -n "$POLICY_ID" ]]; then
    ESCAPED_POLICY_ID=$(printf '%s\n' "$POLICY_ID" | sed 's/[\\/&|]/\\&/g')
    sed -i "" "s|$ESCAPED_POLICY_ID|<replace_with_policy_id>|g" "$SCHEMA_FILE"
  fi
  rm -f "$SCHEMA_FILE.bak"
  rm -f "$READY_FILE"
  exit 0
}
trap cleanup INT TERM EXIT

# Wait forever until killed, so trap always runs
while true; do sleep 1; done 
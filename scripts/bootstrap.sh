#!/bin/bash

set -e

SOURCEHUB_PATH="$1"
ROOTDIR="$(pwd)/.shinzohub"
LOGDIR="logs"
SOURCEHUB_LOG_PATH="$LOGDIR/sourcehub_logs.txt"
SHINZOHUBD_LOG_PATH="$LOGDIR/shinzohubd_logs.txt"
REGISTRAR_LOG_PATH="$LOGDIR/registrar_logs.txt"

# Expand ~ to $HOME if present
SOURCEHUB_PATH="${SOURCEHUB_PATH/#\~/$HOME}"

if [[ -z "$SOURCEHUB_PATH" ]]; then
  echo "ERROR: You must pass SOURCEHUB_PATH. Usage:"
  echo "  make bootstrap SOURCEHUB_PATH=/path/to/sourcehub"
  exit 1
fi

SOURCEHUB_ROOT="$(cd "$SOURCEHUB_PATH" && pwd)"

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

sleep 3

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
  rm -f "$READY_FILE"
  exit 0
}
trap cleanup INT TERM

# Check if processes are running
if ! kill -0 $SOURCEHUB_PID 2>/dev/null; then
  echo "ERROR: sourcehubd failed to start (PID $SOURCEHUB_PID not running)" >&2
  echo "--- sourcehubd log errors ---"
  grep -iE 'error|fail|panic|fatal' "$SOURCEHUB_LOG_PATH" || echo "No error lines found in $SOURCEHUB_LOG_PATH"
  exit 1
fi
if ! kill -0 $SHINZOHUBD_PID 2>/dev/null; then
  echo "ERROR: shinzohubd failed to start (PID $SHINZOHUBD_PID not running)" >&2
  echo "--- shinzohubd log errors ---"
  grep -iE 'error|fail|panic|fatal' "$SHINZOHUBD_LOG_PATH" || echo "No error lines found in $SHINZOHUBD_LOG_PATH"
  # exit 1
fi
if ! kill -0 $REGISTRAR_PID 2>/dev/null; then
  echo "ERROR: registrar failed to start (PID $REGISTRAR_PID not running)" >&2
  echo "--- registrar log errors ---"
  grep -iE 'error|fail|panic|fatal' "$REGISTRAR_LOG_PATH" || echo "No error lines found in $REGISTRAR_LOG_PATH"
  exit 1
fi

# Wait forever until killed, so trap always runs
while true; do sleep 1; done 
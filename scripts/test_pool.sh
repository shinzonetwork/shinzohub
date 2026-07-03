#!/usr/bin/env bash
# Run the pool module's tests end-to-end:
#   1. regenerate proto (in case proto/ changed)
#   2. verify the whole module graph compiles
#   3. run the pool keeper unit + integration tests
#   4. run the poolregistry precompile tests
#
# Usage:
#   ./scripts/test_pool.sh                  # full run
#   ./scripts/test_pool.sh --skip-proto     # skip proto-gen step
#   ./scripts/test_pool.sh -v               # verbose `go test -v`

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SKIP_PROTO=0
GOTEST_FLAGS=()

for arg in "$@"; do
    case "$arg" in
        --skip-proto) SKIP_PROTO=1 ;;
        -v|--verbose) GOTEST_FLAGS+=("-v") ;;
        *) echo "Unknown flag: $arg" >&2; exit 1 ;;
    esac
done

step() { echo -e "\n\033[1;36m==> $*\033[0m"; }
ok()   { echo -e "\033[1;32m✓ $*\033[0m"; }
fail() { echo -e "\033[1;31m✗ $*\033[0m"; exit 1; }

if [[ "$SKIP_PROTO" -eq 0 ]]; then
    step "Regenerating protobuf"
    make proto-gen || fail "proto-gen failed"
    ok "protos up to date"
fi

step "Compiling shinzohub"
go build ./... || fail "build failed"
ok "builds clean"

step "Running x/pool keeper tests"
go test "${GOTEST_FLAGS[@]}" -count=1 ./x/pool/... || fail "x/pool tests failed"
ok "x/pool tests passed"

step "Running poolregistry precompile tests"
if compgen -G "./app/precompiles/poolregistry/*_test.go" > /dev/null; then
    go test "${GOTEST_FLAGS[@]}" -count=1 ./app/precompiles/poolregistry/... || fail "poolregistry tests failed"
    ok "poolregistry tests passed"
else
    echo "  (no precompile test files yet — skipping)"
fi

echo -e "\n\033[1;32mAll pool tests passed.\033[0m"

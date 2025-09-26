#!/usr/bin/env bash
set -euo pipefail

# ----- inputs -----
LEDGER_ENABLED="${LEDGER_ENABLED:-}"
OS="${OS:-$(uname -s)}"
COSMOS_BUILD_OPTIONS="${COSMOS_BUILD_OPTIONS:-}"
VERSION="${VERSION:-}"
COMMIT="${COMMIT:-}"
CMT_VERSION="${CMT_VERSION:-}"
LDFLAGS_EXTRA="${LDFLAGS:-}"
BUILD_TAGS_EXTRA="${BUILD_TAGS:-}"

# Where to put the binary and what to call it
BUILD_DIR="${BUILD_DIR:-build}"
BINARY_NAME="${BINARY_NAME:-shinzohubd}"
# Main package to build
MAIN_PKG="${MAIN_PKG:-./cmd/shinzohubd}"

# ----- helpers -----
has_cosmos_build_opt() { [[ "$COSMOS_BUILD_OPTIONS" == *"$1"* ]]; }
trim_spaces() { awk '{$1=$1; print}'; }
warn() { echo "warning: $*" >&2; }
die()  { echo "error: $*" >&2; exit 1; }

# ----- build tags base -----
build_tags="netgo"

# ----- Ledger support checks -----
if [[ "${LEDGER_ENABLED}" == "true" ]]; then
  if [[ "${OS}" == "Windows_NT" ]]; then
    if command -v where >/dev/null 2>&1; then
      where gcc.exe >/dev/null 2>&1 || die "gcc.exe not installed for ledger support, set LEDGER_ENABLED=false or install gcc"
    else
      command -v gcc.exe >/dev/null 2>&1 || die "gcc.exe not installed for ledger support, set LEDGER_ENABLED=false or install gcc"
    fi
    build_tags+=" ledger"
  else
    uname_s="$(uname -s 2>/dev/null || echo unknown)"
    if [[ "${uname_s}" == "OpenBSD" ]]; then
      warn "OpenBSD detected, disabling ledger support"
    else
      command -v gcc >/dev/null 2>&1 || die "gcc not installed for ledger support, set LEDGER_ENABLED=false or install gcc"
      build_tags+=" ledger"
    fi
  fi
fi

# ----- Optional features from COSMOS_BUILD_OPTIONS -----
has_cosmos_build_opt "secp"     && build_tags+=" libsecp256k1_sdk"
has_cosmos_build_opt "legacy"   && build_tags+=" app_v1"
has_cosmos_build_opt "cleveldb" && build_tags+=" gcc"
has_cosmos_build_opt "badgerdb" && build_tags+=" badgerdb"
if has_cosmos_build_opt "rocksdb"; then export CGO_ENABLED=1; build_tags+=" rocksdb"; fi
has_cosmos_build_opt "boltdb"   && build_tags+=" boltdb"

# Append external build tags
[[ -n "${BUILD_TAGS_EXTRA}" ]] && build_tags+=" ${BUILD_TAGS_EXTRA}"
build_tags="$(printf '%s\n' "${build_tags}" | trim_spaces)"
build_tags_comma_sep="$(printf '%s' "${build_tags}" | tr -s ' ' ',')"

# ----- ldflags -----
ldflags=()
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.Name=${BINARY_NAME}")
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.AppName=${BINARY_NAME}")
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.Version=${VERSION}")
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.Commit=${COMMIT}")
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.BuildTags=${build_tags_comma_sep}")
ldflags+=("-X" "github.com/cometbft/cometbft/version.TMCoreSemVer=${CMT_VERSION}")
ldflags+=("-s" "-w")
if ! has_cosmos_build_opt "nostrip"; then ldflags+=("-w" "-s"); fi
if [[ -n "${LDFLAGS_EXTRA}" ]]; then read -r -a extra_arr <<<"${LDFLAGS_EXTRA}"; ldflags+=("${extra_arr[@]}"); fi
ldflags_str="$(printf '%s ' "${ldflags[@]}" | trim_spaces)"

# ----- go build args -----
go_args=(-tags "${build_tags}" -ldflags "${ldflags_str}")
if ! has_cosmos_build_opt "nostrip"; then go_args+=(-trimpath); fi
if has_cosmos_build_opt "debug"; then go_args+=(-gcflags 'all=-N -l'); fi

# ----- build -----
mkdir -p "${BUILD_DIR}"
echo ">> go build -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PKG}"
go build "${go_args[@]}" -o "${BUILD_DIR}/${BINARY_NAME}" "${MAIN_PKG}"

echo "✅ Built ${BUILD_DIR}/${BINARY_NAME}"

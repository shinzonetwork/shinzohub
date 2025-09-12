#!/usr/bin/env bash
set -euo pipefail

# Read inputs from environment (use defaults)
LEDGER_ENABLED="${LEDGER_ENABLED:-}"
OS="${OS:-$(uname -s)}"
COSMOS_BUILD_OPTIONS="${COSMOS_BUILD_OPTIONS:-}"
VERSION="${VERSION:-}"
COMMIT="${COMMIT:-}"
CMT_VERSION="${CMT_VERSION:-}"
LDFLAGS_EXTRA="${LDFLAGS:-}"
BUILD_TAGS_EXTRA="${BUILD_TAGS:-}"
BUILD_DIR="${BUILD_DIR:-}"
SOURCE_PATH="${SOURCE_PATH:-}"

# --- helpers ---
has_cosmos_build_opt() {
  # returns 0 if $1 is a substring of COSMOS_BUILD_OPTIONS
  [[ "$COSMOS_BUILD_OPTIONS" == *"$1"* ]]
}

trim_spaces() {
  # collapse multiple spaces and trim ends
  awk '{$1=$1; print}'
}

warn() { echo "warning: $*" >&2; }
die()  { echo "error: $*" >&2; exit 1; }

# --- build_tags base ---
build_tags="netgo"

# --- Ledger support checks ---
if [[ "${LEDGER_ENABLED}" == "true" ]]; then
  if [[ "${OS}" == "Windows_NT" ]]; then
    # Prefer 'where', fall back to 'which' if unavailable
    if command -v where >/dev/null 2>&1; then
      if ! where gcc.exe >/dev/null 2>&1; then
        die "gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false"
      else
        build_tags+=" ledger"
      fi
    else
      # Git Bash / MSYS may not have 'where'; try which
      if ! command -v gcc.exe >/dev/null 2>&1; then
        die "gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false"
      else
        build_tags+=" ledger"
      fi
    fi
  else
    uname_s="$(uname -s 2>/dev/null || echo unknown)"
    if [[ "${uname_s}" == "OpenBSD" ]]; then
      warn "OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988)"
      # do not add ledger tag
    else
      if ! command -v gcc >/dev/null 2>&1; then
        die "gcc not installed for ledger support, please install or set LEDGER_ENABLED=false"
      else
        build_tags+=" ledger"
      fi
    fi
  fi
fi

# --- Optional features from COSMOS_BUILD_OPTIONS ---
if has_cosmos_build_opt "secp"; then
  build_tags+=" libsecp256k1_sdk"
fi

if has_cosmos_build_opt "legacy"; then
  build_tags+=" app_v1"
fi

# DB backend selection
if has_cosmos_build_opt "cleveldb"; then
  build_tags+=" gcc"
fi

if has_cosmos_build_opt "badgerdb"; then
  build_tags+=" badgerdb"
fi

if has_cosmos_build_opt "rocksdb"; then
  export CGO_ENABLED=1
  build_tags+=" rocksdb"
fi

if has_cosmos_build_opt "boltdb"; then
  build_tags+=" boltdb"
fi

# Append external build tags
if [[ -n "${BUILD_TAGS_EXTRA}" ]]; then
  build_tags+=" ${BUILD_TAGS_EXTRA}"
fi

# Normalize spaces
build_tags="$(printf '%s\n' "${build_tags}" | trim_spaces)"

# Comma-separated version for BuildTags ldflag
build_tags_comma_sep="$(printf '%s' "${build_tags}" | tr -s ' ' ',' )"

# --- ldflags base (include -s -w) ---
ldflags=()
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.Name=bankd")
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.AppName=bankd")
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.Version=${VERSION}")
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.Commit=${COMMIT}")
ldflags+=("-X" "github.com/cosmos/cosmos-sdk/version.BuildTags=${build_tags_comma_sep}")
ldflags+=("-X" "github.com/cometbft/cometbft/version.TMCoreSemVer=${CMT_VERSION}")
ldflags+=("-s" "-w")

# If nostrip is NOT present, add -w -s again
if ! has_cosmos_build_opt "nostrip"; then
  ldflags+=("-w" "-s")
fi

# Append external LDFLAGS (verbatim)
if [[ -n "${LDFLAGS_EXTRA}" ]]; then
  # shellcheck disable=SC2206
  extra_arr=(${LDFLAGS_EXTRA})
  ldflags+=("${extra_arr[@]}")
fi

# Join ldflags to a single string
# We will wrap the whole thing in single quotes later
ldflags_str="$(printf '%s ' "${ldflags[@]}" | trim_spaces)"

# --- BUILD_FLAGS composition ---
BUILD_FLAGS="-tags \"${build_tags}\" -ldflags '${ldflags_str}'"

# Add -trimpath when nostrip is NOT present
go_args=(-tags "${build_tags}" -ldflags "${ldflags_str}")

if ! has_cosmos_build_opt "nostrip"; then
  go_args+=(-trimpath)
fi

if has_cosmos_build_opt "debug"; then
  go_args+=(-gcflags 'all=-N -l')
fi

go build "${go_args[@]}" -o "${BUILD_DIR}/" "${SOURCE_PATH}"

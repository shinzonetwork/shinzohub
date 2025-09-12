# Justfile
set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

coverage_threshold := "15"
packages_unit := "go list ./... | grep -v '/tests/e2e'"
build_dir := env_var_or_default("BUILD_DIR", join(justfile_directory(), "build"))
version := `git describe --tags --always --match "v*" | sed 's/^v//'`
commit := `git log -1 --format='%H'`
cmt_version := `go list -m github.com/cometbft/cometbft | sed 's:.* ::'`
ledger_enabled := env_var_or_default("LEDGER_ENABLED", "true")

###############################################################################
###                                  Build                                  ###
###############################################################################

# Always produce ./build/shinzohubd
build: verify-deps
	@echo "⚙️  Building binary to {{build_dir}}..."
	@mkdir -p "{{build_dir}}"
	@LEDGER_ENABLED={{ledger_enabled}} \
	VERSION={{version}} \
	COMMIT={{commit}} \
	CMT_VERSION={{cmt_version}} \
	BUILD_DIR="{{build_dir}}" \
	BINARY_NAME=shinzohubd \
	MAIN_PKG=./cmd/shinzohubd \
	{{justfile_directory()}}/scripts/build.sh
	@if [ ! -f "{{build_dir}}/shinzohubd" ]; then \
	  echo "Binary not found at {{build_dir}}/shinzohubd. Searching..."; \
	  found="$(find . -type f -name shinzohubd -perm -111 | head -n1)"; \
	  if [ -n "$found" ]; then \
	    echo "Found at $found. Copying to {{build_dir}}/shinzohubd"; \
	    cp "$found" "{{build_dir}}/shinzohubd"; \
	  else \
	    echo "Build script did not produce shinzohubd. Check scripts/build.sh"; \
	    exit 1; \
	  fi; \
	fi
	@echo "✅ OK → {{build_dir}}/shinzohubd"

build-linux-amd64:
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=false just build

build-linux-arm64:
	GOOS=linux GOARCH=arm64 LEDGER_ENABLED=false just build

# Install to ~/.local/bin (no GOPATH/asdf headaches)
install: build
	@echo "🚀 Installing binary..."
	@dest="${HOME}/.local/bin"; \
	mkdir -p "$dest"; \
	cp "{{build_dir}}/shinzohubd" "$dest/shinzohubd"; \
	echo "✅ Installed to $dest/shinzohubd"; \
	if command -v asdf >/dev/null 2>&1; then asdf reshim golang || true; fi

# Optional: install into GOPATH/bin
install-gopath: build
	@echo "🚀 Installing to GOPATH/bin..."
	@gbin="$(go env GOPATH)/bin"; \
	mkdir -p "$gbin"; \
	cp "{{build_dir}}/shinzohubd" "$gbin/shinzohubd"; \
	echo "✅ Installed to $gbin/shinzohubd"; \
	if command -v asdf >/dev/null 2>&1; then asdf reshim golang || true; fi

verify-deps:
	@echo "🛠  Ensuring dependencies have not been modified ..."
	@go mod verify
	@go mod tidy

clean:
	@echo "🧹  Cleaning up..."
	@rm -rf "{{build_dir}}"

###############################################################################
###                                  Tests                                  ###
###############################################################################

test-unit:
	@echo "🧪  Running unit tests..."
	@go test -v $({{packages_unit}}) -count=1 -cover -coverprofile cover.out.tmp
	@cat cover.out.tmp | grep -vE "pb.go|/cmd" > cover.out
	@rm -f cover.out.tmp
	@go tool cover -func cover.out | grep total

test-coverage:
	#!/usr/bin/env bash
	set -euo pipefail
	command -v bc >/dev/null 2>&1 || { echo "bc not found. Install bc."; exit 1; }
	coverage_total="$([ -f cover.out ] && go tool cover -func=cover.out | grep total | grep -Eo '[0-9]+\.[0-9]+' || echo 0)"
	echo "Threshold:                {{coverage_threshold}}%"
	echo "Current test coverage is: $$coverage_total%"
	if [ "$(echo "$$coverage_total < {{coverage_threshold}}" | bc -l)" -eq 1 ]; then \
		echo "Test coverage is lower than threshold"; \
		exit 1; \
	fi

###############################################################################
###                                 Linting                                 ###
###############################################################################

lint:
	@echo "🧹  Linting..."
	@golangci-lint run --config .golangci.yml

lint-fix:
	@echo "🧹  Linting and fixing..."
	@golangci-lint run --config .golangci.yml --fix

###############################################################################
###                                Protobuf                                 ###
###############################################################################

proto-all: proto-format proto-lint proto-gen

proto-deps:
	@echo "⚙️  Installing Protobuf deps..."
	@go install github.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@v1.0.0-beta.5
	@go install github.com/cosmos/gogoproto/protoc-gen-gocosmos@v1.7.0
	@go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.16.0
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.8

proto-gen:
	@echo "⚙️  Generating Protobuf files..."
	@sh ./scripts/protocgen.sh true

proto-lint:
	@echo "⚙️  Linting Protobuf files..."
	@buf lint ./proto --error-format=json

proto-format:
	@echo "⚙️  Formatting Protobuf files..."
	@buf format ./proto -w

###############################################################################
###                                   E2E                                   ###
###############################################################################

ictest-basic:
	@echo "Running basic e2e test"
	@cd interchaintest && go test -race -v -run TestBasicChain .

ictest-ibc:
	@echo "Running IBC e2e test"
	@cd interchaintest && go test -race -v -run TestIBCBasic .

ictest-wasm:
	@echo "Running cosmwasm e2e test"
	@cd interchaintest && go test -race -v -run TestCosmWasmIntegration .

ictest-packetforward:
	@echo "Running packet forward middleware e2e test"
	@cd interchaintest && go test -race -v -run TestPacketForwardMiddleware .

ictest-poa:
	@echo "Running proof of authority e2e test"
	@cd interchaintest && go test -race -v -run TestPOA .

ictest-ratelimit:
	@echo "Running rate limit e2e test"
	@cd interchaintest && go test -race -v -run TestIBCRateLimit .

###############################################################################
###                                   Dev                                    ###
###############################################################################

sh-testnet: verify-deps
	CHAIN_ID="9001" BLOCK_TIME="1000ms" CLEAN=true sh scripts/test_node.sh

doctor:
	@echo "artifact   = {{build_dir}}/shinzohubd"
	@echo "exists?    = $$(test -f {{build_dir}}/shinzohubd && echo yes || echo no)"
	@echo "which bin  = $$(command -v shinzohubd || echo 'not on PATH')"

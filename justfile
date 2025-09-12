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

build: verify-deps
	@echo "⚙️  Building binary..."
	@LEDGER_ENABLED={{ledger_enabled}} \
	SOURCE_PATH=./... \
	VERSION={{version}} \
	COMMIT={{commit}} \
	CMT_VERSION={{cmt_version}} \
	BUILD_DIR={{build_dir}} \
	{{justfile_directory()}}/scripts/build.sh

build-linux-amd64:
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=false just build

build-linux-arm64:
	GOOS=linux GOARCH=arm64 LEDGER_ENABLED=false just build

install: build
	@echo "🚀 Installing binary to GOPATH/bin..."
	@cp {{build_dir}}/shinzohubd $(go env GOPATH)/bin/shinzohubd

verify-deps:
	@echo "🛠  Ensuring dependencies have not been modified ..."
	@go mod verify
	@go mod tidy

clean:
	@echo "🧹  Cleaning up..."
	@rm -rf {{build_dir}}

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

	coverage_total="$([ -f cover.out ] && go tool cover -func=cover.out | grep total | grep -Eo '[0-9]+\.[0-9]+' || echo 0)"
	echo "Threshold:                {{coverage_threshold}}%"
	echo "Current test coverage is: $coverage_total%"
	if [ "$(echo "$coverage_total < {{coverage_threshold}}" | bc -l)" -eq 1 ]; then \
		echo "Test coverage is lower than threshold ☠️"; \
		exit 1; \
	fi

###############################################################################
###                                 Linting                                 ###
###############################################################################

lint:
	@echo "🧹  Linting..."
	@go tool golangci-lint run --config .golangci.yml

lint-fix:
	@echo "🧹  Linting and fixing..."
	@go tool golangci-lint run --config .golangci.yml --fix

###############################################################################
###                                Protobuf                                 ###
###############################################################################

proto-all: proto-format proto-lint proto-gen

proto-deps:
	# Note: These should not be required to manually install locally since tools
	# can be executed via Go's toolchain directly. However, Buf, specifically in
	# remote environments (e.g. CI), expects plugins to be installed in $PATH.
	@echo "⚙️  Installing Protobuf dependencies..."
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
	@go tool buf lint ./proto --error-format=json

proto-format:
	@echo "⚙️  Formatting Protobuf files..."
	@go tool buf format ./proto -w

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
###                                 Testnet                                 ###
###############################################################################

# setup-testnet: sh-testnet is-localic-installed install local-image set-testnet-configs setup-testnet-keys

# # Run this before testnet keys are added
# # This chain id is used in the testnet.json as well
# set-testnet-configs:
# 	shinzohubd config set client chain-id 9001
# 	shinzohubd config set client keyring-backend test
# 	shinzohubd config set client output text

# # import keys from testnet.json into test keyring
# setup-testnet-keys:
# 	-`echo "decorate bright ozone fork gallery riot bus exhaust worth way bone indoor calm squirrel merry zero scheme cotton until shop any excess stage laundry" | shinzohubd keys add acc0 --recover`
# 	-`echo "wealth flavor believe regret funny network recall kiss grape useless pepper cram hint member few certain unveil rather brick bargain curious require crowd raise" | shinzohubd keys add acc1 --recover`

# testnet: setup-testnet
# 	spawn local-ic start testnet

sh-testnet: verify-deps
	CHAIN_ID="9001" BLOCK_TIME="1000ms" CLEAN=true sh scripts/test_node.sh

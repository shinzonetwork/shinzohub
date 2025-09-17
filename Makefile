SHELL := bash -eu -o pipefail -c

BUILD_DIR ?= $(CURDIR)/build
VERSION := $(shell git describe --tags --always --match "v*" | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
CMT_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')
LEDGER_ENABLED ?= true

###############################################################################
###                                  Build                                  ###
###############################################################################

# Always produce ./build/shinzohubd
build: verify-deps
	@echo "⚙️  Building binary to $(BUILD_DIR)..."
	@mkdir -p "$(BUILD_DIR)"
	@LEDGER_ENABLED=$(LEDGER_ENABLED) \
	VERSION=$(VERSION) \
	COMMIT=$(COMMIT) \
	CMT_VERSION=$(CMT_VERSION) \
	BUILD_DIR="$(BUILD_DIR)" \
	BINARY_NAME=shinzohubd \
	MAIN_PKG=./cmd/shinzohubd \
	$(CURDIR)/scripts/build.sh
	@if [ ! -f "$(BUILD_DIR)/shinzohubd" ]; then \
	  echo "Binary not found at $(BUILD_DIR)/shinzohubd. Searching..."; \
	  found="$$(find . -type f -name shinzohubd -perm -111 | head -n1)"; \
	  if [ -n "$$found" ]; then \
	    echo "Found at $$found. Copying to $(BUILD_DIR)/shinzohubd"; \
	    cp "$$found" "$(BUILD_DIR)/shinzohubd"; \
	  else \
	    echo "Build script did not produce shinzohubd. Check scripts/build.sh"; \
	    exit 1; \
	  fi; \
	fi
	@echo "✅ OK → $(BUILD_DIR)/shinzohubd"

build-linux-amd64:
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=false $(MAKE) build

build-linux-arm64:
	GOOS=linux GOARCH=arm64 LEDGER_ENABLED=false $(MAKE) build

# Install to ~/.local/bin (no GOPATH/asdf headaches)
install: build
	@echo "🚀 Installing binary..."
	@dest="$$HOME/.local/bin"; \
	mkdir -p "$$dest"; \
	cp "$(BUILD_DIR)/shinzohubd" "$$dest/shinzohubd"; \
	echo "✅ Installed to $$dest/shinzohubd"; \
	if command -v asdf >/dev/null 2>&1; then asdf reshim golang || true; fi

# Optional: install into GOPATH/bin
install-gopath: build
	@echo "🚀 Installing to GOPATH/bin..."
	@gbin="$$(go env GOPATH)/bin"; \
	mkdir -p "$$gbin"; \
	cp "$(BUILD_DIR)/shinzohubd" "$$gbin/shinzohubd"; \
	echo "✅ Installed to $$gbin/shinzohubd"; \
	if command -v asdf >/dev/null 2>&1; then asdf reshim golang || true; fi

verify-deps:
	@echo "🛠  Ensuring dependencies have not been modified ..."
	@go mod verify
	@go mod tidy

clean:
	@echo "🧹  Cleaning up..."
	@rm -rf "$(BUILD_DIR)"

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
###                                   Dev                                    ###
###############################################################################

sh-testnet: verify-deps
	CHAIN_ID="9001" BLOCK_TIME="1000ms" CLEAN=true sh scripts/test_node.sh

doctor:
	@echo "artifact   = $(BUILD_DIR)/shinzohubd"
	@echo "exists?    = $$(test -f $(BUILD_DIR)/shinzohubd && echo yes || echo no)"
	@echo "which bin  = $$(command -v shinzohubd || echo 'not on PATH')"

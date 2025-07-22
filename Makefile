BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')
APPNAME := shinzohub

# do not override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  # if VERSION is empty, then populate it with branch name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

# Update the ldflags with the app, client & server names
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=$(APPNAME) \
	-X github.com/cosmos/cosmos-sdk/version.AppName=$(APPNAME)d \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'

##############
###  Test  ###
##############

test-unit:
	@echo Running unit tests...
	@go test -mod=readonly -v -timeout 30m ./...

test-race:
	@echo Running unit tests with race condition reporting...
	@go test -mod=readonly -v -race -timeout 30m ./...

test-cover:
	@echo Running unit tests and creating coverage report...
	@go test -mod=readonly -v -timeout 30m -coverprofile=$(COVER_FILE) -covermode=atomic ./...
	@go tool cover -html=$(COVER_FILE) -o $(COVER_HTML_FILE)
	@rm $(COVER_FILE)

bench:
	@echo Running unit tests with benchmarking...
	@go test -mod=readonly -v -timeout 30m -bench=. ./...

test: govet govulncheck test-unit

.PHONY: test test-unit test-race test-cover bench

#################
###  Install  ###
#################

all: install

install:
	@echo "--> ensure dependencies have not been modified"
	@go mod verify
	@echo "--> installing $(APPNAME)d"
	@go install $(BUILD_FLAGS) -mod=readonly ./cmd/$(APPNAME)d

.PHONY: all install

##################
###  Protobuf  ###
##################

# Use this target if you do not want to use Ignite for generating proto files

proto-deps:
	@echo "Installing proto deps"
	@echo "Proto deps present, run 'go tool' to see them"

proto-gen:
	@echo "Generating protobuf files..."
	@ignite generate proto-go --yes

.PHONY: proto-gen

#################
###  Linting  ###
#################

lint:
	@echo "--> Running linter"
	@go tool github.com/golangci/golangci-lint/cmd/golangci-lint run ./... --timeout 15m

lint-fix:
	@echo "--> Running linter and fixing issues"
	@go tool github.com/golangci/golangci-lint/cmd/golangci-lint run ./... --fix --timeout 15m

.PHONY: lint lint-fix

###################
### Development ###
###################

govet:
	@echo Running go vet...
	@go vet ./...

govulncheck:
	@echo Running govulncheck...
	@go tool golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...

DEFRA_PATH ?=

bootstrap:
	@if [ -z "$(SOURCEHUB_PATH)" ]; then \
		echo "ERROR: You must pass SOURCEHUB_PATH or set it as an environment variable. Usage:"; \
		echo " make bootstrap SOURCEHUB_PATH=../path/to/sourcehub"; \
		exit 1; \
	fi
	@scripts/bootstrap.sh "$(SOURCEHUB_PATH)"

stop:
	@echo "===> Stopping sourcehubd if running..."
	@SHINZO_ROOTDIR="$(shell pwd)/.shinzohub"; \
	SOURCEHUB_PIDS=$$(ps aux | grep '[s]ourcehubd' | awk '{print $$2}'); \
	if [ -n "$$SOURCEHUB_PIDS" ]; then \
	  echo "Killing sourcehubd PIDs: $$SOURCEHUB_PIDS"; \
	  echo "$$SOURCEHUB_PIDS" | xargs -r kill -9 2>/dev/null; \
	  echo "Stopped all sourcehubd processes"; \
	else \
	  echo "No sourcehubd processes found"; \
	fi; \
	rm -f .shinzohub/sourcehubd.pid;
	@echo "===> Stopping shinzohubd if running..."
	SHINZOHUBD_PIDS=$$(ps aux | grep '[s]hinzohubd' | awk '{print $$2}'); \
	if [ -n "$$SHINZOHUBD_PIDS" ]; then \
	  echo "Killing shinzohubd PIDs: $$SHINZOHUBD_PIDS"; \
	  echo "$$SHINZOHUBD_PIDS" | xargs -r kill -9 2>/dev/null; \
	  echo "Stopped all shinzohubd processes"; \
	else \
	  echo "No shinzohubd processes found"; \
	fi; \
	rm -f .shinzohub/shinzohubd.pid;
	@echo "===> Stopping registrar if running..."
	REGISTRAR_PIDS=$$(ps aux | grep '[r]egistrar' | awk '{print $$2}'); \
	if [ -n "$$REGISTRAR_PIDS" ]; then \
	  echo "Killing registrar PIDs: $$REGISTRAR_PIDS"; \
	  echo "$$REGISTRAR_PIDS" | xargs -r kill -9 2>/dev/null; \
	  echo "Stopped all registrar processes"; \
	else \
	  echo "No registrar processes found"; \
	fi; \
	rm -f .shinzohub/registrar.pid;
	@rm -f .shinzohub/ready

.PHONY: govet govulncheck bootstrap stop
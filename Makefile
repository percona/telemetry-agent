.DEFAULT_GOAL := help
BIN_DIR := $(CURDIR)/bin

.PHONY: help clean init build format check test test-cover test-crosscover run prepare-pr

# --- Go-related variables ----------------------------------------------------------------
export GOPRIVATE := github.com/percona
export GONOSUMDB := github.com/percona
export GONOPROXY := github.com/percona

# --- Git variables ------------------------------------------------------------------------
COMPONENT_VERSION := $(shell git describe --abbrev=0 --always --tags)
BUILD_TIME := $(shell date +%FT%T%z)
TELEMETRY_AGENT_RELEASE_FULLCOMMIT := $(shell git rev-parse HEAD)

# --- Build variables ----------------------------------------------------------------------
TELEMETRY_AGENT_BINARY_NAME := $(BIN_DIR)/telemetry-agent
GO_BUILD_LDFLAGS := -X github.com/percona/telemetry-agent/config.Version=${COMPONENT_VERSION} \
	-X github.com/percona/telemetry-agent/config.BuildDate=${BUILD_TIME} \
	-X github.com/percona/telemetry-agent/config.Commit=${TELEMETRY_AGENT_RELEASE_FULLCOMMIT} \
	-extldflags -static
GOARCH?=amd64

# --- Tools variables ---------------------------------------------------------------------
GOLANGCI_LINT_VERSION := v2.12.2 # Version should match specified in CI
# ------------------------------------------------------------------------------------------

help: ## Display this help message
	@echo "Please use \`make <target>\`, where <target> is one of the following:"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9-]+:.*## / {printf "  %-23s%s\n", $$1, $$2}' $(MAKEFILE_LIST)

clean:
	@echo "🧹 Cleaning up binaries..."
	rm -f $(BIN_DIR)/*
	@echo "✅ Cleanup completed."

init: clean                  ## Install development tools
	@echo "Installing development tools..."
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(CURDIR)/bin $(GOLANGCI_LINT_VERSION)
	@echo "✅ Development tools installation completed."

build:                ## Compile using plain go build
	@echo "🚀 Building binaries."
	CGO_ENABLED=0 \
	GOARCH=$(GOARCH) \
	go build -a -ldflags="$(GO_BUILD_LDFLAGS)" -o $(TELEMETRY_AGENT_BINARY_NAME) $(CURDIR)/cmd/telemetry-agent
	@echo "✅ Building binaries completed."

format:                 ## Format source code
	$(BIN_DIR)/golangci-lint fmt -c=.golangci.yml
	go mod tidy

GOLANG_CI_LINT_RUN_OPTS ?=
check:                  ## Run checks/linters for the changes
	LOG_LEVEL=error $(BIN_DIR)/golangci-lint run -c=.golangci.yml --new-from-rev=$(shell git merge-base main HEAD) --new $(GOLANG_CI_LINT_RUN_OPTS)

test:                   ## Run tests
	go clean -testcache
	go test -race -timeout=10m $(CURDIR)/...

test-cover:             ## Run tests and collect per-package coverage information
	go clean -testcache
	go test -race -timeout=10m -count=1 -coverprofile=cover.out -covermode=atomic $(CURDIR)/...

test-crosscover:        ## Run tests and collect cross-package coverage information
	go clean -testcache
	go test -race -timeout=10m -count=1 -coverprofile=crosscover.out -covermode=atomic -p=1 -coverpkg=./... $(CURDIR)/...

run:                    ## Run telemetry-agent with race detector
	go run -race $(CURDIR)/cmd/telemetry-agent/main.go \
		--log.verbose --log.dev-mode

prepare-pr: format 	## Prepare code for PR commit
	$(MAKE) GOLANG_CI_LINT_RUN_OPTS=--fix check

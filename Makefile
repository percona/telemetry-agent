.PHONY: all build clean default help init test format check test-cover test-crosscover run
default: help

GOPRIVATE=github.com/Percona-Lab/telemetry-agent,github.com/percona-platform/saas,github.com/percona
COMPONENT_VERSION ?= $(shell git describe --abbrev=0 --always)
BUILD ?= $(shell date +%FT%T%z)
TELEMETRY_AGENT_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
GO_BUILD_LDFLAGS = -X github.com/percona-lab/telemetry-agent/config.Version=${COMPONENT_VERSION} \
	-X github.com/percona-lab/telemetry-agent/config.BuildDate=${BUILD} \
	-X github.com/percona-lab/telemetry-agent/config.Commit=${TELEMETRY_AGENT_RELEASE_FULLCOMMIT} \
	-extldflags -static

help:                   ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

init:                   ## Install development tools
	cd tools && go generate -x -tags=tools

build:                ## Compile using plain go build
	CGO_ENABLED=0 \
	GOARCH=amd64 \
	go build -a -ldflags="${GO_BUILD_LDFLAGS}" -o ./bin/telemetry-agent ./cmd/telemetry-agent

gen:                    ## Generate code
	go generate ./...
	make format

format:                 ## Format source code
	go mod tidy
	bin/gofumpt -l -w .
	bin/goimports -local github.com/percona-lab/telemetry-agent -l -w .

check:                  ## Run checks/linters for the whole project
	bin/go-consistent -exclude=tools -pedantic ./...
	LOG_LEVEL=error bin/golangci-lint run

test:                   ## Run tests
	go test -race -timeout=10m ./...

test-cover:             ## Run tests and collect per-package coverage information
	go test -race -timeout=10m -count=1 -coverprofile=cover.out -covermode=atomic ./...

test-crosscover:        ## Run tests and collect cross-package coverage information
	go test -race -timeout=10m -count=1 -coverprofile=crosscover.out -covermode=atomic -p=1 -coverpkg=./... ./...

run:                    ## Run authed with race detector
	go run -race cmd/telemetry-agent/main.go \
		--verbose --dev-mode

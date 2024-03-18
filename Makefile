GOPRIVATE=github.com/Percona-Lab/telemetry-agent,github.com/percona-platform/saas,github.com/percona

default: help

help:                   ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

init:                   ## Install development tools
	cd tools && go generate -x -tags=tools

install:                ## Install binaries
	CGO_ENABLED=0 \
	GOARCH=amd64 \
	go build -a -ldflags '-extldflags "-static"' -o ./bin/telemetry-agent ./cmd/telemetry-agent

gen:                    ## Generate code
	go generate ./...
	make format

format:                 ## Format source code
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

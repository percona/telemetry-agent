//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/quasilyte/go-consistent"
	_ "github.com/reviewdog/reviewdog/cmd/reviewdog"
	_ "github.com/vektra/mockery/v2"
	_ "golang.org/x/tools/cmd/goimports"
	_ "gopkg.in/reform.v1/reform"
	_ "mvdan.cc/gofumpt"
)

//go:generate go build -o ../bin/reform gopkg.in/reform.v1/reform
//go:generate go build -o ../bin/go-consistent github.com/quasilyte/go-consistent
//go:generate go build -o ../bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go build -o ../bin/reviewdog github.com/reviewdog/reviewdog/cmd/reviewdog
//go:generate go build -o ../bin/gofumpt mvdan.cc/gofumpt
//go:generate go build -o ../bin/goimports golang.org/x/tools/cmd/goimports
//go:generate go build -o ../bin/mockery github.com/vektra/mockery/v2

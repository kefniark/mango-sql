//go:build tools
// +build tools

package tools

import (
	_ "github.com/boumenot/gocover-cobertura"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)

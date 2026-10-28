// staticlint is a CLI tool for static analysis of Go code.
//
// Usage:
//
//	staticlint [flags] [packages]
//
// Examples:
//
//	mychecker ./...                   # check the whole module
//	mychecker -exitinmain=false ./... # disable a specific analyzer
//
// Exit codes: 0 (no issues), 1 (issues found), 3 (fatal error).
//
// See internal/staticlint/passes for the list of active analyzers.
package main

import (
	"github.com/mersikovs/korob.git/internal/staticlint/passes"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(passes.Build()...)
}

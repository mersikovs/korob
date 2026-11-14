// Package passes aggregates analyzers for the staticlint CLI tool.
//
// Analyzer groups:
//  1. Standard: printf, shadow, structtag, loopclosure, httpresponse, nilness.
//  2. Staticcheck SA*: all rules prefixed with "SA" (see staticcheck.dev/docs/checks/#SA).
//  3. Staticcheck Non-SA: S1000 (replace for-loop with for-range, sourced from honnef.co/go/tools/simple).
//  4. Third-party: bodyclose, noctx.
//  5. Custom: exitinmain.
//
// Each analyzer is automatically exposed as a CLI flag (e.g., -S1000=false).
package passes

import (
	"strings"

	"github.com/mersikovs/korob.git/internal/staticlint/exitinmain"
	"github.com/sonatard/noctx"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
)

// Build returns the complete set of configured analyzers for staticlint.
//
// The returned slice is ready to be passed directly to multichecker.Main.
func Build() []*analysis.Analyzer {
	analyzers := make([]*analysis.Analyzer, 0, 80)
	analyzers = append(analyzers,
		nilness.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		loopclosure.Analyzer,
		httpresponse.Analyzer,
	)

	for _, sa := range staticcheck.Analyzers {
		if strings.HasPrefix(sa.Analyzer.Name, "SA") {
			analyzers = append(analyzers, sa.Analyzer)
		}
	}

	for _, s := range simple.Analyzers {
		if s.Analyzer.Name == "S1000" {
			analyzers = append(analyzers, s.Analyzer)
			break
		}
	}

	analyzers = append(analyzers, bodyclose.Analyzer, noctx.Analyzer)

	analyzers = append(analyzers, exitinmain.Analyzer)

	return analyzers
}

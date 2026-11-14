package exitinmain_test

import (
	"testing"

	"github.com/mersikovs/korob.git/internal/staticlint/exitinmain"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {

	analysistest.Run(
		t,
		analysistest.TestData(),
		exitinmain.Analyzer,
		"exitinmain",
		"fakeos",
		"inmainfile",
		"intestfile",
		"noerror",
		"packagerename",
	)
}

package passes_test

import (
	"testing"

	"github.com/mersikovs/korob.git/internal/staticlint/passes"
)

func TestBuild(t *testing.T) {
	analyzers := passes.Build()

	if len(analyzers) == 0 {
		t.Fatal("Build() returned empty slice")
	}

	names := make(map[string]bool, len(analyzers))
	for _, a := range analyzers {
		if names[a.Name] {
			t.Errorf("Build() Duplicate: %s", a.Name)
		}
		names[a.Name] = true
	}

	tests := []struct {
		name     string
		analyzer string
		want     bool
	}{
		{"Standard: printf", "printf", true},
		{"Standard: shadow", "shadow", true},
		{"Standard: structtag", "structtag", true},
		{"Standard: loopclosure", "loopclosure", true},
		{"Standard: httpresponse", "httpresponse", true},
		{"Standard: nilness", "nilness", true},

		{"Staticcheck SA: SA1000", "SA1000", true},
		{"Staticcheck SA: SA4006", "SA4006", true},

		{"Staticcheck Non-SA: S1000", "S1000", true},

		{"Staticcheck Excluded: ST1000", "ST1000", false},
		{"Staticcheck Excluded: U1000", "U1000", false},

		{"Third-party: bodyclose", "bodyclose", true},
		{"Third-party: noctx", "noctx", true},

		{"Custom: exitinmain", "exitinmain", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := names[tt.analyzer]
			if got != tt.want {
				if tt.want {
					t.Errorf("Analyzer %q is missing, but expected to be present", tt.analyzer)
				} else {
					t.Errorf("Analyzer %q is present, but expected to be excluded by filter", tt.analyzer)
				}
			}
		})
	}

	if len(analyzers) < 60 {
		t.Errorf("Expected at least 60 analyzers, got %d. Staticcheck broken.", len(analyzers))
	}
}

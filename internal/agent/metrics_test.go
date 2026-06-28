package agent_test

import (
	"testing"

	"github.com/mersikovs/korob.git/internal/agent"
	"github.com/stretchr/testify/assert"
)

func TestGetCountDiff(t *testing.T) {
	tests := []struct {
		name       string
		oldMetrics map[string]string
		newMetrics map[string]string
		want       int
	}{
		{
			name:       "no diff empty",
			oldMetrics: map[string]string{},
			newMetrics: map[string]string{},
			want:       0,
		},
		{
			name:       "no diff one value",
			oldMetrics: map[string]string{"metric": "value"},
			newMetrics: map[string]string{"metric": "value"},
			want:       0,
		},
		{
			name:       "one difference",
			oldMetrics: map[string]string{"metric": "value1"},
			newMetrics: map[string]string{"metric": "value2"},
			want:       1,
		},
		{
			name:       "one difference new key",
			oldMetrics: map[string]string{},
			newMetrics: map[string]string{"metric": "value1"},
			want:       1,
		},
		{
			name:       "one difference new key",
			oldMetrics: map[string]string{"metric": "value1"},
			newMetrics: map[string]string{"metric": "value2", "metric_new": "value3"},
			want:       2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.GetCountDiff(tt.oldMetrics, tt.newMetrics)
			assert.Equal(t, tt.want, got)
		})
	}
}

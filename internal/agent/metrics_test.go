package agent_test

import (
	"testing"

	"github.com/mersikovs/korob.git/internal/agent"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/stretchr/testify/assert"
)

func Float64Ptr(v float64) *float64 {
	return &v
}

func TestGetCountDiff(t *testing.T) {
	tests := []struct {
		name       string
		oldMetrics map[string]models.Metrics
		newMetrics map[string]models.Metrics
		want       int
	}{
		{
			name:       "no diff empty",
			oldMetrics: map[string]models.Metrics{},
			newMetrics: map[string]models.Metrics{},
			want:       0,
		},
		{
			name: "no diff one value",
			oldMetrics: map[string]models.Metrics{
				"metric": {
					ID:    "metric",
					Value: nil,
				},
			},
			newMetrics: map[string]models.Metrics{
				"metric": {
					ID:    "metric",
					Value: nil,
				},
			},
			want: 0,
		},
		{
			name: "one difference",
			oldMetrics: map[string]models.Metrics{
				"metric": {
					ID:    "metric",
					Value: Float64Ptr(1),
				},
			},
			newMetrics: map[string]models.Metrics{
				"metric": {
					ID:    "metric",
					Value: Float64Ptr(1),
				},
			},
			want: 1,
		},
		{
			name: "one difference new key",
			oldMetrics: map[string]models.Metrics{
				"metric": {
					ID:    "metric",
					Value: nil,
				},
			},
			newMetrics: map[string]models.Metrics{
				"metric": {
					ID:    "metric",
					Value: Float64Ptr(1),
				},
			},
			want: 1,
		},
		{
			name: "one difference new key",
			oldMetrics: map[string]models.Metrics{
				"metric": {
					ID:    "metric",
					Value: Float64Ptr(1),
				},
			},
			newMetrics: map[string]models.Metrics{
				"metric": {
					ID:    "metric",
					Value: Float64Ptr(2),
				},
				"metric_new": {
					ID:    "metric_new",
					Value: Float64Ptr(2),
				},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.GetCountDiff(tt.oldMetrics, tt.newMetrics)
			assert.Equal(t, tt.want, got)
		})
	}
}

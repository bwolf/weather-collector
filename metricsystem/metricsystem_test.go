package metricsystem

import (
	"testing"
)

func TestMetricSystem(t *testing.T) {
	const num float64 = 6.25
	cases := []struct {
		prefix string
		want   float64
	}{
		{"k", 62500},
		{"", 6.25},
		{"c", 0.625},
	}
	for _, c := range cases {
		got := ScaleByMetricSystemPrefix(num, c.prefix)
		if got != c.want {
			t.Errorf("Scaling (%f, %q) == %f, want %f", num, c.prefix, got, c.want)
		}
	}
}

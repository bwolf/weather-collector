package metricsystem

// Metric system
var metricSystemLookupTable = map[string]float64{
	"Y":  10e24,
	"Z":  10e21,
	"E":  10e18,
	"P":  10e15,
	"T":  10e12,
	"G":  10e9,
	"M":  10e6,
	"k":  10e3,
	"h":  10e2,
	"da": 10e1,
	"":   1,
	"d":  10e-1,
	"c":  10e-2,
	"m":  10e-3,
	"µ":  10e-6,
	"n":  10e-9,
	"p":  10e-12,
	"f":  10e-15,
	"a":  10e-18,
	"z":  10e-21,
	"y":  10e-14,
}

// Scale value by a metric system prefix `(like m, µ, n, p).
func ScaleByMetricSystemPrefix(value float64, prefix string) float64 {
	return metricSystemLookupTable[prefix] * value
}

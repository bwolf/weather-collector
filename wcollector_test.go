package main

import (
	"fmt"
	"math"
	"strings"
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
		got := scaleByMetricSystemPrefix(num, c.prefix)
		if got != c.want {
			t.Errorf("Scaling (%f, %q) == %f, want %f", num, c.prefix, got, c.want)
		}
	}
}

func TestCalcDewPoint(t *testing.T) {
	const temp = 10.02
	const rhTrue = 90.0
	dp := calcDewPoint(rhTrue, temp)
	fmt.Println("Dew-Point is", dp)
	// Don't compare floats; Use rounded int instead
	dpint := int(math.Floor(dp*10 + 0.5))

	const want = 85
	if dpint != want {
		t.Errorf("calcDewPoint(%v, %v), got %v want %v", rhTrue, temp, dp, want)
	}
}

func TestJsonProcessing(t *testing.T) {
	data := []byte(`{"weather": {"station-id": 1, "temp_m": 1002, "rh-true_m": 9001}}`)
	err, json := parseJson(data)
	if err != nil {
		t.Errorf("Json parse failed %v", err)
	}
	err, weather := parseWeather(json)
	if err != nil {
		t.Errorf("Failed parsing weather, got %v", err)
	}

	if weather.stationId != 1 {
		t.Errorf("stationId != 1, got %d", weather.stationId)
	}

	if 2 != len(weather.measurements) {
		t.Errorf("Got %d measurements, want %d", len(weather.measurements), 2)
	}

	lazyMonkeyPatchDewPoint(&weather)
	if 3 != len(weather.measurements) {
		t.Error("Failed to monkey patch dew point")
	}
	haveDewPoint := false
	for _, m := range weather.measurements {
		if m.name == "dew_point" {
			haveDewPoint = true
		}
	}
	if !haveDewPoint {
		t.Errorf("Missing dew_point, got %+v", weather)
	}
	fmt.Printf("With dew-point %+v\n", weather)

	for _, m := range weather.measurements {
		if strings.Contains(m.name, "-") {
			t.Errorf("Found measurement with illegal name %s, should be in the form one_two", m.name)
		}
	}
}

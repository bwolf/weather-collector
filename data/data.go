package data

import (
	"bitbucket.org/mgeiger/wcollector/metricsystem"

	"encoding/json"
	"fmt"
	"math"
	"strings"
)

type Measurement struct {
	Name  string
	Value float64
}

type Weather struct {
	stationId    int
	measurements []Measurement
}

func (weather *Weather) StationId() int {
	return weather.stationId
}

func (weather *Weather) Measurements() []Measurement {
	return weather.measurements
}

func calcDewPoint(humid, temp float64) float64 {
	m := 17.62
	tn := 243.12

	if temp <= 0 {
		m = 22.46
		tn = 272.62
	}
	k := (math.Log10(humid)-2)/0.4343 + (m*temp)/(tn+temp)
	return tn * k / (m - k)
}

func normalizeMeasurement(m *Measurement) *Measurement {
	index := strings.LastIndex(m.Name, "_")
	if index == -1 {
		fmt.Println("Nothing to normalize for key", m.Name)
		newName := strings.Replace(m.Name, "-", "_", -1)
		return &Measurement{Name: newName, Value: m.Value}
	}
	metricPrefix := m.Name[index+1:]
	newName := strings.Replace(m.Name[0:index], "-", "_", -1)
	newValue := metricsystem.ScaleByMetricSystemPrefix(m.Value, metricPrefix)
	return &Measurement{Name: newName, Value: newValue}
}

func transformMeasurements(meas []Measurement) []Measurement {
	res := make([]Measurement, len(meas))
	for i, m := range meas {
		mm := normalizeMeasurement(&m)
		res[i] = *mm
	}
	return res
}

// Patch dew point into data set based on rh_true and temp
func LazyMonkeyPatchDewPoint(weather *Weather) {
	var gotRhTrue, gotTemp bool
	var rhTrue, temp Measurement
	for _, m := range weather.measurements {
		if m.Name == "rh_true" {
			gotRhTrue = true
			rhTrue = m
		} else if m.Name == "temp" {
			gotTemp = true
			temp = m
		}
		if gotRhTrue && gotTemp {
			break
		}
	}

	if gotRhTrue && gotTemp {
		dp := calcDewPoint(rhTrue.Value, temp.Value)
		fmt.Printf("Dew point from h %f, t %f: %f will be patched in\n",
			rhTrue.Value, temp.Value, dp)

		weather.measurements = append(weather.measurements,
			Measurement{Name: "dew_point", Value: dp})
	}
}

func ParseWeather(js interface{}) (error, *Weather) {
	m := js.(map[string]interface{})

	rawWeather, ok := m["weather"]
	if !ok {
		return fmt.Errorf("No weather in Json %v", js), nil
	}

	weather := Weather{}
	weatherMap := rawWeather.(map[string]interface{})
	for k, v := range weatherMap {
		switch vv := v.(type) {
		case string:
			return fmt.Errorf("%s is unsupported string %s", k, vv), nil
		case int:
			return fmt.Errorf("%s is unsupported int %d", k, vv), nil
		case float64:
			fmt.Println(k, "is float64", vv)
			if k == "station-id" { // Decoded as float64, also look like int
				weather.stationId = int(vv)
			} else {
				meas := Measurement{Name: k, Value: vv}
				weather.measurements = append(weather.measurements, meas)
			}
		default:
			return fmt.Errorf("%s is of unsupported type %v", k, vv), nil
		}
	}

	// Ensure there is a station-id in weather
	if weather.stationId == 0 {
		return fmt.Errorf("Missing 'station-id' in weather dataset %v", rawWeather), nil
	}

	weather.measurements = transformMeasurements(weather.measurements)

	return nil, &weather
}

func ParseJson(rawInput []byte) (error, interface{}) {
	var js interface{}
	err := json.Unmarshal(rawInput, &js)
	if err != nil {
		return fmt.Errorf("Invalid JSON: %v\n", err), nil
	}
	return nil, js
}

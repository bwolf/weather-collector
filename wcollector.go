package main

// TODO use proper logging instaed of fmt.Print

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"math"
	"os"
	"path"
	"strings"
)

// ---------------------------------------------------------------------
// Metric Sytem

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

func scaleByMetricSystemPrefix(value float64, prefix string) float64 {
	return metricSystemLookupTable[prefix] * value
}

// ---------------------------------------------------------------------
// UART, serial logic

func openUart(device string, baud int) *serial.Port {
	c := &serial.Config{Name: device, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func readlineCR(port *serial.Port) []byte {
	buf := make([]byte, 0)
	idx := 0
	for {
		chbuf := make([]byte, 1)
		n, err := port.Read(chbuf) // TODO n is ignored
		if err != nil || n != 1 {
			log.Fatal(err) // TODO fixme, just return an error
		}

		ch := chbuf[0]
		buf = append(buf, ch)
		idx++

		if len(buf) > 2 && buf[len(buf)-2] == '\r' && buf[len(buf)-1] == '\n' {
			// return string(buf[:idx])
			return buf[:idx]
		}
	}
}

// ---------------------------------------------------------------------
// JSON weather processing

type Measurement struct {
	name  string
	value float64
}

type Weather struct {
	stationId    int
	measurements []Measurement
}

func measurementConsumer(stationId int, meas *Measurement) error {
	return nil
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
	index := strings.LastIndex(m.name, "_")
	if index == -1 {
		fmt.Println("Nothing to normalize for key", m.name)
		return m
	}
	metricPrefix := m.name[index+1:]
	newName := strings.Replace(m.name[0:index], "-", "_", -1)
	newValue := scaleByMetricSystemPrefix(m.value, metricPrefix)
	return &Measurement{name: newName, value: newValue}
}

func transformMeasurements(meas []Measurement) []Measurement {
	res := make([]Measurement, len(meas))
	for i, m := range meas {
		mm := normalizeMeasurement(&m)
		res[i] = *mm
	}
	return res
}

// Patch dew point into dataset based on rh_true and temp
func lazyMonkeyPatchDewPoint(weather *Weather) {
	var gotRhTrue, gotTemp bool
	var rhTrue, temp Measurement
	for _, m := range weather.measurements {
		if m.name == "rh_true" {
			gotRhTrue = true
			rhTrue = m
		} else if m.name == "temp" {
			gotTemp = true
			temp = m
		}
		if gotRhTrue && gotTemp {
			break
		}
	}

	if gotRhTrue && gotTemp {
		dp := calcDewPoint(rhTrue.value, temp.value)
		fmt.Printf("Dew point from h %f, t %f: %f will be patched in\n",
			rhTrue.value, temp.value, dp)

		weather.measurements = append(weather.measurements,
			Measurement{name: "dew_point", value: dp})
	} else {
		fmt.Println("Not patching in dew-point")
	}
}

func parseWeather(js interface{}) (error, *Weather) {
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
				meas := Measurement{name: k, value: vv}
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
	fmt.Printf("Transformed %+v\n", weather)

	fmt.Printf("Parsed weather %+v\n", weather)

	return nil, &weather
}

func parseJson(rawInput []byte) (error, interface{}) {
	fmt.Printf("Attempt to parse '%s' as json\n", rawInput)
	var js interface{}
	err := json.Unmarshal(rawInput, &js)
	if err != nil {
		fmt.Printf("Invalid JSON: %v\n", err)
		return err, nil
	}
	fmt.Printf("Valid: '%s'\n", js)
	return nil, js
}

// ---------------------------------------------------------------------
// InfluxDB client

type InfluxDbClient struct {
	baseURL string // TODO use URL type if possible
	port    int
	dbName  string
}

func NewInfluxDbClient(baseURL string, port int, dbName string) *InfluxDbClient {
	return &InfluxDbClient{baseURL, port, dbName}
}

// ---------------------------------------------------------------------
// Main entry point

func main() {
	// verbose := flag.Bool("verbose", false, "verbose processing")
	device := flag.String("device", "/dev/ttyAMA0", "Device or filename")
	baud := flag.Int("baud", 4800, "Baudrate")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	ss := openUart(*device, *baud)
	defer ss.Close()
	for {
		line := readlineCR(ss)
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("# ")) {
			err, json := parseJson(line)
			if err != nil {
				// TODO don't fatal/exit here
				log.Fatal(fmt.Sprintf("Cannot process line '%s', because of: %v", line, err))
			}
			err, weather := parseWeather(json)
			if err != nil {
				log.Fatalf("Cannot parse weather %v", err)
			}
			log.Print("Lazy patching in dew point")
			lazyMonkeyPatchDewPoint(weather)
			fmt.Printf("%+v\n", weather)
			// TODO consume weather
		}
	}
}

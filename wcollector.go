package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tarm/serial"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"strings"
)

const ()

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
		logger.Fatalf("Can't open serial device %s: %v", device, err)
	}
	return s
}

func readlineCR(port *serial.Port) ([]byte, error) {
	buf := make([]byte, 0)
	idx := 0
	for {
		chbuf := make([]byte, 1)   // TODO empty array also works?
		n, err := port.Read(chbuf) // TODO n is ignored
		if err != nil || n != 1 {
			return nil, err
		}

		ch := chbuf[0]
		buf = append(buf, ch)
		idx++

		if len(buf) > 2 && buf[len(buf)-2] == '\r' && buf[len(buf)-1] == '\n' {
			return buf[:idx], nil
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
		newName := strings.Replace(m.name, "-", "_", -1)
		return &Measurement{name: newName, value: m.value}
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
	logger.Printf("Transformed %+v\n", weather)

	return nil, &weather
}

func parseJson(rawInput []byte) (error, interface{}) {
	logger.Printf("Attempt to parse '%s' as json\n", rawInput)
	var js interface{}
	err := json.Unmarshal(rawInput, &js)
	if err != nil {
		logger.Printf("Invalid JSON: %v\n", err)
		return err, nil
	}
	logger.Printf("Valid: '%s'\n", js)
	return nil, js
}

// ---------------------------------------------------------------------
// Weather db (InfluxDB) client

type WeatherDbClient struct {
	influxUrl string
	data      bytes.Buffer
}

func NewWeatherDbClient(host string, port int, dbName string) *WeatherDbClient {
	url := fmt.Sprintf("http://%s:%d/write?db=%s", host, port, dbName)
	return &WeatherDbClient{influxUrl: url}
}

// see https://influxdb.com/docs/v0.9/guides/writing_data.html
func (weatherDb *WeatherDbClient) AddValue(stationId int, key string, value float64) {
	fmt.Fprintf(&weatherDb.data, "%s,station=%d value=%f\n", key, stationId, value)
}

func (weatherDb *WeatherDbClient) Post() error {
	resp, err := http.Post(weatherDb.influxUrl, "text/plain", &weatherDb.data)
	if err != nil {
		return fmt.Errorf("HTTP POST failed to InfluxDB: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP POST failed to InfluxDB: %s\n", resp.Status)
	}

	weatherDb.data.Reset()

	return nil
}

// ---------------------------------------------------------------------
// Main entry point

var logger *log.Logger

func storeWeather(db *WeatherDbClient, weather *Weather) error {
	for _, meas := range weather.measurements {
		db.AddValue(weather.stationId, meas.name, meas.value)
	}
	return db.Post()
}

func main() {
	// Flag setup
	verbose := flag.Bool("verbose", false, "Verbose processing (default false)")
	device := flag.String("device", "/dev/ttyAMA0", "Device or filename (default ttyAMA0)")
	baud := flag.Int("baud", 4800, "Baudrate of serial device (default 4800)")
	influxHost := flag.String("influxhost", "localhost", "Influxdb hostname")
	influxPort := flag.Int("influxport", 8086, "Influxdb port")
	influxDb := flag.String("influxdbname", "weather", "Influxdb DB name")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	// Logging setup
	logfilename := "wcollector.log"
	logfile, err := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Panicf("Failed to open logfile %s for writing: %v\n", logfilename, err)
	}
	defer logfile.Close()

	var logwriter io.Writer = logfile
	if *verbose {
		logwriter = io.MultiWriter(logfile, os.Stdout)
	}
	logger = log.New(logwriter, "[wcollector] ", log.LstdFlags)
	log.Println("Starting up")

	// TODO reopen logfile on SIGHUP or some other signal

	db := NewWeatherDbClient(*influxHost, *influxPort, *influxDb)

	// Main logic
	ss := openUart(*device, *baud)
	defer ss.Close()
	for {
		line, err := readlineCR(ss)
		if err == nil {
			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("# ")) {
				err, json := parseJson(line)
				if err != nil {
					logger.Printf("Cannot process line '%s', because of: %v\n", line, err)
				}
				err, weather := parseWeather(json)
				if err != nil {
					logger.Printf("Cannot parse weather %v\n", err)
				}
				logger.Println("Lazy patching in dew point")
				lazyMonkeyPatchDewPoint(weather)
				logger.Printf("Patched %+v\n", weather)

				if err := storeWeather(db, weather); err != nil {
					logger.Printf("Failed storing values: %v\n", err)
				}
			}
		}
	}
}

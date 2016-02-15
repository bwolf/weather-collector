package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"bitbucket.org/mgeiger/wcollector/data"
	"bitbucket.org/mgeiger/wcollector/db"
	"bitbucket.org/mgeiger/wcollector/input"
)

var logger *log.Logger

func storeWeather(d db.DB, weather *data.Weather) error {
	for _, meas := range weather.Measurements() {
		d.AddValue(weather.StationId(), meas.Name, meas.Value)
	}
	return d.Save()
}

func main() {
	// Flag setup
	verbose := flag.Bool("verbose", false, "Verbose processing (default false)")
	deviceName := flag.String("device", "/dev/ttyAMA0", "Device or filename (default ttyAMA0)")
	baudRate := flag.Int("baud", 4800, "Baudrate of serial device (default 4800)")
	influxHost := flag.String("influxhost", "localhost", "Influxdb hostname")
	influxPort := flag.Int("influxport", 8086, "Influxdb port")
	influxDBName := flag.String("influxdbname", "weather", "Influxdb DB name")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	// Logging setup
	logFilename := "wcollector.log"
	logfile, err := os.OpenFile(logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Panicf("Failed to open logfile %s for writing: %v\n", logFilename, err)
	}
	defer logfile.Close()

	var logWriter io.Writer = logfile
	if *verbose {
		logWriter = io.MultiWriter(logfile, os.Stdout)
	}
	logger = log.New(logWriter, "[wcollector] ", log.LstdFlags)
	log.Println("Starting up")

	// TODO reopen logfile on SIGHUP or some other signal

	// Main logic
	db := db.NewInfluxDBClient(*influxHost, *influxPort, *influxDBName)

	err, in := input.OpenUART(*deviceName, *baudRate)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	for {
		err, line := in.ReadLine()
		if err == nil {
			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("# ")) {
				err, json := data.ParseJson(line)
				if err != nil {
					logger.Printf("Cannot process line '%s', because of: %v\n", line, err)
				}
				err, weather := data.ParseWeather(json)
				if err != nil {
					logger.Printf("Cannot parse weather %v\n", err)
				}
				logger.Println("Lazy patching in dew point")
				data.LazyMonkeyPatchDewPoint(weather)
				logger.Printf("Patched %+v\n", weather)

				if err := storeWeather(db, weather); err != nil {
					logger.Printf("Failed storing values: %v\n", err)
				}
			}
		}
	}
}

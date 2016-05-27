package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"
)

const (
	SunSyncDaysPast = 5
)

func main() {
	// Flag setup
	verbose := flag.Bool("verbose", false, "Verbose processing (default false)")
	randomize := flag.Bool("randomize", false, "Random input (default false)")
	deviceName := flag.String("device", "/dev/ttyAMA0", "Device or filename (default ttyAMA0)")
	baudRate := flag.Int("baud", 4800, "Baudrate of serial device (default 4800)")
	influxHost := flag.String("influxhost", "localhost", "Influxdb hostname")
	influxPort := flag.Int("influxport", 8086, "Influxdb port")
	influxDBName := flag.String("influxdbname", "weather", "Influxdb DB name")
	latitude := flag.Float64("latitude", 48.137222, "Geographic latitude (default 48.137222, munich)")
	longitude := flag.Float64("longitude", 11.575556, "Geographic latitude (default 11.575556, munich)")
	location := flag.String("location", "munich", "Geographic location (default 'munich')")
	logFilename := flag.String("logfile", "/var/log/weather/collector.log", "Logfile fully qualified path")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	// Logging setup
	logfile, err := os.OpenFile(*logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Panicf("Failed to open logfile %s for writing: %v\n", logFilename, err)
	}
	defer logfile.Close()

	var logWriter io.Writer = logfile
	if *verbose {
		logWriter = io.MultiWriter(logfile, os.Stdout)
	}

	var logger *log.Logger = log.New(logWriter, "[wcollector] ", log.LstdFlags)
	log.Println("Starting up")

	// TSDB client
	tsdb := NewInfluxDBClient(*influxHost, *influxPort, *influxDBName)
	if *verbose {
		tsdb.SetDebug(true)
	}

	// Input source
	var in Input
	if *randomize {
		in = NewRandomInput()
	} else {
		err, in = OpenUART(*deviceName, *baudRate)
		if err != nil {
			logger.Fatal(err)
		}
	}
	defer in.Close()

	// Sun rise and set calculation
	go func() {
		for {
			logger.Println("Sun rise/set calculation")
			for i := 0; i < 2*SunSyncDaysPast; i++ { // For -5 to +5 days from now
				day := time.Now().AddDate(0, 0, (-SunSyncDaysPast + i))
				err := EnsureSunRiseAndSet(tsdb, day, *latitude, *longitude, *location)
				if err != nil {
					logger.Printf("Error: Failed ensuring sun information: %v\n", err)
				}
			}

			time.Sleep(30 * time.Minute)
		}
	}()

	// Loop over input, process data, store in TSDB
	Consume(in, tsdb, logger)
}

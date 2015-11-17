package main

import (
	"flag"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"os"
	"path"
)

// ---------------------------------------------------------------------
// Metric Sytem

var metricSystemLookupTable = map[string]float32{
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

func scaleByMetricSystemPrefix(value float32, prefix string) float32 {
	return metricSystemLookupTable[prefix] * value
}

// ---------------------------------------------------------------------
// foo

func openUart(device string, baud int) *serial.Port {
	c := &serial.Config{Name: device, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	return s
}

// ---------------------------------------------------------------------
// Main entry point

func main() {
	fmt.Printf("hello world!\n")

	xx := scaleByMetricSystemPrefix(1024.3, "c")
	fmt.Printf("%f", xx)

	ss := openUart("AMA0", 4800)

	verbose := flag.Bool("verbose", false, "verbose processing")
	device := flag.String("device", "/dev/ttyAMA0", "Device or filename")
	baud := flag.Int("baud", 4800, "Baudrate")

	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", path.Base(os.Args[0]))
	}
}

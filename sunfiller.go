package main

import (
	"bytes"
	"fmt"
	"github.com/bwolf/suncal"
	"net/http"
	"os"
	"time"
)

func main() {
	fmt.Println("Hallo")

	const lat = 47.89681
	const lon = 11.69945

	coords := suncal.GeoCoordinates{Latitude: lat, Longitude: lon}
	sun := suncal.SunCal(coords, time.Now())
	fmt.Printf("sun %+v\n", sun)

	url := "http://localhost:8086/write?db=weather"
	var buf bytes.Buffer
	for p := sun.Rise; !p.After(sun.Set); p = p.Add(1 * time.Minute) {
		fmt.Println(p, p.UnixNano())
		// The timestamp value is an integer representing nanoseconds since the epoch.
		fmt.Fprintf(&buf, "sun,station=1,location=\"Erlkam\" value=1 %v\n", p.UnixNano())

		resp, err := http.Post(url, "text/plain", &buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "HTTP POST failed to InfluxDB: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			fmt.Fprintf(os.Stderr, "HTTP POST failed to InfluxDB: %s\n", resp.Status)
		}
		buf.Reset()
	}
}

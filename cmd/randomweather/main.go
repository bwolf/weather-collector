package main

import (
	"fmt"
	"math/rand"
	"time"

	"bitbucket.org/mgeiger/wcollector/data"
	"bitbucket.org/mgeiger/wcollector/db"
	"bytes"
	"io"
)

type RandomInput struct {
	src *rand.Rand
	ts  time.Time
}

func NewRandomInput() *RandomInput {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	ts := time.Now().Add(-11 * time.Second)
	return &RandomInput{src: r, ts: ts}
}

func getRandomData(in *RandomInput) []byte {
	temp := in.src.Intn(1000)
	rhtrue := in.src.Intn(9000)
	press := in.src.Intn(13000)
	cups := in.src.Intn(150)

	data := fmt.Sprintf("{ \"weather\": { \"station-id\": 1, \"temp_m\": %d, "+
		"\"rh-true_m\": %d, \"pressure-nn_c\": %d, \"rain-cupfills\": %d } }",
		temp, rhtrue, press, cups)
	return []byte(data)
}

func (in *RandomInput) ReadLine() (error, []byte) {
	for !time.Now().After(in.ts.Add(10 * time.Second)) {
		time.Sleep(2 * time.Second)
	}
	in.ts = time.Now()
	data := getRandomData(in)
	return nil, data
}

func (in *RandomInput) Close() {}

func storeWeather(d db.DB, weather *data.Weather) error {
	for _, meas := range weather.Measurements() {
		d.AddValue(weather.StationId(), meas.Name, meas.Value)
	}
	return d.Save()
}

func main() {
	in := NewRandomInput()
	d := db.NewInfluxDBClient("localhost", 8086, "rndweather")

	// Copied and adapted from wcollector/main
	for {
		err, line := in.ReadLine()
		if err == nil {
			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("# ")) {
				err, json := data.ParseJson(line)
				if err != nil {
					fmt.Printf("Cannot process line '%s', because of: %v\n", line, err)
				}
				err, weather := data.ParseWeather(json)
				if err != nil {
					fmt.Printf("Cannot parse weather %v\n", err)
				}
				fmt.Println("Lazy patching in dew point")
				data.LazyMonkeyPatchDewPoint(weather)
				fmt.Printf("Patched %+v\n", weather)

				if err := storeWeather(d, weather); err != nil {
					fmt.Printf("Failed storing values: %v\n", err)
				}
			}
		} else {
			if err != io.EOF {
				fmt.Printf("I/O error: %v\n", err)
			}
			break
		}
	}
}

// curl -i -XPOST 'http://localhost:8086/write?db=rndweather' --data-binary 'sun value=0'

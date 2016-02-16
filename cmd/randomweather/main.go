package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"bitbucket.org/mgeiger/wcollector/data"
	"bitbucket.org/mgeiger/wcollector/db"
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
	rhTrue := in.src.Intn(9000)
	press := in.src.Intn(13000)
	cups := in.src.Intn(150)

	data := fmt.Sprintf("{ \"weather\": { \"station-id\": 1, \"temp_m\": %d, "+
		"\"rh-true_m\": %d, \"pressure-nn_c\": %d, \"rain-cupfills\": %d } }",
		temp, rhTrue, press, cups)

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

func main() {
	var logger *log.Logger = log.New(os.Stdout, "[wcollector] ", log.LstdFlags)

	in := NewRandomInput()
	db := db.NewInfluxDBClient("localhost", 8086, "rndweather")

	data.Consume(in, db, logger)
}

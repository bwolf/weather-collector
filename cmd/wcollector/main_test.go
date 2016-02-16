package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"bitbucket.org/mgeiger/wcollector/data"
)

type MockInput struct {
	data  [][]byte
	index int
}

func NewMockInput(data [][]byte) *MockInput {
	return &MockInput{data: data, index: 0}
}

func (in *MockInput) ReadLine() (error, []byte) {
	if in.index >= len(in.data) {
		fmt.Println("EOF")
		return io.EOF, nil
	}
	result := in.data[in.index]
	in.index++
	return nil, result
}

func (in *MockInput) Close() {}

type MockDB struct {
	data bytes.Buffer
}

func (d *MockDB) AddValue(stationId int, key string, value float64) {
	fmt.Fprintf(&d.data, "%s,station=%d value=%f\n", key, stationId, value)
}

func (d *MockDB) Save() error {
	fmt.Printf("Would POST %v", d.data.String())
	d.data.Reset()
	return nil
}

func TestMain(t *testing.T) {
	s := "{ \"weather\": { \"station-id\": 1, \"temp_m\": 727, " +
		"\"rh-true_m\": 7223, \"pressure-nn_c\": 10078, \"rain-cupfills\": 133 } }"

	in := NewMockInput([][]byte{[]byte(s)})
	db := &MockDB{}
	logger := log.New(os.Stdout, "", log.LstdFlags)

	data.Consume(in, db, logger)
}

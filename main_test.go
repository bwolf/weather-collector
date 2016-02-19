package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"
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

func (d *MockDB) Query(query string) (error, *InfluxQueryResult) {
	fmt.Printf("Would GET %v\n", query)
	var r InfluxQueryResult
	return nil, &r
}

func (d *MockDB) AddValue(stationId int, key string, value float64) {
	fmt.Fprintf(&d.data, "%s,station=%d value=%f\n", key, stationId, value)
}

func (ic *MockDB) AddText(key, value string, tags []TSDBTuple, timestamp time.Time) {
	var b bytes.Buffer
	for _, el := range tags {
		fmt.Fprintf(&b, ",%s=%s", el.key, el.val) // First colon separates measurement from tags
	}
	fmt.Fprintf(&ic.data, "%s%s text=\"%s\" %d", key, b.String(), value, timestamp.UnixNano())
}

func (d *MockDB) Save() error {
	fmt.Printf("Would POST %v\n", d.data.String())
	d.data.Reset()
	return nil
}

func TestMain(m *testing.M) {
	s := "{ \"weather\": { \"station-id\": 1, \"temp_m\": 727, " +
		"\"rh-true_m\": 7223, \"pressure-nn_c\": 10078, \"rain-cupfills\": 133 } }"

	in := NewMockInput([][]byte{[]byte(s)})
	db := &MockDB{}
	logger := log.New(os.Stdout, "", log.LstdFlags)

	Consume(in, db, logger)
}

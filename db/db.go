package db

import (
	"bytes"
	"fmt"
	"net/http"
)

type DB interface {
	AddValue(stationId int, key string, value float64)
	Save() error
}

// InfluxDB client for weather data
type InfluxDBClient struct {
	url  string
	data bytes.Buffer
}

func NewInfluxDBClient(host string, port int, dbName string) *InfluxDBClient {
	url := fmt.Sprintf("http://%s:%d/write?db=%s", host, port, dbName)
	return &InfluxDBClient{url: url}
}

func (ic *InfluxDBClient) AddValue(stationId int, key string, value float64) {
	// see https://influxdb.com/docs/v0.9/guides/writing_data.html
	fmt.Fprintf(&ic.data, "%s,station=%d value=%f\n", key, stationId, value)
}

func (ic *InfluxDBClient) Save() error {
	resp, err := http.Post(ic.url, "text/plain", &ic.data)
	if err != nil {
		return fmt.Errorf("HTTP POST to InfluxDB failed: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP POST to InfluxDB failed: %s\n", resp.Status)
	}

	ic.data.Reset()

	return nil
}

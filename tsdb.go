package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Query result of InfluxDB
type InfluxQueryResult struct {
	Results []struct {
		Series []struct {
			Name    string     `json:"name"`
			Columns []string   `json:"columns"`
			Values  [][]string `json:"values"`
		} `json:"series"`
	} `json:"results"`
}

// Tuple with key and value
type TSDBTuple struct {
	key string
	val string
}

// Timeseries DB interface
type TSDBClient interface {
	Query(query string) (error, *InfluxQueryResult)
	AddValue(stationId int, key string, value float64)
	AddText(key, value string, tags []TSDBTuple, timestamp time.Time)
	Save() error
}

// InfluxDB client for weather data
type InfluxDBClient struct {
	insertUrl string
	queryUrl  string
	data      bytes.Buffer
	debug     bool
}

// Create an InfluxDB client for a specific database
func NewInfluxDBClient(host string, port int, dbName string) *InfluxDBClient {
	insertUrl := fmt.Sprintf("http://%s:%d/write?db=%s", host, port, dbName)
	queryUrl := fmt.Sprintf("http://%s:%d/query?db=%s", host, port, dbName)
	return &InfluxDBClient{insertUrl: insertUrl, queryUrl: queryUrl}
}

// Enable or disable debug printing
func (ic *InfluxDBClient) SetDebug(flag bool) {
	ic.debug = flag
}

// Add value to the set of values which are stored by the Save() method.
func (ic *InfluxDBClient) AddValue(stationId int, key string, value float64) {
	// see https://influxdb.com/docs/v0.9/guides/writing_data.html
	fmt.Fprintf(&ic.data, "%s,station=%d value=%f\n", key, stationId, value)
}

// Add text value to the set of values which are stored by the Save() method.
func (ic *InfluxDBClient) AddText(key, value string, tags []TSDBTuple, timestamp time.Time) {
	var b bytes.Buffer
	for _, el := range tags {
		fmt.Fprintf(&b, ",%s=%s", el.key, el.val) // First colon separates measurement from tags
	}
	fmt.Fprintf(&ic.data, "%s%s text=\"%s\" %d", key, b.String(), value, timestamp.UnixNano())
}

// Save stored values.
func (ic *InfluxDBClient) Save() error {
	requ, errReq := http.NewRequest("POST", ic.insertUrl, strings.NewReader(ic.data.String()))
	if errReq != nil {
		return fmt.Errorf("Cannot create request: %v", errReq)
	}

	if ic.debug {
		dump, _ := httputil.DumpRequestOut(requ, true)
		fmt.Printf("%q\n", dump)
	}

	client := &http.Client{}
	resp, err := client.Do(requ)
	ic.data.Reset()

	if err != nil {
		return fmt.Errorf("HTTP POST to InfluxDB failed: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP POST to InfluxDB failed: %s", resp.Status)
	}

	return nil
}

// Query database.
func (ic *InfluxDBClient) Query(query string) (error, *InfluxQueryResult) {
	url := fmt.Sprintf("%s&q=%s", ic.queryUrl, url.QueryEscape(query))
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP Query to InfluxDB failed: %v", err), nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP GET to Influx DB failed: %s", resp.Status), nil
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read HTTP response: %v", err), nil
	}

	var result InfluxQueryResult
	err = json.Unmarshal(content, &result)
	if err != nil {
		return fmt.Errorf("Failed to parse json: %v", err), nil
	}

	return nil, &result
}

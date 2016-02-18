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

type InfluxQueryResult struct {
	Results []struct {
		Series []struct {
			Name    string     `json:"name"`
			Columns []string   `json:"columns"`
			Values  [][]string `json:"values"`
		} `json:"series"`
	} `json:"results"`
}

type TSDBClient interface {
	Query(query string) (error, *InfluxQueryResult)
	AddValue(stationId int, key string, value float64)
	AddText(key, value string, timestamp time.Time)
	Save() error
}

// InfluxDB client for weather data
type InfluxDBClient struct {
	insertUrl string
	queryUrl  string
	data      bytes.Buffer
	debug     bool
}

func NewInfluxDBClient(host string, port int, dbName string) *InfluxDBClient {
	insertUrl := fmt.Sprintf("http://%s:%d/write?db=%s", host, port, dbName)
	queryUrl := fmt.Sprintf("http://%s:%d/query?db=%s", host, port, dbName)
	return &InfluxDBClient{insertUrl: insertUrl, queryUrl: queryUrl}
}

func (ic *InfluxDBClient) SetDebug(flag bool) {
	ic.debug = flag
}

func (ic *InfluxDBClient) AddValue(stationId int, key string, value float64) {
	// see https://influxdb.com/docs/v0.9/guides/writing_data.html
	fmt.Fprintf(&ic.data, "%s,station=%d value=%f\n", key, stationId, value)
}

func (ic *InfluxDBClient) AddText(key, value string, timestamp time.Time) {
	fmt.Fprintf(&ic.data, "%s text=\"%s\" %d", key, value, timestamp.UnixNano())
}

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

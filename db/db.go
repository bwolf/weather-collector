package db

import (
	"bytes"
	"fmt"
	"net/http"
)

// Weather db (InfluxDB) client
type DB struct {
	influxUrl string
	data      bytes.Buffer
}

func NewDB(host string, port int, dbName string) *DB {
	url := fmt.Sprintf("http://%s:%d/write?db=%s", host, port, dbName)
	return &DB{influxUrl: url}
}

// see https://influxdb.com/docs/v0.9/guides/writing_data.html
func (db *DB) AddValue(stationId int, key string, value float64) {
	fmt.Fprintf(&db.data, "%s,station=%d value=%f\n", key, stationId, value)
}

func (db *DB) Post() error {
	resp, err := http.Post(db.influxUrl, "text/plain", &db.data)
	if err != nil {
		return fmt.Errorf("HTTP POST failed to InfluxDB: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP POST failed to InfluxDB: %s\n", resp.Status)
	}

	db.data.Reset()

	return nil
}

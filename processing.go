package main

import (
	"bytes"
	"fmt"
	"github.com/bwolf/suncal"
	"io"
	"log"
	"time"
)

func storeWeather(db TSDBClient, weather *Weather) error {
	for _, meas := range weather.Measurements() {
		db.AddValue(weather.StationId(), meas.Name, meas.Value)
	}
	return db.Save()
}

func Consume(in Input, db TSDBClient, logger *log.Logger) {
	for {
		err, line := in.ReadLine()
		if err == nil {
			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("# ")) {
				err, json := ParseJson(line)
				if err != nil {
					logger.Printf("Cannot process line '%s', because of: %v\n", line, err)
				}
				err, weather := ParseWeather(json)
				if err != nil {
					logger.Printf("Cannot parse weather %v\n", err)
				}
				logger.Println("Lazy patching in dew point")
				LazyMonkeyPatchDewPoint(weather)
				logger.Printf("Patched %+v\n", weather)

				if err := storeWeather(db, weather); err != nil {
					logger.Printf("Failed storing values: %v\n", err)
				}
			}
		} else {
			if err != io.EOF {
				logger.Printf("I/O error: %v\n", err)
			}
			break
		}
	}
}

// Use Grafana query 'SELECT * FROM "sun" WHERE $timeFilter' to get the annoation.
func EnsureSunRiseAndSet(db TSDBClient, day time.Time, lat, lon float64) error {
	geo := suncal.GeoCoordinates{Latitude: lat, Longitude: lon}
	sunInfo := suncal.SunCal(geo, day)

	for i, t := range []time.Time{sunInfo.Rise, sunInfo.Set} {
		err, res := db.Query(fmt.Sprintf("SELECT * FROM sun WHERE time = '%s'", t.Format(time.RFC3339)))
		if err != nil {
			return err
		}

		if res.Results != nil && len(res.Results) > 0 && len(res.Results[0].Series) > 0 {
			continue
		}

		// Insert missing value
		var text string
		if i == 0 {
			text = "Rise"
		} else {
			text = "Set"
		}

		db.AddText("sun", text, t)
		err = db.Save()
		if err != nil {
			return fmt.Errorf("Failed saving sun info: %v\n", err)
		}
	}

	return nil
}

package data

import (
	"bitbucket.org/mgeiger/wcollector/db"
	"bitbucket.org/mgeiger/wcollector/input"
	"bytes"
	"io"
	"log"
)

func storeWeather(db db.DB, weather *Weather) error {
	for _, meas := range weather.Measurements() {
		db.AddValue(weather.StationId(), meas.Name, meas.Value)
	}
	return db.Save()
}

func Consume(in input.Input, db db.DB, logger *log.Logger) {
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

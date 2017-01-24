package main

import (
	"log"
	"os"
	"sync"

	"time"

	"github.com/influxdata/influxdb/client/v2"
)

const (
	db            = "roomcontrol"
	intervalWrite = 15 * time.Minute
)

func startInfluxDBLogger(temps <-chan float32, hums <-chan float32, wg *sync.WaitGroup, quit <-chan struct{}) error {

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     os.Getenv("INFLUXADDR"),
		Username: os.Getenv("INFLUXUSER"),
		Password: os.Getenv("INFLUXPASS"),
	})
	if err != nil {
		return err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		bp, err := createNewBatchPoints()
		if err != nil {
			log.Printf("Couldn't create influxdb batch points %v.\nDon't log.", err)
			return
		}
		defer func() {
			// Try to write latest data.
			if bp != nil {
				if err := c.Write(bp); err != nil {
					log.Printf("Couldn't save latest batch of data: %v", err)
				}
			}
		}()

		writeToDisk := time.After(intervalWrite)
		for {
			select {
			case t := <-temps:
				addPoint("temp", t, bp)
			case h := <-hums:
				addPoint("hum", h, bp)
			case <-writeToDisk:
				writeToDisk = time.After(intervalWrite)
				log.Printf("Save data")
				if err := c.Write(bp); err != nil {
					log.Printf("Couldn't save last batch of data: %v", err)
					continue
				}
				bp, err = createNewBatchPoints()
				if err != nil {
					log.Printf("Couldn't create influxdb batch points %v.\nDon't log.", err)
					return
				}
			case <-quit:
				return
			}
		}

	}()

	return nil
}

func createNewBatchPoints() (client.BatchPoints, error) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  db,
		Precision: "s",
	})
	if err != nil {
		return nil, err
	}

	return bp, err
}

func addPoint(table string, data float32, bp client.BatchPoints) {
	field := map[string]interface{}{"value": data}
	pt, err := client.NewPoint(table, make(map[string]string), field, time.Now())
	if err != nil {
		log.Printf("Couldn't add %s data for %f", table, data)
		return
	}
	bp.AddPoint(pt)
}

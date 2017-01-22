package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

var lastWrite time.Time

func startLogger(temps <-chan float32, wg *sync.WaitGroup, quit <-chan struct{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		writeTempToDisk(temps, quit)
	}()
}

func writeTempToDisk(temps <-chan float32, quit <-chan struct{}) error {
	f, err := os.OpenFile("temp.logs", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()
	for {
		var t float32
		select {
		case t = <-temps:
		case <-quit:
			return nil
		}

		if _, err := w.WriteString(fmt.Sprint(t, "\n")); err != nil {
			fmt.Printf("Error while writing %+v\n", err)
			continue
		}
		// If we didn't write to the file in the last hour, do it.
		now := time.Now()
		if lastWrite.Before(now.Add(-time.Duration(time.Hour))) {
			w.Flush()
			lastWrite = now
		}
	}

}

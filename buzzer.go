package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/didrocks/roomcontrol/grovepi"
)

const (
	pinBuzz      = grovepi.D3
	buzzTreshold = 3
	gracePeriod  = 30 * time.Minute
)

func startBuzzer(tempOK <-chan bool, bEvents <-chan ButtonEvent, g grovepi.GrovePi, wg *sync.WaitGroup, quit <-chan struct{}) (chan bool, chan bool, error) {

	if err := g.PinMode(pinBuzz, "OUTPUT"); err != nil {
		return nil, nil, err
	}

	buzzEnabled, buzzTempDisabled := make(chan bool), make(chan bool)
	wg.Add(1)
	childWait := sync.WaitGroup{}

	go func() {
		defer wg.Done()
		defer buzz(g, false)

		numTempKO := 0
		buzzOn := false
		canBuzz := true
		buzzEnabled <- canBuzz
		tempDisabled := false
		buzzTempDisabled <- tempDisabled

		var reenableTimeout *time.Timer

		for {

			select {
			case tOK := <-tempOK:
				fmt.Println(tOK)
				if tOK {
					if buzzOn {
						buzz(g, false)
						buzzOn = false
					}
					numTempKO = 0
				} else { // Temperature isn't ok.
					numTempKO++
					if numTempKO >= buzzTreshold && !buzzOn {
						buzzOn = true
						childWait.Add(1)
						// Method handling buzz pattern.
						go func() {
							defer childWait.Done()
							buzzTicker := time.NewTicker(time.Duration(5 * time.Second))
							defer buzzTicker.Stop()
							for {
								select {
								case <-buzzTicker.C:
									if !canBuzz || tempDisabled {
										continue
									}
									buzz(g, true)
									time.Sleep(20 * time.Millisecond)
									buzz(g, false)
									time.Sleep(10 * time.Millisecond)
									buzz(g, true)
									time.Sleep(20 * time.Millisecond)
									buzz(g, false)
								case <-quit:
									return
								}
							}

						}()
					}
				}
			case e := <-bEvents:
				if e == LONGPRESS {
					tempDisabled = true
					buzzTempDisabled <- tempDisabled

					// Cancel previous timeout.
					if reenableTimeout != nil {
						reenableTimeout.Stop()
					}
					// Give a new grace period.
					reenableTimeout = time.AfterFunc(gracePeriod, func() {
						tempDisabled = false
						buzzTempDisabled <- tempDisabled
					})
				} else if e == DOUBLECLICK {
					canBuzz = !canBuzz
					buzzEnabled <- canBuzz
				}

			case <-quit:
				childWait.Wait()
				return
			}

		}
	}()

	return buzzEnabled, buzzTempDisabled, nil
}

func buzz(g grovepi.GrovePi, on bool) {
	beep := byte(0)
	if on {
		beep = 1
	}
	if err := g.DigitalWrite(pinBuzz, beep); err != nil {
		log.Printf("Couldn't set buzz to %d: %v", beep, err)
	}
}

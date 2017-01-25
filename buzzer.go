package main

import (
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

// BuzzMode can be disabled, low or high volume.
type BuzzMode int

const (
	// Disabled BuzzMode.
	Disabled = iota
	// LowVolume BuzzMode.
	LowVolume
	// HighVolume BuzzMode.
	HighVolume
)

func startBuzzer(tempOK <-chan bool, bEvents <-chan ButtonEvent, g grovepi.GrovePi, wg *sync.WaitGroup, quit <-chan struct{}) (chan BuzzMode, chan bool, error) {

	if err := g.PinMode(pinBuzz, "OUTPUT"); err != nil {
		return nil, nil, err
	}

	buzzMode, buzzTempDisabled := make(chan BuzzMode), make(chan bool)
	wg.Add(1)
	childWait := sync.WaitGroup{}

	go func() {
		defer wg.Done()
		defer buzz(g, Disabled, false)

		numTempKO := 0
		buzzOn := false
		curBuzzMode := BuzzMode(LowVolume)
		buzzMode <- curBuzzMode
		tempDisabled := false
		buzzTempDisabled <- tempDisabled

		var reenableTimeout *time.Timer

		for {

			select {
			case tOK := <-tempOK:
				if tOK {
					if buzzOn {
						buzz(g, Disabled, false)
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
									if curBuzzMode == Disabled || tempDisabled {
										continue
									}
									buzz(g, curBuzzMode, true)
									time.Sleep(20 * time.Millisecond)
									buzz(g, curBuzzMode, false)
									time.Sleep(10 * time.Millisecond)
									buzz(g, curBuzzMode, true)
									time.Sleep(20 * time.Millisecond)
									buzz(g, curBuzzMode, false)
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
					switch curBuzzMode {
					case Disabled:
						curBuzzMode = LowVolume
					case LowVolume:
						curBuzzMode = HighVolume
					case HighVolume:
						curBuzzMode = Disabled
					}
					buzzMode <- curBuzzMode
				}

			case <-quit:
				childWait.Wait()
				return
			}

		}
	}()

	return buzzMode, buzzTempDisabled, nil
}

func buzz(g grovepi.GrovePi, b BuzzMode, on bool) {
	beep := byte(0)
	if on {
		switch b {
		case LowVolume:
			beep = 1
		case HighVolume:
			beep = 10
		}
	}
	if err := g.AnalogWrite(pinBuzz, beep); err != nil {
		log.Printf("Couldn't set buzz to %d: %v", beep, err)
	}
}

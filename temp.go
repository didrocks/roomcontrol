package main

import (
	"math"
	"sync"

	"time"

	"github.com/didrocks/roomcontrol/grovepi"
)

// Our sensor is 1 degree celsius above real temperature.
const tempOffset float32 = -1

const pinDHT = grovepi.D8

func startMesTempAndHum(g grovepi.GrovePi, wg *sync.WaitGroup, quit <-chan struct{}) (<-chan float32, <-chan float32) {
	temps, hums := make(chan float32, 1), make(chan float32, 1)
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Get average temp and humidity over 1000 reads.
		mTemp, mHum := float32(0.0), float32(0.0)
		nTemp := 0
		nHum := 0
		tSent, hSent := false, false
		for {

			// Immediate exit if requested.
			select {
			case <-quit:
				return
			default:
			}

			t, h, err := g.ReadDHT(pinDHT)
			// Ignore invalid temp and humidity.
			if err != nil {
				continue
			}

			if !math.IsNaN(float64(t)) && t > -50 {
				mTemp += t
				nTemp++

				if nTemp == 1000 {
					temps <- mTemp/float32(nTemp) + tempOffset
					tSent = true
					mTemp = 0
					nTemp = 0
				}
			}

			if !math.IsNaN(float64(h)) && h > -50 {
				mHum += h
				nHum++
				if nHum == 1000 {
					hums <- mHum / float32(nHum)
					hSent = true
					mHum = 0
					nHum = 0
				}
			}

			// Exit if quit was requested, otherwise take next value in 5 minutes.
			if tSent && hSent {
				tSent, hSent = false, false
				select {
				case <-quit:
					return
				case <-time.After(5 * time.Minute):
				}
			} else {
				// Give an extra millisecond to let other goroutines working as expected
				time.Sleep(time.Millisecond)
			}

		}
	}()

	return temps, hums
}

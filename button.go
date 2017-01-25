package main

import (
	"log"
	"sync"
	"time"

	"github.com/didrocks/roomcontrol/grovepi"
)

// ButtonEvent is different type of button clics
type ButtonEvent int

const (
	// SINGLECLICK button click
	SINGLECLICK ButtonEvent = iota
	// DOUBLECLICK button click
	DOUBLECLICK
	// LONGPRESS button click
	LONGPRESS

	pinButton         = grovepi.D7
	doubleClickTime   = time.Second
	defaultResolution = 50 * time.Millisecond
)

func startButtonListener(g grovepi.GrovePi, wg *sync.WaitGroup, quit <-chan struct{}) (<-chan ButtonEvent, error) {
	err := g.PinMode(pinButton, "input")
	if err != nil {
		return nil, err
	}

	ev := make(chan ButtonEvent)

	wg.Add(1)
	go func() {
		defer wg.Done()

		inClick := false
		var firstClick time.Time
		var singleClickTimeout *time.Timer
		res := defaultResolution

		for {
			select {
			case <-time.After(res):
				val, err := g.DigitalRead(pinButton)
				if val == 1 {
					// Can be first or second click.
					if !inClick {
						inClick = true
						// First click.
						if time.Now().Sub(firstClick) > doubleClickTime {
							firstClick = time.Now()
							res = 10 * time.Millisecond
							singleClickTimeout = time.AfterFunc(doubleClickTime, func() {
								// Only one click happened

								// If we are still actively clicking, it's a long press
								if inClick {
									ev <- LONGPRESS
								} else {
									// It was only a single click.
									ev <- SINGLECLICK
								}
								res = defaultResolution
							})
						} else {
							// Double click event, fire away!
							singleClickTimeout.Stop()
							ev <- DOUBLECLICK
							res = defaultResolution
						}
					}
				} else {
					inClick = false
				}
				if err != nil {
					log.Println("Button read error")
				}
			case <-quit:
				return
			}
		}

	}()

	return ev, nil
}

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/didrocks/roomcontrol/grovepi"
)

// ButtonEvent is different type of button clics
type ButtonEvent int

const (
	// SINGLE button clic
	SINGLECLICK ButtonEvent = iota
	// DOUBLE button clic
	DOUBLECLICK

	pin             = grovepi.D8
	doubleClickTime = time.Second
)

func startButtonListener(g grovepi.GrovePi, wg *sync.WaitGroup, quit <-chan struct{}) (<-chan ButtonEvent, error) {
	err := g.PinMode(pin, "input")
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

		for {
			select {
			case <-time.After(100 * time.Millisecond):
				val, err := g.DigitalRead(pin)
				if val == 1 {
					// Can be first or second click.
					if !inClick {
						inClick = true
						// First click.
						if time.Now().Sub(firstClick) > doubleClickTime {
							firstClick = time.Now()
							singleClickTimeout = time.AfterFunc(doubleClickTime, func() {
								// It was only a single click.
								ev <- SINGLECLICK
							})
						} else {
							// Double click event, fire away!
							singleClickTimeout.Stop()
							ev <- DOUBLECLICK
						}
					}
				} else {
					inClick = false
				}
				if err != nil {
					fmt.Println("Button read error")
				}
			case <-quit:
				return
			}
		}

	}()

	return ev, nil
}

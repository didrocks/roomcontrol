package main

import (
	"image/color"
	"sync"
	"time"

	"fmt"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/chip"
)

type display struct {
	colorOn  bool
	curColor color.RGBA
	screen   *i2c.GroveLcdDriver
	r        *gobot.Robot
}

func startDisplay(temps <-chan float32, humids <-chan float32, wg *sync.WaitGroup, quit <-chan struct{}) {
	wg.Add(1)
	d := display{}

	go func() {
		defer wg.Done()
		board := chip.NewAdaptor()
		d.screen = i2c.NewGroveLcdDriver(board)

		var mainloop = func() {
			screen := d.screen

			// Tear down LCD by erasing and clearing the screen.
			defer func() {
				screen.SetRGB(0, 0, 0)
				screen.Clear()
				d.r.Stop()
				// wait for some milliseconds for the robot to send pending commands
				time.Sleep(100 * time.Millisecond)
			}()

			screen.Clear()

			for {
				select {
				case t := <-temps:
					screen.Home()
					screen.Write(fmt.Sprintf("Temp : %.1fC", t))
					d.updateColor()
				case h := <-humids:
					screen.Home()
					screen.Write(fmt.Sprintf("\nHum :  %.0f%%", h))
				case <-quit:
					d.screen.Clear()
					return
				}
			}

		}

		d.r = gobot.NewRobot("display",
			[]gobot.Connection{board},
			[]gobot.Device{d.screen},
			mainloop,
		)

		// We don't want gobot to handle SIGINT and quit itself.
		d.r.Start(false)
	}()
}

func (d *display) updateColor() {
	c := d.curColor
	if d.colorOn {
		d.screen.SetRGB(int(c.R), int(c.G), int(c.B))
	} else {
		d.screen.SetRGB(0, 0, 0)
	}
}

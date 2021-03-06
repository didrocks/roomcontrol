package main

import (
	"image/color"
	"math"
	"sync"

	"fmt"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/chip"
)

const tempColorNorm = 19.0

var (
	colorMax  = color.RGBA{255, 62, 0, 0}
	colorNorm = color.RGBA{96, 129, 5, 0}
	colorMin  = color.RGBA{116, 106, 255, 0}
	bell      = [...]byte{4, 14, 14, 14, 31, 4, 0, 0}
	bellloud  = [...]byte{21, 14, 14, 14, 31, 4, 17, 0}
	clock     = [...]byte{0, 14, 21, 23, 17, 14, 0, 0}
)

type display struct {
	colorOn  bool
	curColor color.RGBA
	screen   *i2c.GroveLcdDriver
	r        *gobot.Robot
}

func startDisplay(temps <-chan float32, humids <-chan float32, bEvents <-chan ButtonEvent, buzzMode <-chan BuzzMode, buzzTempDisabled <-chan bool, tempOk chan<- bool, wg *sync.WaitGroup, quit <-chan struct{}) {
	wg.Add(1)
	d := display{colorOn: true}

	workDone := make(chan struct{})

	go func() {
		defer wg.Done()
		board := chip.NewAdaptor()
		d.screen = i2c.NewGroveLcdDriver(board)

		// Tear down LCD by erasing and clearing the screen.
		defer func() {
			d.screen.SetRGB(0, 0, 0)
			d.screen.Clear()
			d.r.Stop()
		}()

		var mainloop = func() {
			screen := d.screen
			screen.Clear()

			for {
				select {
				case t := <-temps:
					screen.Home()
					screen.Write(fmt.Sprintf("Temp : %.1fC", t))
					c := d.evaluateTemp(float64(t))
					tempOk <- c
					d.updateColor()
				case h := <-humids:
					screen.Home()
					d.screen.SetPosition(16)
					screen.Write(fmt.Sprintf("Hum :  %.0f%%", h))
				case e := <-bEvents:
					if e == SINGLECLICK {
						d.colorOn = !d.colorOn
						d.updateColor()
					}
				case e := <-buzzMode:
					screen.Home()
					d.screen.SetPosition(15)
					switch e {
					case Disabled:
						d.screen.Write(" ")
					case LowVolume:
						d.screen.Write(string(byte(0)))
						d.screen.SetCustomChar(0, bell)
					case HighVolume:
						d.screen.Write(string(byte(1)))
						d.screen.SetCustomChar(1, bellloud)
					}
				case td := <-buzzTempDisabled:
					screen.Home()
					d.screen.SetPosition(31)
					if td {
						d.screen.Write(string(byte(2)))
						d.screen.SetCustomChar(2, clock)
					} else {
						d.screen.Write(" ")
					}
				case <-quit:
					close(workDone)
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
		<-workDone
	}()
}

// evaluateTemp based on Norm. Signal if any deficiency and update current color
// we have a gradient of 1 degree.
func (d *display) evaluateTemp(t float64) bool {
	proRata := t - tempColorNorm
	tempOk := true
	if proRata > 2.0 || proRata < -1.5 {
		tempOk = false
	}
	proRata = math.Min(math.Max(proRata, -1), 1)

	// Temp superior to norm
	if proRata > 0 {
		d.curColor = color.RGBA{
			uint8(float64(colorNorm.R) + proRata*(float64(int(colorMax.R)-int(colorNorm.R)))),
			uint8(float64(colorNorm.G) + proRata*(float64(int(colorMax.G)-int(colorNorm.G)))),
			uint8(float64(colorNorm.B) + proRata*(float64(int(colorMax.B)-int(colorNorm.B)))),
			0,
		}
	} else {
		d.curColor = color.RGBA{
			uint8(float64(colorNorm.R) - proRata*(float64(int(colorMin.R)-int(colorNorm.R)))),
			uint8(float64(colorNorm.G) - proRata*(float64(int(colorMin.G)-int(colorNorm.G)))),
			uint8(float64(colorNorm.B) - proRata*(float64(int(colorMin.B)-int(colorNorm.B)))),
			0,
		}

	}

	return tempOk
}

func (d *display) updateColor() {
	c := d.curColor
	if d.colorOn {
		d.screen.SetRGB(int(c.R), int(c.G), int(c.B))
	} else {
		d.screen.SetRGB(0, 0, 0)
	}
}

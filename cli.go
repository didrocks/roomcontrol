package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/didrocks/roomcontrol/grovepi"
)

func main() {

	// Disable logging for now.
	log.SetOutput(ioutil.Discard)

	g := *grovepi.InitGrovePi(0x04)
	defer g.CloseDevice()

	wg := &sync.WaitGroup{}
	quit := make(chan struct{})

	// Handle user generated stop requests.
	userstop := make(chan os.Signal)
	signal.Notify(userstop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-userstop
		close(quit)
	}()

	temps, hums := startMesTempAndHum(g, wg, quit)
	var tempListeners []chan float32
	var humListeners []chan float32
	var buttonListeners []chan ButtonEvent

	// Disk loggers.
	/*tempListeners, t := newlistener(tempListeners)
	startLogger(t, wg, quit)*/

	// Influxdb logger.
	tempListeners, t := newlistener(tempListeners)
	humListeners, h := newlistener(humListeners)
	if err := startInfluxDBLogger(t, h, wg, quit); err != nil {
		log.Printf("InfluxDB connect: %v", err)
		return
	}

	// Listen on button.
	bEvents, err := startButtonListener(g, wg, quit)
	if err != nil {
		log.Printf("Connect to button error: %v", err)
		return
	}

	// Buzzer enablement.
	tempOk := make(chan bool)
	buttonListeners, b := newbuttonlistener(buttonListeners)
	buzzEnabled, buzzTempDisabled, err := startBuzzer(tempOk, b, g, wg, quit)
	if err != nil {
		log.Printf("Couldn't connect buzzer: %v", err)
		return
	}

	// LCD screen display.
	tempListeners, t = newlistener(tempListeners)
	humListeners, h = newlistener(humListeners)
	buttonListeners, b = newbuttonlistener(buttonListeners)
	startDisplay(t, h, b, buzzEnabled, buzzTempDisabled, tempOk, wg, quit)

	// Values multipler.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case t := <-temps:
				for _, l := range tempListeners {
					l <- t
				}
			case h := <-hums:
				for _, l := range humListeners {
					l <- h
				}
			case e := <-bEvents:
				for _, l := range buttonListeners {
					l <- e
				}
			case <-quit:
				return
			}
		}

	}()

	wg.Wait()

	for _, l := range tempListeners {
		close(l)
	}
	for _, l := range humListeners {
		close(l)
	}
	for _, l := range buttonListeners {
		close(l)
	}
}

func newlistener(listener []chan float32) ([]chan float32, chan float32) {
	t := make(chan float32)
	listener = append(listener, t)

	return listener, t
}

func newbuttonlistener(listener []chan ButtonEvent) ([]chan ButtonEvent, chan ButtonEvent) {
	t := make(chan ButtonEvent)
	listener = append(listener, t)

	return listener, t
}

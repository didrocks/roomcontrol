package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/didrocks/roomcontrol/grovepi"
)

func main() {

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

	tempListeners, t := newlistener(tempListeners)
	startLogger(t, wg, quit)

	// LCD screen display.
	tempListeners, t = newlistener(tempListeners)
	humListeners, h := newlistener(humListeners)
	startDisplay(t, h, wg, quit)

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
}

func newlistener(listner []chan float32) ([]chan float32, chan float32) {
	t := make(chan float32)
	listner = append(listner, t)

	return listner, t
}

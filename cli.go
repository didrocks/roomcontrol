package main

import (
	"fmt"
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

	tempListeners, t := newlistener(tempListeners)
	startLogger(t, wg, quit)

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
				fmt.Print("humidity is ")
				fmt.Println(h)
			case <-quit:
				return
			}
		}

	}()

	wg.Wait()

	for _, l := range tempListeners {
		close(l)
	}

}

func newlistener(listner []chan float32) ([]chan float32, chan float32) {
	t := make(chan float32)
	listner = append(listner, t)

	return listner, t
}

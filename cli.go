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

	// Values multipler.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case t := <-temps:
				fmt.Print("temperature is")
				fmt.Println(t)
			case h := <-hums:
				fmt.Print("humidity is ")
				fmt.Println(h)
			case <-quit:
				return
			}
		}

	}()

	wg.Wait()

}

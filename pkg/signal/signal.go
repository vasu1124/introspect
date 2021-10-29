package signal

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func SignalHandler() (stopChanel <-chan int) {

	stop := make(chan int)
	signalChanel := make(chan os.Signal, 1)
	signal.Notify(signalChanel,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		for {
			s := <-signalChanel
			switch s {
			// kill -SIGHUP
			case syscall.SIGHUP:
				fmt.Println("Signal hang up.")
				stop <- 1

			// kill -SIGINT or Ctrl+c
			case syscall.SIGINT:
				fmt.Println("Signal interrupt.")
				stop <- 2

			// kill -SIGTERM
			case syscall.SIGTERM:
				fmt.Println("Signal terminate.")
				stop <- 3

			// kill -SIGQUIT
			case syscall.SIGQUIT:
				fmt.Println("Signal quit.")
				stop <- 4

			default:
				fmt.Println("Signal unknown.")
				stop <- 99
			}
		}
	}()

	return stop
}

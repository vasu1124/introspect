package signal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vasu1124/introspect/pkg/logger"
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
				logger.Log.Info("[signal] Signal hang up")
				stop <- 1

			// kill -SIGINT or Ctrl+c
			case syscall.SIGINT:
				logger.Log.Info("[signal] Signal interrupt")
				stop <- 2

			// kill -SIGTERM
			case syscall.SIGTERM:
				logger.Log.Info("[signal] Signal terminate")
				stop <- 3

			// kill -SIGQUIT
			case syscall.SIGQUIT:
				logger.Log.Info("[signal] Signal quit")
				stop <- 4

			default:
				logger.Log.Info("[signal] Signal unknown")
				stop <- 99
			}
		}
	}()

	return stop
}

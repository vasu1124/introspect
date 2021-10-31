// Inspired from https://github.com/kelseyhightower/inspector/blob/master/logger.go

package main

import (
	"log"

	"github.com/vasu1124/introspect/pkg/server"
	"github.com/vasu1124/introspect/pkg/signal"
	"github.com/vasu1124/introspect/pkg/version"
)

func main() {
	log.Printf("[introspect] Version = %s/%s/%s", version.Version, version.Commit, version.Branch)

	stop := signal.SignalHandler()
	srv := server.NewServer()
	srv.Run(stop)
}

package version

import (
	"flag"
)

var (
	// Version of inspect
	Version = "v0.0"
	// Commit of inspect
	Commit = "dev"
	// Branch of inspect
	Branch = "dev"

	//Port to bind
	Port *int
	//TLSPort to bind
	TLSPort *int
)

func init() {
	Port = flag.Int("port", 9090, "Port to bind server.")
	TLSPort = flag.Int("tlsport", 10443, "TLS Port to bind server.")
	flag.Parse()
}

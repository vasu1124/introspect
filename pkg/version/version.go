package version

import (
	"flag"
)

var (
	// Version of inspect
	Version = "0.0.0"
	// Commit of inspect
	Commit = "dev"
	// Branch of inspect
	Branch = "dev"

	//Port to bind
	Port       *int
	SecurePort *int
)

func init() {
	Port = flag.Int("port", 9090, "Port to bind server.")
	SecurePort = flag.Int("secureport", 9443, "Secureport to bind server.")
	flag.Parse()
}

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
)

func init() {
	Port = flag.Int("port", 9090, "Port to bind server.")
	flag.Parse()
}

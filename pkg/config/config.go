package config

import "time"

// Config struct
type Config struct {
	Port        int
	SecurePort  int
	LogLevel    string
	Development bool

	AssetDir     string
	TLSCertFile  string
	TLSKeyFile   string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Default configuration
var Default = &Config{
	Port:        9090,
	SecurePort:  9443,
	LogLevel:    "info",
	Development: false,

	AssetDir:     "assets",
	TLSCertFile:  "etc/tls/server.crt",
	TLSKeyFile:   "etc/tls/server.key",
	ReadTimeout:  30 * time.Second,
	WriteTimeout: 30 * time.Second,
	IdleTimeout:  60 * time.Second,
}

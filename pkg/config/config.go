package config

// Config struct
type Config struct {
	Port       int
	SecurePort int
	//[debug|info|warn|error|fatal|panic]
	LogLevel    string
	Development bool
}

// Default configuration
var Default = &Config{
	Port:        9090,
	SecurePort:  9443,
	LogLevel:    "",
	Development: false,
}

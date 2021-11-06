package config

type Config struct {
	Port       int
	SecurePort int
}

var Default = &Config{
	Port:       9090,
	SecurePort: 9443,
}

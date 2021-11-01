package config

type config struct {
	//Port to bind
	Port int
	//TLS Port to bind
	SecurePort int
}

var Config = &config{
	Port:       9090,
	SecurePort: 9443,
}

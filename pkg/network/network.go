package network

import (
	"io/ioutil"
	"log"
	"net"
)

// Data .
type Data struct {
	ResolvConf string
	Interfaces []net.Interface
}

// NetworkData .
var NetworkData Data

func init() {

	resolvConf, err := ioutil.ReadFile("/etc/resolv.conf")
	if err != nil {
		log.Println("[network] ReadFile: ", err)
		return
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println("[network] Interfaces: ", err)
		return
	}

	NetworkData.ResolvConf = string(resolvConf)
	NetworkData.Interfaces = ifaces
}

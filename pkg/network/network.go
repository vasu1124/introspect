package network

import (
	"io/ioutil"
	"net"

	"github.com/vasu1124/introspect/pkg/logger"
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
		logger.Log.Error(err, "[network] ReadFile")
		return
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		logger.Log.Error(err, "[network] Interfaces")
		return
	}

	NetworkData.ResolvConf = string(resolvConf)
	NetworkData.Interfaces = ifaces
}

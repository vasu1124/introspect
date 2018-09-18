package version

import (
	"flag"
	"log"
	"os"
	"path/filepath"
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
	//MasterURL to kubernetes cluster
	MasterURL *string
	//Kubeconfig to kubernetes cluster
	Kubeconfig *string
)

func init() {
	MasterURL = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	if kc := os.Getenv("KUBECONFIG"); kc != "" {
		Kubeconfig = flag.String("kubeconfig", kc, "(optional) Path to a kube config. Only required if out-of-cluster.")
	} else {
		kcdefault := filepath.Join(homeDir(), ".kube", "config")
		if _, err := os.Stat(kcdefault); err == nil {
			Kubeconfig = flag.String("kubeconfig", kcdefault, "(optional) Path to a kube config. Only required if out-of-cluster.")
		} else {
			Kubeconfig = flag.String("kubeconfig", "", "(optional) Path to a kube config. Only required if out-of-cluster.")
		}
	}
	Port = flag.Int("port", 8080, "Port to bind server.")
	TLSPort = flag.Int("tlsport", 10443, "TLS Port to bind server.")
	flag.Parse()

	if *Kubeconfig != "" {
		log.Printf("[introspect] KUBECONFIG=%s\n", *Kubeconfig)
	}
	if *MasterURL != "" {
		log.Printf("[introspect] Kubernets API=%s\n", *MasterURL)
	}
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return os.Getenv("USERPROFILE") // windows
}

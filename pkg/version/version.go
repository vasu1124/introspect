package version

import "flag"

var (
	// Version of inspect
	Version = "v0.0"
	// Commit of inspect
	Commit = "dev"
	// Branch of inspect
	Branch = "dev"

	//Port to bind
	Port *int
	//MasterURL to kubernetes cluster
	MasterURL *string
	//Kubeconfig to kubernetes cluster
	Kubeconfig *string
)

func init() {
	MasterURL = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	Kubeconfig = flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	Port = flag.Int("port", 8080, "Port to bind server.")
	flag.Parse()
}

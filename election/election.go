package election

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vasu1124/introspect/version"

	corev1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

//Leader am I a Leader?
//TODO: secure with mutex
var Leader = false

//Handler .
type Handler struct {
	config         *rest.Config
	kubeClient     *clientset.Clientset
	rc             record.EventRecorder
	LeaderElection *leaderelection.LeaderElectionConfig
}

// New .
func New() *Handler {
	var h Handler
	var err error

	// Create the client config. Use masterURL and kubeconfig if given, otherwise assume in-cluster.
	h.config, err = clientcmd.BuildConfigFromFlags(*version.MasterURL, *version.Kubeconfig)
	if err != nil {
		log.Printf("[election] KubeConfig error: %v", err)
		return &h
	}

	h.kubeClient, err = clientset.NewForConfig(h.config)
	if err != nil {
		log.Printf("[election] ClientSet error: %v", err)
		return &h
	}

	// v, err := h.kubeClient.ServerVersion()
	// if v.Major < 1 && v.Minor < 7 {
	// 	log.Fatal("Version is not enough")
	// }

	// Set up leader election if enabled and prepare event recorder.
	recorder := createRecorder(h.kubeClient)

	leaderElectionConfig, err := makeLeaderElectionConfig(h.kubeClient, recorder)
	if err != nil {
		log.Printf("[election] leaderElectionConfig error: %v", err)
	}

	leaderElectionConfig.Callbacks = leaderelection.LeaderCallbacks{
		OnStartedLeading: func(stop <-chan struct{}) {
			Leader = true
			log.Println("[election] Got leadership.")
			<-stop
		},
		OnStoppedLeading: func() {
			Leader = false
			log.Println("[election] Lost leadership.")
		},
	}
	leaderElector, err := leaderelection.NewLeaderElector(*leaderElectionConfig)
	if err != nil {
		log.Printf("[election] leaderElection error: %v", err)
	}
	go leaderElector.Run()

	return &h
}

func makeLeaderElectionConfig(client *clientset.Clientset, recorder record.EventRecorder) (*leaderelection.LeaderElectionConfig, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %v", err)
	}

	lock, err := resourcelock.New(resourcelock.ConfigMapsResourceLock,
		"default",
		"introspect-config",
		client.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      hostname,
			EventRecorder: recorder,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't create resource lock: %v", err)
	}

	return &leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
	}, nil
}

func createRecorder(kubeClient *clientset.Clientset) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Printf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: typedcorev1.New(kubeClient.CoreV1().RESTClient()).Events("")})
	return eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "introspect-election"})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("index").Parse(`
		<!DOCTYPE html>
		<html>
			<head>
				<link rel="stylesheet" href="css/bootstrap.css">
				<style>
				{{if eq .Version "v1.0" }}
				body { background-color: #F0FFF0; }
				{{end}}
				{{if eq .Version "v2.0" }}
				body { background-color: #F0F0FF; }
				{{end}}
				</style>
			</head>
			<div class="container">
				<body>
				<h1>Introspection-{{.Version}}</h1>
				{{if eq .Leader true }}
				I am an active Leader
				{{else}}
				I am on standby
				{{end}}
				</body>
			</div>
			</html>
  `)
	if err != nil {
		log.Println("[election] parse template:", err)
		fmt.Fprint(w, "[election] parse template: ", err)
		return
	}

	type EnvData struct {
		Version string
		Leader  bool
	}
	data := EnvData{version.Version, Leader}

	err = t.Execute(w, data)
	if err != nil {
		log.Println("[election] executing template:", err)
		fmt.Fprint(w, "[election] executing template: ", err)
	}

}

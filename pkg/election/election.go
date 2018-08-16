package election

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vasu1124/introspect/pkg/version"

	corev1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

//Leader ... am I a Leader?
//TODO: secure with mutex
var Leader = false

//Fail ... did the electionprocess fail?
//TODO: secure with mutex
var Fail = true

//Handler .
type Handler struct {
	leaderElector *leaderelection.LeaderElector
}

// New .
func New() *Handler {
	var h Handler
	var err error

	// Create the client config. Use masterURL and kubeconfig if given, otherwise assume in-cluster.
	config, err := clientcmd.BuildConfigFromFlags(*version.MasterURL, *version.Kubeconfig)
	if err != nil {
		log.Printf("[election] KubeConfig error: %v", err)
		return &h
	}

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		log.Printf("[election] ClientSet error: %v", err)
		return &h
	}

	// v, err := h.kubeClient.ServerVersion()
	// if v.Major < 1 && v.Minor < 7 {
	// 	log.Fatal("Version is not enough")
	// }

	// Set up leader election if enabled and prepare event recorder.
	recorder := createRecorder(kubeClient)

	leaderElectionConfig, err := makeLeaderElectionConfig(kubeClient, recorder)
	if err != nil {
		log.Printf("[election] leaderElectionConfig error: %v", err)
	}

	leaderElectionConfig.Callbacks = leaderelection.LeaderCallbacks{
		OnStartedLeading: func(stop <-chan struct{}) {
			Fail = false
			Leader = true
			log.Println("[election] Got leadership.")
			<-stop
		},
		OnStoppedLeading: func() {
			Fail = false
			Leader = false
			log.Println("[election] Lost leadership.")
		},
	}
	h.leaderElector, err = leaderelection.NewLeaderElector(*leaderElectionConfig)
	if err != nil {
		log.Printf("[election] leaderElection error: %v", err)
	}
	go h.leaderElector.Run()

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
				{{if eq .Fail true }}
					No election could be negotiated
				{{else}}
					{{if eq .Leader true }}
						I am an <b>active</b> Leader<br>
					{{else}}
						I am on <b>standby</b><br>
					{{end}}
					<br>
					Leader is {{.LeaderElection.GetLeader}}
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
		Version        string
		Leader         bool
		Fail           bool
		LeaderElection *leaderelection.LeaderElector
	}

	data := EnvData{version.Version, Leader, Fail, h.leaderElector}

	err = t.Execute(w, data)
	if err != nil {
		log.Println("[election] executing template:", err)
		fmt.Fprint(w, "[election] executing template: ", err)
	}

}

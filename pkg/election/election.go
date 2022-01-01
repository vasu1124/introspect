package election

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
	rc, err := config.GetConfig()
	if err != nil {
		logger.Log.Error(err, "[election] KubeConfig error")
		return &h
	}
	kubeClient, err := clientset.NewForConfig(rc)
	if err != nil {
		logger.Log.Error(err, "[election] ClientSet error")
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
		logger.Log.Error(err, "[election] leaderElectionConfig error")
	}

	leaderElectionConfig.Callbacks = leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			Fail = false
			Leader = true
			logger.Log.Info("[election] Got leadership")
			<-ctx.Done()
		},
		OnStoppedLeading: func() {
			Fail = false
			Leader = false
			logger.Log.Info("[election] Lost leadership")
		},
		OnNewLeader: func(identity string) {
			Fail = false
			logger.Log.Info("[election] Got informed. Leadership is with", "leadership", identity)
		},
	}
	h.leaderElector, err = leaderelection.NewLeaderElector(*leaderElectionConfig)
	if err != nil {
		logger.Log.Error(err, "[election] leaderElection error")
	}
	go h.leaderElector.Run(context.TODO())

	return &h
}

func makeLeaderElectionConfig(client *clientset.Clientset, recorder record.EventRecorder) (*leaderelection.LeaderElectionConfig, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %v", err)
	}

	namespace, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		namespace = "default"
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "election.introspect.actvirtual.com",
			Namespace: namespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: hostname,
		},
	}

	return &leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
	}, nil
}

func createRecorder(kubeClient *clientset.Clientset) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logger.Printf)
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
				{{if .Flag }}
				body { background-color: #F0FFF0; }
				{{else}}
				body { background-color: #F0F0FF; }
				{{end}}
				</style>
			</head>
			<div class="container">
				<body>
				<h1>Introspection-{{.Version}}</h1>
				I am an <b>{{.Hostname}}</b>.<br>
				{{if eq .Fail true }}
					No election could be negotiated<br>
				{{else}}
					{{if .Leader }}
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
		logger.Log.Error(err, "[election] error parsing template")
		fmt.Fprint(w, "[election] error parsing template: ", err)
		return
	}

	type EnvData struct {
		Version        string
		Flag           bool
		Leader         bool
		Fail           bool
		LeaderElection *leaderelection.LeaderElector
		Hostname       string
	}

	hostname, err := os.Hostname()
	if err != nil {
		logger.Log.Error(err, "[election] Unable to get hostname")
		fmt.Fprint(w, "[election] unable to get hostname: ", err)
	}
	data := EnvData{version.Get().GitVersion, version.GetPatchVersion()%2 == 0, Leader, Fail, h.leaderElector, hostname}

	err = t.Execute(w, data)
	if err != nil {
		logger.Log.Error(err, "[election] error executing template")
		fmt.Fprint(w, "[election] error executing template: ", err)
	}

}

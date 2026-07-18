package election

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
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

// ElectionState holds the leader election state with mutex protection.
type ElectionState struct {
	mu     sync.RWMutex
	leader bool
	fail   bool
}

// Leader returns whether this instance is the leader.
func (s *ElectionState) Leader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.leader
}

// SetLeader sets the leader state.
func (s *ElectionState) SetLeader(leader bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leader = leader
}

// Fail returns whether the election process failed.
func (s *ElectionState) Fail() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fail
}

// SetFail sets the fail state.
func (s *ElectionState) SetFail(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fail = fail
}

// NewElectionState creates a new ElectionState with default values.
func NewElectionState() *ElectionState {
	return &ElectionState{
		leader: false,
		fail:   true,
	}
}

// Handler .
type Handler struct {
	leaderElector *leaderelection.LeaderElector
	state         *ElectionState
}

// New .
func New() *Handler {
	var h Handler
	var err error

	h.state = NewElectionState()

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

	// Set up leader election if enabled and prepare event recorder.
	recorder := createRecorder(kubeClient)

	leaderElectionConfig, err := makeLeaderElectionConfig(kubeClient, recorder)
	if err != nil {
		logger.Log.Error(err, "[election] leaderElectionConfig error")
	}

	leaderElectionConfig.Callbacks = leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			h.state.SetFail(false)
			h.state.SetLeader(true)
			logger.Log.Info("[election] Got leadership")
			<-ctx.Done()
		},
		OnStoppedLeading: func() {
			h.state.SetFail(false)
			h.state.SetLeader(false)
			logger.Log.Info("[election] Lost leadership")
		},
		OnNewLeader: func(identity string) {
			h.state.SetFail(false)
			logger.Log.Info("[election] Got informed. Leadership is with", "leadership", identity)
		},
	}
	h.leaderElector, err = leaderelection.NewLeaderElector(*leaderElectionConfig)
	if err != nil {
		logger.Log.Error(err, "[election] leaderElection error")
	}
	go h.leaderElector.Run(context.Background())

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
	eventBroadcaster.StartLogging(logger.Log.Info)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: typedcorev1.New(kubeClient.CoreV1().RESTClient()).Events("")})
	return eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "introspect-election"})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("tmpl/layout.html", "tmpl/leadership.html")
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
	data := EnvData{version.Get().GitVersion, version.GetPatchVersion()%2 == 0, h.state.Leader(), h.state.Fail(), h.leaderElector, hostname}

	err = t.Execute(w, data)
	if err != nil {
		logger.Log.Error(err, "[election] error executing template")
		fmt.Fprint(w, "[election] error executing template: ", err)
	}

}

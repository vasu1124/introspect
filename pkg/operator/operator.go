package operator

import (
	"context"
	"net/http"
	"os"

	"github.com/olahol/melody"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/operator/useless/api/v1alpha1"
	"github.com/vasu1124/introspect/pkg/operator/useless/controllers"
	"github.com/vasu1124/introspect/pkg/operator/websocket"
	"github.com/vasu1124/introspect/pkg/version"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	controllerRuntime "sigs.k8s.io/controller-runtime"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

// Handler implements handler.Handler for the operator endpoint.
type Handler struct {
	Melody *melody.Melody
	mgr    controllerRuntime.Manager
}

// New creates a new operator handler.
func New() *Handler {
	h := &Handler{
		Melody: melody.New(),
	}

	namespace, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		namespace = "default"
	}

	controllerRuntime.SetLogger(logger.Log)

	mgr, err := controllerRuntime.NewManager(controllerRuntime.GetConfigOrDie(), controllerRuntime.Options{
		Scheme:                  scheme,
		LeaderElection:          true,
		LeaderElectionNamespace: namespace,
		LeaderElectionID:        "useless.introspect.actvirtual.com",
		Logger:                  controllerRuntime.Log.WithName("operator"),
	})
	if err != nil {
		logger.Log.Error(err, "unable to start UselessMachine manager", "controller", "UselessMachine")
		return h
	}

	if err = (&controllers.UselessMachineReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Notifier: websocket.NewNotifier(h.Melody, mgr.GetClient()),
	}).SetupWithManager(mgr); err != nil {
		logger.Log.Error(err, "[operator] unable to create controller", "controller", "UselessMachine")
		return h
	}

	h.mgr = mgr
	return h
}

// Name implements handler.Handler.
func (h *Handler) Name() string {
	return "operator"
}

// RegisterRoutes implements handler.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	mux.HandleFunc("/operator", h.ServeHTTP)
	mux.HandleFunc("/operatorws", func(w http.ResponseWriter, r *http.Request) {
		h.Melody.HandleRequest(w, r)
	})
	logger.Log.Info("[operator] registered /operator and /operatorws")

	// Start controller manager with context
	go func() {
		logger.Log.Info("[operator] starting manager")
		if err := h.mgr.Start(ctx); err != nil {
			logger.Log.Error(err, "[operator] problem running manager", "controller", "UselessMachine")
		}
	}()
}

// ServeHTTP handles the operator UI request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := struct {
		assets.CommonData
	}{
		CommonData: assets.CommonData{Version: version.Version, Flag: version.Flag},
	}

	if err := assets.ExecuteTemplate(w, "operator.html", data); err != nil {
		logger.Log.Error(err, "[operator] executing template")
	}
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)

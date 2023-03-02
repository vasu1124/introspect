package operator

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/vasu1124/introspect/pkg/logger"
	uselessmachinev1alpha1 "github.com/vasu1124/introspect/pkg/operator/useless/api/v1alpha1"
	"github.com/vasu1124/introspect/pkg/operator/websocket"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/vasu1124/introspect/pkg/operator/useless/controllers"
	"github.com/vasu1124/introspect/pkg/version"
	melody "gopkg.in/olahol/melody.v1"
	"k8s.io/apimachinery/pkg/runtime"
	controller_runtime "sigs.k8s.io/controller-runtime"
)

// Handler .
type Handler struct {
	Melody *melody.Melody
}

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(uselessmachinev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// New .
func New() *Handler {
	var h Handler

	h.Melody = melody.New()

	namespace, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		namespace = "default"
	}

	controller_runtime.SetLogger(logger.Log)

	mgr, err := controller_runtime.NewManager(controller_runtime.GetConfigOrDie(), controller_runtime.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      "0", //turn off,
		LeaderElection:          true,
		LeaderElectionNamespace: namespace,
		LeaderElectionID:        "useless.introspect.actvirtual.com",
		Logger:                  controller_runtime.Log.WithName("operator"),
	})
	if err != nil {
		logger.Log.Error(err, "unable to start UselessMachine manager", "controller", "UselessMachine")
		return nil
	}

	if err = (&controllers.UselessMachineReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Notifier: websocket.NewNotifier(h.Melody, mgr.GetClient()),
	}).SetupWithManager(mgr); err != nil {
		logger.Log.Error(err, "[operator] unable to create controller", "controller", "UselessMachine")
		return nil
	}
	// +kubebuilder:scaffold:builder

	go func() {
		logger.Log.Info("[operator] starting manager")
		if err := mgr.Start(controller_runtime.SetupSignalHandler()); err != nil {
			logger.Log.Error(err, "[operator] problem running manager", "controller", "UselessMachine")
		}
	}()

	return &h
}

// ServeHTTP .
// This is an UI to the state of the Operator
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	type EnvData struct {
		Flag bool
	}

	data := EnvData{version.GetPatchVersion()%2 == 0}

	homeTemplate, err := template.ParseFiles("tmpl/operator.html")
	if err != nil {
		fmt.Fprint(w, "[operator] parsing template: ", err)
		logger.Log.Error(err, "[operator] parsing template")
		return
	}

	err = homeTemplate.Execute(w, data)
	if err != nil {
		fmt.Fprint(w, "[operator] executing template: ", err)
		logger.Log.Error(err, "[operator] executing template")
	}

}

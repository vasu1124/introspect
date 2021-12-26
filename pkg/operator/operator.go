package operator

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/vasu1124/introspect/pkg/logger"
	uselessmachinev1alpha1 "github.com/vasu1124/introspect/pkg/operator/useless/api/v1alpha1"
	"github.com/vasu1124/introspect/pkg/operator/websocket"
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
	log    = controller_runtime.Log.WithName("operator")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = uselessmachinev1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

// New .
func New() *Handler {
	var h Handler

	// TODO: use gin
	// r := gin.Default()
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
	})
	if err != nil {
		log.Error(err, "unable to start manager", "operator", "UselessMachine")
		return nil
	}

	if err = (&controllers.UselessMachineReconciler{
		Client:   mgr.GetClient(),
		Log:      controller_runtime.Log.WithName("controller").WithName("UselessMachine"),
		Scheme:   mgr.GetScheme(),
		Notifier: websocket.NewNotifier(h.Melody, mgr.GetClient()),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "operator", "UselessMachine")
		return nil
	}
	// +kubebuilder:scaffold:builder

	go func() {
		log.Info("starting manager")
		if err := mgr.Start(controller_runtime.SetupSignalHandler()); err != nil {
			log.Error(err, "problem running manager", "operator", "UselessMachine")
		}
	}()

	return &h
}

//ServeHTTP .
//This is an UI to the state of the Operator
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	type EnvData struct {
		version string
	}

	data := EnvData{version.Version}

	homeTemplate, err := template.ParseFiles("tmpl/operator.html")
	if err != nil {
		fmt.Fprint(w, "[operator] parsing template: ", err)
		log.Error(err, "[operator] parsing template")
		return
	}

	err = homeTemplate.Execute(w, data)
	if err != nil {
		fmt.Fprint(w, "[operator] executing template: ", err)
		log.Error(err, "[operator] executing template")
	}

}

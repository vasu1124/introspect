package operator

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/vasu1124/introspect/pkg/apis"
	introspect_v1alpha1 "github.com/vasu1124/introspect/pkg/apis/introspect/v1alpha1"
	"github.com/vasu1124/introspect/pkg/controller/useless"
	websocket_controller "github.com/vasu1124/introspect/pkg/controller/websocket"
	"github.com/vasu1124/introspect/pkg/operator/websocket"
	melody "gopkg.in/olahol/melody.v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/vasu1124/introspect/pkg/version"
)

// Handler .
type Handler struct {
	Melody *melody.Melody
}

// New .
func New() *Handler {
	var h Handler

	// TODO: use gin
	// r := gin.Default()
	h.Melody = melody.New()
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		log.Fatalf("[operator] could not create manager: %v", err)
	}

	// Setup Scheme for all resources
	log.Println("[operator] setting up scheme")
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatalf("[operator] unable add APIs to scheme :%v", err)
	}

	log.Println("[operator] registering components.")
	_, err = builder.
		ControllerManagedBy(mgr).
		For(&introspect_v1alpha1.UselessMachine{}).
		Build(&useless.ReconcileUselessMachine{})
	if err != nil {
		log.Fatalf("[operator] could not create useless controller: %v", err)
	}

	n := websocket.NewNotifier(h.Melody, mgr.GetClient())
	_, err = builder.
		ControllerManagedBy(mgr).
		For(&introspect_v1alpha1.UselessMachine{}).
		Build(websocket_controller.NewReconcileUselessMachine(n))
	if err != nil {
		log.Fatalf("[operator] could not create webhook controller: %v", err)
	}

	go func() {
		if err := mgr.Start(nil); err != nil {
			log.Fatalf("[operator] could not create start manager: %v", err)
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
		log.Println("[operator] parsing template: ", err)
		return
	}

	err = homeTemplate.Execute(w, data)
	if err != nil {
		fmt.Fprint(w, "[operator] executing template: ", err)
		log.Println("[operator] executing template: ", err)
	}

}

package operator

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/vasu1124/introspect/pkg/version"
)

// Handler .
type Handler struct{}

// New .
func New() *Handler {
	var h Handler
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

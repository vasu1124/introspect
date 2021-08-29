package validate

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"

	"github.com/vasu1124/introspect/pkg/version"
	admission "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Handler .
type Handler struct {
	AdmissionReviews map[types.UID]*admission.AdmissionReview
	Regexp           string
}

// New .
func New() *Handler {
	var h Handler
	h.AdmissionReviews = map[types.UID]*admission.AdmissionReview{}
	h.Regexp = ".*"

	return &h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.UserAgent() != "kube-apiserver-admission" {
		h.userUI(w, r)
	} else {
		h.validate(w, r)
	}
}

func (h *Handler) userUI(w http.ResponseWriter, r *http.Request) {
	log.Println("[validate] rendering ui")

	log.Printf("[validate] Regexp is %s", h.Regexp)
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			log.Print("[validate] parsing form", err)
		}
		if r.Form["Regexp"] != nil {
			h.Regexp = r.Form["Regexp"][0]
			log.Printf("[validate] setting Regexp to %s", h.Regexp)
		}
	}

	t, err := template.ParseFiles("tmpl/validate.html")
	if err != nil {
		log.Println("[validate] parse template:", err)
		fmt.Fprint(w, "[validate] parse template: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type EnvData struct {
		Version string
		Handler *Handler
	}

	data := EnvData{version.Version, h}

	err = t.Execute(w, data)
	if err != nil {
		log.Println("[validate] executing template:", err)
		fmt.Fprint(w, "[validate] executing template: ", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	log.Println("[validate] rendering webhook")

	ar := new(admission.AdmissionReview)
	err := json.NewDecoder(r.Body).Decode(ar)
	if err != nil {
		handleError(w, nil, err)
		return
	}

	response := &admission.AdmissionResponse{
		Allowed: true,
		UID:     ar.Request.UID,
	}
	pod := &corev1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pod); err != nil {
		handleError(w, nil, err)
		return
	}

	re := regexp.MustCompile(h.Regexp)

	for _, c := range pod.Spec.Containers {
		if !re.MatchString(c.Image) {
			response.Allowed = false
			break
		}
	}

	responseAR := &admission.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: response,
	}
	ar.Response = response
	h.AdmissionReviews[ar.Request.UID] = ar
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseAR)
}

func handleError(w http.ResponseWriter, ar *admission.AdmissionReview, err error) {
	w.WriteHeader(http.StatusOK)
	if err != nil {
		log.Println("[validate] webhook error ", err.Error())
	}

	response := &admission.AdmissionResponse{
		Allowed: false,
	}
	if ar != nil {
		response.UID = ar.Request.UID
	}

	ar.Response = response
	json.NewEncoder(w).Encode(ar)
}

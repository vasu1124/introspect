package validate

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/vasu1124/introspect/pkg/version"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Handler .
type Handler struct {
	AdmissionReviews  map[types.UID]v1beta1.AdmissionReview
	Pods              map[types.UID]v1.Pod
	AdmissionResponse map[string]bool
}

// New .
func New() *Handler {
	var h Handler
	h.AdmissionReviews = map[types.UID]v1beta1.AdmissionReview{}
	h.Pods = map[types.UID]v1.Pod{}
	h.AdmissionResponse = map[string]bool{}
	return &h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/validate?ui" {
		h.userUI(w, r)
	} else {
		h.validate(w, r)
	}
}

func (h *Handler) userUI(w http.ResponseWriter, r *http.Request) {
	log.Println("[validate] rendering ui")

	err := r.ParseForm()
	if err != nil {
		log.Print("[validate] parsing form", err)
	}

	if r.Method == "POST" {
		//set all to false
		for uid := range h.AdmissionResponse {
			h.AdmissionResponse[uid] = false
		}

		//set only checked to true
		for _, uid := range r.Form["AdmissionResponse"] {
			if _, ok := h.AdmissionResponse[uid]; ok {
				h.AdmissionResponse[uid] = true
			}
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
		Version           string
		AdmissionReviews  map[types.UID]v1beta1.AdmissionReview
		Pods              map[types.UID]v1.Pod
		AdmissionResponse map[string]bool
	}

	data := EnvData{version.Version, h.AdmissionReviews, h.Pods, h.AdmissionResponse}

	err = t.Execute(w, data)
	if err != nil {
		log.Println("[validate] executing template:", err)
		fmt.Fprint(w, "[validate] executing template: ", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	//w.WriteHeader(http.StatusOK)
}

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	log.Println("[validate] rendering webhook")

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println(string(data))

	ar := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(data, &ar); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	uid := ar.Request.UID
	h.AdmissionReviews[uid] = ar

	pod := v1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.Pods[uid] = pod

	allowed := true
	message := ""
	for _, c := range pod.Spec.Containers {
		if _, ok := h.AdmissionResponse[c.Image]; !ok {
			h.AdmissionResponse[c.Image] = false
			allowed = false
			break
		} else if !h.AdmissionResponse[c.Image] {
			message = fmt.Sprintf("[instrospect] human operator denied Container: %s", c.Image)
			allowed = false
			break
		}
	}

	admissionResponse := v1beta1.AdmissionResponse{UID: uid, Allowed: allowed}
	if !allowed {
		admissionResponse.Result = &metav1.Status{
			Reason: metav1.StatusReasonUnauthorized,
			Details: &metav1.StatusDetails{
				Causes: []metav1.StatusCause{
					{Message: message},
				},
			},
		}
	}

	ar = v1beta1.AdmissionReview{
		Response: &admissionResponse,
	}

	data, err = json.Marshal(ar)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

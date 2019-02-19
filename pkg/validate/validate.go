package validate

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/vasu1124/introspect/pkg/version"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Handler .
type Handler struct {
	AdmissionReviews  map[types.UID][]v1.Container
	ContainerResponse map[string]bool
	ContainerLabels   map[string]map[string]string
}

// New .
func New() *Handler {
	var h Handler
	h.AdmissionReviews = map[types.UID][]v1.Container{}
	h.ContainerResponse = map[string]bool{}
	h.ContainerLabels = map[string]map[string]string{}

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

	err := r.ParseForm()
	if err != nil {
		log.Print("[validate] parsing form", err)
	}

	if r.Method == "POST" {
		//set all to false
		for uid := range h.ContainerResponse {
			h.ContainerResponse[uid] = false
		}

		//set only checked to true
		for _, uid := range r.Form["ContainerResponse"] {
			if _, ok := h.ContainerResponse[uid]; ok {
				h.ContainerResponse[uid] = true
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
		AdmissionReviews  map[types.UID][]v1.Container
		ContainerResponse map[string]bool
		ContainerLabels   map[string]map[string]string
	}

	data := EnvData{version.Version, h.AdmissionReviews, h.ContainerResponse, h.ContainerLabels}

	err = t.Execute(w, data)
	if err != nil {
		log.Println("[validate] executing template:", err)
		fmt.Fprint(w, "[validate] executing template: ", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
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

	pod := v1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.AdmissionReviews[uid] = pod.Spec.Containers

	allowed := true
	message := ""
	for _, c := range pod.Spec.Containers {
		cref := c.Image
		ref, err := name.ParseReference(cref, name.WeakValidation)
		if err != nil {
			log.Printf("[validate] parsing reference %q: %v", cref, err)
		}

		cname := ref.Name()

		img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			log.Printf("[validate] reading image %q: %v", ref, err)
		}

		cf, err := partial.ConfigFile(img)
		if err != nil {
			log.Printf("[validate] parsing image config %v: %v", cf, err)
		}

		h.ContainerLabels[cname] = cf.Config.Labels
		log.Printf("[validate] image labels: %v\n", cf.Config.Labels)

		if _, ok := h.ContainerResponse[cname]; !ok {
			h.ContainerResponse[cname] = false
			allowed = false
			break
		} else if !h.ContainerResponse[cname] {
			message = fmt.Sprintf("[instrospect] human operator denied Container: %s", cname)
			allowed = false
			break
		}
	}

	ContainerResponse := v1beta1.AdmissionResponse{UID: uid, Allowed: allowed}
	if !allowed {
		ContainerResponse.Result = &metav1.Status{
			Reason: metav1.StatusReasonNotAcceptable,
			Details: &metav1.StatusDetails{
				Causes: []metav1.StatusCause{
					{Message: message},
				},
			},
		}
	}

	ar = v1beta1.AdmissionReview{
		Response: &ContainerResponse,
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

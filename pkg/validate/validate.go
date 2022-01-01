package validate

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"regexp"

	"github.com/vasu1124/introspect/pkg/logger"
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
	logger.Log.Info("[validate] rendering ui")

	logger.Log.Info("[validate] Regexp", "RegExp", h.Regexp)
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			logger.Log.Error(err, "[validate] error parsing form")
		}
		if r.Form["Regexp"] != nil {
			h.Regexp = r.Form["Regexp"][0]
			logger.Log.Info("[validate] setting Regexp", "RegExp", h.Regexp)
		}
	}

	t, err := template.ParseFiles("tmpl/validate.html")
	if err != nil {
		logger.Log.Error(err, "[validate] error parsing template")
		fmt.Fprint(w, "[validate] parse template: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type EnvData struct {
		Version string
		Flag    bool
		Handler *Handler
	}

	data := EnvData{version.Get().GitVersion, version.GetPatchVersion()%2 == 0, h}

	err = t.Execute(w, data)
	if err != nil {
		logger.Log.Error(err, "[validate] error executing template")
		fmt.Fprint(w, "[validate] executing template: ", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("[validate] rendering webhook")

	ar := new(admission.AdmissionReview)
	err := json.NewDecoder(r.Body).Decode(ar)
	if err != nil {
		logger.Log.Error(err, "[validate] error decoding")
		handleError(w, nil, err)
		return
	}

	response := &admission.AdmissionResponse{
		Allowed: true,
		UID:     ar.Request.UID,
	}
	pod := &corev1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pod); err != nil {
		logger.Log.Error(err, "[validate] error unmarshalling")
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
		logger.Log.Error(err, "[validate] error webhook")
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

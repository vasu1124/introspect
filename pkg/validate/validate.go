package validate

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sync"

	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
	admission "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Handler implements server.Handler for the validate endpoint.
type Handler struct {
	mu              sync.RWMutex
	admissionReviews map[types.UID]*admission.AdmissionReview
	regexp          string
}

// New creates a new validate handler.
func New() *Handler {
	return &Handler{
		admissionReviews: make(map[types.UID]*admission.AdmissionReview),
		regexp:           ".*",
	}
}

// Name implements server.Handler.
func (h *Handler) Name() string {
	return "validate"
}

// RegisterRoutes implements server.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	mux.HandleFunc("/validate", h.ServeHTTP)
	logger.Log.Info("[validate] registered /validate")
}

// ServeHTTP handles both the webhook and the UI.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.UserAgent() != "kube-apiserver-admission" {
		h.userUI(w, r)
	} else {
		h.validate(w, r)
	}
}

func (h *Handler) userUI(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("[validate] rendering ui")

	h.mu.RLock()
	regexp := h.regexp
	h.mu.RUnlock()

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			logger.Log.Error(err, "[validate] error parsing form")
		}
		if r.Form["Regexp"] != nil {
			h.mu.Lock()
			h.regexp = r.Form["Regexp"][0]
			h.mu.Unlock()
			logger.Log.Info("[validate] setting Regexp", "RegExp", h.regexp)
		}
	}

	data := struct {
		assets.CommonData
		Regexp string
	}{
		CommonData: assets.CommonData{Version: version.Version, Flag: version.Flag},
		Regexp:     regexp,
	}

	if err := assets.ExecuteTemplate(w, "validate.html", data); err != nil {
		logger.Log.Error(err, "[validate] executing template")
	}
}

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("[validate] rendering webhook")

	ar := new(admission.AdmissionReview)
	err := json.NewDecoder(r.Body).Decode(ar)
	if err != nil {
		logger.Log.Error(err, "[validate] error decoding")
		h.handleError(w, nil, err)
		return
	}

	response := &admission.AdmissionResponse{
		Allowed: true,
		UID:     ar.Request.UID,
	}
	pod := &corev1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pod); err != nil {
		logger.Log.Error(err, "[validate] error unmarshalling")
		h.handleError(w, nil, err)
		return
	}

	re := regexp.MustCompile(h.getRegexp())

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

	h.mu.Lock()
	h.admissionReviews[ar.Request.UID] = ar
	// Keep map bounded - remove oldest if over 128 entries
	if len(h.admissionReviews) > 128 {
		for k := range h.admissionReviews {
			delete(h.admissionReviews, k)
			break
		}
	}
	h.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseAR)
}

func (h *Handler) getRegexp() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.regexp
}

func (h *Handler) handleError(w http.ResponseWriter, ar *admission.AdmissionReview, err error) {
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

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)
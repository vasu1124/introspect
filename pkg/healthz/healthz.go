package healthz

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
)

// Handler implements handler.Handler for the healthz endpoint.
type Handler struct {
	mu      sync.Mutex
	statusl int
	statusr int
}

// New creates a new healthz handler.
func New() *Handler {
	return &Handler{statusl: http.StatusOK, statusr: http.StatusOK}
}

// Name implements handler.Handler.
func (h *Handler) Name() string {
	return "healthz"
}

// RegisterRoutes implements handler.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	mux.HandleFunc("/healthz", h.handleHealthz)
	mux.HandleFunc("/healthzr", h.handleHealthzr)
	mux.HandleFunc("/healthz/ui", h.handleHealthzUI)
	logger.Log.Info("[healthz] registered /healthz, /healthzr, and /healthz/ui")
}

// handleHealthz handles the liveness check endpoint.
// Query params: ?die (sets 500), ?live (sets 200)
func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[healthz] ParseForm error")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if r.Form["die"] != nil {
		h.statusl = http.StatusInternalServerError
	} else if r.Form["live"] != nil {
		h.statusl = http.StatusOK
	}
	logger.Log.Info("[healthz]",
		"status", h.statusl,
	)
	w.WriteHeader(h.statusl)
	w.Write(fmt.Appendf(nil, "Status: %d", h.statusl))
}

// handleHealthzr handles the readiness check endpoint.
// Query params: ?die (sets 503), ?live (sets 200)
func (h *Handler) handleHealthzr(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[healthzr] ParseForm error")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if r.Form["die"] != nil {
		h.statusr = http.StatusServiceUnavailable
	} else if r.Form["live"] != nil {
		h.statusr = http.StatusOK
	}
	logger.Log.Info("[healthzr]",
		"status", h.statusr,
	)
	w.WriteHeader(h.statusr)
	w.Write(fmt.Appendf(nil, "Status: %d", h.statusr))
}

// handleHealthzUI serves the health check UI page.
func (h *Handler) handleHealthzUI(w http.ResponseWriter, r *http.Request) {
	data := struct {
		assets.CommonData
		LivenessStatus  int
		ReadinessStatus int
	}{
		CommonData:      assets.CommonData{Version: version.Version, Flag: version.Flag},
		LivenessStatus:  h.getLivenessStatus(),
		ReadinessStatus: h.getReadinessStatus(),
	}

	if err := assets.ExecuteTemplate(w, "healthz.html", data); err != nil {
		logger.Log.Error(err, "[healthz] executing template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// getLivenessStatus returns the current liveness status (thread-safe).
func (h *Handler) getLivenessStatus() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.statusl
}

// getReadinessStatus returns the current readiness status (thread-safe).
func (h *Handler) getReadinessStatus() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.statusr
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)

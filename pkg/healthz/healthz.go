package healthz

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
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
	mux.Handle("/healthz", h)
	mux.Handle("/healthzr", h)
	logger.Log.Info("[healthz] registered /healthz and /healthzr")
}

// ServeHTTP handles the health check request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[healthz|r] ParseForm error")
	}

	h.mu.Lock()
	if strings.Contains(r.URL.Path, "/healthzr") {
		if r.Form["die"] != nil {
			h.statusr = http.StatusInternalServerError
		} else if r.Form["live"] != nil {
			h.statusr = http.StatusOK
		}
		logger.Log.Info("[healthzr]",
			"status", h.statusr,
		)
		w.WriteHeader(h.statusr)
		w.Write([]byte(fmt.Sprintf("Status: %d", h.statusr)))
	} else {
		if r.Form["die"] != nil {
			h.statusl = http.StatusInternalServerError
		} else if r.Form["live"] != nil {
			h.statusl = http.StatusOK
		}
		logger.Log.Info("[healthz]",
			"status", h.statusl,
		)
		w.WriteHeader(h.statusl)
		w.Write([]byte(fmt.Sprintf("Status: %d", h.statusl)))
	}
	h.mu.Unlock()
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)

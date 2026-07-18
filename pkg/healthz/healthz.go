package healthz

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
)

// Handler implements handler.Handler for the healthz endpoint.
type Handler struct {
	mu     sync.Mutex
	status int
}

// New creates a new healthz handler.
func New() *Handler {
	return &Handler{status: http.StatusOK}
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
	if r.Form["die"] != nil {
		h.status = http.StatusInternalServerError
	} else if r.Form["live"] != nil {
		h.status = http.StatusOK
	}
	status := h.status
	h.mu.Unlock()

	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf("Status: %d", status)))
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)
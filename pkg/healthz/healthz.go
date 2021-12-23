package healthz

import (
	"fmt"
	"net/http"

	"github.com/vasu1124/introspect/pkg/logger"
)

// Handler .
type Handler struct {
	status int
}

// New .
func New() *Handler {
	var h Handler
	h.status = http.StatusOK
	return &h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[healthz|r] ParseForm error")
	}
	if r.Form["die"] != nil {
		h.status = http.StatusInternalServerError
	} else if r.Form["live"] != nil {
		h.status = http.StatusOK
	}

	w.WriteHeader(h.status)
	w.Write([]byte(fmt.Sprintf("Status: %d", h.status)))
}

package healthz

import (
	"log"
	"net/http"
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
		log.Print("[healthz|r] ParseForm error: ", err)
	}
	if r.Form["die"] != nil {
		h.status = http.StatusInternalServerError
	} else if r.Form["live"] != nil {
		h.status = http.StatusOK
	}

	w.WriteHeader(h.status)
}

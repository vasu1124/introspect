package cookie

import (
	"context"
	"net/http"
	"time"

	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
)

// Handler implements server.Handler for the cookie endpoint.
type Handler struct{}

// New creates a new cookie handler.
func New() *Handler {
	return &Handler{}
}

// Name implements server.Handler.
func (h *Handler) Name() string {
	return "cookie"
}

// RegisterRoutes implements server.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	mux.HandleFunc("/cookie", h.ServeHTTP)
	logger.Log.Info("[cookie] registered /cookie")
}

// ServeHTTP handles the cookie request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[cookie] can't parse form")
	}

	if r.Form["cookie"] != nil && r.Form["value"] != nil && r.Form["expiry"] != nil &&
		r.Form["cookie"][0] != "" && r.Form["value"][0] != "" && r.Form["expiry"][0] != "" {

		expsec, err := time.ParseDuration(r.Form["expiry"][0] + "s")
		if err != nil {
			expsec = 60 * time.Second
		}
		cookie := http.Cookie{Name: r.Form["cookie"][0], Value: r.Form["value"][0], Expires: time.Now().Add(expsec)}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/cookie", http.StatusMovedPermanently)
		return
	}

	data := struct {
		assets.CommonData
		Cookie []*http.Cookie
	}{
		CommonData: assets.CommonData{Version: version.Version, Flag: version.Flag},
		Cookie:     r.Cookies(),
	}

	if err := assets.ExecuteTemplate(w, "cookie.html", data); err != nil {
		logger.Log.Error(err, "[cookie] executing template")
	}
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)

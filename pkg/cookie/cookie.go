package cookie

import (
	"html/template"
	"net/http"
	"time"

	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
)

// Handler .
type Handler struct{}

// New .
func New() *Handler {
	var h Handler
	return &h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("tmpl/layout.html", "tmpl/cookie.html")
	if err != nil {
		logger.Log.Error(err, "[cookie] can't parse template")
		return
	}

	err = r.ParseForm()
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
	}

	type EnvData struct {
		Version string
		Flag    bool
		Cookie  []*http.Cookie
	}

	data := EnvData{version.Get().GitVersion, version.GetPatchVersion()%2 == 0, r.Cookies()}

	err = t.Execute(w, data)
	if err != nil {
		logger.Log.Error(err, "[cookie] can't exectute template")
	}

}

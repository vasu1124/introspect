package cookie

import (
	"html/template"
	"log"
	"net/http"
	"time"
)

// Handler .
type Handler struct{}

// New .
func New() *Handler {
	var h Handler
	return &h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("tmpl/cookie.html")
	if err != nil {
		log.Print(err)
		return
	}

	err = r.ParseForm()
	if err != nil {
		log.Print(err)
	}

	if r.Form["cookie"] != nil && r.Form["value"] != nil && r.Form["expiry"] != nil &&
		r.Form["cookie"][0] != "" && r.Form["value"][0] != "" && r.Form["expiry"][0] != "" {

		expsec, err := time.ParseDuration(r.Form["expiry"][0] + "s")
		if err != nil {
			expsec = 60 * time.Second
		}
		cookie := http.Cookie{Name: r.Form["cookie"][0], Value: r.Form["value"][0], Expires: time.Now().Add(expsec)}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	}

	err = t.Execute(w, h)
	if err != nil {
		log.Println("executing template:", err)
	}

}

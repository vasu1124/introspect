package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed ../../tmpl/*
var templateFS embed.FS

//go:embed ../../scss/*
var cssFS embed.FS

// CommonData contains data shared across all templates.
type CommonData struct {
	Version string
	Flag    bool
}

// Common returns the common template data (version and flag).
func Common() CommonData {
	return CommonData{Version: "0.0.0-dev", Flag: false}
}

// InitTemplates is kept for compatibility but does nothing.
// Templates are parsed per-request with layout + page.
func InitTemplates() error {
	return nil
}

// CSSHandler serves embedded CSS files.
func CSSHandler() http.Handler {
	// The embedded FS has files at css/switch.css, etc.
	// We need to serve them at /css/switch.css
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Remove /css/ prefix from path
		path := strings.TrimPrefix(r.URL.Path, "/css/")
		if path == "" {
			http.NotFound(w, r)
			return
		}

		// Look up file in embedded FS at css/{path}
		fullPath := "css/" + path
		data, err := fs.ReadFile(cssFS, fullPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Set content type based on extension
		if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		w.Write(data)
	})
}

// FaviconHandler serves the embedded favicon.
func FaviconHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(templateFS, "tmpl/favicon.ico")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(data)
	}
}

// ExecuteTemplate executes a page template with the layout.
// It parses layout.html + the named page template together to avoid block conflicts.
func ExecuteTemplate(w http.ResponseWriter, pageName string, data any) error {
	// Read layout template
	layoutContent, err := fs.ReadFile(templateFS, "tmpl/layout.html")
	if err != nil {
		return err
	}

	// Read page template
	pageContent, err := fs.ReadFile(templateFS, "tmpl/"+pageName)
	if err != nil {
		return err
	}

	// Parse the layout first under its file name ("layout.html") so that
	// templates that invoke {{template "layout.html" .}} can resolve it.
	tmpl, err := template.New("layout.html").Parse(string(layoutContent))
	if err != nil {
		return err
	}

	_, err = tmpl.New(pageName).Parse(string(pageContent))
	if err != nil {
		return err
	}

	// Execute the page template (which includes layout via {{template "layout.html" .}})
	return tmpl.ExecuteTemplate(w, pageName, data)
}

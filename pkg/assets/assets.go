package assets

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	// TemplateDir is the directory containing HTML templates
	TemplateDir = "tmpl"
	// CSSDir is the directory containing CSS/JS files
	CSSDir = "css"
)

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

// CSSHandler serves CSS/JS files from the filesystem.
func CSSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Remove /css/ prefix from path
		path := strings.TrimPrefix(r.URL.Path, "/css/")
		if path == "" {
			http.NotFound(w, r)
			return
		}

		// Prevent directory traversal
		path = filepath.Clean(path)
		if strings.HasPrefix(path, "..") {
			http.NotFound(w, r)
			return
		}

		fullPath := filepath.Join(CSSDir, path)
		data, err := os.ReadFile(fullPath)
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

// FaviconHandler serves the favicon from the filesystem.
func FaviconHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(filepath.Join(TemplateDir, "favicon.ico"))
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
	layoutPath := filepath.Join(TemplateDir, "layout.html")
	layoutContent, err := os.ReadFile(layoutPath)
	if err != nil {
		return err
	}

	// Read page template
	pagePath := filepath.Join(TemplateDir, pageName)
	pageContent, err := os.ReadFile(pagePath)
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

// StaticFS returns an http.FileSystem for serving static files.
// Deprecated: Use CSSHandler and individual handlers instead.
func StaticFS() http.FileSystem {
	return http.FS(os.DirFS("."))
}
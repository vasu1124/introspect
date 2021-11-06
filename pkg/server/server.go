package server

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vasu1124/introspect/pkg/config"
	"github.com/vasu1124/introspect/pkg/cookie"
	"github.com/vasu1124/introspect/pkg/dynconfig"
	"github.com/vasu1124/introspect/pkg/election"
	"github.com/vasu1124/introspect/pkg/environ"
	"github.com/vasu1124/introspect/pkg/guestbook"
	"github.com/vasu1124/introspect/pkg/healthz"
	"github.com/vasu1124/introspect/pkg/mandelbrot"
	"github.com/vasu1124/introspect/pkg/middleware"
	"github.com/vasu1124/introspect/pkg/operator"
	"github.com/vasu1124/introspect/pkg/validate"
	"github.com/vasu1124/introspect/pkg/version"
)

type Server struct {
	router *mux.Router
}

func NewServer() *Server {
	srv := &Server{
		router: mux.NewRouter(),
	}

	return srv
}

func (s *Server) Run(stop <-chan int) {
	s.registerHandlers()
	s.registerMiddlewares()

	srv := s.startServer()
	srvTLS := s.startServerTLS()

	// wait for Shutdown
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("[server] Initiated graceful shutdown of HTTP(S) server")
	time.Sleep(1 * time.Second)

	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			log.Println("[server] Graceful HTTP server shutdown failed", err)
		}
	}

	if srvTLS != nil {
		if err := srvTLS.Shutdown(ctx); err != nil {
			log.Println("[server] Graceful HTTPS server shutdown failed", err)
		}
	}
}

func (s *Server) startServer() *http.Server {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Default.Port),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      s.router,
	}

	go func() {
		log.Println("[server] Serving HTTP  ", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("[server] HTTP server crashed", err)
		}
	}()

	return srv
}

func (s *Server) startServerTLS() *http.Server {
	if _, err := os.Stat("etc/tls/server.key"); err != nil {
		return nil
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Default.SecurePort),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      s.router,
	}

	go func() {
		log.Println("[server] Serving HTTPS ", srv.Addr)
		if err := srv.ListenAndServeTLS("etc/tls/server.crt", "etc/tls/server.key"); err != http.ErrServerClosed {
			log.Fatal("[server] HTTPS server crashed", err)
		}
	}()

	return srv
}

func (s *Server) registerHandlers() {
	s.router.HandleFunc("/", serveMenu)
	log.Println("[server] registered /")
	s.router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "tmpl/favicon.ico")
	})
	log.Println("[server] registered /favicon.ico")
	s.router.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
	log.Println("[server] registered /css")
	s.router.Handle("/environ", environ.New())
	log.Println("[server] registered /environ")
	s.router.Handle("/mandelbrot", mandelbrot.New())
	log.Println("[server] registered /mandelbrot")
	s.router.Handle("/dynconfig", dynconfig.New())
	log.Println("[server] registered /dynconfig")
	s.router.Handle("/cookie", cookie.New())
	log.Println("[server] registered /cookie")
	s.router.Handle("/metrics", promhttp.Handler())
	log.Println("[server] registered /metrics")
	s.router.Handle("/validate", validate.New())
	log.Println("[server] registered /validate")
	s.router.Handle("/healthz", healthz.New())
	s.router.Handle("/healthzr", healthz.New())
	log.Println("[server] registered /healthz|r")
	s.router.Handle("/guestbook", guestbook.New())
	log.Println("[server] registered /guestbook")
	s.router.Handle("/election", election.New())
	log.Println("[server] registered /election")
	o := operator.New()
	if o != nil {
		s.router.Handle("/operator", o)
		s.router.HandleFunc("/operatorws", func(w http.ResponseWriter, r *http.Request) {
			o.Melody.HandleRequest(w, r)
		})
		log.Println("[server] registered /operator")
	}

}

func (s *Server) registerMiddlewares() {
	s.router.Use(middleware.NewRequestLoggerHandler)
}

func serveMenu(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("tmpl/menu.html")
	if err != nil {
		log.Println("[server] template parse error: ", err)
		return
	}
	err = r.ParseForm()
	if err != nil {
		log.Println("[server] ParseForm error:", err)
	}

	type EnvData struct {
		Version string
	}
	data := EnvData{version.Version}

	err = t.Execute(w, data)
	if err != nil {
		log.Println("[server] executing template:", err)
		fmt.Fprint(w, "[server] executing template: ", err)
	}
}

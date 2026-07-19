package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
)

// Server struct
type Server struct {
	router   *http.ServeMux
	handlers []Handler
	config   *Config
}

// Config holds server configuration.
type Config struct {
	Port         int
	SecurePort   int
	AssetDir     string
	TLSCertFile  string
	TLSKeyFile   string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultConfig returns the default server configuration.
func DefaultConfig() *Config {
	return &Config{
		Port:         8080,
		SecurePort:   8443,
		AssetDir:     "assets",
		TLSCertFile:  "etc/tls/server.crt",
		TLSKeyFile:   "etc/tls/server.key",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// NewServer creates a new Server with the given config.
func NewServer(cfg *Config) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	srv := &Server{
		router:   http.NewServeMux(),
		config:   cfg,
		handlers: []Handler{}, // Handlers will be registered via RegisterHandlers
	}

	return srv
}

// RegisterHandlers registers all feature handlers with the server.
// This is separate from NewServer to avoid import cycles.
func (s *Server) RegisterHandlers(handlers []Handler) {
	s.handlers = handlers
}

// Run starts the Server and blocks until the stop channel is closed.
func (s *Server) Run(stop <-chan int) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.registerHandlers(rootCtx)

	// Apply middlewares to the router
	finalHandler := MiddlewareChain(s.router, DefaultMiddlewares()...)

	srv := s.startServer(rootCtx, finalHandler)
	srvTLS := s.startServerTLS(rootCtx, finalHandler)

	// Wait for shutdown signal
	<-stop
	logger.Log.Info("[server] Initiated graceful shutdown of HTTP(S) server")
	cancel()

	// Give in-flight requests a moment to complete
	time.Sleep(500 * time.Millisecond)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if srv != nil {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error(err, "[server] Graceful HTTP server shutdown failed")
		}
	}
	if srvTLS != nil {
		if err := srvTLS.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error(err, "[server] Graceful HTTPS server shutdown failed")
		}
	}
}

func (s *Server) startServer(ctx context.Context, handler http.Handler) *http.Server {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.Port),
		ReadHeaderTimeout: s.config.ReadTimeout,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
		Handler:           handler,
	}

	go func() {
		logger.Log.Info("[server] Serving HTTP", "HTTP", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error(err, "[server] HTTP server crashed")
		}
	}()

	// Small delay to ensure server is ready
	time.Sleep(100 * time.Millisecond)

	return srv
}

func (s *Server) startServerTLS(ctx context.Context, handler http.Handler) *http.Server {
	if _, err := os.Stat(s.config.TLSKeyFile); err != nil {
		return nil
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.SecurePort),
		ReadHeaderTimeout: s.config.ReadTimeout,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
		Handler:           handler,
	}

	go func() {
		logger.Log.Info("[server] Serving HTTPS", "HTTPS", srv.Addr)
		if err := srv.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile); err != nil && err != http.ErrServerClosed {
			logger.Log.Error(err, "[server] HTTPS server crashed")
		}
	}()

	return srv
}

func (s *Server) registerHandlers(ctx context.Context) {
	// Initialize embedded templates
	if err := assets.InitTemplates(); err != nil {
		logger.Log.Error(err, "[server] failed to initialize templates")
	}

	// Static routes
	s.router.HandleFunc("/", serveMenu)
	s.router.HandleFunc("/favicon.ico", assets.FaviconHandler())
	s.router.Handle("/css/", assets.CSSHandler())
	s.router.Handle("/metrics", promhttp.Handler())

	// Register all feature handlers
	for _, h := range s.handlers {
		h.RegisterRoutes(s.router, ctx)
	}

	// Log all registered routes
	logger.Log.Info("[server] All handlers registered")
}

func serveMenu(w http.ResponseWriter, r *http.Request) {
	data := struct {
		assets.CommonData
	}{
		CommonData: assets.CommonData{Version: version.Version, Flag: version.Flag},
	}

	if err := assets.ExecuteTemplate(w, "menu.html", data); err != nil {
		logger.Log.Error(err, "[server] executing menu template")
		http.Error(w, "Failed to render menu", http.StatusInternalServerError)
	}
}

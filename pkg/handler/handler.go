package handler

import (
	"context"
	"net/http"
)

// Handler is the interface that all feature handlers must implement.
// This allows for consistent route registration and lifecycle management.
type Handler interface {
	// Name returns the unique name of this handler/module.
	Name() string

	// RegisterRoutes registers all HTTP routes for this handler on the given ServeMux.
	// The context is derived from the server's root context and can be used for
	// cancellation/timeouts during startup.
	RegisterRoutes(mux *http.ServeMux, ctx context.Context)
}

// Closer is an optional interface for handlers that need cleanup on shutdown.
type Closer interface {
	Close() error
}

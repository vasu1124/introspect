package server

import (
	"context"
	"net/http"
)

// Handler is the interface that all feature handlers must implement.
// This allows for consistent route registration and lifecycle management.
type Handler interface {
	// Name returns the unique name of this handler/module.
	Name() string

	RegisterRoutes(mux *http.ServeMux, ctx context.Context)
}

// Closer is an optional interface for handlers that need cleanup on shutdown.
type Closer interface {
	Close() error
}


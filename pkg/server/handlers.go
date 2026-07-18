package server

import (
	"github.com/vasu1124/introspect/pkg/cookie"
	"github.com/vasu1124/introspect/pkg/dynconfig"
	"github.com/vasu1124/introspect/pkg/election"
	"github.com/vasu1124/introspect/pkg/environ"
	"github.com/vasu1124/introspect/pkg/guestbook"
	"github.com/vasu1124/introspect/pkg/healthz"
	"github.com/vasu1124/introspect/pkg/mandelbrot"
	"github.com/vasu1124/introspect/pkg/operator"
	"github.com/vasu1124/introspect/pkg/validate"
)

// BuildHandlers returns all feature handlers for the server.
// This is in a separate file to avoid import cycles.
func BuildHandlers() []Handler {
	return []Handler{
		environ.New(),
		cookie.New(),
		dynconfig.New(),
		validate.New(),
		guestbook.New(),
		election.New(),
		mandelbrot.New(),
		healthz.New(),
		operator.New(),
	}
}
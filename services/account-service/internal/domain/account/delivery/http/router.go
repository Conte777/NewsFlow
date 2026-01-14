package http

import (
	"github.com/fasthttp/router"
	"github.com/rs/zerolog"
)

// Router registers account-related HTTP routes
type Router struct {
	handler *HealthHandler
	logger  zerolog.Logger
}

// NewRouter creates a new account router
func NewRouter(handler *HealthHandler, logger zerolog.Logger) *Router {
	return &Router{
		handler: handler,
		logger:  logger,
	}
}

// RegisterRoutes registers account routes on the router
func (r *Router) RegisterRoutes(rt *router.Router) {
	rt.GET("/health", r.handler.Handle)
}

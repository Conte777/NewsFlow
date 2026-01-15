package http

import (
	"github.com/fasthttp/router"
	"github.com/rs/zerolog"
)

// Router registers QR auth HTTP routes
type Router struct {
	handler *QRAuthHandler
	logger  zerolog.Logger
}

// NewRouter creates a new QR auth router
func NewRouter(handler *QRAuthHandler, logger zerolog.Logger) *Router {
	return &Router{
		handler: handler,
		logger:  logger,
	}
}

// RegisterRoutes registers QR auth routes on the router
func (r *Router) RegisterRoutes(rt *router.Router) {
	// QR authentication endpoints
	rt.POST("/api/v1/auth/qr/start", r.handler.StartAuth)
	rt.GET("/api/v1/auth/qr/{session_id}/status", r.handler.GetStatus)
	rt.POST("/api/v1/auth/qr/{session_id}/password", r.handler.SubmitPassword)
	rt.DELETE("/api/v1/auth/qr/{session_id}", r.handler.Cancel)

	r.logger.Info().Msg("QR auth routes registered")
}

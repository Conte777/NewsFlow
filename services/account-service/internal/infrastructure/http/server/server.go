package server

import (
	"context"
	"fmt"

	"github.com/fasthttp/router"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// Server represents fasthttp server
type Server struct {
	server *fasthttp.Server
	Router *router.Router
	addr   string
	logger zerolog.Logger
}

// NewServer creates a new fasthttp server
func NewServer(port string, logger zerolog.Logger) *Server {
	r := router.New()

	srv := &fasthttp.Server{
		Handler:          r.Handler,
		Name:             "account-service",
		ReadTimeout:      5_000_000_000,  // 5 seconds
		WriteTimeout:     10_000_000_000, // 10 seconds
		IdleTimeout:      120_000_000_000, // 120 seconds
	}

	return &Server{
		server: srv,
		Router: r,
		addr:   fmt.Sprintf(":%s", port),
		logger: logger,
	}
}

// RegisterMetrics registers Prometheus metrics endpoint
func (s *Server) RegisterMetrics() {
	// Adapt promhttp.Handler to fasthttp
	prometheusHandler := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
	s.Router.GET("/metrics", prometheusHandler)
}

// Start starts the HTTP server in a separate goroutine
func (s *Server) Start() error {
	s.logger.Info().
		Str("addr", s.addr).
		Msg("Starting HTTP server")

	go func() {
		if err := s.server.ListenAndServe(s.addr); err != nil {
			s.logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down HTTP server")

	if err := s.server.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.logger.Info().Msg("HTTP server stopped gracefully")
	return nil
}

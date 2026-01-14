package httputil

import (
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// Middleware is a function that wraps a handler
type Middleware func(fasthttp.RequestHandler) fasthttp.RequestHandler

// MiddlewareGroup wraps a router group with middleware support
type MiddlewareGroup struct {
	group      *router.Group
	middleware []Middleware
}

// NewMiddlewareGroup creates a new middleware group
func NewMiddlewareGroup(group *router.Group) *MiddlewareGroup {
	return &MiddlewareGroup{
		group:      group,
		middleware: make([]Middleware, 0),
	}
}

// Use adds middleware to the group
func (g *MiddlewareGroup) Use(m ...Middleware) *MiddlewareGroup {
	g.middleware = append(g.middleware, m...)
	return g
}

// Group creates a new sub-group with inherited middleware
func (g *MiddlewareGroup) Group(path string) *MiddlewareGroup {
	subGroup := g.group.Group(path)
	return &MiddlewareGroup{
		group:      subGroup,
		middleware: append([]Middleware{}, g.middleware...),
	}
}

// applyMiddleware applies all middleware to a handler in reverse order
func (g *MiddlewareGroup) applyMiddleware(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
	for i := len(g.middleware) - 1; i >= 0; i-- {
		handler = g.middleware[i](handler)
	}
	return handler
}

// GET registers a GET handler
func (g *MiddlewareGroup) GET(path string, handler fasthttp.RequestHandler) {
	g.group.GET(path, g.applyMiddleware(handler))
}

// POST registers a POST handler
func (g *MiddlewareGroup) POST(path string, handler fasthttp.RequestHandler) {
	g.group.POST(path, g.applyMiddleware(handler))
}

// PUT registers a PUT handler
func (g *MiddlewareGroup) PUT(path string, handler fasthttp.RequestHandler) {
	g.group.PUT(path, g.applyMiddleware(handler))
}

// PATCH registers a PATCH handler
func (g *MiddlewareGroup) PATCH(path string, handler fasthttp.RequestHandler) {
	g.group.PATCH(path, g.applyMiddleware(handler))
}

// DELETE registers a DELETE handler
func (g *MiddlewareGroup) DELETE(path string, handler fasthttp.RequestHandler) {
	g.group.DELETE(path, g.applyMiddleware(handler))
}

// ANY registers a handler for any method
func (g *MiddlewareGroup) ANY(path string, handler fasthttp.RequestHandler) {
	g.group.ANY(path, g.applyMiddleware(handler))
}

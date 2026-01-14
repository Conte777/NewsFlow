package httputil

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// WriteResponse writes a successful JSON response
func WriteResponse(ctx *fasthttp.RequestCtx, data interface{}) {
	WriteResponseWithStatus(ctx, data, fasthttp.StatusOK)
}

// WriteResponseWithStatus writes a successful JSON response with custom status
func WriteResponseWithStatus(ctx *fasthttp.RequestCtx, data interface{}, status int) {
	resp := Response{
		Success: true,
		Data:    data,
	}
	writeJSON(ctx, resp, status)
}

// WriteErrorResponse writes an error JSON response
func WriteErrorResponse(ctx *fasthttp.RequestCtx, message string, status int) {
	resp := Response{
		Success: false,
		Error:   message,
	}
	writeJSON(ctx, resp, status)
}

// WriteError writes an error response with error object
func WriteError(ctx *fasthttp.RequestCtx, err error, status int) {
	message := "internal server error"
	if err != nil {
		message = err.Error()
	}
	WriteErrorResponse(ctx, message, status)
}

// writeJSON writes JSON response to context
func writeJSON(ctx *fasthttp.RequestCtx, data interface{}, status int) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)

	body, err := json.Marshal(data)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte(`{"success":false,"error":"failed to marshal response"}`))
		return
	}

	ctx.SetBody(body)
}

// WriteHealthResponse writes a health check response
func WriteHealthResponse(ctx *fasthttp.RequestCtx, data interface{}, healthy bool) {
	status := fasthttp.StatusOK
	if !healthy {
		status = fasthttp.StatusServiceUnavailable
	}
	writeJSON(ctx, data, status)
}

package http

import (
	"encoding/json"
	"errors"

	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/dto"
	qrerrors "github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/errors"
)

// QRAuthHandler handles QR authentication HTTP requests
type QRAuthHandler struct {
	useCase deps.QRAuthService
	logger  zerolog.Logger
}

// NewQRAuthHandler creates a new QR auth handler
func NewQRAuthHandler(useCase deps.QRAuthService, logger zerolog.Logger) *QRAuthHandler {
	return &QRAuthHandler{
		useCase: useCase,
		logger:  logger.With().Str("handler", "qr_auth").Logger(),
	}
}

// StartAuth handles POST /api/v1/auth/qr/start
func (h *QRAuthHandler) StartAuth(ctx *fasthttp.RequestCtx) {
	var req dto.StartQRAuthRequest
	if len(ctx.PostBody()) > 0 {
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			h.writeError(ctx, fasthttp.StatusBadRequest, "invalid request body")
			return
		}
	}

	session, err := h.useCase.StartAuth(ctx, req.SessionName)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to start QR auth")
		h.handleError(ctx, err)
		return
	}

	h.writeJSON(ctx, fasthttp.StatusOK, dto.StartQRAuthResponse{
		SessionID:    session.ID,
		QRURL:        session.QRURL,
		QRCodeBase64: session.QRCodeBase64,
		Status:       string(session.Status),
		ExpiresAt:    session.ExpiresAt,
	})
}

// GetStatus handles GET /api/v1/auth/qr/{session_id}/status
func (h *QRAuthHandler) GetStatus(ctx *fasthttp.RequestCtx) {
	sessionID, ok := ctx.UserValue("session_id").(string)
	if !ok || sessionID == "" {
		h.writeError(ctx, fasthttp.StatusBadRequest, "session_id is required")
		return
	}

	session, err := h.useCase.GetStatus(ctx, sessionID)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	resp := dto.QRAuthStatusResponse{
		SessionID: session.ID,
		Status:    string(session.Status),
	}
	if session.PhoneNumber != "" {
		resp.PhoneNumber = &session.PhoneNumber
	}
	if session.AccountID != nil {
		resp.AccountID = session.AccountID
	}
	if session.Error != "" {
		resp.Error = &session.Error
	}

	h.writeJSON(ctx, fasthttp.StatusOK, resp)
}

// SubmitPassword handles POST /api/v1/auth/qr/{session_id}/password
func (h *QRAuthHandler) SubmitPassword(ctx *fasthttp.RequestCtx) {
	sessionID, ok := ctx.UserValue("session_id").(string)
	if !ok || sessionID == "" {
		h.writeError(ctx, fasthttp.StatusBadRequest, "session_id is required")
		return
	}

	var req dto.SubmitPasswordRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		h.writeError(ctx, fasthttp.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" {
		h.writeError(ctx, fasthttp.StatusBadRequest, "password is required")
		return
	}

	session, err := h.useCase.SubmitPassword(ctx, sessionID, req.Password)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	resp := dto.SubmitPasswordResponse{
		SessionID: session.ID,
		Status:    string(session.Status),
	}
	if session.PhoneNumber != "" {
		resp.PhoneNumber = &session.PhoneNumber
	}
	if session.AccountID != nil {
		resp.AccountID = session.AccountID
	}
	if session.Error != "" {
		resp.Error = &session.Error
	}

	h.writeJSON(ctx, fasthttp.StatusOK, resp)
}

// Cancel handles DELETE /api/v1/auth/qr/{session_id}
func (h *QRAuthHandler) Cancel(ctx *fasthttp.RequestCtx) {
	sessionID, ok := ctx.UserValue("session_id").(string)
	if !ok || sessionID == "" {
		h.writeError(ctx, fasthttp.StatusBadRequest, "session_id is required")
		return
	}

	if err := h.useCase.Cancel(ctx, sessionID); err != nil {
		h.handleError(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// handleError maps domain errors to HTTP status codes
func (h *QRAuthHandler) handleError(ctx *fasthttp.RequestCtx, err error) {
	switch {
	case errors.Is(err, qrerrors.ErrSessionNotFound):
		h.writeError(ctx, fasthttp.StatusNotFound, "session not found")
	case errors.Is(err, qrerrors.ErrSessionExpired):
		h.writeError(ctx, fasthttp.StatusGone, "session expired")
	case errors.Is(err, qrerrors.ErrInvalidSessionState):
		h.writeError(ctx, fasthttp.StatusConflict, "invalid session state")
	case errors.Is(err, qrerrors.ErrInvalidPassword):
		h.writeError(ctx, fasthttp.StatusUnauthorized, "invalid password")
	case errors.Is(err, qrerrors.ErrPasswordRequired):
		h.writeError(ctx, fasthttp.StatusBadRequest, "password is required")
	case errors.Is(err, qrerrors.ErrMaxSessionsReached):
		h.writeError(ctx, fasthttp.StatusTooManyRequests, "too many active sessions")
	case errors.Is(err, qrerrors.ErrQRGenerationFailed):
		h.writeError(ctx, fasthttp.StatusInternalServerError, "failed to generate QR code")
	default:
		h.logger.Error().Err(err).Msg("unexpected error")
		h.writeError(ctx, fasthttp.StatusInternalServerError, "internal server error")
	}
}

// writeJSON writes JSON response
func (h *QRAuthHandler) writeJSON(ctx *fasthttp.RequestCtx, status int, data interface{}) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	if err := json.NewEncoder(ctx).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode response")
	}
}

// writeError writes error response
func (h *QRAuthHandler) writeError(ctx *fasthttp.RequestCtx, status int, message string) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	json.NewEncoder(ctx).Encode(dto.ErrorResponse{Error: message})
}

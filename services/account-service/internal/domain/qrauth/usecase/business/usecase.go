package business

import (
	"context"
	"strings"

	"github.com/rs/zerolog"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/entities"
	qrerrors "github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/errors"
)

// QRAuthUseCase implements QR authentication business logic
type QRAuthUseCase struct {
	service deps.QRAuthService
	logger  zerolog.Logger
}

// NewQRAuthUseCase creates a new QR auth use case
func NewQRAuthUseCase(service deps.QRAuthService, logger zerolog.Logger) *QRAuthUseCase {
	return &QRAuthUseCase{
		service: service,
		logger:  logger.With().Str("usecase", "qrauth").Logger(),
	}
}

// StartAuth initiates QR authentication
func (uc *QRAuthUseCase) StartAuth(ctx context.Context, sessionName string) (*entities.QRAuthSession, error) {
	// Sanitize session name
	sessionName = strings.TrimSpace(sessionName)
	if len(sessionName) > 100 {
		sessionName = sessionName[:100]
	}

	uc.logger.Info().Str("session_name", sessionName).Msg("starting QR auth")

	session, err := uc.service.StartAuth(ctx, sessionName)
	if err != nil {
		uc.logger.Error().Err(err).Msg("failed to start QR auth")
		return nil, err
	}

	return session, nil
}

// GetStatus returns current authentication status
func (uc *QRAuthUseCase) GetStatus(ctx context.Context, sessionID string) (*entities.QRAuthSession, error) {
	if sessionID == "" {
		return nil, qrerrors.ErrSessionNotFound
	}

	return uc.service.GetStatus(ctx, sessionID)
}

// SubmitPassword submits 2FA password for authentication
func (uc *QRAuthUseCase) SubmitPassword(ctx context.Context, sessionID, password string) (*entities.QRAuthSession, error) {
	if sessionID == "" {
		return nil, qrerrors.ErrSessionNotFound
	}
	if password == "" {
		return nil, qrerrors.ErrPasswordRequired
	}

	uc.logger.Debug().Str("session_id", sessionID).Msg("submitting 2FA password")

	return uc.service.SubmitPassword(ctx, sessionID, password)
}

// Cancel cancels ongoing authentication
func (uc *QRAuthUseCase) Cancel(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return qrerrors.ErrSessionNotFound
	}

	uc.logger.Info().Str("session_id", sessionID).Msg("cancelling QR auth")

	return uc.service.Cancel(ctx, sessionID)
}

// Ensure QRAuthUseCase implements deps.QRAuthService
var _ deps.QRAuthService = (*QRAuthUseCase)(nil)

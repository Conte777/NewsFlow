package deps

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/entities"
)

// QRAuthService defines QR authentication operations
type QRAuthService interface {
	// StartAuth initiates QR authentication and returns session with QR code
	StartAuth(ctx context.Context, sessionName string) (*entities.QRAuthSession, error)

	// GetStatus returns current authentication status
	GetStatus(ctx context.Context, sessionID string) (*entities.QRAuthSession, error)

	// SubmitPassword submits 2FA password for authentication
	SubmitPassword(ctx context.Context, sessionID, password string) (*entities.QRAuthSession, error)

	// Cancel cancels ongoing authentication
	Cancel(ctx context.Context, sessionID string) error
}

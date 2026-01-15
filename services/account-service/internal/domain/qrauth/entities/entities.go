package entities

import "time"

// QRAuthStatus represents the current state of QR authentication
type QRAuthStatus string

const (
	StatusPending         QRAuthStatus = "pending"          // QR created, waiting for scan
	StatusWaitingPassword QRAuthStatus = "waiting_password" // 2FA password required
	StatusSuccess         QRAuthStatus = "success"          // Authentication successful
	StatusFailed          QRAuthStatus = "failed"           // Authentication failed
	StatusExpired         QRAuthStatus = "expired"          // Session expired
	StatusCancelled       QRAuthStatus = "cancelled"        // User cancelled
)

// QRAuthSession represents an active QR authentication session
type QRAuthSession struct {
	ID           string
	SessionName  string
	Status       QRAuthStatus
	QRURL        string // tg://login?token=... URL
	QRCodeBase64 string // Base64 encoded PNG image
	AccountID    *uint
	PhoneNumber  string
	Error        string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	UpdatedAt    time.Time
}

// IsTerminal returns true if the session is in a terminal state
func (s *QRAuthSession) IsTerminal() bool {
	return s.Status == StatusSuccess ||
		s.Status == StatusFailed ||
		s.Status == StatusExpired ||
		s.Status == StatusCancelled
}

// IsExpired returns true if the session has expired
func (s *QRAuthSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

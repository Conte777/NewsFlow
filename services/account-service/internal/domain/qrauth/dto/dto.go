package dto

import "time"

// StartQRAuthRequest request to start QR authentication
type StartQRAuthRequest struct {
	SessionName string `json:"session_name,omitempty"` // Optional friendly name
}

// StartQRAuthResponse response with QR code data
type StartQRAuthResponse struct {
	SessionID    string    `json:"session_id"`
	QRURL        string    `json:"qr_url"`         // tg://login?token=...
	QRCodeBase64 string    `json:"qr_code_base64"` // PNG image base64
	Status       string    `json:"status"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// QRAuthStatusResponse response for status check
type QRAuthStatusResponse struct {
	SessionID   string  `json:"session_id"`
	Status      string  `json:"status"`
	PhoneNumber *string `json:"phone_number,omitempty"` // Set after success
	AccountID   *uint   `json:"account_id,omitempty"`   // Set after success
	Error       *string `json:"error,omitempty"`        // Set on failure
}

// SubmitPasswordRequest request to submit 2FA password
type SubmitPasswordRequest struct {
	Password string `json:"password"`
}

// SubmitPasswordResponse response after password submission
type SubmitPasswordResponse struct {
	SessionID   string  `json:"session_id"`
	Status      string  `json:"status"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	AccountID   *uint   `json:"account_id,omitempty"`
	Error       *string `json:"error,omitempty"`
}

// ErrorResponse generic error response
type ErrorResponse struct {
	Error string `json:"error"`
}

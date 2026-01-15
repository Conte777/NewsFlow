package telegram

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"rsc.io/qr"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/entities"
	qrerrors "github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/errors"
)

// QRAuthManager implements QR authentication for Telegram accounts
type QRAuthManager struct {
	db             *gorm.DB
	telegramCfg    *config.TelegramConfig
	accountManager domain.AccountManager
	sessionStore   *QRSessionStore
	logger         zerolog.Logger
}

// NewQRAuthManager creates a new QR authentication manager
func NewQRAuthManager(
	db *gorm.DB,
	telegramCfg *config.TelegramConfig,
	accountManager domain.AccountManager,
	sessionStore *QRSessionStore,
	logger zerolog.Logger,
) *QRAuthManager {
	return &QRAuthManager{
		db:             db,
		telegramCfg:    telegramCfg,
		accountManager: accountManager,
		sessionStore:   sessionStore,
		logger:         logger.With().Str("component", "qr_auth_manager").Logger(),
	}
}

// StartAuth initiates QR authentication and returns session with QR code
func (m *QRAuthManager) StartAuth(ctx context.Context, sessionName string) (*entities.QRAuthSession, error) {
	sessionID := uuid.New().String()

	m.logger.Info().
		Str("session_id", sessionID).
		Str("session_name", sessionName).
		Msg("starting QR authentication")

	// Create internal session
	now := time.Now()
	session := &InternalQRSession{
		QRAuthSession: &entities.QRAuthSession{
			ID:          sessionID,
			SessionName: sessionName,
			Status:      entities.StatusPending,
			CreatedAt:   now,
			ExpiresAt:   now.Add(5 * time.Minute), // Telegram QR tokens expire in ~5 minutes
			UpdatedAt:   now,
		},
		PasswordChan: make(chan string, 1),
	}

	// Store session
	if err := m.sessionStore.Store(session); err != nil {
		return nil, err
	}

	// Channel to receive QR code or error
	qrReady := make(chan error, 1)

	// Start authentication goroutine
	go m.runQRAuth(session, qrReady)

	// Wait for QR code to be generated
	select {
	case err := <-qrReady:
		if err != nil {
			m.sessionStore.Delete(sessionID)
			return nil, err
		}
	case <-time.After(30 * time.Second):
		session.SetError(fmt.Errorf("timeout waiting for QR code generation"))
		m.sessionStore.Delete(sessionID)
		return nil, qrerrors.ErrQRGenerationFailed
	case <-ctx.Done():
		session.SetError(ctx.Err())
		m.sessionStore.Delete(sessionID)
		return nil, ctx.Err()
	}

	return session.GetSnapshot(), nil
}

// GetStatus returns current authentication status
func (m *QRAuthManager) GetStatus(ctx context.Context, sessionID string) (*entities.QRAuthSession, error) {
	session, err := m.sessionStore.Load(sessionID)
	if err != nil {
		return nil, err
	}
	return session.GetSnapshot(), nil
}

// SubmitPassword submits 2FA password for authentication
func (m *QRAuthManager) SubmitPassword(ctx context.Context, sessionID, password string) (*entities.QRAuthSession, error) {
	session, err := m.sessionStore.Load(sessionID)
	if err != nil {
		return nil, err
	}

	snapshot := session.GetSnapshot()
	if snapshot.Status != entities.StatusWaitingPassword {
		return nil, qrerrors.ErrInvalidSessionState
	}

	// Send password to the authentication goroutine
	select {
	case session.PasswordChan <- password:
		m.logger.Debug().Str("session_id", sessionID).Msg("password submitted")
	default:
		return nil, qrerrors.ErrInvalidSessionState
	}

	// Wait a bit for status update
	time.Sleep(500 * time.Millisecond)

	return session.GetSnapshot(), nil
}

// Cancel cancels ongoing authentication
func (m *QRAuthManager) Cancel(ctx context.Context, sessionID string) error {
	session, err := m.sessionStore.Load(sessionID)
	if err != nil {
		return err
	}

	session.UpdateStatus(entities.StatusCancelled)

	if session.CancelFunc != nil {
		session.CancelFunc()
	}

	m.sessionStore.Delete(sessionID)
	m.logger.Info().Str("session_id", sessionID).Msg("QR auth cancelled")

	return nil
}

// runQRAuth runs the QR authentication process in a goroutine
func (m *QRAuthManager) runQRAuth(session *InternalQRSession, qrReady chan<- error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	session.CancelFunc = cancel
	defer cancel()

	// Create temporary session storage for this auth attempt
	tempStorage := NewMemorySessionStorage()

	// Create Telegram client
	client := telegram.NewClient(
		m.telegramCfg.APIID,
		m.telegramCfg.APIHash,
		telegram.Options{
			SessionStorage: tempStorage,
		},
	)
	session.Client = client

	err := client.Run(ctx, func(ctx context.Context) error {
		api := client.API()

		// Create QR login handler
		qrLogin := qrlogin.NewQR(api, m.telegramCfg.APIID, m.telegramCfg.APIHash, qrlogin.Options{})

		// Export initial login token
		token, err := qrLogin.Export(ctx)
		if err != nil {
			qrReady <- fmt.Errorf("export token: %w", err)
			return err
		}

		// Generate QR code image
		qrCode, err := qr.Encode(token.URL(), qr.L)
		if err != nil {
			qrReady <- fmt.Errorf("encode QR: %w", err)
			return err
		}

		// Update session with QR data
		session.SetQRCode(token.URL(), base64.StdEncoding.EncodeToString(qrCode.PNG()))

		m.logger.Info().
			Str("session_id", session.ID).
			Str("qr_url", token.URL()).
			Msg("QR code generated")

		// Signal that QR is ready
		qrReady <- nil

		// Loop to handle Accept retries and token refresh
		for {
			// Wait for user to scan and accept
			m.logger.Debug().Str("session_id", session.ID).Msg("Waiting for QR scan...")
			authorization, err := qrLogin.Accept(ctx, token)
			if err != nil {
				m.logger.Debug().Err(err).Str("session_id", session.ID).Msg("Accept returned error")

				// Check if token expired - need to regenerate
				if tgerr.Is(err, "AUTH_TOKEN_EXPIRED") {
					m.logger.Info().Str("session_id", session.ID).Msg("QR token expired, regenerating...")

					// Export new token
					token, err = qrLogin.Export(ctx)
					if err != nil {
						session.SetError(err)
						return fmt.Errorf("export token: %w", err)
					}

					// Generate new QR code
					qrCode, err = qr.Encode(token.URL(), qr.L)
					if err != nil {
						session.SetError(err)
						return fmt.Errorf("encode QR: %w", err)
					}

					// Update session with new QR data
					session.SetQRCode(token.URL(), base64.StdEncoding.EncodeToString(qrCode.PNG()))
					m.logger.Info().
						Str("session_id", session.ID).
						Str("qr_url", token.URL()).
						Msg("New QR code generated")
					continue
				}

				// AUTH_KEY_UNREGISTERED - retry Accept with same token after delay
				if tgerr.Is(err, "AUTH_KEY_UNREGISTERED") {
					m.logger.Debug().Str("session_id", session.ID).Msg("Auth key not registered yet, retrying Accept...")
					time.Sleep(1 * time.Second)
					continue
				}

				// AUTH_TOKEN_ALREADY_ACCEPTED means user scanned and accepted - we're logged in!
				if tgerr.Is(err, "AUTH_TOKEN_ALREADY_ACCEPTED") {
					m.logger.Info().Str("session_id", session.ID).Msg("Token already accepted, getting user info...")
					self, err := client.Self(ctx)
					if err != nil {
						session.SetError(err)
						return fmt.Errorf("get self after accept: %w", err)
					}
					return m.finalizeAuth(ctx, session, self, tempStorage)
				}

				// Check if 2FA is required
				if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
					session.UpdateStatus(entities.StatusWaitingPassword)
					m.logger.Info().Str("session_id", session.ID).Msg("2FA password required")

					// Wait for password
					select {
					case password := <-session.PasswordChan:
						m.logger.Debug().Str("session_id", session.ID).Msg("received 2FA password")

						// Authenticate with password
						_, err = client.Auth().Password(ctx, password)
						if err != nil {
							if tgerr.Is(err, "PASSWORD_HASH_INVALID") {
								session.SetError(qrerrors.ErrInvalidPassword)
								return qrerrors.ErrInvalidPassword
							}
							session.SetError(err)
							return fmt.Errorf("2FA auth failed: %w", err)
						}

						// Get user info after password auth
						self, err := client.Self(ctx)
						if err != nil {
							session.SetError(err)
							return fmt.Errorf("get self failed: %w", err)
						}

						return m.finalizeAuth(ctx, session, self, tempStorage)

					case <-ctx.Done():
						session.SetError(ctx.Err())
						return ctx.Err()
					}
				}

				session.SetError(err)
				return fmt.Errorf("accept failed: %w", err)
			}

			// QR login succeeded, get user info
			_ = authorization // Authorization contains user ID but we need full user info

			self, err := client.Self(ctx)
			if err != nil {
				session.SetError(err)
				return fmt.Errorf("get self failed: %w", err)
			}

			return m.finalizeAuth(ctx, session, self, tempStorage)
		}
	})

	if err != nil {
		snapshot := session.GetSnapshot()
		if snapshot.Status != entities.StatusSuccess && snapshot.Status != entities.StatusCancelled {
			session.SetError(err)
		}
		m.logger.Error().Err(err).Str("session_id", session.ID).Msg("QR auth failed")
	}
}

// finalizeAuth completes the authentication process
func (m *QRAuthManager) finalizeAuth(ctx context.Context, session *InternalQRSession, user *tg.User, tempStorage *MemorySessionStorage) error {
	phoneNumber := user.Phone
	if phoneNumber == "" {
		phoneNumber = fmt.Sprintf("user_%d", user.ID)
	}

	m.logger.Info().
		Str("session_id", session.ID).
		Str("phone", phoneNumber).
		Int64("user_id", user.ID).
		Msg("QR authentication successful")

	// Create proper MTProto client with persistent storage
	clientCfg := MTProtoClientConfig{
		APIID:       m.telegramCfg.APIID,
		APIHash:     m.telegramCfg.APIHash,
		PhoneNumber: phoneNumber,
		Logger:      m.logger,
	}

	newClient, err := NewMTProtoClientWithDB(clientCfg, m.db)
	if err != nil {
		session.SetError(err)
		return fmt.Errorf("create client: %w", err)
	}

	// Copy session data from temp storage to persistent storage
	if newClient.postgresStorage != nil {
		sessionData, err := tempStorage.LoadSession(ctx)
		if err == nil && sessionData != nil {
			if err := newClient.postgresStorage.StoreSession(ctx, sessionData); err != nil {
				m.logger.Warn().Err(err).Msg("failed to store session data")
			}
		}
	}

	// Connect the new client to Telegram
	if err := newClient.Connect(ctx); err != nil {
		session.SetError(err)
		return fmt.Errorf("connect client: %w", err)
	}

	// Add to AccountManager
	if err := m.accountManager.AddAccount(newClient); err != nil {
		newClient.Disconnect(ctx) // Cleanup on failure
		session.SetError(err)
		return fmt.Errorf("add account: %w", err)
	}

	// Get account ID
	var accountID *uint
	if newClient.postgresStorage != nil {
		id := newClient.postgresStorage.GetAccountID()
		accountID = &id
	}

	session.SetSuccess(phoneNumber, accountID)

	m.logger.Info().
		Str("session_id", session.ID).
		Str("phone", phoneNumber).
		Msg("account added to manager")

	return nil
}

// MemorySessionStorage is a simple in-memory session storage for temporary use
type MemorySessionStorage struct {
	data []byte
}

// NewMemorySessionStorage creates a new memory session storage
func NewMemorySessionStorage() *MemorySessionStorage {
	return &MemorySessionStorage{}
}

// LoadSession loads session data from memory
func (s *MemorySessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	return s.data, nil
}

// StoreSession stores session data in memory
func (s *MemorySessionStorage) StoreSession(ctx context.Context, data []byte) error {
	s.data = make([]byte, len(data))
	copy(s.data, data)
	return nil
}

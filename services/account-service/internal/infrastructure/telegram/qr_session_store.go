package telegram

import (
	"context"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/rs/zerolog"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/entities"
	qrerrors "github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/errors"
)

// InternalQRSession holds runtime data for QR authentication
type InternalQRSession struct {
	*entities.QRAuthSession
	Client       *telegram.Client
	PasswordChan chan string
	CancelFunc   context.CancelFunc
	mu           sync.RWMutex
}

// UpdateStatus safely updates the session status
func (s *InternalQRSession) UpdateStatus(status entities.QRAuthStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	s.UpdatedAt = time.Now()
}

// SetQRCode safely sets the QR code data
func (s *InternalQRSession) SetQRCode(url, base64 string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.QRURL = url
	s.QRCodeBase64 = base64
	s.UpdatedAt = time.Now()
}

// SetError safely sets an error on the session
func (s *InternalQRSession) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = entities.StatusFailed
	if err != nil {
		s.Error = err.Error()
	}
	s.UpdatedAt = time.Now()
}

// SetSuccess safely marks the session as successful
func (s *InternalQRSession) SetSuccess(phoneNumber string, accountID *uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = entities.StatusSuccess
	s.PhoneNumber = phoneNumber
	s.AccountID = accountID
	s.UpdatedAt = time.Now()
}

// GetSnapshot returns a thread-safe copy of the session state
func (s *InternalQRSession) GetSnapshot() *entities.QRAuthSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &entities.QRAuthSession{
		ID:           s.ID,
		SessionName:  s.SessionName,
		Status:       s.Status,
		QRURL:        s.QRURL,
		QRCodeBase64: s.QRCodeBase64,
		AccountID:    s.AccountID,
		PhoneNumber:  s.PhoneNumber,
		Error:        s.Error,
		CreatedAt:    s.CreatedAt,
		ExpiresAt:    s.ExpiresAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// QRSessionStore stores active QR authentication sessions in memory
type QRSessionStore struct {
	sessions        sync.Map // map[string]*InternalQRSession
	sessionTTL      time.Duration
	cleanupInterval time.Duration
	maxSessions     int
	sessionCount    int
	countMu         sync.Mutex
	stopCleanup     chan struct{}
	logger          zerolog.Logger
}

// NewQRSessionStore creates a new QR session store
func NewQRSessionStore(sessionTTL, cleanupInterval time.Duration, maxSessions int, logger zerolog.Logger) *QRSessionStore {
	store := &QRSessionStore{
		sessionTTL:      sessionTTL,
		cleanupInterval: cleanupInterval,
		maxSessions:     maxSessions,
		stopCleanup:     make(chan struct{}),
		logger:          logger.With().Str("component", "qr_session_store").Logger(),
	}

	// Start cleanup goroutine
	go store.runCleanup()

	return store
}

// Store saves a session to the store
func (s *QRSessionStore) Store(session *InternalQRSession) error {
	s.countMu.Lock()
	if s.sessionCount >= s.maxSessions {
		s.countMu.Unlock()
		return qrerrors.ErrMaxSessionsReached
	}
	s.sessionCount++
	s.countMu.Unlock()

	s.sessions.Store(session.ID, session)
	s.logger.Debug().Str("session_id", session.ID).Msg("session stored")
	return nil
}

// Load retrieves a session by ID
func (s *QRSessionStore) Load(sessionID string) (*InternalQRSession, error) {
	value, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil, qrerrors.ErrSessionNotFound
	}

	session := value.(*InternalQRSession)

	// Check if expired
	if session.IsExpired() {
		s.Delete(sessionID)
		return nil, qrerrors.ErrSessionExpired
	}

	return session, nil
}

// Delete removes a session from the store
func (s *QRSessionStore) Delete(sessionID string) {
	if _, loaded := s.sessions.LoadAndDelete(sessionID); loaded {
		s.countMu.Lock()
		s.sessionCount--
		s.countMu.Unlock()
		s.logger.Debug().Str("session_id", sessionID).Msg("session deleted")
	}
}

// Cleanup removes all expired sessions and returns the count of removed sessions
func (s *QRSessionStore) Cleanup() int {
	var removed int
	var toDelete []string

	s.sessions.Range(func(key, value any) bool {
		sessionID := key.(string)
		session := value.(*InternalQRSession)

		if session.IsExpired() || session.IsTerminal() {
			toDelete = append(toDelete, sessionID)
		}
		return true
	})

	for _, sessionID := range toDelete {
		// Cancel the session context if still running
		if session, err := s.Load(sessionID); err == nil && session.CancelFunc != nil {
			session.CancelFunc()
		}
		s.Delete(sessionID)
		removed++
	}

	if removed > 0 {
		s.logger.Info().Int("removed", removed).Msg("cleaned up expired sessions")
	}

	return removed
}

// Count returns the current number of active sessions
func (s *QRSessionStore) Count() int {
	s.countMu.Lock()
	defer s.countMu.Unlock()
	return s.sessionCount
}

// Stop stops the cleanup goroutine
func (s *QRSessionStore) Stop() {
	close(s.stopCleanup)
}

// runCleanup periodically removes expired sessions
func (s *QRSessionStore) runCleanup() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	s.logger.Info().
		Dur("interval", s.cleanupInterval).
		Dur("ttl", s.sessionTTL).
		Msg("QR session cleanup started")

	for {
		select {
		case <-s.stopCleanup:
			s.logger.Info().Msg("QR session cleanup stopped")
			return
		case <-ticker.C:
			s.Cleanup()
		}
	}
}

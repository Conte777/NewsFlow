package telegram

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotd/td/session"
)

// FileSessionStorage implements session.Storage interface for persistent session storage
type FileSessionStorage struct {
	sessionDir  string
	phoneNumber string
	filePath    string
}

// NewFileSessionStorage creates a new file-based session storage
func NewFileSessionStorage(sessionDir, phoneNumber string) (*FileSessionStorage, error) {
	// Create session directory if it doesn't exist
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	// Hash phone number for privacy - use first 64 bits of SHA256
	hash := sha256.Sum256([]byte(phoneNumber))
	fileName := fmt.Sprintf("session_%x.json", hash[:8])
	filePath := filepath.Join(sessionDir, fileName)

	return &FileSessionStorage{
		sessionDir:  sessionDir,
		phoneNumber: phoneNumber,
		filePath:    filePath,
	}, nil
}

// LoadSession loads session data from file
func (s *FileSessionStorage) LoadSession(ctx context.Context) (data []byte, err error) {
	// Check if session file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// No session file exists, return empty data (new session)
		return nil, session.ErrNotFound
	}

	// Read session data from file
	data, err = os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	// If file is empty, treat as no session
	if len(data) == 0 {
		return nil, session.ErrNotFound
	}

	return data, nil
}

// StoreSession stores session data to file
func (s *FileSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	// Write session data to file with restricted permissions
	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// GetFilePath returns the path to the session file
func (s *FileSessionStorage) GetFilePath() string {
	return s.filePath
}

// DeleteSession removes the session file
func (s *FileSessionStorage) DeleteSession() error {
	if err := os.Remove(s.filePath); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, nothing to delete
			return nil
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}

// SessionExists checks if a session file exists
func (s *FileSessionStorage) SessionExists() bool {
	_, err := os.Stat(s.filePath)
	return err == nil
}

// Ensure FileSessionStorage implements session.Storage interface
var _ session.Storage = (*FileSessionStorage)(nil)

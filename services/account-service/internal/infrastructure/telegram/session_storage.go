package telegram

import (
	"context"
	"encoding/json"
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

	// Create a unique filename based on phone number
	fileName := fmt.Sprintf("session_%s.json", phoneNumber)
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

// sessionData represents the structure of session data for JSON serialization
type sessionData struct {
	DC        int    `json:"dc"`
	AuthKey   []byte `json:"auth_key"`
	AuthKeyID []byte `json:"auth_key_id"`
	Salt      int64  `json:"salt"`
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

// ExportSession exports session data as JSON for backup or transfer
func (s *FileSessionStorage) ExportSession(ctx context.Context) (string, error) {
	data, err := s.LoadSession(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load session: %w", err)
	}

	// Convert to JSON string
	jsonData, err := json.Marshal(map[string]interface{}{
		"phone":   s.phoneNumber,
		"session": data,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	return string(jsonData), nil
}

// ImportSession imports session data from JSON
func (s *FileSessionStorage) ImportSession(ctx context.Context, jsonData string) error {
	var sessionMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &sessionMap); err != nil {
		return fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	sessionBytes, ok := sessionMap["session"].([]byte)
	if !ok {
		return fmt.Errorf("invalid session data format")
	}

	return s.StoreSession(ctx, sessionBytes)
}

// Ensure FileSessionStorage implements session.Storage interface
var _ session.Storage = (*FileSessionStorage)(nil)

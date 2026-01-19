package telegram

import (
	"context"

	"github.com/gotd/td/session"
)

// mockSessionStorage implements session.Storage for testing
type mockSessionStorage struct {
	data []byte
}

// LoadSession loads session data from memory
func (m *mockSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	if m.data == nil {
		return nil, session.ErrNotFound
	}
	return m.data, nil
}

// StoreSession stores session data to memory
func (m *mockSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	m.data = data
	return nil
}

// Ensure mockSessionStorage implements session.Storage interface
var _ session.Storage = (*mockSessionStorage)(nil)

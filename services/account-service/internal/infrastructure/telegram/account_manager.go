package telegram

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// accountManager manages a pool of Telegram MTProto accounts
type accountManager struct {
	accounts   map[string]domain.TelegramClient // accountID -> client
	accountIDs []string                          // ordered list for round-robin iteration
	mu         sync.RWMutex
	currentIdx atomic.Int32 // current index for round-robin load balancing
}

// NewAccountManager creates a new account manager
func NewAccountManager() domain.AccountManager {
	return &accountManager{
		accounts:   make(map[string]domain.TelegramClient),
		accountIDs: make([]string, 0),
	}
}

// GetAvailableAccount returns an available account for operation
// Uses round-robin load balancing and health checks (IsConnected)
func (m *accountManager) GetAvailableAccount() (domain.TelegramClient, error) {
	m.mu.RLock()
	if len(m.accountIDs) == 0 {
		m.mu.RUnlock()
		return nil, domain.ErrNoActiveAccounts
	}

	// Try to find a healthy connected account using round-robin
	startIdx := int(m.currentIdx.Load()) % len(m.accountIDs)
	accountsCount := len(m.accountIDs)

	// Create snapshot of accountIDs to avoid holding lock during IsConnected() calls
	accountIDsCopy := make([]string, accountsCount)
	copy(accountIDsCopy, m.accountIDs)

	// Keep accounts map reference for lookups
	accountsRef := m.accounts
	m.mu.RUnlock()

	// Search for connected account
	for i := 0; i < accountsCount; i++ {
		idx := (startIdx + i) % accountsCount
		accountID := accountIDsCopy[idx]

		m.mu.RLock()
		account, exists := accountsRef[accountID]
		m.mu.RUnlock()

		if !exists {
			continue
		}

		// Health check: only return connected accounts
		if account.IsConnected() {
			// Update currentIdx for next call (round-robin)
			// Always increment to ensure fair distribution
			m.currentIdx.Add(1)
			return account, nil
		}
	}

	return nil, domain.ErrNoActiveAccounts
}

// GetAllAccounts returns all managed accounts
func (m *accountManager) GetAllAccounts() []domain.TelegramClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	accounts := make([]domain.TelegramClient, 0, len(m.accountIDs))
	for _, id := range m.accountIDs {
		accounts = append(accounts, m.accounts[id])
	}
	return accounts
}

// AddAccount adds a new account to the pool
func (m *accountManager) AddAccount(client domain.TelegramClient) error {
	if client == nil {
		return fmt.Errorf("cannot add nil client")
	}

	// Use GetAccountID() from interface instead of type assertion
	accountID := client.GetAccountID()
	if accountID == "" {
		return fmt.Errorf("client account ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if account already exists
	if _, exists := m.accounts[accountID]; exists {
		return fmt.Errorf("account already exists: %s", maskPhoneNumber(accountID))
	}

	m.accounts[accountID] = client
	m.accountIDs = append(m.accountIDs, accountID)

	return nil
}

// RemoveAccount removes an account from the pool by account ID
func (m *accountManager) RemoveAccount(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.accounts[accountID]; !exists {
		return fmt.Errorf("account not found: %s", maskPhoneNumber(accountID))
	}

	delete(m.accounts, accountID)

	// Remove from accountIDs slice
	for i, id := range m.accountIDs {
		if id == accountID {
			m.accountIDs = append(m.accountIDs[:i], m.accountIDs[i+1:]...)
			break
		}
	}

	// Reset currentIdx if needed
	if len(m.accountIDs) > 0 {
		// Normalize currentIdx to be within bounds
		currentIdx := int(m.currentIdx.Load())
		if currentIdx >= len(m.accountIDs) {
			m.currentIdx.Store(int32(currentIdx % len(m.accountIDs)))
		}
	} else {
		m.currentIdx.Store(0)
	}

	return nil
}

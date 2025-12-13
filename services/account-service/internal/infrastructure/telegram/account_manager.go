package telegram

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// ClientFactory is a function type for creating Telegram clients
type ClientFactory func(cfg MTProtoClientConfig) (domain.TelegramClient, error)

// accountManager manages a pool of Telegram MTProto accounts
type accountManager struct {
	accounts   map[string]domain.TelegramClient // accountID -> client
	accountIDs []string                          // ordered list for round-robin iteration
	mu         sync.RWMutex
	currentIdx atomic.Int32 // current index for round-robin load balancing

	// clientFactory is used to create new clients (can be overridden for testing)
	clientFactory ClientFactory
}

// NewAccountManager creates a new account manager
func NewAccountManager() domain.AccountManager {
	return &accountManager{
		accounts:      make(map[string]domain.TelegramClient),
		accountIDs:    make([]string, 0),
		clientFactory: defaultClientFactory,
	}
}

// defaultClientFactory is the default factory that creates real MTProtoClient instances
func defaultClientFactory(cfg MTProtoClientConfig) (domain.TelegramClient, error) {
	return NewMTProtoClient(cfg)
}

// InitializeAccounts loads and initializes Telegram accounts from configuration
// It creates MTProtoClient for each phone number and connects them in parallel.
//
// The method implements the following safety measures:
// - Worker pool pattern to limit concurrent goroutines (prevents resource exhaustion)
// - Context cancellation support (graceful shutdown)
// - Proper cleanup of failed connections (timeout-based disconnect)
// - Phone number masking in error reports (security)
//
// Returns a detailed report about initialization success/failure.
func (m *accountManager) InitializeAccounts(ctx context.Context, cfg domain.AccountInitConfig) *domain.InitializationReport {
	report := &domain.InitializationReport{
		TotalAccounts: len(cfg.Accounts),
		Errors:        make(map[string]error),
	}

	if len(cfg.Accounts) == 0 {
		cfg.Logger.Warn().Msg("No accounts configured for initialization")
		return report
	}

	// Set default MaxConcurrent if not specified
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Default limit to prevent resource exhaustion
	}

	cfg.Logger.Info().
		Int("count", len(cfg.Accounts)).
		Int("max_concurrent", maxConcurrent).
		Msg("Starting account initialization")

	// Use WaitGroup for parallel connection
	var wg sync.WaitGroup
	reportMu := sync.Mutex{} // Renamed for consistency

	// Semaphore for limiting concurrent goroutines (worker pool pattern)
	semaphore := make(chan struct{}, maxConcurrent)

	for _, phoneNumber := range cfg.Accounts {
		wg.Add(1)
		go func(phone string) {
			defer wg.Done()

			// Check for context cancellation before starting work
			select {
			case <-ctx.Done():
				cfg.Logger.Debug().
					Str("phone", maskPhoneNumber(phone)).
					Msg("Skipping account initialization due to context cancellation")
				reportMu.Lock()
				report.Errors[maskPhoneNumber(phone)] = ctx.Err()
				report.FailedAccounts++
				reportMu.Unlock()
				return
			default:
			}

			// Acquire semaphore (worker pool)
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				reportMu.Lock()
				report.Errors[maskPhoneNumber(phone)] = ctx.Err()
				report.FailedAccounts++
				reportMu.Unlock()
				return
			}

			maskedPhone := maskPhoneNumber(phone)
			logger := cfg.Logger.With().Str("phone", maskedPhone).Logger()

			// Create MTProto client using factory
			client, err := m.clientFactory(MTProtoClientConfig{
				APIID:       cfg.APIID,
				APIHash:     cfg.APIHash,
				PhoneNumber: phone,
				SessionDir:  cfg.SessionDir,
				Logger:      logger,
			})

			if err != nil {
				logger.Warn().Err(err).Msg("Failed to create MTProto client")
				reportMu.Lock()
				report.Errors[maskedPhone] = fmt.Errorf("create client: %w", err)
				report.FailedAccounts++
				reportMu.Unlock()
				return
			}

			// Connect to Telegram with context
			if err := client.Connect(ctx); err != nil {
				logger.Warn().Err(err).Msg("Failed to connect account")
				reportMu.Lock()
				report.Errors[maskedPhone] = fmt.Errorf("connect: %w", err)
				report.FailedAccounts++
				reportMu.Unlock()
				return
			}

			// Add to account manager
			if err := m.AddAccount(client); err != nil {
				logger.Warn().Err(err).Msg("Failed to add account to manager")

				// Disconnect the client since we can't add it
				// Use timeout context for graceful shutdown (not context.Background())
				disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if disconnectErr := client.Disconnect(disconnectCtx); disconnectErr != nil {
					logger.Warn().Err(disconnectErr).Msg("Failed to disconnect client during cleanup")
				}

				reportMu.Lock()
				report.Errors[maskedPhone] = fmt.Errorf("add account: %w", err)
				report.FailedAccounts++
				reportMu.Unlock()
				return
			}

			logger.Info().Msg("Account initialized successfully")
			reportMu.Lock()
			report.SuccessfulAccounts++
			reportMu.Unlock()
		}(phoneNumber)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	cfg.Logger.Info().
		Int("total", report.TotalAccounts).
		Int("successful", report.SuccessfulAccounts).
		Int("failed", report.FailedAccounts).
		Msg("Account initialization completed")

	return report
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

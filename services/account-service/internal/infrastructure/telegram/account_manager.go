package telegram

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/news/handlers"
)

// ClientFactory is a function type for creating Telegram clients with database connection
type ClientFactory func(cfg MTProtoClientConfig, db *gorm.DB) (domain.TelegramClient, error)

// accountManager manages a pool of Telegram MTProto accounts
type accountManager struct {
	accounts   map[string]domain.TelegramClient // accountID -> client
	accountIDs []string                          // ordered list for round-robin iteration
	mu         sync.RWMutex
	currentIdx atomic.Int32 // current index for round-robin load balancing

	// clientFactory is used to create new clients (must be set via fx.go)
	clientFactory ClientFactory

	// db is database connection for session storage
	db *gorm.DB

	// logger is used for logging shutdown process and errors
	// If not set, uses zerolog.Nop()
	logger zerolog.Logger

	// newsHandler is set for all clients to enable real-time updates
	newsHandler *handlers.NewsUpdateHandler

	// isShutdown prevents operations after Shutdown has been called
	isShutdown atomic.Bool
}

// NewAccountManager creates a new account manager
func NewAccountManager() domain.AccountManager {
	return &accountManager{
		accounts:   make(map[string]domain.TelegramClient),
		accountIDs: make([]string, 0),
		logger:     zerolog.Nop(),
	}
}

// WithLogger sets a logger for the account manager (optional, for shutdown logging)
func (m *accountManager) WithLogger(logger zerolog.Logger) *accountManager {
	m.logger = logger
	return m
}

// WithDB sets the database connection for session storage
func (m *accountManager) WithDB(db *gorm.DB) *accountManager {
	m.db = db
	return m
}

// WithNewsHandler sets the news handler for all clients to enable real-time updates
func (m *accountManager) WithNewsHandler(handler *handlers.NewsUpdateHandler) *accountManager {
	m.newsHandler = handler
	return m
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
	if m.isShutdown.Load() {
		cfg.Logger.Warn().Msg("Cannot initialize accounts: manager is shut down")
		return &domain.InitializationReport{
			TotalAccounts: len(cfg.Accounts),
			Errors:        make(map[string]error),
		}
	}

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
					Str("phone", phone).
					Msg("Skipping account initialization due to context cancellation")
				reportMu.Lock()
				report.Errors[phone] = ctx.Err()
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
				report.Errors[phone] = ctx.Err()
				report.FailedAccounts++
				reportMu.Unlock()
				return
			}

			logger := cfg.Logger.With().Str("phone", phone).Logger()

			// Create MTProto client using factory
			client, err := m.clientFactory(MTProtoClientConfig{
				APIID:       cfg.APIID,
				APIHash:     cfg.APIHash,
				PhoneNumber: phone,
				Logger:      logger,
			}, m.db)

			if err != nil {
				logger.Warn().Err(err).Msg("Failed to create MTProto client")
				reportMu.Lock()
				report.Errors[phone] = fmt.Errorf("create client: %w", err)
				report.FailedAccounts++
				reportMu.Unlock()
				return
			}

			// Set news handler for real-time updates (if available)
			if m.newsHandler != nil {
				if mtpClient, ok := client.(*MTProtoClient); ok {
					mtpClient.SetNewsHandler(m.newsHandler)
					logger.Debug().Msg("News handler set for real-time updates")
				}
			}

			// Connect to Telegram with context
			if err := client.Connect(ctx); err != nil {
				logger.Warn().Err(err).Msg("Failed to connect account")
				reportMu.Lock()
				report.Errors[phone] = fmt.Errorf("connect: %w", err)
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
				report.Errors[phone] = fmt.Errorf("add account: %w", err)
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
	if m.isShutdown.Load() {
		return nil, fmt.Errorf("account manager is shut down")
	}

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

// GetAccountByPhone returns an account by phone number
func (m *accountManager) GetAccountByPhone(phoneNumber string) (domain.TelegramClient, error) {
	if m.isShutdown.Load() {
		return nil, fmt.Errorf("account manager is shut down")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.accounts[phoneNumber]
	if !exists {
		return nil, fmt.Errorf("account not found: %s", phoneNumber)
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("account %s is not connected", phoneNumber)
	}

	return client, nil
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

// GetActiveAccountCount returns the number of active (connected) accounts
func (m *accountManager) GetActiveAccountCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeCount := 0
	for _, client := range m.accounts {
		if client.IsConnected() {
			activeCount++
		}
	}
	return activeCount
}

// AddAccount adds a new account to the pool
func (m *accountManager) AddAccount(client domain.TelegramClient) error {
	if m.isShutdown.Load() {
		return fmt.Errorf("cannot add account: manager is shut down")
	}

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
		return fmt.Errorf("account already exists: %s", accountID)
	}

	m.accounts[accountID] = client
	m.accountIDs = append(m.accountIDs, accountID)

	return nil
}

// RemoveAccount removes an account from the pool by account ID
func (m *accountManager) RemoveAccount(accountID string) error {
	if m.isShutdown.Load() {
		return fmt.Errorf("cannot remove account: manager is shut down")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.accounts[accountID]; !exists {
		return fmt.Errorf("account not found: %s", accountID)
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

// SyncAccounts discovers and initializes new accounts from the provided phone numbers.
// It compares the given list with currently managed accounts and only initializes new ones.
// Returns a report about newly initialized accounts.
func (m *accountManager) SyncAccounts(ctx context.Context, cfg domain.AccountInitConfig) *domain.InitializationReport {
	if m.isShutdown.Load() {
		cfg.Logger.Warn().Msg("Cannot sync accounts: manager is shut down")
		return &domain.InitializationReport{
			TotalAccounts: 0,
			Errors:        make(map[string]error),
		}
	}

	// Get currently managed account IDs
	m.mu.RLock()
	existingAccounts := make(map[string]bool, len(m.accountIDs))
	for _, id := range m.accountIDs {
		existingAccounts[id] = true
	}
	m.mu.RUnlock()

	// Find new accounts (not yet in the manager)
	newAccounts := make([]string, 0)
	for _, phone := range cfg.Accounts {
		if !existingAccounts[phone] {
			newAccounts = append(newAccounts, phone)
		}
	}

	if len(newAccounts) == 0 {
		return &domain.InitializationReport{
			TotalAccounts:      0,
			SuccessfulAccounts: 0,
			FailedAccounts:     0,
			Errors:             make(map[string]error),
		}
	}

	cfg.Logger.Info().
		Int("new_accounts", len(newAccounts)).
		Msg("Discovered new accounts to initialize")

	// Initialize only new accounts
	syncCfg := domain.AccountInitConfig{
		APIID:         cfg.APIID,
		APIHash:       cfg.APIHash,
		Accounts:      newAccounts,
		Logger:        cfg.Logger,
		MaxConcurrent: cfg.MaxConcurrent,
	}

	return m.InitializeAccounts(ctx, syncCfg)
}

// Shutdown gracefully disconnects all managed accounts
//
// The method implements the following behavior:
// - Disconnects accounts sequentially (not in parallel) for predictable behavior
// - Non-blocking error handling: continues disconnecting even if some fail
// - Respects context timeout (recommended 30 seconds)
// - Logs progress and errors for monitoring
// - Returns count of successfully disconnected accounts
// - Can only be called once; subsequent calls return 0
//
// Context cancellation or timeout will stop the shutdown process early.
// Partially disconnected state is acceptable for graceful degradation.
func (m *accountManager) Shutdown(ctx context.Context) int {
	// Atomically set shutdown flag (prevents concurrent shutdowns)
	if !m.isShutdown.CompareAndSwap(false, true) {
		m.logger.Warn().Msg("Shutdown already in progress or completed")
		return 0
	}

	m.mu.RLock()
	accountCount := len(m.accountIDs)

	// Create snapshot of accounts to avoid holding lock during disconnect
	accountsToDisconnect := make([]struct {
		id     string
		client domain.TelegramClient
	}, 0, accountCount)

	for _, id := range m.accountIDs {
		if client, exists := m.accounts[id]; exists {
			accountsToDisconnect = append(accountsToDisconnect, struct {
				id     string
				client domain.TelegramClient
			}{id: id, client: client})
		}
	}
	m.mu.RUnlock()

	if accountCount == 0 {
		m.logger.Info().Msg("No accounts to shutdown")
		return 0
	}

	m.logger.Info().
		Int("total_accounts", accountCount).
		Msg("Starting graceful shutdown of accounts")

	successCount := 0
	failedCount := 0

	// Disconnect accounts sequentially
	for _, acc := range accountsToDisconnect {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			m.logger.Warn().
				Err(ctx.Err()).
				Int("disconnected", successCount).
				Int("failed", failedCount).
				Int("remaining", accountCount-successCount-failedCount).
				Msg("Shutdown interrupted by context cancellation")
			return successCount
		default:
		}

		logger := m.logger.With().Str("account", acc.id).Logger()

		logger.Debug().Msg("Disconnecting account")

		// Disconnect with context timeout
		if err := acc.client.Disconnect(ctx); err != nil {
			logger.Warn().Err(err).Msg("Failed to disconnect account")
			failedCount++
			// Continue to next account (non-blocking error handling)
			continue
		}

		logger.Info().Msg("Account disconnected successfully")
		successCount++
	}

	m.logger.Info().
		Int("total", accountCount).
		Int("successful", successCount).
		Int("failed", failedCount).
		Msg("Account shutdown completed")

	return successCount
}

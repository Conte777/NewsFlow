package telegram

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
)

// mockTelegramClient is a test mock that implements domain.TelegramClient
type mockTelegramClient struct {
	accountID string
	connected bool
	mu        sync.RWMutex
}

func (m *mockTelegramClient) Connect(ctx context.Context) error {
	return nil
}

func (m *mockTelegramClient) Disconnect(ctx context.Context) error {
	return nil
}

func (m *mockTelegramClient) JoinChannel(ctx context.Context, channelID string) error {
	return nil
}

func (m *mockTelegramClient) LeaveChannel(ctx context.Context, channelID string) error {
	return nil
}

func (m *mockTelegramClient) GetChannelMessages(ctx context.Context, channelID string, limit, offset int) ([]domain.NewsItem, error) {
	return nil, nil
}

func (m *mockTelegramClient) GetChannelInfo(ctx context.Context, channelID string) (*domain.ChannelInfo, error) {
	return nil, nil
}

func (m *mockTelegramClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

func (m *mockTelegramClient) GetAccountID() string {
	return m.accountID
}

func (m *mockTelegramClient) setConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// createTestClient creates a mock TelegramClient for testing
func createTestClient(accountID string, connected bool) *mockTelegramClient {
	return &mockTelegramClient{
		accountID: accountID,
		connected: connected,
	}
}

func TestNewAccountManager(t *testing.T) {
	manager := NewAccountManager()
	if manager == nil {
		t.Fatal("NewAccountManager returned nil")
	}

	accounts := manager.GetAllAccounts()
	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(accounts))
	}
}

func TestAddAccount(t *testing.T) {
	manager := NewAccountManager()

	client1 := createTestClient("+1234567890", true)
	err := manager.AddAccount(client1)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	accounts := manager.GetAllAccounts()
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(accounts))
	}

	// Add another account
	client2 := createTestClient("+0987654321", true)
	err = manager.AddAccount(client2)
	if err != nil {
		t.Fatalf("Failed to add second account: %v", err)
	}

	accounts = manager.GetAllAccounts()
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accounts))
	}
}

func TestAddAccount_NilClient(t *testing.T) {
	manager := NewAccountManager()

	err := manager.AddAccount(nil)
	if err == nil {
		t.Error("Expected error when adding nil client, got nil")
	}
}

func TestAddAccount_EmptyAccountID(t *testing.T) {
	manager := NewAccountManager()

	client := createTestClient("", true)
	err := manager.AddAccount(client)
	if err == nil {
		t.Error("Expected error when adding client with empty account ID, got nil")
	}
}

func TestAddAccount_DuplicateAccount(t *testing.T) {
	manager := NewAccountManager()

	client := createTestClient("+1234567890", true)
	err := manager.AddAccount(client)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Try to add the same account ID again
	client2 := createTestClient("+1234567890", true)
	err = manager.AddAccount(client2)
	if err == nil {
		t.Error("Expected error when adding duplicate account, got nil")
	}
}

func TestGetAvailableAccount(t *testing.T) {
	manager := NewAccountManager()

	client := createTestClient("+1234567890", true)
	err := manager.AddAccount(client)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	availableClient, err := manager.GetAvailableAccount()
	if err != nil {
		t.Fatalf("Failed to get available account: %v", err)
	}

	if availableClient == nil {
		t.Error("Expected non-nil client, got nil")
	}
}

func TestGetAvailableAccount_NoAccounts(t *testing.T) {
	manager := NewAccountManager()

	_, err := manager.GetAvailableAccount()
	if err != domain.ErrNoActiveAccounts {
		t.Errorf("Expected ErrNoActiveAccounts, got %v", err)
	}
}

func TestGetAvailableAccount_NoConnectedAccounts(t *testing.T) {
	manager := NewAccountManager()

	// Add disconnected account
	client := createTestClient("+1234567890", false)
	err := manager.AddAccount(client)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	_, err = manager.GetAvailableAccount()
	if err != domain.ErrNoActiveAccounts {
		t.Errorf("Expected ErrNoActiveAccounts when all accounts disconnected, got %v", err)
	}
}

func TestGetAvailableAccount_RoundRobin(t *testing.T) {
	manager := NewAccountManager()

	client1 := createTestClient("+1111111111", true)
	client2 := createTestClient("+2222222222", true)
	client3 := createTestClient("+3333333333", true)

	manager.AddAccount(client1)
	manager.AddAccount(client2)
	manager.AddAccount(client3)

	// Get accounts in round-robin order
	accounts := make([]domain.TelegramClient, 6)
	for i := 0; i < 6; i++ {
		client, err := manager.GetAvailableAccount()
		if err != nil {
			t.Fatalf("Failed to get available account at iteration %d: %v", i, err)
		}
		accounts[i] = client
	}

	// Verify round-robin: should cycle through accounts
	// With the improved algorithm, it always increments
	// So we should see each account appear twice
	accountIDs := make([]string, 6)
	for i, acc := range accounts {
		accountIDs[i] = acc.GetAccountID()
	}

	// Count occurrences
	counts := make(map[string]int)
	for _, id := range accountIDs {
		counts[id]++
	}

	// Each account should be selected 2 times in 6 calls
	for id, count := range counts {
		if count != 2 {
			t.Errorf("Account %s selected %d times, expected 2", id, count)
		}
	}
}

func TestGetAllAccounts(t *testing.T) {
	manager := NewAccountManager()

	client1 := createTestClient("+1111111111", true)
	client2 := createTestClient("+2222222222", true)

	manager.AddAccount(client1)
	manager.AddAccount(client2)

	accounts := manager.GetAllAccounts()
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accounts))
	}

	// Verify that modifying returned slice doesn't affect internal state
	accounts[0] = nil
	accountsAgain := manager.GetAllAccounts()
	if accountsAgain[0] == nil {
		t.Error("Internal state was modified by external slice modification")
	}
}

func TestRemoveAccount(t *testing.T) {
	manager := NewAccountManager()

	client1 := createTestClient("+1111111111", true)
	client2 := createTestClient("+2222222222", true)

	manager.AddAccount(client1)
	manager.AddAccount(client2)

	// Remove first account
	err := manager.RemoveAccount("+1111111111")
	if err != nil {
		t.Fatalf("Failed to remove account: %v", err)
	}

	accounts := manager.GetAllAccounts()
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account after removal, got %d", len(accounts))
	}

	if accounts[0].GetAccountID() != "+2222222222" {
		t.Error("Wrong account remained after removal")
	}
}

func TestRemoveAccount_NotFound(t *testing.T) {
	manager := NewAccountManager()

	err := manager.RemoveAccount("+9999999999")
	if err == nil {
		t.Error("Expected error when removing non-existent account, got nil")
	}
}

func TestRemoveAccount_AdjustsRoundRobinIndex(t *testing.T) {
	manager := NewAccountManager()

	client1 := createTestClient("+1111111111", true)
	client2 := createTestClient("+2222222222", true)
	client3 := createTestClient("+3333333333", true)

	manager.AddAccount(client1)
	manager.AddAccount(client2)
	manager.AddAccount(client3)

	// Get first account to advance index
	manager.GetAvailableAccount() // Should return client1, advance index
	manager.GetAvailableAccount() // Should return client2, advance index

	// Remove client3 (last in list)
	err := manager.RemoveAccount("+3333333333")
	if err != nil {
		t.Fatalf("Failed to remove account: %v", err)
	}

	// Next call should still work (index should be adjusted)
	client, err := manager.GetAvailableAccount()
	if err != nil {
		t.Fatalf("Failed to get available account after removal: %v", err)
	}
	if client == nil {
		t.Error("Expected non-nil client after removal")
	}
}

func TestGetAvailableAccount_SkipsDisconnected(t *testing.T) {
	manager := NewAccountManager()

	client1 := createTestClient("+1111111111", false) // disconnected
	client2 := createTestClient("+2222222222", true)  // connected
	client3 := createTestClient("+3333333333", false) // disconnected

	manager.AddAccount(client1)
	manager.AddAccount(client2)
	manager.AddAccount(client3)

	// Should return client2 as it's the only connected one
	client, err := manager.GetAvailableAccount()
	if err != nil {
		t.Fatalf("Failed to get available account: %v", err)
	}

	if client.GetAccountID() != "+2222222222" {
		t.Errorf("Expected to get client2, got %s", client.GetAccountID())
	}

	// Second call should also return client2
	client, err = manager.GetAvailableAccount()
	if err != nil {
		t.Fatalf("Failed to get available account on second call: %v", err)
	}

	if client.GetAccountID() != "+2222222222" {
		t.Errorf("Expected to get client2 on second call, got %s", client.GetAccountID())
	}
}

// Concurrency tests
func TestAccountManager_ConcurrentGetAvailableAccount(t *testing.T) {
	manager := NewAccountManager()

	// Add multiple accounts
	for i := 0; i < 5; i++ {
		client := createTestClient(string(rune('+'))+string(rune('1'+i))+"000000000", true)
		if err := manager.AddAccount(client); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
	}

	// Concurrently get available accounts
	const goroutines = 20
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := manager.GetAvailableAccount()
				if err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent GetAvailableAccount error: %v", err)
	}
}

func TestAccountManager_ConcurrentAddRemove(t *testing.T) {
	manager := NewAccountManager()

	// Add initial accounts
	for i := 0; i < 3; i++ {
		client := createTestClient(string(rune('+'))+string(rune('1'+i))+"000000000", true)
		if err := manager.AddAccount(client); err != nil {
			t.Fatalf("Failed to add initial account: %v", err)
		}
	}

	var wg sync.WaitGroup
	const goroutines = 10

	// Concurrent reads (GetAvailableAccount)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				manager.GetAvailableAccount()
			}
		}()
	}

	// Concurrent reads (GetAllAccounts)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				manager.GetAllAccounts()
			}
		}()
	}

	// Concurrent writes (AddAccount/RemoveAccount)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		accountID := string(rune('+')) + string(rune('5'+i)) + "000000000"
		go func(id string) {
			defer wg.Done()
			client := createTestClient(id, true)
			manager.AddAccount(client)
			manager.RemoveAccount(id)
		}(accountID)
	}

	wg.Wait()

	// Verify manager is still functional
	accounts := manager.GetAllAccounts()
	if len(accounts) < 3 {
		t.Errorf("Expected at least 3 accounts after concurrent operations, got %d", len(accounts))
	}
}

func TestAccountManager_ConcurrentConnectionChanges(t *testing.T) {
	manager := NewAccountManager()

	// Add accounts
	clients := make([]*mockTelegramClient, 5)
	for i := 0; i < 5; i++ {
		clients[i] = createTestClient(string(rune('+'))+string(rune('1'+i))+"000000000", true)
		if err := manager.AddAccount(clients[i]); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
	}

	var wg sync.WaitGroup
	const goroutines = 20
	const iterations = 50

	// Goroutines that get available accounts
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				manager.GetAvailableAccount()
			}
		}()
	}

	// Goroutines that toggle connection status
	for i := 0; i < 5; i++ {
		wg.Add(1)
		client := clients[i]
		go func(c *mockTelegramClient) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				c.setConnected(j%2 == 0)
			}
		}(client)
	}

	wg.Wait()

	// Verify manager is still functional
	accounts := manager.GetAllAccounts()
	if len(accounts) != 5 {
		t.Errorf("Expected 5 accounts after concurrent operations, got %d", len(accounts))
	}
}

// Tests for InitializeAccounts

func TestInitializeAccounts_Success(t *testing.T) {
	manager := NewAccountManager().(*accountManager)

	// Set up mock client factory that returns connected clients
	callCount := 0
	manager.clientFactory = func(cfg MTProtoClientConfig) (domain.TelegramClient, error) {
		callCount++
		client := createTestClient(cfg.PhoneNumber, true)
		return client, nil
	}

	// Create test logger
	logger := createTestLogger()

	// Initialize accounts
	ctx := context.Background()
	cfg := domain.AccountInitConfig{
		APIID:      12345,
		APIHash:    "test_hash",
		SessionDir: "./test_sessions",
		Accounts:   []string{"+1111111111", "+2222222222", "+3333333333"},
		Logger:     logger,
	}

	report := manager.InitializeAccounts(ctx, cfg)

	// Verify report
	if report.TotalAccounts != 3 {
		t.Errorf("Expected TotalAccounts=3, got %d", report.TotalAccounts)
	}
	if report.SuccessfulAccounts != 3 {
		t.Errorf("Expected SuccessfulAccounts=3, got %d", report.SuccessfulAccounts)
	}
	if report.FailedAccounts != 0 {
		t.Errorf("Expected FailedAccounts=0, got %d", report.FailedAccounts)
	}
	if len(report.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(report.Errors))
	}

	// Verify factory was called 3 times
	if callCount != 3 {
		t.Errorf("Expected factory to be called 3 times, got %d", callCount)
	}

	// Verify accounts were added to manager
	accounts := manager.GetAllAccounts()
	if len(accounts) != 3 {
		t.Errorf("Expected 3 accounts in manager, got %d", len(accounts))
	}
}

func TestInitializeAccounts_EmptyList(t *testing.T) {
	manager := NewAccountManager().(*accountManager)

	// Create test logger
	logger := createTestLogger()

	// Initialize with empty account list
	ctx := context.Background()
	cfg := domain.AccountInitConfig{
		APIID:      12345,
		APIHash:    "test_hash",
		SessionDir: "./test_sessions",
		Accounts:   []string{},
		Logger:     logger,
	}

	report := manager.InitializeAccounts(ctx, cfg)

	// Verify report
	if report.TotalAccounts != 0 {
		t.Errorf("Expected TotalAccounts=0, got %d", report.TotalAccounts)
	}
	if report.SuccessfulAccounts != 0 {
		t.Errorf("Expected SuccessfulAccounts=0, got %d", report.SuccessfulAccounts)
	}
	if report.FailedAccounts != 0 {
		t.Errorf("Expected FailedAccounts=0, got %d", report.FailedAccounts)
	}

	// Verify no accounts were added
	accounts := manager.GetAllAccounts()
	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts in manager, got %d", len(accounts))
	}
}

func TestInitializeAccounts_PartialFailure(t *testing.T) {
	manager := NewAccountManager().(*accountManager)

	// Set up mock client factory that fails for specific phone numbers
	manager.clientFactory = func(cfg MTProtoClientConfig) (domain.TelegramClient, error) {
		if cfg.PhoneNumber == "+2222222222" {
			return nil, fmt.Errorf("connection failed")
		}
		client := createTestClient(cfg.PhoneNumber, true)
		return client, nil
	}

	// Create test logger
	logger := createTestLogger()

	// Initialize accounts
	ctx := context.Background()
	cfg := domain.AccountInitConfig{
		APIID:      12345,
		APIHash:    "test_hash",
		SessionDir: "./test_sessions",
		Accounts:   []string{"+1111111111", "+2222222222", "+3333333333"},
		Logger:     logger,
	}

	report := manager.InitializeAccounts(ctx, cfg)

	// Verify report
	if report.TotalAccounts != 3 {
		t.Errorf("Expected TotalAccounts=3, got %d", report.TotalAccounts)
	}
	if report.SuccessfulAccounts != 2 {
		t.Errorf("Expected SuccessfulAccounts=2, got %d", report.SuccessfulAccounts)
	}
	if report.FailedAccounts != 1 {
		t.Errorf("Expected FailedAccounts=1, got %d", report.FailedAccounts)
	}
	if len(report.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(report.Errors))
	}

	// Verify the specific error
	if _, exists := report.Errors["+2222222222"]; !exists {
		t.Errorf("Expected error for +2222222222")
	}

	// Verify only successful accounts were added
	accounts := manager.GetAllAccounts()
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts in manager, got %d", len(accounts))
	}
}

func TestInitializeAccounts_AllFailures(t *testing.T) {
	manager := NewAccountManager().(*accountManager)

	// Set up mock client factory that always fails
	manager.clientFactory = func(cfg MTProtoClientConfig) (domain.TelegramClient, error) {
		return nil, fmt.Errorf("connection failed")
	}

	// Create test logger
	logger := createTestLogger()

	// Initialize accounts
	ctx := context.Background()
	cfg := domain.AccountInitConfig{
		APIID:      12345,
		APIHash:    "test_hash",
		SessionDir: "./test_sessions",
		Accounts:   []string{"+1111111111", "+2222222222"},
		Logger:     logger,
	}

	report := manager.InitializeAccounts(ctx, cfg)

	// Verify report
	if report.TotalAccounts != 2 {
		t.Errorf("Expected TotalAccounts=2, got %d", report.TotalAccounts)
	}
	if report.SuccessfulAccounts != 0 {
		t.Errorf("Expected SuccessfulAccounts=0, got %d", report.SuccessfulAccounts)
	}
	if report.FailedAccounts != 2 {
		t.Errorf("Expected FailedAccounts=2, got %d", report.FailedAccounts)
	}
	if len(report.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(report.Errors))
	}

	// Verify no accounts were added
	accounts := manager.GetAllAccounts()
	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts in manager, got %d", len(accounts))
	}
}

func TestInitializeAccounts_ConnectFailure(t *testing.T) {
	manager := NewAccountManager().(*accountManager)

	// Set up mock client factory that returns disconnected clients
	manager.clientFactory = func(cfg MTProtoClientConfig) (domain.TelegramClient, error) {
		// Create client that fails on Connect
		client := &mockTelegramClientWithConnectFailure{
			mockTelegramClient: mockTelegramClient{
				accountID: cfg.PhoneNumber,
				connected: false,
			},
		}
		return client, nil
	}

	// Create test logger
	logger := createTestLogger()

	// Initialize accounts
	ctx := context.Background()
	cfg := domain.AccountInitConfig{
		APIID:      12345,
		APIHash:    "test_hash",
		SessionDir: "./test_sessions",
		Accounts:   []string{"+1111111111"},
		Logger:     logger,
	}

	report := manager.InitializeAccounts(ctx, cfg)

	// Verify report shows connection failure
	if report.TotalAccounts != 1 {
		t.Errorf("Expected TotalAccounts=1, got %d", report.TotalAccounts)
	}
	if report.SuccessfulAccounts != 0 {
		t.Errorf("Expected SuccessfulAccounts=0, got %d", report.SuccessfulAccounts)
	}
	if report.FailedAccounts != 1 {
		t.Errorf("Expected FailedAccounts=1, got %d", report.FailedAccounts)
	}

	// Verify no accounts were added
	accounts := manager.GetAllAccounts()
	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts in manager, got %d", len(accounts))
	}
}

// Helper: mockTelegramClientWithConnectFailure simulates connection failures
type mockTelegramClientWithConnectFailure struct {
	mockTelegramClient
}

func (m *mockTelegramClientWithConnectFailure) Connect(ctx context.Context) error {
	return fmt.Errorf("connection failed")
}

// Helper: mockTelegramClientWithDisconnectFailure simulates disconnect failures
type mockTelegramClientWithDisconnectFailure struct {
	mockTelegramClient
	disconnectError error
}

func (m *mockTelegramClientWithDisconnectFailure) Disconnect(ctx context.Context) error {
	if m.disconnectError != nil {
		return m.disconnectError
	}
	return fmt.Errorf("disconnect failed")
}

// Tests for Shutdown

func TestShutdown_Success(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Add multiple accounts
	client1 := createTestClient("+1111111111", true)
	client2 := createTestClient("+2222222222", true)
	client3 := createTestClient("+3333333333", true)

	manager.AddAccount(client1)
	manager.AddAccount(client2)
	manager.AddAccount(client3)

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	successCount := manager.Shutdown(ctx)

	// Verify all accounts were disconnected
	if successCount != 3 {
		t.Errorf("Expected 3 successful disconnections, got %d", successCount)
	}
}

func TestShutdown_NoAccounts(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Shutdown with no accounts
	ctx := context.Background()
	successCount := manager.Shutdown(ctx)

	// Verify count is 0
	if successCount != 0 {
		t.Errorf("Expected 0 successful disconnections, got %d", successCount)
	}
}

func TestShutdown_PartialFailure(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Add accounts - one will fail to disconnect
	client1 := createTestClient("+1111111111", true)
	client2 := &mockTelegramClientWithDisconnectFailure{
		mockTelegramClient: mockTelegramClient{
			accountID: "+2222222222",
			connected: true,
		},
		disconnectError: fmt.Errorf("network error"),
	}
	client3 := createTestClient("+3333333333", true)

	manager.AddAccount(client1)
	manager.AddAccount(client2)
	manager.AddAccount(client3)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	successCount := manager.Shutdown(ctx)

	// Verify 2 out of 3 succeeded (non-blocking error handling)
	if successCount != 2 {
		t.Errorf("Expected 2 successful disconnections, got %d", successCount)
	}
}

func TestShutdown_ContextTimeout(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Add multiple accounts with slow disconnect (100ms each)
	for i := 0; i < 5; i++ {
		client := &mockTelegramClientWithSlowDisconnect{
			mockTelegramClient: mockTelegramClient{
				accountID: fmt.Sprintf("+%d000000000", i+1),
				connected: true,
			},
			delay: 100 * time.Millisecond,
		}
		manager.AddAccount(client)
	}

	// Context timeout of 250ms allows only 2 disconnects (100ms + 100ms)
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	successCount := manager.Shutdown(ctx)

	// Should disconnect at least 1 but not all 5 accounts
	if successCount == 0 {
		t.Error("Expected at least 1 disconnect before timeout")
	}
	if successCount >= 5 {
		t.Errorf("Expected timeout to interrupt shutdown, but all %d accounts disconnected", successCount)
	}
	// Typical result: 2 successful disconnects (200ms < 250ms < 300ms)
	t.Logf("Disconnected %d out of 5 accounts before timeout", successCount)
}

func TestShutdown_AllDisconnectErrors(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Add accounts that will all fail to disconnect
	client1 := &mockTelegramClientWithDisconnectFailure{
		mockTelegramClient: mockTelegramClient{
			accountID: "+1111111111",
			connected: true,
		},
	}
	client2 := &mockTelegramClientWithDisconnectFailure{
		mockTelegramClient: mockTelegramClient{
			accountID: "+2222222222",
			connected: true,
		},
	}

	manager.AddAccount(client1)
	manager.AddAccount(client2)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	successCount := manager.Shutdown(ctx)

	// Verify all failed
	if successCount != 0 {
		t.Errorf("Expected 0 successful disconnections, got %d", successCount)
	}
}

func TestShutdown_SequentialDisconnect(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Track disconnect order
	var disconnectOrder []string
	var mu sync.Mutex

	// Create mock clients that record disconnect order
	for i := 0; i < 3; i++ {
		accountID := fmt.Sprintf("+%d000000000", i+1)
		client := &mockTelegramClientWithOrderTracking{
			mockTelegramClient: mockTelegramClient{
				accountID: accountID,
				connected: true,
			},
			onDisconnect: func(id string) {
				mu.Lock()
				disconnectOrder = append(disconnectOrder, id)
				mu.Unlock()
			},
		}
		manager.AddAccount(client)
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.Shutdown(ctx)

	// Verify sequential disconnect (order should match)
	if len(disconnectOrder) != 3 {
		t.Errorf("Expected 3 disconnects, got %d", len(disconnectOrder))
	}
}

// Helper: mockTelegramClientWithOrderTracking tracks disconnect order
type mockTelegramClientWithOrderTracking struct {
	mockTelegramClient
	onDisconnect func(string)
}

func (m *mockTelegramClientWithOrderTracking) Disconnect(ctx context.Context) error {
	if m.onDisconnect != nil {
		m.onDisconnect(m.accountID)
	}
	return nil
}

// Helper: mockTelegramClientWithSlowDisconnect simulates slow disconnect with delay
type mockTelegramClientWithSlowDisconnect struct {
	mockTelegramClient
	delay time.Duration
}

func (m *mockTelegramClientWithSlowDisconnect) Disconnect(ctx context.Context) error {
	select {
	case <-time.After(m.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Tests for shutdown flag and concurrent shutdown

func TestShutdown_ConcurrentCalls(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Add accounts
	for i := 0; i < 3; i++ {
		client := createTestClient(fmt.Sprintf("+%d000000000", i+1), true)
		manager.AddAccount(client)
	}

	// Call Shutdown concurrently
	var wg sync.WaitGroup
	results := make([]int, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			results[idx] = manager.Shutdown(ctx)
		}(i)
	}

	wg.Wait()

	// Only one Shutdown should succeed
	successfulShutdowns := 0
	totalDisconnected := 0
	for _, count := range results {
		if count > 0 {
			successfulShutdowns++
			totalDisconnected += count
		}
	}

	if successfulShutdowns != 1 {
		t.Errorf("Expected exactly 1 successful shutdown, got %d", successfulShutdowns)
	}
	if totalDisconnected != 3 {
		t.Errorf("Expected 3 total disconnects, got %d", totalDisconnected)
	}
}

func TestShutdown_OperationsAfterShutdown(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Add accounts
	client1 := createTestClient("+1111111111", true)
	manager.AddAccount(client1)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager.Shutdown(ctx)

	// Try to add account after shutdown
	client2 := createTestClient("+2222222222", true)
	err := manager.AddAccount(client2)
	if err == nil {
		t.Error("Expected error when adding account after shutdown")
	}

	// Try to get available account after shutdown
	_, err = manager.GetAvailableAccount()
	if err == nil {
		t.Error("Expected error when getting account after shutdown")
	}

	// Try to remove account after shutdown
	err = manager.RemoveAccount("+1111111111")
	if err == nil {
		t.Error("Expected error when removing account after shutdown")
	}

	// Try to initialize accounts after shutdown
	report := manager.InitializeAccounts(context.Background(), domain.AccountInitConfig{
		Accounts: []string{"+3333333333"},
		Logger:   createTestLogger(),
	})
	if report.SuccessfulAccounts != 0 {
		t.Errorf("Expected 0 initialized accounts after shutdown, got %d", report.SuccessfulAccounts)
	}
}

func TestShutdown_GetAllAccountsAfterShutdown(t *testing.T) {
	manager := NewAccountManager().(*accountManager)
	manager.logger = createTestLogger()

	// Add accounts
	client1 := createTestClient("+1111111111", true)
	client2 := createTestClient("+2222222222", true)
	manager.AddAccount(client1)
	manager.AddAccount(client2)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager.Shutdown(ctx)

	// GetAllAccounts should still work (read-only operation)
	accounts := manager.GetAllAccounts()
	if len(accounts) != 2 {
		t.Errorf("Expected GetAllAccounts to return 2 accounts even after shutdown, got %d", len(accounts))
	}
}

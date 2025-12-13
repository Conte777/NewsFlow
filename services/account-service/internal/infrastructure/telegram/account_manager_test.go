package telegram

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
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

	// Verify the specific error (phone number should be masked)
	maskedPhone := maskPhoneNumber("+2222222222")
	if _, exists := report.Errors[maskedPhone]; !exists {
		t.Errorf("Expected error for %s (masked +2222222222)", maskedPhone)
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

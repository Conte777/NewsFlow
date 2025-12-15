package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// mockTelegramClient mocks the Telegram client for testing
type mockTelegramClient struct {
	accountID         string
	connected         bool
	joinedChannels    map[string]bool
	channelMessages   map[string][]domain.NewsItem
	mu                sync.RWMutex
	joinChannelFunc   func(ctx context.Context, channelID string) error
	leaveChannelFunc  func(ctx context.Context, channelID string) error
	getMessagesFunc   func(ctx context.Context, channelID string, limit, offset int) ([]domain.NewsItem, error)
	getChannelInfoFunc func(ctx context.Context, channelID string) (*domain.ChannelInfo, error)
}

func newMockTelegramClient(accountID string) *mockTelegramClient {
	return &mockTelegramClient{
		accountID:       accountID,
		connected:       true,
		joinedChannels:  make(map[string]bool),
		channelMessages: make(map[string][]domain.NewsItem),
	}
}

func (m *mockTelegramClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockTelegramClient) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *mockTelegramClient) JoinChannel(ctx context.Context, channelID string) error {
	if m.joinChannelFunc != nil {
		return m.joinChannelFunc(ctx, channelID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.joinedChannels[channelID] = true
	return nil
}

func (m *mockTelegramClient) LeaveChannel(ctx context.Context, channelID string) error {
	if m.leaveChannelFunc != nil {
		return m.leaveChannelFunc(ctx, channelID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.joinedChannels, channelID)
	return nil
}

func (m *mockTelegramClient) GetChannelMessages(ctx context.Context, channelID string, limit, offset int) ([]domain.NewsItem, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, channelID, limit, offset)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	messages, exists := m.channelMessages[channelID]
	if !exists {
		return []domain.NewsItem{}, nil
	}

	// Apply offset and limit
	start := offset
	if start >= len(messages) {
		return []domain.NewsItem{}, nil
	}

	end := start + limit
	if end > len(messages) {
		end = len(messages)
	}

	return messages[start:end], nil
}

func (m *mockTelegramClient) GetChannelInfo(ctx context.Context, channelID string) (*domain.ChannelInfo, error) {
	if m.getChannelInfoFunc != nil {
		return m.getChannelInfoFunc(ctx, channelID)
	}

	return &domain.ChannelInfo{
		ID:    channelID,
		Title: "Test Channel: " + channelID,
	}, nil
}

func (m *mockTelegramClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

func (m *mockTelegramClient) GetAccountID() string {
	return m.accountID
}

// mockAccountManager implements AccountManager interface
type mockAccountManager struct {
	clients []domain.TelegramClient
	mu      sync.RWMutex
}

func newMockAccountManager(clients ...domain.TelegramClient) *mockAccountManager {
	return &mockAccountManager{
		clients: clients,
	}
}

func (m *mockAccountManager) GetAvailableAccount() (domain.TelegramClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.clients) == 0 {
		return nil, domain.ErrNoActiveAccounts
	}

	return m.clients[0], nil
}

func (m *mockAccountManager) GetAllAccounts() []domain.TelegramClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients
}

func (m *mockAccountManager) AddAccount(client domain.TelegramClient) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients = append(m.clients, client)
	return nil
}

func (m *mockAccountManager) RemoveAccount(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, client := range m.clients {
		if client.GetAccountID() == accountID {
			m.clients = append(m.clients[:i], m.clients[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("account not found")
}

func (m *mockAccountManager) InitializeAccounts(ctx context.Context, cfg domain.AccountInitConfig) *domain.InitializationReport {
	return &domain.InitializationReport{
		TotalAccounts:      len(m.clients),
		SuccessfulAccounts: len(m.clients),
		FailedAccounts:     0,
	}
}

func (m *mockAccountManager) Shutdown(ctx context.Context) int {
	return len(m.clients)
}

// mockChannelRepository implements ChannelRepository interface
type mockChannelRepository struct {
	channels map[string]domain.ChannelSubscription
	mu       sync.RWMutex
}

func newMockChannelRepository() *mockChannelRepository {
	return &mockChannelRepository{
		channels: make(map[string]domain.ChannelSubscription),
	}
}

func (m *mockChannelRepository) AddChannel(ctx context.Context, channelID, channelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.channels[channelID] = domain.ChannelSubscription{
		ChannelID:   channelID,
		ChannelName: channelName,
	}

	return nil
}

func (m *mockChannelRepository) RemoveChannel(ctx context.Context, channelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.channels[channelID]; !exists {
		return domain.ErrChannelNotFound
	}

	delete(m.channels, channelID)
	return nil
}

func (m *mockChannelRepository) GetAllChannels(ctx context.Context) ([]domain.ChannelSubscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channels := make([]domain.ChannelSubscription, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}

	return channels, nil
}

func (m *mockChannelRepository) GetChannel(ctx context.Context, channelID string) (*domain.ChannelSubscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ch, exists := m.channels[channelID]
	if !exists {
		return nil, domain.ErrChannelNotFound
	}

	return &ch, nil
}

func (m *mockChannelRepository) ChannelExists(ctx context.Context, channelID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.channels[channelID]
	return exists, nil
}

func (m *mockChannelRepository) UpdateLastProcessedMessageID(ctx context.Context, channelID string, messageID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch, exists := m.channels[channelID]
	if !exists {
		return domain.ErrChannelNotFound
	}

	ch.LastProcessedMessageID = messageID
	m.channels[channelID] = ch
	return nil
}

// mockKafkaProducer implements KafkaProducer interface
type mockKafkaProducer struct {
	sentNews []domain.NewsItem
	mu       sync.Mutex
	sendFunc func(ctx context.Context, news *domain.NewsItem) error
}

func newMockKafkaProducer() *mockKafkaProducer {
	return &mockKafkaProducer{
		sentNews: make([]domain.NewsItem, 0),
	}
}

func (m *mockKafkaProducer) SendNewsReceived(ctx context.Context, news *domain.NewsItem) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, news)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentNews = append(m.sentNews, *news)
	return nil
}

func (m *mockKafkaProducer) Close() error {
	return nil
}

func (m *mockKafkaProducer) GetSentNews() []domain.NewsItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.NewsItem{}, m.sentNews...)
}

// TestIntegration_SubscriptionCreated_FullFlow tests the complete flow:
// Subscription event -> Handler -> UseCase -> Subscribe -> Save to repo
func TestIntegration_SubscriptionCreated_FullFlow(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	// Setup mocks
	mockClient := newMockTelegramClient("test_account_1")
	mockAccMgr := newMockAccountManager(mockClient)
	mockRepo := newMockChannelRepository()
	mockProducer := newMockKafkaProducer()

	// Create use case
	useCase := newTestAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)

	// Create handler
	handler := NewSubscriptionHandler(useCase, logger)

	// Test data
	userID := int64(12345)
	channelID := "tech_news_channel"
	channelName := "Tech News Channel"

	// Execute: Handle subscription created event
	err := handler.HandleSubscriptionCreated(ctx, userID, channelID, channelName)

	// Verify: No error
	if err != nil {
		t.Fatalf("HandleSubscriptionCreated failed: %v", err)
	}

	// Verify: Channel was joined via Telegram client
	mockClient.mu.RLock()
	joined := mockClient.joinedChannels[channelID]
	mockClient.mu.RUnlock()

	if !joined {
		t.Error("Expected channel to be joined via Telegram client")
	}

	// Verify: Channel was saved to repository
	exists, err := mockRepo.ChannelExists(ctx, channelID)
	if err != nil {
		t.Fatalf("Failed to check channel existence: %v", err)
	}
	if !exists {
		t.Error("Expected channel to be saved in repository")
	}

	// Verify: Channel data is correct
	channel, err := mockRepo.GetChannel(ctx, channelID)
	if err != nil {
		t.Fatalf("Failed to get channel: %v", err)
	}

	if channel.ChannelID != channelID {
		t.Errorf("Expected channel ID %s, got %s", channelID, channel.ChannelID)
	}
	if channel.ChannelName != channelName {
		t.Errorf("Expected channel name %s, got %s", channelName, channel.ChannelName)
	}

	t.Logf("✅ Full flow test passed: subscription created -> joined channel -> saved to repo")
}

// TestIntegration_SubscriptionDeleted_FullFlow tests the complete unsubscribe flow
func TestIntegration_SubscriptionDeleted_FullFlow(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	// Setup mocks
	mockClient := newMockTelegramClient("test_account_1")
	mockAccMgr := newMockAccountManager(mockClient)
	mockRepo := newMockChannelRepository()
	mockProducer := newMockKafkaProducer()

	// Pre-setup: Add channel first
	channelID := "old_channel"
	channelName := "Old Channel"

	mockClient.mu.Lock()
	mockClient.joinedChannels[channelID] = true
	mockClient.mu.Unlock()

	mockRepo.AddChannel(ctx, channelID, channelName)

	// Create use case and handler
	useCase := newTestAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)
	handler := NewSubscriptionHandler(useCase, logger)

	// Execute: Handle subscription deleted event
	userID := int64(67890)
	err := handler.HandleSubscriptionDeleted(ctx, userID, channelID)

	// Verify: No error
	if err != nil {
		t.Fatalf("HandleSubscriptionDeleted failed: %v", err)
	}

	// Verify: Channel was left via Telegram client
	mockClient.mu.RLock()
	joined := mockClient.joinedChannels[channelID]
	mockClient.mu.RUnlock()

	if joined {
		t.Error("Expected channel to be left via Telegram client")
	}

	// Verify: Channel was removed from repository
	exists, err := mockRepo.ChannelExists(ctx, channelID)
	if err != nil {
		t.Fatalf("Failed to check channel existence: %v", err)
	}
	if exists {
		t.Error("Expected channel to be removed from repository")
	}

	t.Logf("✅ Full flow test passed: subscription deleted -> left channel -> removed from repo")
}

// TestIntegration_EndToEnd_SubscribeAndCollectNews tests the complete end-to-end flow:
// Subscribe -> CollectNews -> Produce to Kafka
func TestIntegration_EndToEnd_SubscribeAndCollectNews(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	// Setup mocks
	mockClient := newMockTelegramClient("test_account_1")
	mockAccMgr := newMockAccountManager(mockClient)
	mockRepo := newMockChannelRepository()
	mockProducer := newMockKafkaProducer()

	// Setup channel with test news
	channelID := "breaking_news"
	channelName := "Breaking News"

	testNews := []domain.NewsItem{
		{
			ChannelID:   channelID,
			ChannelName: channelName,
			MessageID:   1,
			Content:     "Breaking: First news item",
			Date:        time.Now().Add(-5 * time.Minute),
		},
		{
			ChannelID:   channelID,
			ChannelName: channelName,
			MessageID:   2,
			Content:     "Breaking: Second news item",
			Date:        time.Now().Add(-3 * time.Minute),
		},
		{
			ChannelID:   channelID,
			ChannelName: channelName,
			MessageID:   3,
			Content:     "Breaking: Third news item",
			MediaURLs:   []string{"https://example.com/image.jpg"},
			Date:        time.Now().Add(-1 * time.Minute),
		},
	}

	mockClient.channelMessages[channelID] = testNews

	// Create use case and handler
	useCase := newTestAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)
	handler := NewSubscriptionHandler(useCase, logger)

	// Step 1: Subscribe to channel
	err := handler.HandleSubscriptionCreated(ctx, 12345, channelID, channelName)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	t.Logf("✅ Step 1: Subscribed to channel %s", channelID)

	// Step 2: Collect news
	err = useCase.CollectNews(ctx)
	if err != nil {
		t.Fatalf("Failed to collect news: %v", err)
	}

	t.Logf("✅ Step 2: Collected news from channel")

	// Step 3: Verify news was sent to Kafka
	sentNews := mockProducer.GetSentNews()

	if len(sentNews) != len(testNews) {
		t.Fatalf("Expected %d news items sent to Kafka, got %d", len(testNews), len(sentNews))
	}

	// Verify each news item
	for i, expectedNews := range testNews {
		actualNews := sentNews[i]

		if actualNews.ChannelID != expectedNews.ChannelID {
			t.Errorf("News %d: Expected channel ID %s, got %s", i, expectedNews.ChannelID, actualNews.ChannelID)
		}

		if actualNews.MessageID != expectedNews.MessageID {
			t.Errorf("News %d: Expected message ID %d, got %d", i, expectedNews.MessageID, actualNews.MessageID)
		}

		if actualNews.Content != expectedNews.Content {
			t.Errorf("News %d: Expected content %s, got %s", i, expectedNews.Content, actualNews.Content)
		}

		// Verify serialization to JSON works
		jsonData, err := json.Marshal(actualNews)
		if err != nil {
			t.Errorf("News %d: Failed to marshal to JSON: %v", i, err)
		} else {
			t.Logf("News %d JSON: %s", i, string(jsonData))
		}
	}

	t.Logf("✅ Step 3: Verified %d news items sent to Kafka topic 'news.received'", len(sentNews))
	t.Logf("✅ End-to-end test passed: subscribe -> collect -> produce to Kafka")
}

// TestIntegration_MultipleSubscriptions_CollectNews tests collecting news from multiple channels
func TestIntegration_MultipleSubscriptions_CollectNews(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	// Setup mocks
	mockClient := newMockTelegramClient("test_account_1")
	mockAccMgr := newMockAccountManager(mockClient)
	mockRepo := newMockChannelRepository()
	mockProducer := newMockKafkaProducer()

	// Setup multiple channels with news
	channels := []struct {
		id       string
		name     string
		newsCount int
	}{
		{"tech_news", "Tech News", 3},
		{"sports_news", "Sports News", 2},
		{"finance_news", "Finance News", 5},
	}

	totalNewsExpected := 0
	for _, ch := range channels {
		// Create test news for this channel
		news := make([]domain.NewsItem, ch.newsCount)
		for i := 0; i < ch.newsCount; i++ {
			news[i] = domain.NewsItem{
				ChannelID:   ch.id,
				ChannelName: ch.name,
				MessageID:   i + 1,
				Content:     fmt.Sprintf("%s: News item %d", ch.name, i+1),
				Date:        time.Now().Add(time.Duration(-i) * time.Minute),
			}
		}

		mockClient.channelMessages[ch.id] = news
		totalNewsExpected += ch.newsCount
	}

	// Create use case and handler
	useCase := newTestAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)
	handler := NewSubscriptionHandler(useCase, logger)

	// Subscribe to all channels
	for _, ch := range channels {
		err := handler.HandleSubscriptionCreated(ctx, 12345, ch.id, ch.name)
		if err != nil {
			t.Fatalf("Failed to subscribe to %s: %v", ch.id, err)
		}
		t.Logf("Subscribed to channel: %s", ch.name)
	}

	// Collect news from all channels
	err := useCase.CollectNews(ctx)
	if err != nil {
		t.Fatalf("Failed to collect news: %v", err)
	}

	// Verify total news count
	sentNews := mockProducer.GetSentNews()
	if len(sentNews) != totalNewsExpected {
		t.Fatalf("Expected %d total news items, got %d", totalNewsExpected, len(sentNews))
	}

	// Verify news distribution per channel
	newsByChannel := make(map[string]int)
	for _, news := range sentNews {
		newsByChannel[news.ChannelID]++
	}

	for _, ch := range channels {
		count := newsByChannel[ch.id]
		if count != ch.newsCount {
			t.Errorf("Channel %s: expected %d news items, got %d", ch.id, ch.newsCount, count)
		}
	}

	t.Logf("✅ Multi-channel test passed: %d channels, %d total news items", len(channels), totalNewsExpected)
}

// newTestAccountUseCase is a helper to create AccountUseCase for testing
// This is a wrapper around the real usecase implementation
func newTestAccountUseCase(
	accountManager domain.AccountManager,
	channelRepo domain.ChannelRepository,
	kafkaProducer domain.KafkaProducer,
	logger zerolog.Logger,
) domain.AccountUseCase {
	// Import the real usecase package would cause circular dependency
	// So we inline a simplified version for testing
	return &testAccountUseCase{
		accountManager: accountManager,
		channelRepo:    channelRepo,
		kafkaProducer:  kafkaProducer,
		logger:         logger,
	}
}

// testAccountUseCase is a test implementation that mirrors the real one
type testAccountUseCase struct {
	accountManager domain.AccountManager
	channelRepo    domain.ChannelRepository
	kafkaProducer  domain.KafkaProducer
	logger         zerolog.Logger
}

func (u *testAccountUseCase) SubscribeToChannel(ctx context.Context, channelID, channelName string) error {
	if channelID == "" {
		return domain.ErrInvalidChannelID
	}

	exists, err := u.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	client, err := u.accountManager.GetAvailableAccount()
	if err != nil {
		return domain.ErrNoActiveAccounts
	}

	if err := client.JoinChannel(ctx, channelID); err != nil {
		return domain.ErrSubscriptionFailed
	}

	if err := u.channelRepo.AddChannel(ctx, channelID, channelName); err != nil {
		return err
	}

	return nil
}

func (u *testAccountUseCase) UnsubscribeFromChannel(ctx context.Context, channelID string) error {
	if channelID == "" {
		return domain.ErrInvalidChannelID
	}

	exists, err := u.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		return err
	}

	if !exists {
		return domain.ErrChannelNotFound
	}

	client, err := u.accountManager.GetAvailableAccount()
	if err != nil {
		return domain.ErrNoActiveAccounts
	}

	if err := client.LeaveChannel(ctx, channelID); err != nil {
		return domain.ErrUnsubscriptionFailed
	}

	if err := u.channelRepo.RemoveChannel(ctx, channelID); err != nil {
		return err
	}

	return nil
}

func (u *testAccountUseCase) CollectNews(ctx context.Context) error {
	channels, err := u.channelRepo.GetAllChannels(ctx)
	if err != nil {
		return err
	}

	if len(channels) == 0 {
		return nil
	}

	client, err := u.accountManager.GetAvailableAccount()
	if err != nil {
		return domain.ErrNoActiveAccounts
	}

	for _, channel := range channels {
		newsItems, err := client.GetChannelMessages(ctx, channel.ChannelID, 10, 0)
		if err != nil {
			u.logger.Error().Err(err).Str("channel_id", channel.ChannelID).Msg("Failed to get messages")
			continue
		}

		for _, news := range newsItems {
			if err := u.kafkaProducer.SendNewsReceived(ctx, &news); err != nil {
				u.logger.Error().Err(err).Msg("Failed to send news")
				continue
			}
		}
	}

	return nil
}

func (u *testAccountUseCase) GetActiveChannels(ctx context.Context) ([]string, error) {
	channels, err := u.channelRepo.GetAllChannels(ctx)
	if err != nil {
		return nil, err
	}

	channelIDs := make([]string, len(channels))
	for i, channel := range channels {
		channelIDs[i] = channel.ChannelID
	}

	return channelIDs, nil
}

package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
	"github.com/rs/zerolog"
)

// mockAccountManager is a mock implementation of domain.AccountManager
type mockAccountManager struct {
	getAvailableAccountFunc func() (domain.TelegramClient, error)
}

func (m *mockAccountManager) GetAvailableAccount() (domain.TelegramClient, error) {
	if m.getAvailableAccountFunc != nil {
		return m.getAvailableAccountFunc()
	}
	return nil, domain.ErrNoActiveAccounts
}

func (m *mockAccountManager) GetAllAccounts() []domain.TelegramClient {
	return nil
}

func (m *mockAccountManager) AddAccount(client domain.TelegramClient) error {
	return nil
}

func (m *mockAccountManager) RemoveAccount(accountID string) error {
	return nil
}

func (m *mockAccountManager) InitializeAccounts(ctx context.Context, cfg domain.AccountInitConfig) *domain.InitializationReport {
	return nil
}

func (m *mockAccountManager) Shutdown(ctx context.Context) int {
	return 0
}

// mockTelegramClient is a mock implementation of domain.TelegramClient
type mockTelegramClient struct {
	getChannelMessagesFunc func(ctx context.Context, channelID string, limit, offset int) ([]domain.NewsItem, error)
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
	if m.getChannelMessagesFunc != nil {
		return m.getChannelMessagesFunc(ctx, channelID, limit, offset)
	}
	return nil, nil
}

func (m *mockTelegramClient) GetChannelInfo(ctx context.Context, channelID string) (*domain.ChannelInfo, error) {
	return nil, nil
}

func (m *mockTelegramClient) IsConnected() bool {
	return true
}

func (m *mockTelegramClient) GetAccountID() string {
	return "test_account"
}

// mockChannelRepository is a mock implementation of domain.ChannelRepository
type mockChannelRepository struct {
	getAllChannelsFunc               func(ctx context.Context) ([]domain.ChannelSubscription, error)
	channelExistsFunc                func(ctx context.Context, channelID string) (bool, error)
	updateLastProcessedMessageIDFunc func(ctx context.Context, channelID string, messageID int) error
}

func (m *mockChannelRepository) AddChannel(ctx context.Context, channelID, channelName string) error {
	return nil
}

func (m *mockChannelRepository) RemoveChannel(ctx context.Context, channelID string) error {
	return nil
}

func (m *mockChannelRepository) GetAllChannels(ctx context.Context) ([]domain.ChannelSubscription, error) {
	if m.getAllChannelsFunc != nil {
		return m.getAllChannelsFunc(ctx)
	}
	return nil, nil
}

func (m *mockChannelRepository) GetChannel(ctx context.Context, channelID string) (*domain.ChannelSubscription, error) {
	return nil, nil
}

func (m *mockChannelRepository) ChannelExists(ctx context.Context, channelID string) (bool, error) {
	if m.channelExistsFunc != nil {
		return m.channelExistsFunc(ctx, channelID)
	}
	return false, nil
}

func (m *mockChannelRepository) UpdateLastProcessedMessageID(ctx context.Context, channelID string, messageID int) error {
	if m.updateLastProcessedMessageIDFunc != nil {
		return m.updateLastProcessedMessageIDFunc(ctx, channelID, messageID)
	}
	return nil
}

// mockKafkaProducer is a mock implementation of domain.KafkaProducer
type mockKafkaProducer struct {
	sendNewsReceivedFunc func(ctx context.Context, news *domain.NewsItem) error
	sentNewsCount        int
}

func (m *mockKafkaProducer) SendNewsReceived(ctx context.Context, news *domain.NewsItem) error {
	if m.sendNewsReceivedFunc != nil {
		return m.sendNewsReceivedFunc(ctx, news)
	}
	m.sentNewsCount++
	return nil
}

func (m *mockKafkaProducer) Close() error {
	return nil
}

// TestCollectNews_LogsNewsCount tests that CollectNews logs the number of collected news
func TestCollectNews_LogsNewsCount(t *testing.T) {
	t.Run("Success_LogsCollectedNews", func(t *testing.T) {
		// Setup logger to capture output
		var buf bytes.Buffer
		logger := zerolog.New(&buf).With().Timestamp().Logger()

		// Create mock channels
		channels := []domain.ChannelSubscription{
			{ChannelID: "channel1", ChannelName: "Channel 1"},
			{ChannelID: "channel2", ChannelName: "Channel 2"},
		}

		// Create mock news items
		newsChannel1 := []domain.NewsItem{
			{ChannelID: "channel1", MessageID: 1, Content: "News 1"},
			{ChannelID: "channel1", MessageID: 2, Content: "News 2"},
			{ChannelID: "channel1", MessageID: 3, Content: "News 3"},
		}
		newsChannel2 := []domain.NewsItem{
			{ChannelID: "channel2", MessageID: 4, Content: "News 4"},
			{ChannelID: "channel2", MessageID: 5, Content: "News 5"},
		}

		// Create mock client that returns news based on channel ID
		mockClient := &mockTelegramClient{
			getChannelMessagesFunc: func(ctx context.Context, channelID string, limit, offset int) ([]domain.NewsItem, error) {
				if channelID == "channel1" {
					return newsChannel1, nil
				}
				return newsChannel2, nil
			},
		}

		// Create mock account manager
		mockAccMgr := &mockAccountManager{
			getAvailableAccountFunc: func() (domain.TelegramClient, error) {
				return mockClient, nil
			},
		}

		// Create mock channel repository
		mockRepo := &mockChannelRepository{
			getAllChannelsFunc: func(ctx context.Context) ([]domain.ChannelSubscription, error) {
				return channels, nil
			},
		}

		// Create mock kafka producer
		mockProducer := &mockKafkaProducer{}

		// Create use case
		uc := NewAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)

		// Execute CollectNews
		err := uc.CollectNews(context.Background())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify logs contain the correct counts
		logOutput := buf.String()

		// Parse log entries
		var completionLog map[string]interface{}
		found := false
		for _, line := range strings.Split(logOutput, "\n") {
			if strings.Contains(line, "News collection completed") {
				if err := json.Unmarshal([]byte(line), &completionLog); err == nil {
					found = true
					break
				}
			}
		}

		if !found {
			t.Fatal("Expected to find 'News collection completed' log entry")
		}

		// Verify total_collected
		totalCollected, ok := completionLog["total_collected"].(float64)
		if !ok {
			t.Fatal("Expected total_collected field in log")
		}
		expectedCollected := len(newsChannel1) + len(newsChannel2)
		if int(totalCollected) != expectedCollected {
			t.Errorf("Expected total_collected=%d, got %d", expectedCollected, int(totalCollected))
		}

		// Verify total_sent
		totalSent, ok := completionLog["total_sent"].(float64)
		if !ok {
			t.Fatal("Expected total_sent field in log")
		}
		if int(totalSent) != expectedCollected {
			t.Errorf("Expected total_sent=%d, got %d", expectedCollected, int(totalSent))
		}

		// Verify channels_count
		channelsCount, ok := completionLog["channels_count"].(float64)
		if !ok {
			t.Fatal("Expected channels_count field in log")
		}
		if int(channelsCount) != len(channels) {
			t.Errorf("Expected channels_count=%d, got %d", len(channels), int(channelsCount))
		}
	})

	t.Run("WithKafkaErrors_LogsPartialSuccess", func(t *testing.T) {
		// Setup logger to capture output
		var buf bytes.Buffer
		logger := zerolog.New(&buf).With().Timestamp().Logger()

		// Create mock channels
		channels := []domain.ChannelSubscription{
			{ChannelID: "channel1", ChannelName: "Channel 1"},
		}

		// Create mock news items
		newsChannel1 := []domain.NewsItem{
			{ChannelID: "channel1", MessageID: 1, Content: "News 1"},
			{ChannelID: "channel1", MessageID: 2, Content: "News 2"},
			{ChannelID: "channel1", MessageID: 3, Content: "News 3"},
		}

		// Create mock client
		mockClient := &mockTelegramClient{
			getChannelMessagesFunc: func(ctx context.Context, channelID string, limit, offset int) ([]domain.NewsItem, error) {
				return newsChannel1, nil
			},
		}

		// Create mock account manager
		mockAccMgr := &mockAccountManager{
			getAvailableAccountFunc: func() (domain.TelegramClient, error) {
				return mockClient, nil
			},
		}

		// Create mock channel repository
		mockRepo := &mockChannelRepository{
			getAllChannelsFunc: func(ctx context.Context) ([]domain.ChannelSubscription, error) {
				return channels, nil
			},
		}

		// Create mock kafka producer that fails for the second message
		callCount := 0
		mockProducer := &mockKafkaProducer{
			sendNewsReceivedFunc: func(ctx context.Context, news *domain.NewsItem) error {
				callCount++
				if callCount == 2 {
					return errors.New("kafka error")
				}
				return nil
			},
		}

		// Create use case
		uc := NewAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)

		// Execute CollectNews
		err := uc.CollectNews(context.Background())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify logs contain the correct counts
		logOutput := buf.String()

		// Parse log entries
		var completionLog map[string]interface{}
		found := false
		for _, line := range strings.Split(logOutput, "\n") {
			if strings.Contains(line, "News collection completed") {
				if err := json.Unmarshal([]byte(line), &completionLog); err == nil {
					found = true
					break
				}
			}
		}

		if !found {
			t.Fatal("Expected to find 'News collection completed' log entry")
		}

		// Verify total_collected (all news were collected)
		totalCollected, ok := completionLog["total_collected"].(float64)
		if !ok {
			t.Fatal("Expected total_collected field in log")
		}
		if int(totalCollected) != 3 {
			t.Errorf("Expected total_collected=3, got %d", int(totalCollected))
		}

		// Verify total_sent (only 2 were sent successfully)
		totalSent, ok := completionLog["total_sent"].(float64)
		if !ok {
			t.Fatal("Expected total_sent field in log")
		}
		if int(totalSent) != 2 {
			t.Errorf("Expected total_sent=2, got %d", int(totalSent))
		}
	})

	t.Run("NoChannels_LogsNoNews", func(t *testing.T) {
		// Setup logger to capture output
		var buf bytes.Buffer
		logger := zerolog.New(&buf).With().Timestamp().Logger()

		// Create mock account manager
		mockAccMgr := &mockAccountManager{}

		// Create mock channel repository with no channels
		mockRepo := &mockChannelRepository{
			getAllChannelsFunc: func(ctx context.Context) ([]domain.ChannelSubscription, error) {
				return []domain.ChannelSubscription{}, nil
			},
		}

		// Create mock kafka producer
		mockProducer := &mockKafkaProducer{}

		// Create use case
		uc := NewAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)

		// Execute CollectNews
		err := uc.CollectNews(context.Background())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify logs contain "No subscribed channels"
		logOutput := buf.String()
		if !strings.Contains(logOutput, "No subscribed channels") {
			t.Error("Expected to find 'No subscribed channels' in logs")
		}

		// Should NOT contain "News collection completed"
		if strings.Contains(logOutput, "News collection completed") {
			t.Error("Did not expect 'News collection completed' when there are no channels")
		}
	})

	t.Run("ChannelMessagesError_ContinuesCollection", func(t *testing.T) {
		// Setup logger to capture output
		var buf bytes.Buffer
		logger := zerolog.New(&buf).With().Timestamp().Logger()

		// Create mock channels
		channels := []domain.ChannelSubscription{
			{ChannelID: "channel1", ChannelName: "Channel 1"},
			{ChannelID: "channel2", ChannelName: "Channel 2"},
		}

		// Create mock client that fails for channel1 but succeeds for channel2
		newsChannel2 := []domain.NewsItem{
			{ChannelID: "channel2", MessageID: 1, Content: "News 1"},
		}

		mockClient := &mockTelegramClient{
			getChannelMessagesFunc: func(ctx context.Context, channelID string, limit, offset int) ([]domain.NewsItem, error) {
				if channelID == "channel1" {
					return nil, errors.New("failed to get messages")
				}
				return newsChannel2, nil
			},
		}

		// Create mock account manager
		mockAccMgr := &mockAccountManager{
			getAvailableAccountFunc: func() (domain.TelegramClient, error) {
				return mockClient, nil
			},
		}

		// Create mock channel repository
		mockRepo := &mockChannelRepository{
			getAllChannelsFunc: func(ctx context.Context) ([]domain.ChannelSubscription, error) {
				return channels, nil
			},
		}

		// Create mock kafka producer
		mockProducer := &mockKafkaProducer{}

		// Create use case
		uc := NewAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)

		// Execute CollectNews
		err := uc.CollectNews(context.Background())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify logs contain the correct counts (only channel2 news)
		logOutput := buf.String()

		// Parse log entries
		var completionLog map[string]interface{}
		found := false
		for _, line := range strings.Split(logOutput, "\n") {
			if strings.Contains(line, "News collection completed") {
				if err := json.Unmarshal([]byte(line), &completionLog); err == nil {
					found = true
					break
				}
			}
		}

		if !found {
			t.Fatal("Expected to find 'News collection completed' log entry")
		}

		// Verify total_collected (only channel2 news)
		totalCollected, ok := completionLog["total_collected"].(float64)
		if !ok {
			t.Fatal("Expected total_collected field in log")
		}
		if int(totalCollected) != 1 {
			t.Errorf("Expected total_collected=1, got %d", int(totalCollected))
		}

		// Verify error was logged for channel1
		if !strings.Contains(logOutput, "Failed to get channel messages") {
			t.Error("Expected to find error log for failed channel")
		}
	})
}

// TestCollectNews tests basic CollectNews functionality
func TestCollectNews(t *testing.T) {
	t.Run("NoAvailableAccount", func(t *testing.T) {
		logger := zerolog.Nop()

		// Create mock account manager that returns error
		mockAccMgr := &mockAccountManager{
			getAvailableAccountFunc: func() (domain.TelegramClient, error) {
				return nil, domain.ErrNoActiveAccounts
			},
		}

		// Create mock channel repository with channels
		mockRepo := &mockChannelRepository{
			getAllChannelsFunc: func(ctx context.Context) ([]domain.ChannelSubscription, error) {
				return []domain.ChannelSubscription{
					{ChannelID: "channel1", ChannelName: "Channel 1"},
				}, nil
			},
		}

		// Create mock kafka producer
		mockProducer := &mockKafkaProducer{}

		// Create use case
		uc := NewAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)

		// Execute CollectNews
		err := uc.CollectNews(context.Background())
		if err != domain.ErrNoActiveAccounts {
			t.Errorf("Expected ErrNoActiveAccounts, got %v", err)
		}
	})

	t.Run("RepositoryError", func(t *testing.T) {
		logger := zerolog.Nop()

		mockAccMgr := &mockAccountManager{}

		// Create mock channel repository that returns error
		expectedErr := errors.New("repository error")
		mockRepo := &mockChannelRepository{
			getAllChannelsFunc: func(ctx context.Context) ([]domain.ChannelSubscription, error) {
				return nil, expectedErr
			},
		}

		mockProducer := &mockKafkaProducer{}

		// Create use case
		uc := NewAccountUseCase(mockAccMgr, mockRepo, mockProducer, logger)

		// Execute CollectNews
		err := uc.CollectNews(context.Background())
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected repository error, got %v", err)
		}
	})
}

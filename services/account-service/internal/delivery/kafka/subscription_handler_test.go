package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

// mockAccountUseCase is a mock implementation of domain.AccountUseCase for testing
type mockAccountUseCase struct {
	subscribeToChannelFunc      func(ctx context.Context, channelID, channelName string) error
	unsubscribeFromChannelFunc  func(ctx context.Context, channelID string) error
	subscribeToChannelCalls     []subscribeCall
	unsubscribeFromChannelCalls []unsubscribeCall
}

type subscribeCall struct {
	channelID   string
	channelName string
}

type unsubscribeCall struct {
	channelID string
}

func (m *mockAccountUseCase) SubscribeToChannel(ctx context.Context, channelID, channelName string) error {
	m.subscribeToChannelCalls = append(m.subscribeToChannelCalls, subscribeCall{
		channelID:   channelID,
		channelName: channelName,
	})

	if m.subscribeToChannelFunc != nil {
		return m.subscribeToChannelFunc(ctx, channelID, channelName)
	}
	return nil
}

func (m *mockAccountUseCase) UnsubscribeFromChannel(ctx context.Context, channelID string) error {
	m.unsubscribeFromChannelCalls = append(m.unsubscribeFromChannelCalls, unsubscribeCall{
		channelID: channelID,
	})

	if m.unsubscribeFromChannelFunc != nil {
		return m.unsubscribeFromChannelFunc(ctx, channelID)
	}
	return nil
}

func (m *mockAccountUseCase) CollectNews(ctx context.Context) error {
	return nil
}

func (m *mockAccountUseCase) GetActiveChannels(ctx context.Context) ([]string, error) {
	return nil, nil
}

// TestNewSubscriptionHandler tests handler creation
func TestNewSubscriptionHandler(t *testing.T) {
	mockUseCase := &mockAccountUseCase{}
	logger := zerolog.Nop()

	handler := NewSubscriptionHandler(mockUseCase, logger)

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	if handler.accountUseCase != mockUseCase {
		t.Error("Expected accountUseCase to be set")
	}
}

// TestHandleSubscriptionCreated tests subscription created event handling
func TestHandleSubscriptionCreated(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUseCase := &mockAccountUseCase{}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		ctx := context.Background()
		userID := int64(12345)
		channelID := "tech_news"
		channelName := "Tech News Channel"

		err := handler.HandleSubscriptionCreated(ctx, userID, channelID, channelName)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify SubscribeToChannel was called
		if len(mockUseCase.subscribeToChannelCalls) != 1 {
			t.Fatalf("Expected 1 call to SubscribeToChannel, got %d", len(mockUseCase.subscribeToChannelCalls))
		}

		call := mockUseCase.subscribeToChannelCalls[0]
		if call.channelID != channelID {
			t.Errorf("Expected channelID %s, got %s", channelID, call.channelID)
		}
		if call.channelName != channelName {
			t.Errorf("Expected channelName %s, got %s", channelName, call.channelName)
		}
	})

	t.Run("WithError", func(t *testing.T) {
		expectedErr := errors.New("subscription failed")
		mockUseCase := &mockAccountUseCase{
			subscribeToChannelFunc: func(ctx context.Context, channelID, channelName string) error {
				return expectedErr
			},
		}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		ctx := context.Background()
		err := handler.HandleSubscriptionCreated(ctx, 12345, "test_channel", "Test Channel")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		// Verify error is wrapped
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error to wrap %v, got %v", expectedErr, err)
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		// Test calling HandleSubscriptionCreated multiple times with same channel
		mockUseCase := &mockAccountUseCase{}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		ctx := context.Background()
		channelID := "duplicate_channel"
		channelName := "Duplicate Channel"

		// Call multiple times
		for i := 0; i < 3; i++ {
			err := handler.HandleSubscriptionCreated(ctx, 12345, channelID, channelName)
			if err != nil {
				t.Errorf("Call %d: Expected no error, got %v", i+1, err)
			}
		}

		// All calls should succeed (idempotency handled by use case)
		if len(mockUseCase.subscribeToChannelCalls) != 3 {
			t.Errorf("Expected 3 calls, got %d", len(mockUseCase.subscribeToChannelCalls))
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		mockUseCase := &mockAccountUseCase{
			subscribeToChannelFunc: func(ctx context.Context, channelID, channelName string) error {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					return nil
				}
			},
		}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := handler.HandleSubscriptionCreated(ctx, 12345, "test_channel", "Test Channel")

		// Should propagate context cancellation error
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})
}

// TestHandleSubscriptionDeleted tests subscription deleted event handling
func TestHandleSubscriptionDeleted(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUseCase := &mockAccountUseCase{}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		ctx := context.Background()
		userID := int64(67890)
		channelID := "old_channel"

		err := handler.HandleSubscriptionDeleted(ctx, userID, channelID)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify UnsubscribeFromChannel was called
		if len(mockUseCase.unsubscribeFromChannelCalls) != 1 {
			t.Fatalf("Expected 1 call to UnsubscribeFromChannel, got %d", len(mockUseCase.unsubscribeFromChannelCalls))
		}

		call := mockUseCase.unsubscribeFromChannelCalls[0]
		if call.channelID != channelID {
			t.Errorf("Expected channelID %s, got %s", channelID, call.channelID)
		}
	})

	t.Run("WithError", func(t *testing.T) {
		expectedErr := errors.New("unsubscription failed")
		mockUseCase := &mockAccountUseCase{
			unsubscribeFromChannelFunc: func(ctx context.Context, channelID string) error {
				return expectedErr
			},
		}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		ctx := context.Background()
		err := handler.HandleSubscriptionDeleted(ctx, 67890, "test_channel")

		if err == nil {
			t.Error("Expected error, got nil")
		}

		// Verify error is wrapped
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error to wrap %v, got %v", expectedErr, err)
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		// Test calling HandleSubscriptionDeleted multiple times with same channel
		mockUseCase := &mockAccountUseCase{}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		ctx := context.Background()
		channelID := "duplicate_delete_channel"

		// Call multiple times
		for i := 0; i < 3; i++ {
			err := handler.HandleSubscriptionDeleted(ctx, 67890, channelID)
			if err != nil {
				t.Errorf("Call %d: Expected no error, got %v", i+1, err)
			}
		}

		// All calls should succeed (idempotency handled by use case)
		if len(mockUseCase.unsubscribeFromChannelCalls) != 3 {
			t.Errorf("Expected 3 calls, got %d", len(mockUseCase.unsubscribeFromChannelCalls))
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		mockUseCase := &mockAccountUseCase{
			unsubscribeFromChannelFunc: func(ctx context.Context, channelID string) error {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					return nil
				}
			},
		}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := handler.HandleSubscriptionDeleted(ctx, 67890, "test_channel")

		// Should propagate context cancellation error
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})
}

// TestSubscriptionHandler_BothMethods tests that both methods are implemented
func TestSubscriptionHandler_BothMethods(t *testing.T) {
	mockUseCase := &mockAccountUseCase{}
	logger := zerolog.Nop()
	handler := NewSubscriptionHandler(mockUseCase, logger)

	ctx := context.Background()

	// Test HandleSubscriptionCreated
	err := handler.HandleSubscriptionCreated(ctx, 1, "channel1", "Channel 1")
	if err != nil {
		t.Errorf("HandleSubscriptionCreated failed: %v", err)
	}

	// Test HandleSubscriptionDeleted
	err = handler.HandleSubscriptionDeleted(ctx, 1, "channel1")
	if err != nil {
		t.Errorf("HandleSubscriptionDeleted failed: %v", err)
	}

	// Verify both methods were called
	if len(mockUseCase.subscribeToChannelCalls) != 1 {
		t.Errorf("Expected 1 subscribe call, got %d", len(mockUseCase.subscribeToChannelCalls))
	}
	if len(mockUseCase.unsubscribeFromChannelCalls) != 1 {
		t.Errorf("Expected 1 unsubscribe call, got %d", len(mockUseCase.unsubscribeFromChannelCalls))
	}
}

// TestSubscriptionHandler_ErrorWrapping tests that errors are properly wrapped
func TestSubscriptionHandler_ErrorWrapping(t *testing.T) {
	t.Run("SubscriptionCreatedErrorWrapping", func(t *testing.T) {
		originalErr := errors.New("original error")
		mockUseCase := &mockAccountUseCase{
			subscribeToChannelFunc: func(ctx context.Context, channelID, channelName string) error {
				return originalErr
			},
		}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		err := handler.HandleSubscriptionCreated(context.Background(), 1, "test", "Test")

		// Error should be wrapped with channel ID
		if err.Error() == originalErr.Error() {
			t.Error("Expected error to be wrapped, but it's not")
		}

		// Original error should be in the chain
		if !errors.Is(err, originalErr) {
			t.Error("Original error should be in error chain")
		}
	})

	t.Run("SubscriptionDeletedErrorWrapping", func(t *testing.T) {
		originalErr := errors.New("original error")
		mockUseCase := &mockAccountUseCase{
			unsubscribeFromChannelFunc: func(ctx context.Context, channelID string) error {
				return originalErr
			},
		}
		logger := zerolog.Nop()
		handler := NewSubscriptionHandler(mockUseCase, logger)

		err := handler.HandleSubscriptionDeleted(context.Background(), 1, "test")

		// Error should be wrapped with channel ID
		if err.Error() == originalErr.Error() {
			t.Error("Expected error to be wrapped, but it's not")
		}

		// Original error should be in the chain
		if !errors.Is(err, originalErr) {
			t.Error("Original error should be in error chain")
		}
	})
}

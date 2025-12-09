package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// TestLeaveChannel_Success tests successful channel leave
func TestLeaveChannel_Success(t *testing.T) {
	// This is an integration test that requires:
	// 1. Valid Telegram API credentials
	// 2. Active session
	// 3. Membership in a test channel
	//
	// To run this test:
	// - Set environment variables: TG_API_ID, TG_API_HASH, TG_PHONE
	// - Ensure you're a member of the test channel
	// - Run: go test -v -run TestLeaveChannel_Success
	//
	// Example:
	// export TG_API_ID=12345
	// export TG_API_HASH=abcdef
	// export TG_PHONE=+1234567890
	// go test -v -run TestLeaveChannel_Success

	t.Skip("Skipping integration test - requires valid Telegram credentials and active session")
}

// TestLeaveChannel_ChannelNotFound tests error handling for non-existent channel
func TestLeaveChannel_ChannelNotFound(t *testing.T) {
	// This is an integration test that requires:
	// 1. Valid Telegram API credentials
	// 2. Active session
	//
	// To run this test:
	// - Set environment variables: TG_API_ID, TG_API_HASH, TG_PHONE
	// - Run: go test -v -run TestLeaveChannel_ChannelNotFound
	//
	// The test will attempt to leave a non-existent channel and verify
	// that ErrChannelNotFound is returned

	t.Skip("Skipping integration test - requires valid Telegram credentials and active session")
}

// TestLeaveChannel_NotConnected tests error handling when client is not connected
func TestLeaveChannel_NotConnected(t *testing.T) {
	client := &MTProtoClient{
		connected: false,
	}

	ctx := context.Background()
	err := client.LeaveChannel(ctx, "@testchannel")

	if err != domain.ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got: %v", err)
	}
}

// TestLeaveChannel_InvalidChannelID tests validation of channel ID
func TestLeaveChannel_InvalidChannelID(t *testing.T) {
	client := &MTProtoClient{
		connected: true,
	}

	tests := []struct {
		name      string
		channelID string
		wantErr   error
	}{
		{
			name:      "empty channel ID",
			channelID: "",
			wantErr:   domain.ErrInvalidChannelID,
		},
		{
			name:      "invalid format",
			channelID: "invalid_channel",
			wantErr:   domain.ErrInvalidChannelID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := client.LeaveChannel(ctx, tt.channelID)

			if err != tt.wantErr {
				t.Errorf("Expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestLeaveChannel_ContextCancellation tests context cancellation handling
func TestLeaveChannel_ContextCancellation(t *testing.T) {
	// This test verifies that LeaveChannel respects context cancellation
	// when rate limiting is in effect

	t.Skip("Skipping - requires mocked API client for proper testing")
}

// Integration test example with real Telegram connection
// To use this, you need to:
// 1. Create a .env file or set environment variables
// 2. Uncomment and run manually
/*
func TestLeaveChannel_Integration(t *testing.T) {
	// Load credentials from environment
	apiID := getEnvAsInt("TG_API_ID")
	apiHash := getEnv("TG_API_HASH")
	phone := getEnv("TG_PHONE")
	testChannel := getEnv("TG_TEST_CHANNEL") // e.g., "@test_channel"

	if apiID == 0 || apiHash == "" || phone == "" || testChannel == "" {
		t.Skip("Skipping integration test - missing credentials")
	}

	// Create logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create client
	client, err := NewMTProtoClient(MTProtoClientConfig{
		APIID:       apiID,
		APIHash:     apiHash,
		PhoneNumber: phone,
		SessionDir:  "./test_sessions",
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Connect with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Test 1: Leave known channel
	t.Run("leave known channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := client.LeaveChannel(ctx, testChannel)
		if err != nil {
			t.Errorf("Failed to leave channel: %v", err)
		}
	})

	// Test 2: Leave non-existent channel
	t.Run("leave non-existent channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := client.LeaveChannel(ctx, "@nonexistent_channel_12345")
		if err != domain.ErrChannelNotFound {
			t.Errorf("Expected ErrChannelNotFound, got: %v", err)
		}
	})

	// Test 3: Leave channel user is not a member of
	t.Run("leave channel not a member of", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Should return nil (success) as the end result is the same
		err := client.LeaveChannel(ctx, "@some_public_channel")
		if err != nil {
			t.Errorf("Expected nil for non-member channel, got: %v", err)
		}
	})
}
*/

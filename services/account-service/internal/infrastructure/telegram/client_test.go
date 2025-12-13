package telegram

import (
	"context"
	"testing"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
	"github.com/gotd/td/tg"
	"github.com/rs/zerolog"
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

// TestGetChannelMessages_NotConnected tests error handling when client is not connected
func TestGetChannelMessages_NotConnected(t *testing.T) {
	client := &MTProtoClient{
		connected: false,
	}

	ctx := context.Background()
	_, err := client.GetChannelMessages(ctx, "@testchannel", 10, 0)

	if err != domain.ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got: %v", err)
	}
}

// TestGetChannelMessages_InvalidChannelID tests validation of channel ID
func TestGetChannelMessages_InvalidChannelID(t *testing.T) {
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
			_, err := client.GetChannelMessages(ctx, tt.channelID, 10, 0)

			if err != tt.wantErr {
				t.Errorf("Expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestGetChannelMessages_LimitAndOffset tests limit and offset validation
func TestGetChannelMessages_LimitAndOffset(t *testing.T) {
	// This test verifies that invalid limit and offset values are handled correctly
	// The implementation should:
	// - Use default limit (10) if limit <= 0
	// - Set offset to 0 if offset < 0

	t.Skip("Skipping - requires mocked API client for proper testing")
}

// Integration test for getting last 10 messages
// To use this, you need to:
// 1. Create a .env file or set environment variables
// 2. Uncomment and run manually
/*
func TestGetChannelMessages_Last10Messages(t *testing.T) {
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

	// Test: Get last 10 messages
	t.Run("get last 10 messages", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		newsItems, err := client.GetChannelMessages(ctx, testChannel, 10, 0)
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		// Verify we got results (assuming test channel has messages)
		if len(newsItems) == 0 {
			t.Log("Warning: No messages returned - test channel may be empty")
		}

		// Verify each message has required fields
		for i, item := range newsItems {
			if item.ChannelID != testChannel {
				t.Errorf("Message %d: expected channel ID %s, got %s", i, testChannel, item.ChannelID)
			}
			if item.ChannelName == "" {
				t.Errorf("Message %d: channel name is empty", i)
			}
			if item.MessageID == 0 {
				t.Errorf("Message %d: message ID is zero", i)
			}
			// Content can be empty for media-only messages
			if item.Date.IsZero() {
				t.Errorf("Message %d: date is zero", i)
			}

			// Verify date is in reasonable range (not in future, not too old)
			now := time.Now()
			if item.Date.After(now) {
				t.Errorf("Message %d: date is in the future: %v", i, item.Date)
			}
			// Check date is in ISO 8601 format by marshaling to JSON
			dateStr := item.Date.Format(time.RFC3339)
			if dateStr == "" {
				t.Errorf("Message %d: failed to format date as ISO 8601", i)
			}

			t.Logf("Message %d: ID=%d, Content=%q, MediaURLs=%v, Date=%v",
				i, item.MessageID, truncateString(item.Content, 50), item.MediaURLs, item.Date)
		}

		// Verify limit is respected
		if len(newsItems) > 10 {
			t.Errorf("Expected at most 10 messages, got %d", len(newsItems))
		}
	})

	// Test: Get messages with offset
	t.Run("get messages with offset", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get first 5 messages
		firstBatch, err := client.GetChannelMessages(ctx, testChannel, 5, 0)
		if err != nil {
			t.Fatalf("Failed to get first batch: %v", err)
		}

		// Get next 5 messages with offset
		secondBatch, err := client.GetChannelMessages(ctx, testChannel, 5, 5)
		if err != nil {
			t.Fatalf("Failed to get second batch: %v", err)
		}

		// Verify batches are different (if channel has enough messages)
		if len(firstBatch) > 0 && len(secondBatch) > 0 {
			if firstBatch[0].MessageID == secondBatch[0].MessageID {
				t.Errorf("First message in both batches has same ID - offset not working")
			}
		}

		t.Logf("First batch: %d messages, Second batch: %d messages",
			len(firstBatch), len(secondBatch))
	})

	// Test: Empty channel or excessive offset
	t.Run("handle excessive offset", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Request messages with very large offset
		newsItems, err := client.GetChannelMessages(ctx, testChannel, 10, 10000)
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		// Should return empty array, not error
		if newsItems == nil {
			t.Error("Expected empty array, got nil")
		}
		if len(newsItems) != 0 {
			t.Errorf("Expected 0 messages with excessive offset, got %d", len(newsItems))
		}
	})
}

func TestGetChannelMessages_MediaOnly(t *testing.T) {
	// Load credentials from environment
	apiID := getEnvAsInt("TG_API_ID")
	apiHash := getEnv("TG_API_HASH")
	phone := getEnv("TG_PHONE")
	testChannel := getEnv("TG_TEST_CHANNEL") // e.g., "@test_channel" with media messages

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

	// Test: Get messages and verify media extraction
	t.Run("verify media extraction", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		newsItems, err := client.GetChannelMessages(ctx, testChannel, 50, 0)
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		// Find messages with media
		var messagesWithMedia []domain.NewsItem
		for _, item := range newsItems {
			if len(item.MediaURLs) > 0 {
				messagesWithMedia = append(messagesWithMedia, item)
			}
		}

		if len(messagesWithMedia) == 0 {
			t.Log("Warning: No messages with media found - test channel may not have media messages")
			t.Log("To properly test media extraction, post photos/videos/documents to the test channel")
			return
		}

		t.Logf("Found %d messages with media out of %d total messages",
			len(messagesWithMedia), len(newsItems))

		// Verify media URL formats
		for i, item := range messagesWithMedia {
			t.Logf("Message %d (ID=%d):", i, item.MessageID)
			t.Logf("  Content: %q", truncateString(item.Content, 50))
			t.Logf("  Media URLs (%d):", len(item.MediaURLs))

			for j, url := range item.MediaURLs {
				t.Logf("    [%d] %s", j, url)

				// Verify URL format
				if url == "" {
					t.Errorf("Message %d, Media %d: URL is empty", i, j)
				}

				// Check for expected URL schemes
				validSchemes := []string{"photo://", "video://", "audio://", "document://", "http://", "https://", "geo:", "contact://"}
				hasValidScheme := false
				for _, scheme := range validSchemes {
					if len(url) >= len(scheme) && url[:len(scheme)] == scheme {
						hasValidScheme = true
						break
					}
				}

				if !hasValidScheme {
					t.Errorf("Message %d, Media %d: URL has unexpected scheme: %s", i, j, url)
				}
			}
		}
	})
}
*/

// Helper function to truncate string for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestExtractMediaURLs_Photo tests photo media extraction
func TestExtractMediaURLs_Photo(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaPhoto{
		Photo: &tg.Photo{ID: 12345},
	}

	urls := client.extractMediaURLs(media)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	expected := "photo://12345"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractMediaURLs_Video tests video document extraction
func TestExtractMediaURLs_Video(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			ID: 67890,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeVideo{},
				&tg.DocumentAttributeFilename{FileName: "test.mp4"},
			},
		},
	}

	urls := client.extractMediaURLs(media)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	expected := "video://67890:test.mp4"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractMediaURLs_Audio tests audio document extraction
func TestExtractMediaURLs_Audio(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			ID: 11111,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeAudio{},
				&tg.DocumentAttributeFilename{FileName: "song.mp3"},
			},
		},
	}

	urls := client.extractMediaURLs(media)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	expected := "audio://11111:song.mp3"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractMediaURLs_Document tests generic document extraction
func TestExtractMediaURLs_Document(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			ID: 22222,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeFilename{FileName: "file.pdf"},
			},
		},
	}

	urls := client.extractMediaURLs(media)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	expected := "document://22222:file.pdf"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractMediaURLs_WebPage tests webpage media extraction
func TestExtractMediaURLs_WebPage(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaWebPage{
		Webpage: &tg.WebPage{
			URL: "https://example.com/article",
			Photo: &tg.Photo{ID: 33333},
			Document: &tg.Document{ID: 44444},
		},
	}

	urls := client.extractMediaURLs(media)

	// Should extract webpage URL, photo, and document
	if len(urls) != 3 {
		t.Fatalf("expected 3 URLs, got %d: %v", len(urls), urls)
	}

	expectedURLs := map[string]bool{
		"https://example.com/article": true,
		"photo://33333":               true,
		"document://44444":             true,
	}

	for _, url := range urls {
		if !expectedURLs[url] {
			t.Errorf("unexpected URL: %s", url)
		}
	}
}

// TestExtractMediaURLs_Geo tests geographic location extraction
func TestExtractMediaURLs_Geo(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaGeo{
		Geo: &tg.GeoPoint{
			Lat:  40.748817,
			Long: -73.985428,
		},
	}

	urls := client.extractMediaURLs(media)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	expected := "geo:40.748817,-73.985428"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractMediaURLs_Contact tests contact extraction
func TestExtractMediaURLs_Contact(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaContact{
		PhoneNumber: "+1234567890",
	}

	urls := client.extractMediaURLs(media)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	expected := "contact://tel:+1234567890"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractMediaURLs_Poll tests that polls return empty slice
func TestExtractMediaURLs_Poll(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaPoll{
		Poll: tg.Poll{ID: 55555},
	}

	urls := client.extractMediaURLs(media)

	if len(urls) != 0 {
		t.Errorf("expected 0 URLs for poll, got %d", len(urls))
	}
}

// TestExtractMediaURLs_Empty tests empty/nil media
func TestExtractMediaURLs_Empty(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaEmpty{}

	urls := client.extractMediaURLs(media)

	if len(urls) != 0 {
		t.Errorf("expected 0 URLs for empty media, got %d", len(urls))
	}
}

// createTestLogger creates a no-op logger for tests
func createTestLogger() zerolog.Logger {
	return zerolog.New(nil).Level(zerolog.Disabled)
}

// TestGetChannelInfo_NotConnected tests error handling when client is not connected
func TestGetChannelInfo_NotConnected(t *testing.T) {
	client := &MTProtoClient{
		connected: false,
		logger:    createTestLogger(),
	}

	ctx := context.Background()
	_, err := client.GetChannelInfo(ctx, "@testchannel")

	if err != domain.ErrNotConnected {
		t.Errorf("Expected ErrNotConnected, got: %v", err)
	}
}

// TestGetChannelInfo_InvalidChannelID tests validation of channel ID
func TestGetChannelInfo_InvalidChannelID(t *testing.T) {
	client := &MTProtoClient{
		connected: true,
		logger:    createTestLogger(),
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
			_, err := client.GetChannelInfo(ctx, tt.channelID)

			if err != tt.wantErr {
				t.Errorf("Expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// Integration test for getting channel info
// To use this, you need to:
// 1. Create a .env file or set environment variables
// 2. Uncomment and run manually
/*
func TestGetChannelInfo_KnownChannel(t *testing.T) {
	// Load credentials from environment
	apiID := getEnvAsInt("TG_API_ID")
	apiHash := getEnv("TG_API_HASH")
	phone := getEnv("TG_PHONE")
	testChannel := getEnv("TG_TEST_CHANNEL") // e.g., "@telegram"

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

	// Test: Get channel info
	t.Run("get known channel info", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		info, err := client.GetChannelInfo(ctx, testChannel)
		if err != nil {
			t.Fatalf("Failed to get channel info: %v", err)
		}

		// Verify required fields
		if info.ID == "" {
			t.Error("Expected channel ID to be non-empty")
		}
		if info.Title == "" {
			t.Error("Expected channel title to be non-empty")
		}
		if info.Username == "" {
			t.Log("Warning: Channel username is empty (might be expected for some channels)")
		}

		// Log channel information
		t.Logf("Channel Info:")
		t.Logf("  ID: %s", info.ID)
		t.Logf("  Username: %s", info.Username)
		t.Logf("  Title: %s", info.Title)
		t.Logf("  About: %s", truncateString(info.About, 100))
		t.Logf("  Participants: %d", info.ParticipantsCount)
		t.Logf("  Verified: %v", info.IsVerified)
		t.Logf("  Restricted: %v", info.IsRestricted)
		t.Logf("  Created: %v", info.CreatedAt)

		// Verify participant count is reasonable
		if info.ParticipantsCount < 0 {
			t.Errorf("Participant count should not be negative: %d", info.ParticipantsCount)
		}

		// Verify created date is in the past
		if info.CreatedAt.After(time.Now()) {
			t.Errorf("Created date should be in the past: %v", info.CreatedAt)
		}
	})
}

func TestGetChannelInfo_PrivateChannel(t *testing.T) {
	// Load credentials from environment
	apiID := getEnvAsInt("TG_API_ID")
	apiHash := getEnv("TG_API_HASH")
	phone := getEnv("TG_PHONE")

	if apiID == 0 || apiHash == "" || phone == "" {
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

	// Test: Attempt to get private channel info
	t.Run("handle private channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Try to get info for a private/restricted channel
		// You'll need to replace this with a known private channel username
		privateChannel := "@some_private_channel_that_doesnt_exist"

		info, err := client.GetChannelInfo(ctx, privateChannel)

		// Should either return ErrChannelNotFound or ErrChannelPrivate
		if err == nil {
			// If no error, it means we got basic info (channel exists and is accessible)
			t.Logf("Channel info retrieved (channel is accessible): %s", info.Title)
		} else if err == domain.ErrChannelNotFound {
			t.Logf("Channel not found (expected for non-existent channels)")
		} else if err == domain.ErrChannelPrivate {
			t.Logf("Channel is private (expected for private channels)")
		} else {
			t.Logf("Got error: %v", err)
		}
	})

	// Test: Non-existent channel
	t.Run("handle non-existent channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := client.GetChannelInfo(ctx, "@nonexistent_channel_12345678")

		if err != domain.ErrChannelNotFound {
			t.Errorf("Expected ErrChannelNotFound for non-existent channel, got: %v", err)
		}
	})
}
*/

// Helper functions for environment variables (used in integration tests)
// Uncomment when running integration tests
/*
func getEnv(key string) string {
	return os.Getenv(key)
}

func getEnvAsInt(key string) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return 0
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0
	}
	return val
}
*/

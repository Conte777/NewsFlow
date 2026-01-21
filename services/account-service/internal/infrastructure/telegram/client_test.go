package telegram

import (
	"context"
	"testing"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/gotd/td/tg"
	"github.com/rs/zerolog"
)

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

// Helper function to truncate string for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestExtractSimpleMediaURLs_Photo tests that photo media returns empty slice (handled via S3)
func TestExtractSimpleMediaURLs_Photo(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaPhoto{
		Photo: &tg.Photo{
			ID:   12345,
			DCID: 2,
			Sizes: []tg.PhotoSizeClass{
				&tg.PhotoSize{Type: "y", W: 800, H: 600},
			},
		},
	}

	urls := client.extractSimpleMediaURLs(media)

	// Photo media is now handled via S3 upload, so extractSimpleMediaURLs returns empty
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs for photo (handled via S3), got %d", len(urls))
	}
}

// TestExtractSimpleMediaURLs_Video tests that video document returns empty slice (handled via S3)
func TestExtractSimpleMediaURLs_Video(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			ID:   67890,
			DCID: 2,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeVideo{},
				&tg.DocumentAttributeFilename{FileName: "test.mp4"},
			},
		},
	}

	urls := client.extractSimpleMediaURLs(media)

	// Document media is now handled via S3 upload, so extractSimpleMediaURLs returns empty
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs for video (handled via S3), got %d", len(urls))
	}
}

// TestExtractSimpleMediaURLs_Audio tests that audio document returns empty slice (handled via S3)
func TestExtractSimpleMediaURLs_Audio(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			ID:   11111,
			DCID: 2,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeAudio{},
				&tg.DocumentAttributeFilename{FileName: "song.mp3"},
			},
		},
	}

	urls := client.extractSimpleMediaURLs(media)

	// Document media is now handled via S3 upload, so extractSimpleMediaURLs returns empty
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs for audio (handled via S3), got %d", len(urls))
	}
}

// TestExtractSimpleMediaURLs_Document tests that generic document returns empty slice (handled via S3)
func TestExtractSimpleMediaURLs_Document(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			ID:   22222,
			DCID: 2,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeFilename{FileName: "file.pdf"},
			},
		},
	}

	urls := client.extractSimpleMediaURLs(media)

	// Document media is now handled via S3 upload, so extractSimpleMediaURLs returns empty
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs for document (handled via S3), got %d", len(urls))
	}
}

// TestExtractSimpleMediaURLs_WebPage tests webpage media extraction (only URL, no embedded media)
func TestExtractSimpleMediaURLs_WebPage(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaWebPage{
		Webpage: &tg.WebPage{
			URL: "https://example.com/article",
			Photo: &tg.Photo{
				ID:   33333,
				DCID: 2,
				Sizes: []tg.PhotoSizeClass{
					&tg.PhotoSize{Type: "y", W: 800, H: 600},
				},
			},
			Document: &tg.Document{
				ID:   44444,
				DCID: 2,
			},
		},
	}

	urls := client.extractSimpleMediaURLs(media)

	// extractSimpleMediaURLs now only extracts the webpage URL, not embedded photo/document
	if len(urls) != 1 {
		t.Fatalf("expected 1 URL (only webpage URL), got %d: %v", len(urls), urls)
	}

	expected := "https://example.com/article"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractSimpleMediaURLs_Geo tests geographic location extraction
func TestExtractSimpleMediaURLs_Geo(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaGeo{
		Geo: &tg.GeoPoint{
			Lat:  40.748817,
			Long: -73.985428,
		},
	}

	urls := client.extractSimpleMediaURLs(media)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	expected := "geo:40.748817,-73.985428"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractSimpleMediaURLs_Contact tests contact extraction
func TestExtractSimpleMediaURLs_Contact(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaContact{
		PhoneNumber: "+1234567890",
	}

	urls := client.extractSimpleMediaURLs(media)

	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}

	expected := "contact://tel:+1234567890"
	if urls[0] != expected {
		t.Errorf("expected %s, got %s", expected, urls[0])
	}
}

// TestExtractSimpleMediaURLs_Poll tests that polls return empty slice
func TestExtractSimpleMediaURLs_Poll(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaPoll{
		Poll: tg.Poll{ID: 55555},
	}

	urls := client.extractSimpleMediaURLs(media)

	if len(urls) != 0 {
		t.Errorf("expected 0 URLs for poll, got %d", len(urls))
	}
}

// TestExtractSimpleMediaURLs_Empty tests empty/nil media
func TestExtractSimpleMediaURLs_Empty(t *testing.T) {
	client := &MTProtoClient{
		logger: createTestLogger(),
	}

	media := &tg.MessageMediaEmpty{}

	urls := client.extractSimpleMediaURLs(media)

	if len(urls) != 0 {
		t.Errorf("expected 0 URLs for empty media, got %d", len(urls))
	}
}

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


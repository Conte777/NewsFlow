package telegram

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMTProtoClient tests client creation with various configurations
func TestNewMTProtoClient(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	sessionDir := t.TempDir()

	tests := []struct {
		name        string
		config      MTProtoClientConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: MTProtoClientConfig{
				APIID:       123456,
				APIHash:     "test_hash",
				PhoneNumber: "+1234567890",
				SessionDir:  sessionDir,
				Logger:      logger,
			},
			expectError: false,
		},
		{
			name: "missing API ID",
			config: MTProtoClientConfig{
				APIID:       0,
				APIHash:     "test_hash",
				PhoneNumber: "+1234567890",
				SessionDir:  sessionDir,
				Logger:      logger,
			},
			expectError: true,
			errorMsg:    "APIID is required",
		},
		{
			name: "missing API Hash",
			config: MTProtoClientConfig{
				APIID:       123456,
				APIHash:     "",
				PhoneNumber: "+1234567890",
				SessionDir:  sessionDir,
				Logger:      logger,
			},
			expectError: true,
			errorMsg:    "APIHash is required",
		},
		{
			name: "missing phone number",
			config: MTProtoClientConfig{
				APIID:       123456,
				APIHash:     "test_hash",
				PhoneNumber: "",
				SessionDir:  sessionDir,
				Logger:      logger,
			},
			expectError: true,
			errorMsg:    "PhoneNumber is required",
		},
		{
			name: "default session directory",
			config: MTProtoClientConfig{
				APIID:       123456,
				APIHash:     "test_hash",
				PhoneNumber: "+1234567890",
				SessionDir:  "",
				Logger:      logger,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewMTProtoClient(tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.config.APIID, client.apiID)
				assert.Equal(t, tt.config.APIHash, client.apiHash)
				assert.Equal(t, tt.config.PhoneNumber, client.phoneNumber)
				assert.False(t, client.connected)
			}
		})
	}
}

// TestMTProtoClient_IsConnected tests connection state tracking
func TestMTProtoClient_IsConnected(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	sessionDir := t.TempDir()

	client, err := NewMTProtoClient(MTProtoClientConfig{
		APIID:       123456,
		APIHash:     "test_hash",
		PhoneNumber: "+1234567890",
		SessionDir:  sessionDir,
		Logger:      logger,
	})
	require.NoError(t, err)

	// Initially not connected
	assert.False(t, client.IsConnected())

	// Simulate connection
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()

	assert.True(t, client.IsConnected())

	// Simulate disconnection
	client.mu.Lock()
	client.connected = false
	client.mu.Unlock()

	assert.False(t, client.IsConnected())
}

// TestFileSessionStorage tests session storage operations
func TestFileSessionStorage(t *testing.T) {
	sessionDir := t.TempDir()
	phoneNumber := "+1234567890"

	t.Run("create session storage", func(t *testing.T) {
		storage, err := NewFileSessionStorage(sessionDir, phoneNumber)
		require.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Equal(t, sessionDir, storage.sessionDir)
		assert.Equal(t, phoneNumber, storage.phoneNumber)
		assert.Contains(t, storage.filePath, phoneNumber)
	})

	t.Run("session does not exist initially", func(t *testing.T) {
		storage, err := NewFileSessionStorage(sessionDir, phoneNumber)
		require.NoError(t, err)
		assert.False(t, storage.SessionExists())
	})

	t.Run("store and load session", func(t *testing.T) {
		storage, err := NewFileSessionStorage(sessionDir, phoneNumber)
		require.NoError(t, err)

		ctx := context.Background()
		testData := []byte("test_session_data")

		// Store session
		err = storage.StoreSession(ctx, testData)
		require.NoError(t, err)
		assert.True(t, storage.SessionExists())

		// Load session
		loadedData, err := storage.LoadSession(ctx)
		require.NoError(t, err)
		assert.Equal(t, testData, loadedData)
	})

	t.Run("delete session", func(t *testing.T) {
		storage, err := NewFileSessionStorage(sessionDir, phoneNumber)
		require.NoError(t, err)

		ctx := context.Background()
		testData := []byte("test_session_data")

		// Store session
		err = storage.StoreSession(ctx, testData)
		require.NoError(t, err)
		assert.True(t, storage.SessionExists())

		// Delete session
		err = storage.DeleteSession()
		require.NoError(t, err)
		assert.False(t, storage.SessionExists())

		// Delete non-existent session should not error
		err = storage.DeleteSession()
		require.NoError(t, err)
	})

	t.Run("session file permissions", func(t *testing.T) {
		storage, err := NewFileSessionStorage(sessionDir, phoneNumber)
		require.NoError(t, err)

		ctx := context.Background()
		testData := []byte("test_session_data")

		// Store session
		err = storage.StoreSession(ctx, testData)
		require.NoError(t, err)

		// Check file permissions (should be 0600)
		info, err := os.Stat(storage.GetFilePath())
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})

	t.Run("session directory creation", func(t *testing.T) {
		nestedDir := filepath.Join(sessionDir, "nested", "sessions")
		storage, err := NewFileSessionStorage(nestedDir, phoneNumber)
		require.NoError(t, err)

		// Check directory exists
		info, err := os.Stat(nestedDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})
}

// TestConsoleCodeProvider tests code provider functionality
func TestConsoleCodeProvider(t *testing.T) {
	t.Run("create console code provider", func(t *testing.T) {
		provider := &ConsoleCodeProvider{}
		assert.NotNil(t, provider)
	})
}

// TestConnect_WithExistingSession simulates connection with existing session
func TestConnect_WithExistingSession(t *testing.T) {
	// Note: This is a unit test that cannot actually connect to Telegram
	// In a real scenario, you would use mocks or integration tests
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	sessionDir := t.TempDir()

	client, err := NewMTProtoClient(MTProtoClientConfig{
		APIID:       123456,
		APIHash:     "test_hash",
		PhoneNumber: "+1234567890",
		SessionDir:  sessionDir,
		Logger:      logger,
	})
	require.NoError(t, err)

	// Create a fake session file to simulate existing session
	ctx := context.Background()
	fakeSessionData := []byte("fake_session_data")
	err = client.sessionStorage.StoreSession(ctx, fakeSessionData)
	require.NoError(t, err)

	// Verify session exists
	assert.True(t, client.sessionStorage.SessionExists())

	// Note: Actual connection test would require real Telegram credentials
	// and would be done in integration tests, not unit tests
}

// TestConnect_InvalidPhoneCode simulates handling of invalid phone code
func TestConnect_InvalidPhoneCode(t *testing.T) {
	// Note: This is a conceptual test
	// Actual testing of invalid phone code would require integration tests
	// with real Telegram API or mocked telegram client

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	sessionDir := t.TempDir()

	client, err := NewMTProtoClient(MTProtoClientConfig{
		APIID:       123456,
		APIHash:     "test_hash",
		PhoneNumber: "+1234567890",
		SessionDir:  sessionDir,
		Logger:      logger,
	})
	require.NoError(t, err)
	assert.NotNil(t, client)

	// In a real test with mocked telegram client, we would:
	// 1. Mock the auth flow to return PHONE_CODE_INVALID error
	// 2. Call Connect()
	// 3. Verify it retries up to maxRetries times
	// 4. Verify proper error message is returned
}

// TestDisconnect tests disconnection logic
func TestDisconnect(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	sessionDir := t.TempDir()

	client, err := NewMTProtoClient(MTProtoClientConfig{
		APIID:       123456,
		APIHash:     "test_hash",
		PhoneNumber: "+1234567890",
		SessionDir:  sessionDir,
		Logger:      logger,
	})
	require.NoError(t, err)

	// Simulate connected state
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()

	assert.True(t, client.IsConnected())

	// Disconnect
	err = client.Disconnect()
	require.NoError(t, err)
	assert.False(t, client.IsConnected())

	// Disconnect again should not error
	err = client.Disconnect()
	require.NoError(t, err)
}

// TestConnect_Concurrent tests concurrent connection attempts
func TestConnect_Concurrent(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	sessionDir := t.TempDir()

	client, err := NewMTProtoClient(MTProtoClientConfig{
		APIID:       123456,
		APIHash:     "test_hash",
		PhoneNumber: "+1234567890",
		SessionDir:  sessionDir,
		Logger:      logger,
	})
	require.NoError(t, err)

	// Test concurrent IsConnected calls (should not panic or race)
	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = client.IsConnected()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// TestValidateChannelID tests channel ID validation
func TestValidateChannelID(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		wantErr   bool
	}{
		{
			name:      "valid username with @",
			channelID: "@testchannel",
			wantErr:   false,
		},
		{
			name:      "valid numeric ID",
			channelID: "123456789",
			wantErr:   false,
		},
		{
			name:      "empty channel ID",
			channelID: "",
			wantErr:   true,
		},
		{
			name:      "invalid format",
			channelID: "invalid@channel",
			wantErr:   true,
		},
		{
			name:      "alphanumeric without @",
			channelID: "abc123",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChannelID(tt.channelID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMaskPhoneNumber tests phone number masking for logs
func TestMaskPhoneNumber(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{
			name:     "normal phone number",
			phone:    "+1234567890",
			expected: "+1********90",
		},
		{
			name:     "short phone number",
			phone:    "+123",
			expected: "***",
		},
		{
			name:     "very short",
			phone:    "12",
			expected: "***",
		},
		{
			name:     "four digits",
			phone:    "1234",
			expected: "1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskPhoneNumber(tt.phone)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsNumeric tests numeric string validation
func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "all digits",
			input:    "123456",
			expected: true,
		},
		{
			name:     "with letters",
			input:    "123abc",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "with special chars",
			input:    "123-456",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSessionStorage_HashedFilename tests that phone numbers are hashed in filenames
func TestSessionStorage_HashedFilename(t *testing.T) {
	sessionDir := t.TempDir()
	phoneNumber := "+1234567890"

	storage, err := NewFileSessionStorage(sessionDir, phoneNumber)
	require.NoError(t, err)

	// Verify filename does not contain the phone number
	filename := filepath.Base(storage.GetFilePath())
	assert.NotContains(t, filename, phoneNumber, "phone number should not appear in filename")
	assert.NotContains(t, filename, "1234567890", "phone digits should not appear in filename")
	assert.Contains(t, filename, "session_", "filename should have session_ prefix")
	assert.Contains(t, filename, ".json", "filename should have .json extension")
}

// TestConsoleCodeProvider_Timeout tests code provider timeout
func TestConsoleCodeProvider_Timeout(t *testing.T) {
	provider := &ConsoleCodeProvider{}

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// This should timeout since we're not providing input
	_, err := provider.GetCode(ctx)
	assert.Error(t, err)
	// Error should be either timeout or context cancelled
	assert.True(t, err != nil)
}

// TestConsolePasswordProvider_Timeout tests password provider timeout
func TestConsolePasswordProvider_Timeout(t *testing.T) {
	provider := &ConsolePasswordProvider{}

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// This should timeout since we're not providing input
	_, err := provider.GetPassword(ctx)
	assert.Error(t, err)
	assert.True(t, err != nil)
}

// TestDisconnect_WithCancelFunc tests that Disconnect properly cancels context
func TestDisconnect_WithCancelFunc(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	sessionDir := t.TempDir()

	client, err := NewMTProtoClient(MTProtoClientConfig{
		APIID:       123456,
		APIHash:     "test_hash",
		PhoneNumber: "+1234567890",
		SessionDir:  sessionDir,
		Logger:      logger,
	})
	require.NoError(t, err)

	// Simulate connected state with cancel function
	ctx, cancel := context.WithCancel(context.Background())
	client.mu.Lock()
	client.connected = true
	client.cancelFunc = cancel
	client.mu.Unlock()

	// Track if cancel was called
	cancelCalled := false
	go func() {
		<-ctx.Done()
		cancelCalled = true
	}()

	// Disconnect
	err = client.Disconnect()
	require.NoError(t, err)
	assert.False(t, client.IsConnected())

	// Give a moment for cancel to propagate
	time.Sleep(10 * time.Millisecond)
	assert.True(t, cancelCalled, "cancel function should have been called")
}

// Benchmark session storage operations
func BenchmarkSessionStorage_StoreAndLoad(b *testing.B) {
	sessionDir := b.TempDir()
	storage, err := NewFileSessionStorage(sessionDir, "+1234567890")
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	testData := []byte("benchmark_session_data_with_some_length")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := storage.StoreSession(ctx, testData); err != nil {
			b.Fatal(err)
		}
		if _, err := storage.LoadSession(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

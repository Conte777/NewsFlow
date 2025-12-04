package telegram

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDisconnect tests basic disconnect functionality
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

	// Disconnect with 10 second timeout
	ctx := context.Background()
	disconnectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = client.Disconnect(disconnectCtx)
	require.NoError(t, err)
	assert.False(t, client.IsConnected())

	// Disconnect again should not error
	err = client.Disconnect(disconnectCtx)
	require.NoError(t, err)
}

// TestDisconnect_WithTimeout tests disconnect timeout mechanism
func TestDisconnect_WithTimeout(t *testing.T) {
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

	assert.True(t, client.IsConnected())

	// Start time
	start := time.Now()

	// Disconnect should complete within timeout
	ctx := context.Background()
	disconnectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = client.Disconnect(disconnectCtx)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.False(t, client.IsConnected())
	assert.Less(t, duration, 10*time.Second, "disconnect should complete within 10 seconds")
}

// TestReconnect_AfterDisconnect tests reconnection after disconnect
// This is the key acceptance criteria test
func TestReconnect_AfterDisconnect(t *testing.T) {
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

	// Test scenario:
	// 1. Simulate initial connection
	client.mu.Lock()
	client.connected = true
	ctx1, cancel1 := context.WithCancel(context.Background())
	client.cancelFunc = cancel1
	client.mu.Unlock()

	assert.True(t, client.IsConnected(), "should be connected initially")

	// 2. Disconnect
	ctx := context.Background()
	disconnectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = client.Disconnect(disconnectCtx)
	require.NoError(t, err, "first disconnect should succeed")
	assert.False(t, client.IsConnected(), "should not be connected after disconnect")

	// 3. Verify state is properly reset
	client.mu.RLock()
	assert.Nil(t, client.cancelFunc, "cancelFunc should be nil after disconnect")
	assert.Nil(t, client.client, "client should be nil after disconnect")
	assert.Nil(t, client.api, "api should be nil after disconnect")
	assert.Nil(t, client.runDone, "runDone should be nil after disconnect")
	assert.False(t, client.connected, "connected flag should be false")
	assert.False(t, client.disconnecting, "disconnecting flag should be false")
	client.mu.RUnlock()

	// 4. Simulate reconnection
	client.mu.Lock()
	client.connected = true
	ctx2, cancel2 := context.WithCancel(context.Background())
	client.cancelFunc = cancel2
	client.runDone = make(chan struct{})
	client.mu.Unlock()

	assert.True(t, client.IsConnected(), "should be connected after reconnection")

	// 5. Disconnect again
	disconnectCtx2, cancel2nd := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2nd()

	// Close runDone to simulate Run() completion
	close(client.runDone)

	err = client.Disconnect(disconnectCtx2)
	require.NoError(t, err, "second disconnect should succeed")
	assert.False(t, client.IsConnected(), "should not be connected after second disconnect")
}

// TestIsConnected tests the IsConnected status check
func TestIsConnected(t *testing.T) {
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

// TestIsConnected_Concurrent tests concurrent calls to IsConnected
func TestIsConnected_Concurrent(t *testing.T) {
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

// TestDisconnect_Concurrent tests concurrent disconnect calls
func TestDisconnect_Concurrent(t *testing.T) {
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
	ctx, cancel := context.WithCancel(context.Background())
	client.cancelFunc = cancel
	client.runDone = make(chan struct{})
	client.mu.Unlock()

	// Close runDone to simulate Run() completion
	close(client.runDone)

	// Call Disconnect concurrently
	const goroutines = 10
	done := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			disconnectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			done <- client.Disconnect(disconnectCtx)
		}()
	}

	// All should succeed without error or panic
	for i := 0; i < goroutines; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	assert.False(t, client.IsConnected())
}

// TestSessionPreservation tests that session is preserved across disconnect
func TestSessionPreservation(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	sessionDir := t.TempDir()
	phoneNumber := "+1234567890"

	// Create client and session storage
	client, err := NewMTProtoClient(MTProtoClientConfig{
		APIID:       123456,
		APIHash:     "test_hash",
		PhoneNumber: phoneNumber,
		SessionDir:  sessionDir,
		Logger:      logger,
	})
	require.NoError(t, err)

	// Store a test session
	ctx := context.Background()
	testSessionData := []byte("test_session_data_preserved")
	err = client.sessionStorage.StoreSession(ctx, testSessionData)
	require.NoError(t, err)

	// Simulate connection and disconnection
	client.mu.Lock()
	client.connected = true
	cancelCtx, cancel := context.WithCancel(context.Background())
	client.cancelFunc = cancel
	client.runDone = make(chan struct{})
	client.mu.Unlock()

	// Close runDone to simulate Run() completion
	close(client.runDone)

	// Disconnect
	ctx := context.Background()
	disconnectCtx, disconnectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer disconnectCancel()

	err = client.Disconnect(disconnectCtx)
	require.NoError(t, err)

	// Verify session still exists after disconnect
	assert.True(t, client.sessionStorage.SessionExists(), "session file should exist after disconnect")

	// Verify session data is preserved
	loadedData, err := client.sessionStorage.LoadSession(ctx)
	require.NoError(t, err)
	assert.Equal(t, testSessionData, loadedData, "session data should be preserved after disconnect")
}

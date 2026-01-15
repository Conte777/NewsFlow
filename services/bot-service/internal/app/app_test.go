package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestCreateApp(t *testing.T) {
	// Set required environment variables for test
	os.Setenv("TELEGRAM_BOT_TOKEN", "test-token-123")
	os.Setenv("KAFKA_BROKERS", "localhost:9093")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("KAFKA_BROKERS")
	}()

	// Validate fx dependency graph
	require.NoError(t, fx.ValidateApp(CreateApp()))
}

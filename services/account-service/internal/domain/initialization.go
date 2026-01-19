package domain

import "github.com/rs/zerolog"

// InitializationReport contains statistics about account initialization
//
// This report is returned by AccountManager.InitializeAccounts() and provides
// detailed information about which accounts were successfully initialized
// and which failed with specific errors.
//
// Example usage:
//
//	report := manager.InitializeAccounts(ctx, cfg)
//	if report.FailedAccounts > 0 {
//	    for phone, err := range report.Errors {
//	        log.Error().Err(err).Str("phone", phone).Msg("Failed to initialize account")
//	    }
//	}
type InitializationReport struct {
	// TotalAccounts is the total number of accounts attempted to initialize
	TotalAccounts int

	// SuccessfulAccounts is the number of accounts successfully initialized and connected
	SuccessfulAccounts int

	// FailedAccounts is the number of accounts that failed to initialize
	FailedAccounts int

	// Errors maps masked phone numbers to their initialization errors
	// Phone numbers are masked for security (e.g., "+1***90")
	Errors map[string]error
}

// AccountInitConfig holds configuration for account initialization
//
// This configuration is used by AccountManager.InitializeAccounts() to
// load and initialize multiple Telegram accounts in parallel.
type AccountInitConfig struct {
	// APIID is the Telegram API application ID
	APIID int

	// APIHash is the Telegram API application hash
	APIHash string

	// Accounts is the list of phone numbers to initialize
	// Phone numbers should be in international format (e.g., "+1234567890")
	Accounts []string

	// Logger is the logger instance for initialization logging
	Logger zerolog.Logger

	// MaxConcurrent is the maximum number of accounts to initialize concurrently
	// If 0, defaults to 10 to prevent resource exhaustion
	MaxConcurrent int
}

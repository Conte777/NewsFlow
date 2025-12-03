package telegram

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// CodeProvider defines interface for providing authentication codes
type CodeProvider interface {
	GetCode(ctx context.Context) (string, error)
}

// PasswordProvider defines interface for providing 2FA passwords
type PasswordProvider interface {
	GetPassword(ctx context.Context) (string, error)
}

// ConsoleCodeProvider implements CodeProvider using stdin
type ConsoleCodeProvider struct{}

// GetCode prompts user for authentication code via console with timeout
func (p *ConsoleCodeProvider) GetCode(ctx context.Context) (string, error) {
	fmt.Print("Enter authentication code: ")

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		code, err := reader.ReadString('\n')
		if err != nil {
			errChan <- fmt.Errorf("failed to read code: %w", err)
			return
		}
		codeChan <- strings.TrimSpace(code)
	}()

	select {
	case code := <-codeChan:
		return code, nil
	case err := <-errChan:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("code input cancelled: %w", ctx.Err())
	case <-time.After(2 * time.Minute):
		return "", fmt.Errorf("code input timeout")
	}
}

// ConsolePasswordProvider implements PasswordProvider using stdin
type ConsolePasswordProvider struct{}

// GetPassword prompts user for 2FA password via console with timeout
func (p *ConsolePasswordProvider) GetPassword(ctx context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")

	passwordChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		password, err := reader.ReadString('\n')
		if err != nil {
			errChan <- fmt.Errorf("failed to read password: %w", err)
			return
		}
		passwordChan <- strings.TrimSpace(password)
	}()

	select {
	case password := <-passwordChan:
		return password, nil
	case err := <-errChan:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("password input cancelled: %w", ctx.Err())
	case <-time.After(2 * time.Minute):
		return "", fmt.Errorf("password input timeout")
	}
}

// authenticateWithRetry performs authentication with exponential backoff for FloodWait
func (c *MTProtoClient) authenticateWithRetry(ctx context.Context, maxRetries int) error {
	var lastErr error
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := c.performAuthentication(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check for non-retryable errors that should fail immediately
		if isNonRetryableError(err) {
			c.logger.Error().Err(err).Msg("non-retryable authentication error")
			return fmt.Errorf("authentication failed with non-retryable error: %w", err)
		}

		// Check if it's a FloodWait error
		var floodWait *tgerr.Error
		if errors.As(err, &floodWait) && floodWait.Code == 420 {
			// Extract wait time from error message (FLOOD_WAIT_X where X is seconds)
			waitDuration := time.Duration(floodWait.Argument) * time.Second
			c.logger.Warn().
				Int("attempt", attempt+1).
				Dur("wait_duration", waitDuration).
				Msg("flood wait detected, waiting before retry")

			select {
			case <-time.After(waitDuration):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Check if it's a session revoked error
		if tgerr.Is(err, "SESSION_REVOKED") {
			c.logger.Error().Msg("session has been revoked, need to re-authenticate")
			// Delete old session
			if err := c.sessionStorage.DeleteSession(); err != nil {
				c.logger.Warn().Err(err).Msg("failed to delete revoked session")
			}
			// Try again with fresh session
			continue
		}

		// Check if it's an invalid phone code error
		if tgerr.Is(err, "PHONE_CODE_INVALID") {
			c.logger.Error().Msg("invalid phone code provided")
			// For invalid code, we should retry immediately (up to maxRetries)
			if attempt < maxRetries-1 {
				c.logger.Info().Msg("please try entering the code again")
				continue
			}
			return fmt.Errorf("authentication failed after %d attempts: invalid phone code", maxRetries)
		}

		// For other errors, apply exponential backoff
		delay := baseDelay * (1 << attempt) // 1s, 2s, 4s, 8s...
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		c.logger.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Dur("retry_delay", delay).
			Msg("authentication failed, retrying")

		select {
		case <-time.After(delay):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("authentication failed after %d attempts: %w", maxRetries, lastErr)
}

// isNonRetryableError checks if an error is non-retryable and should fail immediately
func isNonRetryableError(err error) bool {
	// List of errors that should not be retried
	nonRetryableErrors := []string{
		"PHONE_NUMBER_BANNED",      // Phone number is banned from Telegram
		"PHONE_NUMBER_INVALID",     // Phone number format is invalid
		"API_ID_INVALID",           // Invalid API ID
		"API_ID_PUBLISHED_FLOOD",   // API ID has been published and is rate limited
		"AUTH_TOKEN_INVALID",       // Invalid auth token
		"PASSWORD_HASH_INVALID",    // Invalid 2FA password (different from wrong code)
		"PHONE_NUMBER_OCCUPIED",    // Phone number is already registered
		"PHONE_PASSWORD_PROTECTED", // Should use password auth instead
	}

	for _, nonRetryable := range nonRetryableErrors {
		if tgerr.Is(err, nonRetryable) {
			return true
		}
	}

	return false
}

// performAuthentication performs a single authentication attempt
func (c *MTProtoClient) performAuthentication(ctx context.Context) error {
	codeProvider := &ConsoleCodeProvider{}
	passwordProvider := &ConsolePasswordProvider{}

	flow := auth.NewFlow(
		auth.Constant(
			c.phoneNumber,
			"",
			auth.CodeAuthenticatorFunc(func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
				c.logger.Info().Msg("authentication code has been sent")
				return codeProvider.GetCode(ctx)
			}),
		),
		auth.SendCodeOptions{},
	)

	err := c.client.Auth().IfNecessary(ctx, flow)
	if err != nil {
		// Check if 2FA is required
		if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
			c.logger.Info().Msg("2FA is enabled, requesting password")
			password, err := passwordProvider.GetPassword(ctx)
			if err != nil {
				return fmt.Errorf("failed to get 2FA password: %w", err)
			}

			// Authenticate with password
			_, err = c.client.Auth().Password(ctx, password)
			if err != nil {
				c.logger.Error().Err(err).Msg("2FA authentication failed")
				return fmt.Errorf("2FA authentication failed: %w", err)
			}

			c.logger.Info().Msg("2FA authentication successful")
			return nil
		}
		return err
	}

	c.logger.Info().Msg("authentication successful")
	return nil
}

// waitForAuthComplete waits for authentication to complete or timeout
func (c *MTProtoClient) waitForAuthComplete(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("authentication timeout: %w", ctx.Err())
		case <-ticker.C:
			status, err := c.client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to check auth status: %w", err)
			}
			if status.Authorized {
				return nil
			}
		}
	}
}

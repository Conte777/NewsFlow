// Package telegram contains Telegram bot infrastructure
package telegram

import (
	"context"
	"fmt"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

// Bot wraps the Telegram bot for infrastructure layer
type Bot struct {
	bot    *tgbot.Bot
	logger zerolog.Logger
}

// NewBot creates a new Telegram bot wrapper
func NewBot(token string, logger zerolog.Logger) (*Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}

	opts := []tgbot.Option{
		tgbot.WithDefaultHandler(defaultHandler),
	}

	bot, err := tgbot.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	logger.Info().Msg("Telegram bot created successfully")

	return &Bot{
		bot:    bot,
		logger: logger,
	}, nil
}

// Raw returns the underlying telegram bot for handler registration
func (b *Bot) Raw() *tgbot.Bot {
	return b.bot
}

// Start starts the bot (blocking call)
func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info().Msg("Starting Telegram bot...")
	b.bot.Start(ctx)
	b.logger.Info().Msg("Telegram bot stopped")
	return nil
}

// Stop stops the bot
func (b *Bot) Stop() error {
	b.logger.Info().Msg("Stopping Telegram bot...")
	return nil
}

// defaultHandler handles messages without commands
func defaultHandler(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	_, _ = bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "🤖 Используйте команды для взаимодействия с ботом. Напишите /help для списка доступных команд.",
	})
}

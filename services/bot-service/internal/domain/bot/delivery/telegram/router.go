// Package telegram contains Telegram delivery layer
package telegram

import (
	"context"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

// Router registers Telegram bot handlers
type Router struct {
	handlers *Handlers
	logger   zerolog.Logger
}

// NewRouter creates new Telegram router
func NewRouter(handlers *Handlers, logger zerolog.Logger) *Router {
	return &Router{
		handlers: handlers,
		logger:   logger,
	}
}

// RegisterRoutes registers all command handlers on the bot
func (r *Router) RegisterRoutes(bot *tgbot.Bot) {
	// Register command handlers
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, r.handlers.HandleStart)
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/help", tgbot.MatchTypeExact, r.handlers.HandleHelp)
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/subscribe", tgbot.MatchTypePrefix, r.handlers.HandleSubscribe)
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/unsubscribe", tgbot.MatchTypePrefix, r.handlers.HandleUnsubscribe)
	bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/list", tgbot.MatchTypeExact, r.handlers.HandleList)

	r.logger.Info().Msg("All Telegram command handlers registered successfully")
}

// DefaultHandler handles messages without commands
func DefaultHandler(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	_, _ = bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "🤖 Используйте команды для взаимодействия с ботом. Напишите /help для списка доступных команд.",
	})
}

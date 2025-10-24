package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/telegram-news-feed/bot-service/config"
	"github.com/yourusername/telegram-news-feed/bot-service/internal/infrastructure/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	log := logger.New(cfg.Logging.Level)
	log.Info().
		Str("service", cfg.Service.Name).
		Str("port", cfg.Service.Port).
		Msg("Starting bot service")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO: Initialize dependencies
	// - Kafka producer
	// - Kafka consumer
	// - Telegram bot
	// - Use case
	// - Handlers

	log.Info().Msg("Bot service initialized successfully")

	// TODO: Start bot
	// TODO: Start Kafka consumer

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("Shutting down bot service...")

	// TODO: Cleanup
	// - Stop bot
	// - Close Kafka connections

	log.Info().Msg("Bot service stopped")
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/config"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/logger"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/repository/memory"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/usecase"
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
		Msg("Starting account service")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize channel repository
	channelRepo := memory.NewChannelRepository()

	// TODO: Initialize Telegram clients (MTProto)
	// TODO: Initialize account manager
	// TODO: Initialize Kafka producer
	// TODO: Initialize use case
	_ = channelRepo
	_ = usecase.NewAccountUseCase

	// TODO: Initialize Kafka consumer
	// TODO: Start consuming subscription events

	log.Info().Msg("Account service initialized successfully")

	// TODO: Start periodic news collection
	ticker := time.NewTicker(cfg.News.PollInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				log.Debug().Msg("Collecting news from channels...")
				// TODO: Call use case to collect news
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("Shutting down account service...")

	// TODO: Cleanup
	// - Close Telegram clients
	// - Close Kafka connections

	log.Info().Msg("Account service stopped")
}

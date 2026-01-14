package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/telegram-news-feed/news-service/config"
	"github.com/yourusername/telegram-news-feed/news-service/internal/infrastructure/database"
	"github.com/yourusername/telegram-news-feed/news-service/internal/infrastructure/logger"
	"github.com/yourusername/telegram-news-feed/news-service/internal/repository/postgres"
	"github.com/yourusername/telegram-news-feed/news-service/internal/usecase"
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
		Msg("Starting news service")

	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	log.Info().Msg("Database connected successfully")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize repositories
	newsRepo := postgres.NewNewsRepository(db)
	deliveredRepo := postgres.NewDeliveredNewsRepository(db)

	// TODO: Initialize Kafka producer
	// TODO: Initialize subscription service client
	// TODO: Initialize use case
	_ = newsRepo
	_ = deliveredRepo
	_ = usecase.NewNewsUseCase

	// TODO: Initialize Kafka consumer
	// TODO: Start consuming messages

	log.Info().Msg("News service initialized successfully")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("Shutting down news service...")

	// TODO: Cleanup
	// - Close Kafka connections
	// - Close database connection

	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	log.Info().Msg("News service stopped")
}

package main

import (
	//"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Conte777/NewsFlow/services/subscription-service/config"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/database"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/logger"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/repository/postgres"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/usecase"
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
		Msg("Starting subscription service")

	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	log.Info().Msg("Database connected successfully")

	// Create context with cancellation
	//ctx, cancel := context.WithCancel(context.Background()) //временно отключить ctx до появления Kafka/UC
	//defer cancel()

	// Initialize repository
	subscriptionRepo := postgres.NewSubscriptionRepository(db)

	// TODO: Initialize Kafka producer
	// TODO: Initialize use case
	_ = subscriptionRepo
	_ = usecase.NewSubscriptionUseCase

	// TODO: Initialize Kafka consumer
	// TODO: Start consuming messages

	log.Info().Msg("Subscription service initialized successfully")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("Shutting down subscription service...")

	// TODO: Cleanup
	// - Close Kafka connections
	// - Close database connection

	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	log.Info().Msg("Subscription service stopped")
}

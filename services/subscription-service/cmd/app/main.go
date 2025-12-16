package main

import (
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
	// --- Load configuration from .env ---
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}

	// --- Initialize logger ---
	log := logger.New(cfg.Logging.Level)

	// --- Log loaded config (without secrets) ---
	log.Info().
		Str("service", cfg.Service.Name).
		Str("port", cfg.Service.Port).
		Str("db_host", cfg.Database.Host).
		Str("db_name", cfg.Database.DBName).
		Strs("kafka_brokers", cfg.Kafka.Brokers).
		Str("kafka_group", cfg.Kafka.GroupID).
		Str("log_level", cfg.Logging.Level).
		Msg("configuration loaded successfully")

	log.Info().
		Str("service", cfg.Service.Name).
		Msg("starting subscription service")

	// --- Initialize database ---
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	log.Info().Msg("database connected successfully")

	// --- Initialize repository ---
	subscriptionRepo := postgres.NewSubscriptionRepository(db)

	// TODO: Initialize Kafka producer
	// TODO: Initialize use case
	_ = subscriptionRepo
	_ = usecase.NewSubscriptionUseCase

	log.Info().Msg("subscription service initialized successfully")

	// --- Graceful shutdown ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("shutting down subscription service")

	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	log.Info().Msg("subscription service stopped")
}

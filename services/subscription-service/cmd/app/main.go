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

// Global logger
var log = logger.New("info")

func main() {
	// --- Load configuration ---
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}

	// --- Initialize global logger based on config ---
	log = logger.New(cfg.Logging.Level)
	log.Info().
		Str("service", cfg.Service.Name).
		Str("port", cfg.Service.Port).
		Msg("logger initialized successfully")

	// --- Log loaded configuration (without secrets) ---
	log.Info().
		Str("service", cfg.Service.Name).
		Str("port", cfg.Service.Port).
		Str("db_host", cfg.Database.Host).
		Str("db_name", cfg.Database.DBName).
		Strs("kafka_brokers", cfg.Kafka.Brokers).
		Str("kafka_group", cfg.Kafka.GroupID).
		Str("log_level", cfg.Logging.Level).
		Msg("configuration loaded successfully")

	// --- Initialize database ---
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	log.Info().Msg("database connected successfully")

	// --- Run migrations ---
	log.Info().Msg("running database migrations")

	if err := database.RunMigrations(db, cfg.Database); err != nil {
		log.Fatal().Err(err).Msg("failed to run database migrations")
	}

	log.Info().Msg("database migrations completed successfully")

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

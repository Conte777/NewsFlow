package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Conte777/NewsFlow/services/subscription-service/config"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/delivery/kafka"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/database"
	kafkaInfra "github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/kafka"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/infrastructure/logger"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/repository/postgres"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/usecase"
)

// Global logger
var log = logger.New("info")

func main() {
	// ------------------------------------------------------------------
	// Load configuration
	// ------------------------------------------------------------------
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}

	// ------------------------------------------------------------------
	// Initialize global logger
	// ------------------------------------------------------------------
	log = logger.New(cfg.Logging.Level)
	log.Info().
		Str("service", cfg.Service.Name).
		Str("port", cfg.Service.Port).
		Msg("logger initialized successfully")

	log.Info().
		Str("service", cfg.Service.Name).
		Str("port", cfg.Service.Port).
		Str("db_host", cfg.Database.Host).
		Str("db_name", cfg.Database.DBName).
		Strs("kafka_brokers", cfg.Kafka.Brokers).
		Str("kafka_group", cfg.Kafka.GroupID).
		Msg("configuration loaded successfully")

	// ------------------------------------------------------------------
	// Initialize database + migrations
	// ------------------------------------------------------------------
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	log.Info().Msg("database connected successfully")

	if err := database.RunMigrations(db, cfg.Database); err != nil {
		log.Fatal().Err(err).Msg("failed to run database migrations")
	}
	log.Info().Msg("database migrations completed successfully")

	// ------------------------------------------------------------------
	// Repository
	// ------------------------------------------------------------------
	subscriptionRepo := postgres.NewSubscriptionRepository(db)

	// ------------------------------------------------------------------
	// Kafka producer
	// ------------------------------------------------------------------
	producer, err := kafkaInfra.NewKafkaProducer(cfg.Kafka.Brokers, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize kafka producer")
	}
	log.Info().Msg("kafka producer initialized successfully")

	// ------------------------------------------------------------------
	// Use case
	// ------------------------------------------------------------------
	producerAdapter := kafkaInfra.NewProducerAdapter(producer)

	subscriptionUseCase := usecase.NewSubscriptionUseCase(
		subscriptionRepo,
		producerAdapter,
		log,
	)

	// ------------------------------------------------------------------
	// Event handler
	// ------------------------------------------------------------------
	eventHandler := kafka.NewSubscriptionEventHandler(
		subscriptionUseCase,
		log,
	)
	log.Info().Msg("subscription event handler initialized successfully")

	// ------------------------------------------------------------------
	// Kafka consumer
	// ------------------------------------------------------------------
	topics := []string{
		"subscription.created",
		"subscription.cancelled",
		"subscription.updated",
	}

	consumer, err := kafkaInfra.NewKafkaConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.GroupID,
		topics,
		eventHandler,
		log,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize kafka consumer")
	}
	log.Info().Msg("kafka consumer initialized successfully")

	// ------------------------------------------------------------------
	// Start consumer
	// ------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer.Start(ctx)
	log.Info().Msg("subscription service started")

	// ------------------------------------------------------------------
	// Graceful shutdown
	// ------------------------------------------------------------------
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("shutdown signal received")

	cancel()

	if err := consumer.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close kafka consumer")
	}

	if err := producer.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close kafka producer")
	}

	sqlDB, _ := db.DB()
	if sqlDB != nil {
		_ = sqlDB.Close()
	}

	if err := producerAdapter.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close kafka producer")
	}

	log.Info().Msg("subscription service stopped")
}

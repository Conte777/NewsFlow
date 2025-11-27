package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Conte777/newsflow/services/bot-service/config"
	"github.com/Conte777/newsflow/services/bot-service/internal/delivery/telegram"
	"github.com/Conte777/newsflow/services/bot-service/internal/domain"
	"github.com/Conte777/newsflow/services/bot-service/internal/infrastructure/kafka"
	"github.com/Conte777/newsflow/services/bot-service/internal/infrastructure/logger"
	"github.com/Conte777/newsflow/services/bot-service/internal/usecase"
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

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ===== INITIALIZE DEPENDENCIES =====

	// Initialize Kafka Producer
	log.Info().Msg("Initializing Kafka producer...")
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Kafka producer")
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing Kafka producer")
		}
	}()

	// Initialize Use Case first (нужен для Telegram Handler)
	log.Info().Msg("Initializing bot use case...")
	botUseCase := usecase.NewBotUseCase(kafkaProducer, nil, log) // Пока передаем nil для TelegramBot

	// Initialize Telegram Bot (теперь с 3 аргументами)
	log.Info().Msg("Initializing Telegram bot...")
	telegramBot, err := telegram.NewHandler(cfg.Telegram.BotToken, log, botUseCase)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Telegram bot")
	}
	defer func() {
		telegramBot.Stop()
		log.Info().Msg("Telegram bot stopped")
	}()

	// Update use case with actual TelegramBot
	botUseCase.SetTelegramBot(telegramBot)

	// Initialize Kafka Consumer
	log.Info().Msg("Initializing Kafka consumer...")
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Kafka consumer")
	}
	defer func() {
		if err := kafkaConsumer.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing Kafka consumer")
		}
	}()

	// Wire consumer to use case - handler для обработки новостей из Kafka
	newsHandler := func(news *domain.NewsMessage) error {
		log.Info().
			Str("news_id", news.ID).
			Int64("user_id", news.UserID).
			Str("channel_id", news.ChannelID).
			Msg("Received news from Kafka, sending to user")

		return botUseCase.SendNews(ctx, news)
	}

	log.Info().Msg("✅ All components initialized successfully")

	// ===== START COMPONENTS =====

	var wg sync.WaitGroup

	// Start Telegram Bot
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("Starting Telegram bot...")

		if err := telegramBot.Start(ctx); err != nil {
			log.Error().Err(err).Msg("Telegram bot stopped with error")
		} else {
			log.Info().Msg("Telegram bot stopped gracefully")
		}
	}()

	// Start Kafka Consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("Starting Kafka consumer...")

		if err := kafkaConsumer.ConsumeNewsDelivery(ctx, newsHandler); err != nil {
			log.Error().Err(err).Msg("Kafka consumer stopped with error")
		} else {
			log.Info().Msg("Kafka consumer stopped gracefully")
		}
	}()

	// Health check goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Health checker stopped")
				return
			case <-ticker.C:
				log.Debug().
					Str("service", cfg.Service.Name).
					Msg("Service is healthy")
			}
		}
	}()

	// Test Kafka producer (только для разработки)
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Ждем немного чтобы все компоненты запустились
		time.Sleep(3 * time.Second)

		// Тестовое сообщение только если не продакшн
		if cfg.Logging.Level == "debug" {
			log.Info().Msg("Sending test subscription event...")

			testSubscription := &domain.Subscription{
				UserID:      123456789,
				ChannelID:   "@test_channel",
				ChannelName: "Test Channel",
			}

			if err := kafkaProducer.SendSubscriptionCreated(ctx, testSubscription); err != nil {
				log.Error().Err(err).Msg("Failed to send test event")
			} else {
				log.Info().Msg("✅ Test event sent successfully!")
			}
		}
	}()

	log.Info().Msg("🎉 Bot service started successfully!")
	log.Info().Msg("📱 Telegram bot is listening for messages...")
	log.Info().Msg("📨 Kafka consumer is waiting for news...")
	log.Info().Msg("💡 Press Ctrl+C to stop the service")

	// ===== GRACEFUL SHUTDOWN =====

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for either signal or context cancellation
	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	case <-ctx.Done():
		log.Info().Msg("Context cancelled")
	}

	log.Info().Msg("🛑 Shutting down bot service...")

	// Send cancellation signal to all components
	cancel()

	// Wait for all components to stop with timeout
	shutdownDone := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		close(shutdownDone)
	}()

	// Wait for graceful shutdown or timeout
	select {
	case <-shutdownDone:
		log.Info().Msg("✅ All components stopped gracefully")
	case <-time.After(15 * time.Second):
		log.Warn().Msg("⏰ Timeout waiting for components to stop - forcing shutdown")
	}

	log.Info().Msg("👋 Bot service stopped")
}

// HealthCheck provides a simple health check endpoint (для будущего использования)
func HealthCheck() bool {
	// Здесь можно добавить проверки:
	// - Подключение к Kafka
	// - Подключение к Telegram API
	// - Состояние внутренних компонентов
	return true
}

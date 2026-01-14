package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/config"
	httpDelivery "github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/delivery/http"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/delivery/kafka"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/logger"
	kafkaInfra "github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/kafka"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/metrics"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/telegram"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/repository/memory"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/usecase"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/utils"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// 2. Initialize logger
	log := logger.New(cfg.Logging.Level)
	log.Info().
		Str("service", cfg.Service.Name).
		Str("port", cfg.Service.Port).
		Msg("Starting account service")

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup for tracking goroutines
	var wg sync.WaitGroup

	// 3. Initialize Account Manager
	log.Info().Msg("Initializing account manager...")
	accountManager := telegram.NewAccountManager()

	// Initialize Telegram accounts from configuration
	initCfg := domain.AccountInitConfig{
		APIID:      cfg.Telegram.APIID,
		APIHash:    cfg.Telegram.APIHash,
		SessionDir: cfg.Telegram.SessionDir,
		Accounts:   cfg.Telegram.Accounts,
		Logger:     log,
	}

	report := accountManager.InitializeAccounts(ctx, initCfg)

	// Log initialization report
	if report.SuccessfulAccounts > 0 {
		log.Info().
			Int("successful", report.SuccessfulAccounts).
			Int("failed", report.FailedAccounts).
			Int("total", report.TotalAccounts).
			Msg("Account initialization completed")
	} else {
		log.Warn().
			Int("failed", report.FailedAccounts).
			Int("total", report.TotalAccounts).
			Msg("No accounts successfully initialized")
	}

	// Log specific errors if any
	for phone, err := range report.Errors {
		log.Error().
			Str("phone", utils.MaskPhoneNumber(phone)).
			Err(err).
			Msg("Failed to initialize account")
	}

	// CRITICAL: Check minimum required accounts
	if report.SuccessfulAccounts < cfg.Telegram.MinRequiredAccounts {
		log.Fatal().
			Int("required", cfg.Telegram.MinRequiredAccounts).
			Int("successful", report.SuccessfulAccounts).
			Int("total", report.TotalAccounts).
			Msg("Insufficient accounts initialized - cannot start service")
	}

	// 4. Validate Kafka Brokers availability
	log.Info().Msg("Validating Kafka brokers availability...")
	if err := kafkaInfra.ValidateBrokers(cfg.Kafka.Brokers); err != nil {
		log.Fatal().Err(err).Msg("Kafka brokers are not available")
	}
	log.Info().Msg("Kafka brokers validated successfully")

	// 5. Initialize Kafka Producer
	log.Info().Msg("Initializing Kafka producer...")
	producerCfg := kafkaInfra.ProducerConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.TopicNewsReceived,
		Logger:  log,
	}

	kafkaProducer, err := kafkaInfra.NewKafkaProducer(producerCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Kafka producer")
	}

	// 6. Initialize Channel Repository
	log.Info().Msg("Initializing channel repository...")
	channelRepo := memory.NewChannelRepository()

	// 7. Initialize Use Case
	log.Info().Msg("Initializing use case...")
	accountUseCase := usecase.NewAccountUseCase(
		accountManager,
		channelRepo,
		kafkaProducer,
		log,
	)

	// 8. Initialize Kafka Consumer + Subscription Handler
	log.Info().Msg("Initializing Kafka consumer and subscription handler...")

	// Create subscription handler
	subscriptionHandler := kafka.NewSubscriptionHandler(accountUseCase, log)

	// Create Kafka consumer
	consumerCfg := kafka.ConsumerConfig{
		Brokers: cfg.Kafka.Brokers,
		Logger:  log,
		Handler: subscriptionHandler,
	}

	kafkaConsumer, err := kafka.NewKafkaConsumer(consumerCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Kafka consumer")
	}

	// 9. Initialize HTTP Server for health checks
	log.Info().Msg("Initializing HTTP server for health checks...")

	// Create health handler with all components
	healthHandler := httpDelivery.NewHealthHandler(
		accountManager,
		kafkaProducer,
		kafkaConsumer,
		log,
	)

	// Create HTTP mux and register endpoints
	mux := http.NewServeMux()
	mux.Handle("/health", healthHandler)
	mux.Handle("/metrics", promhttp.Handler())

	// Create HTTP server
	httpServer := httpDelivery.NewServer(cfg.Service.Port, mux, &log)

	// Start HTTP server
	if err := httpServer.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start HTTP server")
	}

	log.Info().
		Str("port", cfg.Service.Port).
		Msg("HTTP server started successfully")

	// 10. Initialize metrics
	log.Info().Msg("Initializing metrics...")

	// Update account metrics
	activeAccountCount := accountManager.GetActiveAccountCount()
	totalAccountCount := len(accountManager.GetAllAccounts())
	metrics.DefaultMetrics.UpdateAccounts(activeAccountCount, totalAccountCount)

	// Update subscription metrics
	channels, err := channelRepo.GetAllChannels(ctx)
	if err == nil {
		metrics.DefaultMetrics.UpdateActiveSubscriptions(len(channels))
	}

	log.Info().
		Int("active_accounts", activeAccountCount).
		Int("total_accounts", totalAccountCount).
		Int("active_subscriptions", len(channels)).
		Msg("Metrics initialized")

	// 11. Start periodic news collection
	log.Info().
		Dur("interval", cfg.News.PollInterval).
		Msg("Starting news collection ticker...")

	ticker := time.NewTicker(cfg.News.PollInterval)
	defer ticker.Stop()

	// Prevent overlapping news collection executions
	var isCollecting atomic.Bool

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("stack", string(debug.Stack())).
					Msg("News collection goroutine panic recovered")
			}
		}()

		for {
			select {
			case <-ticker.C:
				// Skip if previous collection is still in progress
				if !isCollecting.CompareAndSwap(false, true) {
					log.Warn().Msg("Previous news collection still in progress, skipping tick")
					continue
				}

				log.Debug().Msg("Collecting news from channels...")

				// Collect news from all subscribed channels with timeout
				func() {
					defer isCollecting.Store(false)

					collectCtx, collectCancel := context.WithTimeout(ctx, cfg.News.CollectionTimeout)
					defer collectCancel()

					if err := accountUseCase.CollectNews(collectCtx); err != nil {
						log.Error().Err(err).Msg("Failed to collect news")
					}
				}()

			case <-ctx.Done():
				log.Info().Msg("News collection stopped")
				return
			}
		}
	}()

	// Start consuming subscription events in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("stack", string(debug.Stack())).
					Msg("Kafka consumer goroutine panic recovered")
			}
		}()

		log.Info().Msg("Starting Kafka consumer...")
		if err := kafkaConsumer.ConsumeSubscriptionEvents(ctx, subscriptionHandler); err != nil {
			if err == context.Canceled {
				log.Info().Msg("Kafka consumer stopped by context cancellation")
			} else {
				log.Error().Err(err).Msg("Kafka consumer error")
			}
		}
	}()

	log.Info().Msg("Account service initialized successfully")

	// Wait for interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info().Msg("Received shutdown signal, starting graceful shutdown...")

	// Cancel context to stop all goroutines
	cancel()

	// Wait for goroutines to stop with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("All goroutines stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Warn().Msg("Timeout waiting for goroutines to stop")
	}

	// Explicit shutdown sequence (not using defer to control order)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Service.ShutdownTimeout)
	defer shutdownCancel()

	// 1. Shutdown HTTP Server (stop accepting new health check requests)
	log.Info().Msg("Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error shutting down HTTP server")
	}

	// 2. Close Kafka Consumer (stop accepting new messages)
	log.Info().Msg("Closing Kafka consumer...")
	if err := kafkaConsumer.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing Kafka consumer")
	}

	// 3. Close Kafka Producer (flush pending messages)
	log.Info().Msg("Closing Kafka producer...")
	if err := kafkaProducer.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing Kafka producer")
	}

	// 4. Shutdown Account Manager (disconnect all Telegram clients)
	log.Info().Msg("Shutting down account manager...")
	disconnectedCount := accountManager.Shutdown(shutdownCtx)
	log.Info().
		Int("disconnected", disconnectedCount).
		Msg("Account manager shutdown completed")

	log.Info().Msg("Account service stopped successfully")
}

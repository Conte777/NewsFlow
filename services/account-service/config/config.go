package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

// Config holds all configuration for the account service
type Config struct {
	Database DatabaseConfig
	Telegram TelegramConfig
	Kafka    KafkaConfig
	Logging  LoggingConfig
	Service  ServiceConfig
	News     NewsConfig
}

// DatabaseConfig holds PostgreSQL database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// GetDSN returns database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// TelegramConfig holds Telegram MTProto configuration
type TelegramConfig struct {
	APIID               int
	APIHash             string
	AccountSyncInterval time.Duration // Interval for checking new accounts in DB
	MinRequiredAccounts int           // Minimum number of accounts required to start service
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers           []string
	GroupID           string
	TopicNewsReceived string // Topic for publishing collected news

	// Saga: Subscription flow (consumed)
	TopicSubscriptionPending string // subscription.pending -> account-service
	// Saga: Subscription flow (produced)
	TopicSubscriptionActivated string // account-service -> subscription-service
	TopicSubscriptionFailed    string // account-service -> subscription-service

	// Saga: Unsubscription flow (consumed)
	TopicUnsubscriptionPending string // unsubscription.pending -> account-service
	// Saga: Unsubscription flow (produced)
	TopicUnsubscriptionCompleted string // account-service -> subscription-service
	TopicUnsubscriptionFailed    string // account-service -> subscription-service
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	Name            string
	Port            string
	ShutdownTimeout time.Duration // Graceful shutdown timeout
}

// NewsConfig holds news collection configuration
type NewsConfig struct {
	PollInterval         time.Duration // Interval between news collection cycles (when real-time updates are disabled)
	FallbackPollInterval time.Duration // Fallback interval when real-time updates ARE working (longer, for catching missed updates)
	CollectionTimeout    time.Duration // Timeout for single collection cycle
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	apiID, err := strconv.Atoi(getEnv("TELEGRAM_API_ID", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_API_ID: %w", err)
	}

	pollInterval, err := time.ParseDuration(getEnv("NEWS_POLL_INTERVAL", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid NEWS_POLL_INTERVAL: %w", err)
	}

	fallbackPollInterval, err := time.ParseDuration(getEnv("NEWS_FALLBACK_POLL_INTERVAL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid NEWS_FALLBACK_POLL_INTERVAL: %w", err)
	}

	collectionTimeout, err := time.ParseDuration(getEnv("NEWS_COLLECTION_TIMEOUT", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid NEWS_COLLECTION_TIMEOUT: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(getEnv("SERVICE_SHUTDOWN_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVICE_SHUTDOWN_TIMEOUT: %w", err)
	}

	accountSyncInterval, err := time.ParseDuration(getEnv("TELEGRAM_ACCOUNT_SYNC_INTERVAL", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_ACCOUNT_SYNC_INTERVAL: %w", err)
	}

	minRequiredAccounts := 0
	if minStr := getEnv("TELEGRAM_MIN_REQUIRED_ACCOUNTS", ""); minStr != "" {
		minReq, err := strconv.Atoi(minStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TELEGRAM_MIN_REQUIRED_ACCOUNTS: %w", err)
		}
		minRequiredAccounts = minReq
	}

	cfg := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DATABASE_HOST", "localhost"),
			Port:     getEnv("DATABASE_PORT", "5434"),
			User:     getEnv("DATABASE_USER", "accounts_user"),
			Password: getEnv("DATABASE_PASSWORD", "accounts_pass"),
			DBName:   getEnv("DATABASE_NAME", "accounts_db"),
			SSLMode:  getEnv("DATABASE_SSLMODE", "disable"),
		},
		Telegram: TelegramConfig{
			APIID:               apiID,
			APIHash:             getEnv("TELEGRAM_API_HASH", ""),
			AccountSyncInterval: accountSyncInterval,
			MinRequiredAccounts: minRequiredAccounts,
		},
		Kafka: KafkaConfig{
			Brokers:           strings.Split(getEnv("KAFKA_BROKERS", "localhost:9093"), ","),
			GroupID:           getEnv("KAFKA_GROUP_ID", "account-service-group"),
			TopicNewsReceived: getEnv("KAFKA_TOPIC_NEWS_RECEIVED", "news.received"),
			// Saga: Subscription flow
			TopicSubscriptionPending:   getEnv("KAFKA_TOPIC_SUBSCRIPTION_PENDING", "subscription.pending"),
			TopicSubscriptionActivated: getEnv("KAFKA_TOPIC_SUBSCRIPTION_ACTIVATED", "subscription.activated"),
			TopicSubscriptionFailed:    getEnv("KAFKA_TOPIC_SUBSCRIPTION_FAILED", "subscription.failed"),
			// Saga: Unsubscription flow
			TopicUnsubscriptionPending:   getEnv("KAFKA_TOPIC_UNSUBSCRIPTION_PENDING", "unsubscription.pending"),
			TopicUnsubscriptionCompleted: getEnv("KAFKA_TOPIC_UNSUBSCRIPTION_COMPLETED", "unsubscription.completed"),
			TopicUnsubscriptionFailed:    getEnv("KAFKA_TOPIC_UNSUBSCRIPTION_FAILED", "unsubscription.failed"),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Service: ServiceConfig{
			Name:            getEnv("SERVICE_NAME", "account-service"),
			Port:            getEnv("SERVICE_PORT", "8084"),
			ShutdownTimeout: shutdownTimeout,
		},
		News: NewsConfig{
			PollInterval:         pollInterval,
			FallbackPollInterval: fallbackPollInterval,
			CollectionTimeout:    collectionTimeout,
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Telegram validation
	if c.Telegram.APIID == 0 {
		return fmt.Errorf("TELEGRAM_API_ID is required")
	}

	if c.Telegram.APIHash == "" {
		return fmt.Errorf("TELEGRAM_API_HASH is required")
	}

	if c.Telegram.MinRequiredAccounts < 0 {
		return fmt.Errorf("TELEGRAM_MIN_REQUIRED_ACCOUNTS cannot be negative")
	}

	if c.Telegram.AccountSyncInterval < 10*time.Second {
		return fmt.Errorf("TELEGRAM_ACCOUNT_SYNC_INTERVAL must be at least 10 seconds, got %v", c.Telegram.AccountSyncInterval)
	}

	// Kafka validation
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}

	if c.Kafka.GroupID == "" {
		return fmt.Errorf("KAFKA_GROUP_ID is required")
	}

	if c.Kafka.TopicNewsReceived == "" {
		return fmt.Errorf("KAFKA_TOPIC_NEWS_RECEIVED is required")
	}

	// News collection validation
	if c.News.PollInterval < 1*time.Second {
		return fmt.Errorf("NEWS_POLL_INTERVAL must be at least 1 second, got %v", c.News.PollInterval)
	}

	if c.News.CollectionTimeout < 1*time.Second {
		return fmt.Errorf("NEWS_COLLECTION_TIMEOUT must be at least 1 second, got %v", c.News.CollectionTimeout)
	}

	// Service validation
	if c.Service.ShutdownTimeout < 1*time.Second {
		return fmt.Errorf("SERVICE_SHUTDOWN_TIMEOUT must be at least 1 second, got %v", c.Service.ShutdownTimeout)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// ConfigOut provides configuration components for fx DI
type ConfigOut struct {
	fx.Out

	Config   *Config
	Database *DatabaseConfig
	Telegram *TelegramConfig
	Kafka    *KafkaConfig
	Logging  *LoggingConfig
	Service  *ServiceConfig
	News     *NewsConfig
}

// Out loads and provides configuration for fx
func Out() (ConfigOut, error) {
	cfg, err := Load()
	if err != nil {
		return ConfigOut{}, err
	}

	return ConfigOut{
		Config:   cfg,
		Database: &cfg.Database,
		Telegram: &cfg.Telegram,
		Kafka:    &cfg.Kafka,
		Logging:  &cfg.Logging,
		Service:  &cfg.Service,
		News:     &cfg.News,
	}, nil
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the account service
type Config struct {
	Telegram TelegramConfig
	Kafka    KafkaConfig
	Logging  LoggingConfig
	Service  ServiceConfig
	News     NewsConfig
}

// TelegramConfig holds Telegram MTProto configuration
type TelegramConfig struct {
	APIID              int
	APIHash            string
	SessionDir         string
	Accounts           []string // List of phone numbers or session identifiers
	MinRequiredAccounts int      // Minimum number of accounts required to start service
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers                   []string
	GroupID                   string
	TopicNewsReceived         string // Topic for publishing collected news
	TopicSubscriptionsCreated string // Topic for subscription created events
	TopicSubscriptionsDeleted string // Topic for subscription deleted events
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
	PollInterval      time.Duration // Interval between news collection cycles
	CollectionTimeout time.Duration // Timeout for single collection cycle
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

	collectionTimeout, err := time.ParseDuration(getEnv("NEWS_COLLECTION_TIMEOUT", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid NEWS_COLLECTION_TIMEOUT: %w", err)
	}

	shutdownTimeout, err := time.ParseDuration(getEnv("SERVICE_SHUTDOWN_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVICE_SHUTDOWN_TIMEOUT: %w", err)
	}

	accounts := []string{}
	accountsStr := getEnv("TELEGRAM_ACCOUNTS", "")
	if accountsStr != "" {
		accounts = strings.Split(accountsStr, ",")
	}

	minRequiredAccounts := 0
	if len(accounts) > 0 {
		// Default: require at least 50% of configured accounts
		minRequiredAccounts = (len(accounts) + 1) / 2
	}
	if minStr := getEnv("TELEGRAM_MIN_REQUIRED_ACCOUNTS", ""); minStr != "" {
		minReq, err := strconv.Atoi(minStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TELEGRAM_MIN_REQUIRED_ACCOUNTS: %w", err)
		}
		minRequiredAccounts = minReq
	}

	cfg := &Config{
		Telegram: TelegramConfig{
			APIID:              apiID,
			APIHash:            getEnv("TELEGRAM_API_HASH", ""),
			SessionDir:         getEnv("TELEGRAM_SESSION_DIR", "./sessions"),
			Accounts:           accounts,
			MinRequiredAccounts: minRequiredAccounts,
		},
		Kafka: KafkaConfig{
			Brokers:                   strings.Split(getEnv("KAFKA_BROKERS", "localhost:9093"), ","),
			GroupID:                   getEnv("KAFKA_GROUP_ID", "account-service-group"),
			TopicNewsReceived:         getEnv("KAFKA_TOPIC_NEWS_RECEIVED", "news.received"),
			TopicSubscriptionsCreated: getEnv("KAFKA_TOPIC_SUBSCRIPTIONS_CREATED", "subscriptions.created"),
			TopicSubscriptionsDeleted: getEnv("KAFKA_TOPIC_SUBSCRIPTIONS_DELETED", "subscriptions.deleted"),
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
			PollInterval:      pollInterval,
			CollectionTimeout: collectionTimeout,
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

	if c.Telegram.MinRequiredAccounts > len(c.Telegram.Accounts) {
		return fmt.Errorf("TELEGRAM_MIN_REQUIRED_ACCOUNTS (%d) cannot exceed total accounts (%d)",
			c.Telegram.MinRequiredAccounts, len(c.Telegram.Accounts))
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

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

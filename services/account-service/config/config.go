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
	APIID      int
	APIHash    string
	SessionDir string
	Accounts   []string // List of phone numbers or session identifiers
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers []string
	GroupID string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	Name string
	Port string
}

// NewsConfig holds news collection configuration
type NewsConfig struct {
	PollInterval time.Duration
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

	accounts := []string{}
	accountsStr := getEnv("TELEGRAM_ACCOUNTS", "")
	if accountsStr != "" {
		accounts = strings.Split(accountsStr, ",")
	}

	cfg := &Config{
		Telegram: TelegramConfig{
			APIID:      apiID,
			APIHash:    getEnv("TELEGRAM_API_HASH", ""),
			SessionDir: getEnv("TELEGRAM_SESSION_DIR", "./sessions"),
			Accounts:   accounts,
		},
		Kafka: KafkaConfig{
			Brokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9093"), ","),
			GroupID: getEnv("KAFKA_GROUP_ID", "account-service-group"),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Service: ServiceConfig{
			Name: getEnv("SERVICE_NAME", "account-service"),
			Port: getEnv("SERVICE_PORT", "8084"),
		},
		News: NewsConfig{
			PollInterval: pollInterval,
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Telegram.APIID == 0 {
		return fmt.Errorf("TELEGRAM_API_ID is required")
	}

	if c.Telegram.APIHash == "" {
		return fmt.Errorf("TELEGRAM_API_HASH is required")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
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

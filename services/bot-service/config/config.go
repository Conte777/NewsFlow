package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the bot service
type Config struct {
	Telegram TelegramConfig
	Kafka    KafkaConfig
	Logging  LoggingConfig
	Service  ServiceConfig
}

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	BotToken string
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

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	cfg := &Config{
		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		},
		Kafka: KafkaConfig{
			Brokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9093"), ","),
			GroupID: getEnv("KAFKA_GROUP_ID", "bot-service-group"),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Service: ServiceConfig{
			Name: getEnv("SERVICE_NAME", "bot-service"),
			Port: getEnv("SERVICE_PORT", "8081"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
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

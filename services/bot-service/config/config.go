package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

// Config holds all configuration for the bot service
type Config struct {
	Telegram TelegramConfig
	Kafka    KafkaConfig
	GRPC     GRPCConfig
	Logging  LoggingConfig
	Service  ServiceConfig
}

// GRPCConfig holds gRPC client configuration
type GRPCConfig struct {
	SubscriptionServiceAddr string
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

// Result provides config parts for fx dependency injection using fx.Out pattern
type Result struct {
	fx.Out

	Config   *Config
	Telegram *TelegramConfig
	Kafka    *KafkaConfig
	GRPC     *GRPCConfig
	Logging  *LoggingConfig
	Service  *ServiceConfig
}

// Out loads configuration and returns Result for fx injection
func Out() (Result, error) {
	cfg, err := Load()
	if err != nil {
		return Result{}, err
	}

	return Result{
		Config:   cfg,
		Telegram: &cfg.Telegram,
		Kafka:    &cfg.Kafka,
		GRPC:     &cfg.GRPC,
		Logging:  &cfg.Logging,
		Service:  &cfg.Service,
	}, nil
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
		GRPC: GRPCConfig{
			SubscriptionServiceAddr: getEnv("SUBSCRIPTION_SERVICE_GRPC_ADDR", "localhost:50051"),
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

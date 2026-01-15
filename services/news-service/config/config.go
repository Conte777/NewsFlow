package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

// Config holds all configuration for the news service
type Config struct {
	Database            DatabaseConfig
	Kafka               KafkaConfig
	Logging             LoggingConfig
	Service             ServiceConfig
	SubscriptionService SubscriptionServiceConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
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

// SubscriptionServiceConfig holds subscription service client configuration
type SubscriptionServiceConfig struct {
	URL     string
	Timeout time.Duration
}

// Result is fx.Out struct for providing config dependencies
type Result struct {
	fx.Out

	Config                    *Config
	DatabaseConfig            *DatabaseConfig
	KafkaConfig               *KafkaConfig
	LoggingConfig             *LoggingConfig
	ServiceConfig             *ServiceConfig
	SubscriptionServiceConfig *SubscriptionServiceConfig
}

// Out returns fx-compatible config result
func Out() (Result, error) {
	cfg, err := Load()
	if err != nil {
		return Result{}, err
	}

	return Result{
		Config:                    cfg,
		DatabaseConfig:            &cfg.Database,
		KafkaConfig:               &cfg.Kafka,
		LoggingConfig:             &cfg.Logging,
		ServiceConfig:             &cfg.Service,
		SubscriptionServiceConfig: &cfg.SubscriptionService,
	}, nil
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DATABASE_HOST", "localhost"),
			Port:     getEnv("DATABASE_PORT", "5433"),
			User:     getEnv("DATABASE_USER", "news_user"),
			Password: getEnv("DATABASE_PASSWORD", "news_pass"),
			DBName:   getEnv("DATABASE_NAME", "news_db"),
			SSLMode:  getEnv("DATABASE_SSLMODE", "disable"),
		},
		Kafka: KafkaConfig{
			Brokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9093"), ","),
			GroupID: getEnv("KAFKA_GROUP_ID", "news-service-group"),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Service: ServiceConfig{
			Name: getEnv("SERVICE_NAME", "news-service"),
			Port: getEnv("SERVICE_PORT", "8083"),
		},
		SubscriptionService: SubscriptionServiceConfig{
			URL:     getEnv("SUBSCRIPTION_SERVICE_URL", "http://subscription-service:8082"),
			Timeout: getEnvDuration("SUBSCRIPTION_SERVICE_TIMEOUT", 30*time.Second),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DATABASE_HOST is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("DATABASE_USER is required")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("DATABASE_NAME is required")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}

	return nil
}

// GetDSN returns database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvDuration gets environment variable as duration with default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

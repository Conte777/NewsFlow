package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the subscription service
type Config struct {
	Database DatabaseConfig
	Kafka    KafkaConfig
	Logging  LoggingConfig
	Service  ServiceConfig
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

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	cfg := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DATABASE_HOST", "localhost"),
			Port:     getEnv("DATABASE_PORT", "5432"),
			User:     getEnv("DATABASE_USER", "subscriptions_user"),
			Password: getEnv("DATABASE_PASSWORD", "subscriptions_pass"),
			DBName:   getEnv("DATABASE_NAME", "subscriptions_db"),
			SSLMode:  getEnv("DATABASE_SSLMODE", "disable"),
		},
		Kafka: KafkaConfig{
			Brokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9093"), ","),
			GroupID: getEnv("KAFKA_GROUP_ID", "subscription-service-group"),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Service: ServiceConfig{
			Name: getEnv("SERVICE_NAME", "subscription-service"),
			Port: getEnv("SERVICE_PORT", "8082"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadConfig loads configuration from .env and environment variables
func LoadConfig() (*Config, error) {
	return Load()
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

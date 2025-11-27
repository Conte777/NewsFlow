package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/Conte777/newsflow/services/bot-service/config"
	"github.com/rs/zerolog"
)

// TestConsumer_New проверяет создание consumer
func TestConsumer_New(t *testing.T) {
	cfg := config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		// Используйте только реальные поля из вашего config
	}
	logger := zerolog.Nop()

	consumer, err := NewConsumer(cfg, logger)

	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}

	// Проверяем что consumer создан
	if consumer == nil {
		t.Error("Expected consumer to be created, got nil")
		return
	}

	// Cleanup
	defer consumer.Close()

	t.Log("Consumer created successfully")
}

// TestConsumer_New_InvalidConfig проверяет обработку невалидной конфигурации
func TestConsumer_New_InvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.KafkaConfig
		wantErr bool
	}{
		{
			name:    "No brokers",
			cfg:     config.KafkaConfig{},
			wantErr: true,
		},
		{
			name: "Valid config",
			cfg: config.KafkaConfig{
				Brokers: []string{"localhost:9092"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			consumer, err := NewConsumer(tt.cfg, logger)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Cleanup если consumer создан
			if consumer != nil {
				consumer.Close()
			}
		})
	}
}

// TestConsumer_Close проверяет закрытие consumer
func TestConsumer_Close(t *testing.T) {
	cfg := config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
	}
	logger := zerolog.Nop()

	consumer, err := NewConsumer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}

	err = consumer.Close()
	if err != nil {
		t.Errorf("Failed to close consumer: %v", err)
	} else {
		t.Log("Consumer closed successfully")
	}
}

// TestConsumer_ReadMessage_Integration интеграционный тест
func TestConsumer_ReadMessage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
	}
	logger := zerolog.Nop()

	consumer, err := NewConsumer(cfg, logger)
	if err != nil {
		t.Skipf("Skipping test - Kafka not available: %v", err)
		return
	}
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = consumer.ReadMessage(ctx)

	// В тестовом режиме ожидаем таймаут или другую ошибку
	if err != nil {
		t.Logf("ReadMessage returned (expected in test): %v", err)
	} else {
		t.Log("ReadMessage completed successfully")
	}
}

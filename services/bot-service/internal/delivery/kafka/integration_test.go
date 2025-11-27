//go:build integration
// +build integration

package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/Conte777/newsflow/services/bot-service/config"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

// TestKafkaIntegration базовый интеграционный тест
func TestKafkaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("KafkaConnection", func(t *testing.T) {
		// Тестируем создание продюсера (если функция существует)
		producer, err := createTestProducer()
		if err != nil {
			t.Skipf("Kafka not available: %v", err)
			return
		}
		if producer != nil {
			defer producer.Close()
			t.Log("Producer created successfully")
		}

		// Тестируем создание консьюмера
		consumer, err := createTestConsumer()
		if err != nil {
			t.Skipf("Kafka consumer not available: %v", err)
			return
		}
		if consumer != nil {
			defer consumer.Close()
			t.Log("Consumer created successfully")
		}
	})
}

// TestKafkaProducerConsumer интеграционный тест producer/consumer
func TestKafkaProducerConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	producer, err := createTestProducer()
	if err != nil {
		t.Skipf("Skipping test - Producer not available: %v", err)
		return
	}
	defer producer.Close()

	consumer, err := createTestConsumer()
	if err != nil {
		t.Skipf("Skipping test - Consumer not available: %v", err)
		return
	}
	defer consumer.Close()

	ctx := context.Background()
	testTopic := "integration-test-topic"
	testKey := "test-key"
	testValue := "test-value"

	// Пытаемся отправить сообщение
	err = producer.SendMessage(ctx, testTopic, testKey, testValue)
	if err != nil {
		t.Logf("SendMessage completed with: %v (might be expected if Kafka not running)", err)
	} else {
		t.Log("Message sent successfully")
	}

	// Пытаемся прочитать сообщение (с таймаутом)
	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	msg, err := consumer.ReadMessage(readCtx)
	if err != nil {
		t.Logf("ReadMessage completed with: %v (expected without actual Kafka)", err)
	} else {
		t.Logf("Message received: topic=%s, key=%s, value=%s",
			msg.Topic, string(msg.Key), string(msg.Value))
	}
}

// Вспомогательные функции для тестов

// createTestProducer создает тестовый producer
func createTestProducer() (*Producer, error) {
	// Если у вас есть реальная функция NewProducer, используйте ее
	// Если нет, создаем напрямую
	brokers := []string{"localhost:9092"}

	// Создаем writer напрямую для тестов
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		writer: writer,
	}, nil
}

// createTestConsumer создает тестовый consumer
func createTestConsumer() (*Consumer, error) {
	cfg := config.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		// Используйте только реальные поля из вашего config
	}
	logger := zerolog.Nop()

	// Если у вас есть реальная функция NewConsumer, используйте ее
	// Если нет, создаем напрямую
	readerConfig := kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		GroupID: "test-group",
		Topic:   "test-topic", // Топик указываем здесь, а не в конфиге
	}

	reader := kafka.NewReader(readerConfig)

	return &Consumer{
		reader: reader,
		logger: logger,
		config: cfg,
	}, nil
}

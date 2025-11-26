package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

// IntegrationTestSuite содержит общие настройки для интеграционных тестов
type IntegrationTestSuite struct {
	kafkaContainer *kafka.KafkaContainer
	brokers        []string
}

// SetupSuite настраивает Kafka контейнер перед запуском тестов
func (suite *IntegrationTestSuite) SetupSuite(t *testing.T) {
	ctx := context.Background()

	// Запускаем Kafka контейнер
	container, err := kafka.Run(ctx, "confluentinc/cp-kafka:7.3.2")
	assert.NoError(t, err)

	suite.kafkaContainer = container
	suite.brokers = []string{container.GetBootstrapServers(ctx)}
}

// TearDownSuite очищает ресурсы после тестов
func (suite *IntegrationTestSuite) TearDownSuite(t *testing.T) {
	ctx := context.Background()
	if suite.kafkaContainer != nil {
		assert.NoError(t, suite.kafkaContainer.Terminate(ctx))
	}
}

func TestKafkaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропуск интеграционных тестов в short mode")
	}

	suite := &IntegrationTestSuite{}
	suite.SetupSuite(t)
	defer suite.TearDownSuite(t)

	t.Run("Интеграционный тест Producer и Consumer", func(t *testing.T) {
		ctx := context.Background()
		topic := "integration-test-topic"
		groupID := "integration-test-group"

		// Создаем Producer
		producer := NewProducer(suite.brokers)
		defer producer.Close()

		// Создаем Consumer
		consumer := NewConsumer(suite.brokers, topic, groupID)
		defer consumer.Close()

		// Тестовые данные
		testKey := "integration-key"
		testValue := "integration-value"

		// Отправляем сообщение
		err := producer.SendMessage(ctx, topic, testKey, testValue)
		assert.NoError(t, err)

		// Даем время для обработки сообщения
		time.Sleep(3 * time.Second)

		// Читаем сообщение
		msg, err := consumer.ReadMessage(ctx)
		assert.NoError(t, err)
		assert.Equal(t, testKey, string(msg.Key))
		assert.Equal(t, testValue, string(msg.Value))
	})

	t.Run("Интеграционный тест множественных сообщений", func(t *testing.T) {
		ctx := context.Background()
		topic := "integration-batch-test-topic"
		groupID := "integration-batch-test-group"

		producer := NewProducer(suite.brokers)
		defer producer.Close()

		consumer := NewConsumer(suite.brokers, topic, groupID)
		defer consumer.Close()

		// Отправляем несколько сообщений
		messages := []struct {
			key   string
			value string
		}{
			{"key1", "value1"},
			{"key2", "value2"},
			{"key3", "value3"},
		}

		for _, msg := range messages {
			err := producer.SendMessage(ctx, topic, msg.key, msg.value)
			assert.NoError(t, err)
		}

		// Даем время для обработки сообщений
		time.Sleep(5 * time.Second)

		// Читаем и проверяем сообщения
		for i := 0; i < len(messages); i++ {
			msg, err := consumer.ReadMessage(ctx)
			assert.NoError(t, err)

			expected := messages[i]
			assert.Equal(t, expected.key, string(msg.Key))
			assert.Equal(t, expected.value, string(msg.Value))
		}
	})
}

func TestKafkaTopicCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропуск интеграционных тестов в short mode")
	}

	suite := &IntegrationTestSuite{}
	suite.SetupSuite(t)
	defer suite.TearDownSuite(t)

	t.Run("Тест создания топика и отправки сообщений", func(t *testing.T) {
		ctx := context.Background()
		topic := "new-topic-test"

		// Создаем соединение с Kafka для администрирования
		conn, err := kafka.Dial("tcp", suite.brokers[0])
		assert.NoError(t, err)
		defer conn.Close()

		// Создаем топик
		topicConfigs := []kafka.TopicConfig{
			{
				Topic:             topic,
				NumPartitions:     1,
				ReplicationFactor: 1,
			},
		}

		err = conn.CreateTopics(topicConfigs...)
		assert.NoError(t, err)

		// Тестируем отправку и чтение сообщений в новом топике
		producer := NewProducer(suite.brokers)
		defer producer.Close()

		consumer := NewConsumer(suite.brokers, topic, "test-group")
		defer consumer.Close()

		testMessage := "test message for new topic"
		err = producer.SendMessage(ctx, topic, "test-key", testMessage)
		assert.NoError(t, err)

		time.Sleep(2 * time.Second)

		msg, err := consumer.ReadMessage(ctx)
		assert.NoError(t, err)
		assert.Equal(t, testMessage, string(msg.Value))
	})
}

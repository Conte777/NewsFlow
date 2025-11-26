package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockKafkaReader для тестирования Consumer
type MockKafkaReader struct {
	mock.Mock
}

func (m *MockKafkaReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).(kafka.Message), args.Error(1)
}

func (m *MockKafkaReader) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockKafkaReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).(kafka.Message), args.Error(1)
}

func (m *MockKafkaReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func TestConsumer_ReadMessage(t *testing.T) {
	tests := []struct {
		name          string
		expectedMsg   kafka.Message
		expectedError error
		setupMock     func(*MockKafkaReader)
	}{
		{
			name: "Успешное чтение сообщения",
			expectedMsg: kafka.Message{
				Topic: "test-topic",
				Key:   []byte("test-key"),
				Value: []byte("test-value"),
			},
			setupMock: func(mockReader *MockKafkaReader) {
				mockReader.On("ReadMessage", mock.Anything).
					Return(kafka.Message{
						Topic: "test-topic",
						Key:   []byte("test-key"),
						Value: []byte("test-value"),
					}, nil)
			},
		},
		{
			name:          "Ошибка при чтении сообщения",
			expectedError: context.DeadlineExceeded,
			setupMock: func(mockReader *MockKafkaReader) {
				mockReader.On("ReadMessage", mock.Anything).
					Return(kafka.Message{}, context.DeadlineExceeded)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем mock reader
			mockReader := new(MockKafkaReader)
			tt.setupMock(mockReader)

			// Создаем consumer с mock reader
			consumer := &Consumer{
				reader: mockReader,
			}

			// Вызываем тестируемый метод
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			msg, err := consumer.ReadMessage(ctx)

			// Проверяем результаты
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMsg.Topic, msg.Topic)
				assert.Equal(t, tt.expectedMsg.Key, msg.Key)
				assert.Equal(t, tt.expectedMsg.Value, msg.Value)
			}

			// Проверяем, что mock методы были вызваны
			mockReader.AssertCalled(t, "ReadMessage", mock.Anything)
		})
	}
}

func TestConsumer_Close(t *testing.T) {
	mockReader := new(MockKafkaReader)
	mockReader.On("Close").Return(nil)

	consumer := &Consumer{
		reader: mockReader,
	}

	err := consumer.Close()

	assert.NoError(t, err)
	mockReader.AssertCalled(t, "Close")
}

func TestNewConsumer(t *testing.T) {
	brokers := []string{"localhost:9092"}
	topic := "test-topic"
	groupID := "test-group"

	consumer := NewConsumer(brokers, topic, groupID)

	assert.NotNil(t, consumer)
	assert.NotNil(t, consumer.reader)
}

func TestConsumer_ConsumeMessages(t *testing.T) {
	mockReader := new(MockKafkaReader)

	// Настраиваем mock для возврата нескольких сообщений
	testMessages := []kafka.Message{
		{
			Topic: "test-topic",
			Key:   []byte("key1"),
			Value: []byte("value1"),
		},
		{
			Topic: "test-topic",
			Key:   []byte("key2"),
			Value: []byte("value2"),
		},
	}

	callCount := 0
	mockReader.On("ReadMessage", mock.Anything).Return(func(ctx context.Context) kafka.Message {
		if callCount < len(testMessages) {
			msg := testMessages[callCount]
			callCount++
			return msg
		}
		// После всех сообщений возвращаем ошибку для завершения теста
		return kafka.Message{}
	}, func(ctx context.Context) error {
		if callCount < len(testMessages) {
			return nil
		}
		return context.Canceled
	})

	consumer := &Consumer{
		reader: mockReader,
	}

	// Тестируем потребление сообщений
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	messages := make(chan kafka.Message, 2)
	errors := make(chan error, 1)

	go consumer.ConsumeMessages(ctx, messages, errors)

	// Читаем сообщения из канала
	receivedMessages := []kafka.Message{}
	for i := 0; i < 2; i++ {
		select {
		case msg := <-messages:
			receivedMessages = append(receivedMessages, msg)
		case <-ctx.Done():
			break
		}
	}

	assert.Len(t, receivedMessages, 2)
	assert.Equal(t, testMessages[0].Value, receivedMessages[0].Value)
	assert.Equal(t, testMessages[1].Value, receivedMessages[1].Value)
}

package deps

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/news/entities"
)

// KafkaProducer defines interface for sending messages to Kafka
type KafkaProducer interface {
	SendNewsReceived(ctx context.Context, news *entities.NewsItem) error
	Close() error
	IsHealthy() bool
}

// KafkaConsumer defines interface for receiving messages from Kafka
type KafkaConsumer interface {
	Close() error
	IsHealthy() bool
}

package kafka

import (
	"context"
	"strconv"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/events"
)

type ProducerAdapter struct {
	producer *KafkaProducer
}

func NewProducerAdapter(producer *KafkaProducer) *ProducerAdapter {
	return &ProducerAdapter{producer: producer}
}

func (a *ProducerAdapter) NotifyAccountService(
	ctx context.Context,
	eventType string,
	subscription *domain.Subscription,
) error {

	event := &events.SubscriptionEvent{
		Type:        eventType,
		UserID:      strconv.FormatInt(subscription.UserID, 10),
		ChannelID:   subscription.ChannelID,
		ChannelName: subscription.ChannelName,
	}

	return a.producer.NotifyAccountService(ctx, event)
}

// ✅ ВАЖНО: реализуем Close
func (a *ProducerAdapter) Close() error {
	return a.producer.Close()
}

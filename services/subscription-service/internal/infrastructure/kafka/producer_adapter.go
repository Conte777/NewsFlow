package kafka

import (
	"context"
	"strconv"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/dto"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/entities"
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
	subscription *entities.Subscription,
) error {
	event := &dto.SubscriptionEvent{
		Type:        eventType,
		UserID:      strconv.FormatInt(subscription.UserID, 10),
		ChannelID:   subscription.ChannelID,
		ChannelName: subscription.ChannelName,
	}

	return a.producer.NotifyAccountService(ctx, event)
}

func (a *ProducerAdapter) Close() error {
	return a.producer.Close()
}

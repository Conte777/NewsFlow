package kafka

import (
	"context"
	"strconv"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/consts"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/dto"
)

type ProducerAdapter struct {
	producer *KafkaProducer
}

func NewProducerAdapter(producer *KafkaProducer) *ProducerAdapter {
	return &ProducerAdapter{producer: producer}
}

// Saga: Subscription flow

// SendSubscriptionPending sends subscription.pending event to account-service
func (a *ProducerAdapter) SendSubscriptionPending(
	ctx context.Context,
	subscriptionID uint,
	telegramUserID int64,
	channelID, channelName string,
) error {
	event := dto.NewPendingEvent(
		dto.EventTypeSubscriptionPending,
		subscriptionID,
		strconv.FormatInt(telegramUserID, 10),
		channelID,
		channelName,
	)

	return a.producer.SendToTopic(ctx, consts.TopicSubscriptionPending, event.UserID, event)
}

// SendSubscriptionRejected sends subscription.rejected event to bot-service
func (a *ProducerAdapter) SendSubscriptionRejected(
	ctx context.Context,
	telegramUserID int64,
	channelID, channelName, reason string,
) error {
	event := dto.NewRejectedEvent(
		dto.EventTypeSubscriptionRejected,
		strconv.FormatInt(telegramUserID, 10),
		channelID,
		channelName,
		reason,
	)

	return a.producer.SendToTopic(ctx, consts.TopicSubscriptionRejected, event.UserID, event)
}

// SendSubscriptionConfirmed sends subscription.confirmed event to bot-service
func (a *ProducerAdapter) SendSubscriptionConfirmed(
	ctx context.Context,
	telegramUserID int64,
	channelID, channelName string,
) error {
	event := dto.NewConfirmedEvent(
		dto.EventTypeSubscriptionConfirmed,
		strconv.FormatInt(telegramUserID, 10),
		channelID,
		channelName,
	)

	return a.producer.SendToTopic(ctx, consts.TopicSubscriptionConfirmed, event.UserID, event)
}

// Saga: Unsubscription flow

// SendUnsubscriptionPending sends unsubscription.pending event to account-service
func (a *ProducerAdapter) SendUnsubscriptionPending(
	ctx context.Context,
	telegramUserID int64,
	channelID string,
) error {
	event := dto.NewPendingEvent(
		dto.EventTypeUnsubscriptionPending,
		0,
		strconv.FormatInt(telegramUserID, 10),
		channelID,
		"",
	)

	return a.producer.SendToTopic(ctx, consts.TopicUnsubscriptionPending, event.UserID, event)
}

// SendUnsubscriptionRejected sends unsubscription.rejected event to bot-service
func (a *ProducerAdapter) SendUnsubscriptionRejected(
	ctx context.Context,
	telegramUserID int64,
	channelID, channelName, reason string,
) error {
	event := dto.NewRejectedEvent(
		dto.EventTypeUnsubscriptionRejected,
		strconv.FormatInt(telegramUserID, 10),
		channelID,
		channelName,
		reason,
	)

	return a.producer.SendToTopic(ctx, consts.TopicUnsubscriptionRejected, event.UserID, event)
}

// SendUnsubscriptionConfirmed sends unsubscription.confirmed event to bot-service
func (a *ProducerAdapter) SendUnsubscriptionConfirmed(
	ctx context.Context,
	telegramUserID int64,
	channelID, channelName string,
) error {
	event := dto.NewConfirmedEvent(
		dto.EventTypeUnsubscriptionConfirmed,
		strconv.FormatInt(telegramUserID, 10),
		channelID,
		channelName,
	)

	return a.producer.SendToTopic(ctx, consts.TopicUnsubscriptionConfirmed, event.UserID, event)
}

func (a *ProducerAdapter) Close() error {
	return a.producer.Close()
}

package kafka

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/events"

	"github.com/rs/zerolog"
)

type SubscriptionEventHandler struct {
	usecase domain.SubscriptionUseCase
	logger  zerolog.Logger

	processed uint64
	errors    uint64
}

func NewSubscriptionEventHandler(
	usecase domain.SubscriptionUseCase,
	logger zerolog.Logger,
) *SubscriptionEventHandler {
	return &SubscriptionEventHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *SubscriptionEventHandler) Handle(event *events.SubscriptionEvent) error {
	ctx := context.Background()

	switch event.Type {
	case "subscription_created":
		return h.HandleSubscriptionCreated(ctx, event)

	case "subscription_cancelled":
		return h.HandleSubscriptionDeleted(ctx, event)

	default:
		h.logger.Warn().
			Str("event_type", event.Type).
			Msg("received unknown subscription event type")
		return nil // идемпотентно
	}
}

func (h *SubscriptionEventHandler) HandleSubscriptionCreated(
	ctx context.Context,
	event *events.SubscriptionEvent,
) error {

	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		duration := time.Since(start)

		h.logger.Info().
			Dur("duration", duration).
			Uint64("processed_total", atomic.LoadUint64(&h.processed)).
			Uint64("errors_total", atomic.LoadUint64(&h.errors)).
			Msg("subscription created event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" || event.ChannelName == "" {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Str("channel_name", event.ChannelName).
			Msg("subscription created event missing required fields")
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Msg("invalid UserID, cannot convert to int64")
		return err
	}

	// --- Создаём подписку через usecase ---
	err = h.usecase.CreateSubscription(ctx, userID, event.ChannelID, event.ChannelName)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionAlreadyExists) {
			h.logger.Warn().
				Str("user_id", event.UserID).
				Str("channel_id", event.ChannelID).
				Msg("subscription already exists, skipping creation")
			return nil // идемпотентно: успех
		}
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("failed to create subscription")
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription successfully created")

	return nil
}

func (h *SubscriptionEventHandler) HandleSubscriptionDeleted(
	ctx context.Context,
	event *events.SubscriptionEvent,
) error {

	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		duration := time.Since(start)

		h.logger.Info().
			Dur("duration", duration).
			Uint64("processed_total", atomic.LoadUint64(&h.processed)).
			Uint64("errors_total", atomic.LoadUint64(&h.errors)).
			Msg("subscription deleted event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("subscription deleted event missing required fields")
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Msg("invalid UserID, cannot convert to int64")
		return err
	}

	// --- Удаляем подписку через usecase ---
	err = h.usecase.DeleteSubscription(ctx, userID, event.ChannelID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			h.logger.Warn().
				Str("user_id", event.UserID).
				Str("channel_id", event.ChannelID).
				Msg("subscription does not exist, skipping deletion")
			return nil // идемпотентно: успех
		}
		h.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("failed to delete subscription")
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription successfully deleted")

	return nil
}
